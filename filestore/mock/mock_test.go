package mock_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

// readFile returns the contents and content type of filename.
// tryReadFile is readFile for goroutines: it reports failures through the
// returned error instead of stopping the calling goroutine.
func tryReadFile(client *mock.Client, filename string) (string, error) {
	rc, _, err := client.Get(context.Background(), filename)
	if err != nil {
		return "", fmt.Errorf("getting %q: %w", filename, err)
	}
	content, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil {
		return "", fmt.Errorf("reading %q: %w", filename, readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("closing %q: %w", filename, closeErr)
	}
	return string(content), nil
}

func readFile(t *testing.T, client *mock.Client, filename string) (string, string) {
	t.Helper()

	rc, contentType, err := client.Get(context.Background(), filename)
	if err != nil {
		t.Fatalf("getting %q: %v", filename, err)
	}
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading %q: %v", filename, err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("closing %q: %v", filename, err)
	}
	return string(content), contentType
}

func TestSeededObjects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		file            *mock.MemoryFile
		wantContentType string
	}{
		{name: "constructor", file: mock.NewMemoryFile([]byte("hello world"), "text/plain"), wantContentType: "text/plain"},
		{name: "zero value written by hand", file: writtenFile("hello world"), wantContentType: mock.DefaultContentType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			objects := map[string]*mock.MemoryFile{"greeting.txt": tt.file}
			client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test", Objects: objects}})

			data, contentType := readFile(t, client, "greeting.txt")
			if data != "hello world" || contentType != tt.wantContentType {
				t.Fatalf("Get = (%q, %q), want (%q, %q)", data, contentType, "hello world", tt.wantContentType)
			}
			// The seeded map stays the client's backing store.
			if err := client.Purge(context.Background(), "greeting.txt"); err != nil {
				t.Fatal(err)
			}
			if _, ok := objects["greeting.txt"]; ok {
				t.Fatal("Purge did not remove the object from the seeded map")
			}
		})
	}
}

func writtenFile(content string) *mock.MemoryFile {
	f := &mock.MemoryFile{}
	_, _ = f.WriteString(content)
	return f
}

// TestBasePath proves BasePath is applied as a forward-slash key prefix
// rather than ignored: two clients share the same underlying object map so
// the composed key can be observed from outside the scoped client.
func TestBasePath(t *testing.T) {
	t.Parallel()

	objects := map[string]*mock.MemoryFile{}
	scoped := mock.NewClient(mock.Config{BasePath: "tenant-a", Bucket: mock.Bucket{Objects: objects}})
	unscoped := mock.NewClient(mock.Config{Bucket: mock.Bucket{Objects: objects}})

	if err := scoped.Set(context.Background(), "greeting.txt", []byte("hello world"), "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}
	if got, _ := readFile(t, scoped, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q but got %q", "hello world", got)
	}
	if _, _, err := unscoped.Get(context.Background(), "greeting.txt"); !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("file set under a base path visible without it: error = %v", err)
	}
	if got, _ := readFile(t, unscoped, "tenant-a/greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q at the composed key but got %q", "hello world", got)
	}

	url, err := scoped.GetPresignedURL(context.Background(), "greeting.txt", time.Minute)
	if err != nil {
		t.Fatalf("presigning: %v", err)
	}
	if !strings.HasSuffix(url, "/tenant-a/greeting.txt") {
		t.Fatalf("presigned URL %q does not end with the composed key", url)
	}
}

// TestReadersAreIndependent covers the historical bug where Get handed out
// the stored buffer itself, so reading or closing the reader destroyed the
// file.
func TestReadersAreIndependent(t *testing.T) {
	t.Parallel()

	client := mock.NewClient(mock.Config{})
	if err := client.Set(context.Background(), "a.txt", []byte("abcdef"), "text/plain"); err != nil {
		t.Fatal(err)
	}

	first, _, err := client.Get(context.Background(), "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Read(make([]byte, 2)); err != nil {
		t.Fatalf("partial read: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Read(make([]byte, 1)); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("read after close: error = %v, want fs.ErrClosed", err)
	}

	if got, _ := readFile(t, client, "a.txt"); got != "abcdef" {
		t.Fatalf("stored content = %q after a partial read and close, want %q", got, "abcdef")
	}
	if err := client.Copy(context.Background(), "a.txt", "b.txt"); err != nil {
		t.Fatal(err)
	}
	if got, _ := readFile(t, client, "b.txt"); got != "abcdef" {
		t.Fatalf("copied content = %q, want %q", got, "abcdef")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})
	ctx := context.Background()
	const workers, rounds = 8, 50

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := fmt.Sprintf("w%d.txt", w)
			copyName := fmt.Sprintf("w%d-copy.txt", w)
			for range rounds {
				if err := client.Set(ctx, name, []byte(name), "text/plain"); err != nil {
					t.Error(err)
				}
				if got, err := tryReadFile(client, name); err != nil {
					t.Error(err)
				} else if got != name {
					t.Errorf("got %q, want %q", got, name)
				}
				if err := client.Copy(ctx, name, copyName); err != nil {
					t.Error(err)
				}
				if err := client.Move(ctx, copyName, name); !errors.Is(err, filestore.ErrFileExists) {
					t.Errorf("Move onto taken key: error = %v", err)
				}
				if _, err := client.GetPresignedURL(ctx, name, time.Minute); err != nil {
					t.Error(err)
				}
				for _, n := range []string{name, copyName} {
					if err := client.Purge(ctx, n); err != nil {
						t.Error(err)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func TestMemoryFileReadAfterClose(t *testing.T) {
	t.Parallel()

	file := mock.NewMemoryFile([]byte("hello world"), "")
	if err := file.Close(); err != nil {
		t.Fatalf("closing file: %v", err)
	}
	if _, err := file.Read(make([]byte, 1)); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("read after close: error = %v, want fs.ErrClosed", err)
	}
}
