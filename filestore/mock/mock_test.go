package mock_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	data, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatalf("reading and closing %q: %v", filename, err)
	}
	return string(data), contentType
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
			checkSeededObject(t, tt.file, tt.wantContentType)
		})
	}
}

func checkSeededObject(t *testing.T, file *mock.MemoryFile, wantContentType string) {
	t.Helper()
	objects := map[string]*mock.MemoryFile{"greeting.txt": file}
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test", Objects: objects}})
	data, contentType := readFile(t, client, "greeting.txt")
	if data != "hello world" || contentType != wantContentType {
		t.Fatalf("Get = (%q, %q), want (%q, %q)", data, contentType, "hello world", wantContentType)
	}
	if err := client.Purge(context.Background(), "greeting.txt"); err != nil {
		t.Fatal(err)
	}
	if objects["greeting.txt"] != file {
		t.Fatal("Purge mutated seed data")
	}
}

func writtenFile(content string) *mock.MemoryFile {
	f := &mock.MemoryFile{}
	_, _ = f.WriteString(content)
	return f
}

func checkBasePath(t *testing.T, filename string) {
	t.Helper()
	bucket := mock.NewSharedBucket(mock.Bucket{})
	scoped := mock.NewClientFromSharedBucket(bucket, "tenant-a")
	unscoped := mock.NewClientFromSharedBucket(bucket, "")
	if err := scoped.Set(context.Background(), filename, []byte("payload"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	got, _ := readFile(t, unscoped, "tenant-a/"+filename)
	if got != "payload" {
		t.Fatalf("stored bytes = %q", got)
	}
	if _, _, err := unscoped.Get(context.Background(), filename); !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("unprefixed key visible: %v", err)
	}
	url, err := scoped.GetPresignedURL(context.Background(), filename, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://not-a-real-presigned-url.example/tenant-a/" + filename; url != want {
		t.Fatalf("url = %q, want %q", url, want)
	}
}

func TestBasePath(t *testing.T) {
	for _, filename := range []string{"", "/a", "a//b", ".", "..", "../outside", "greeting.txt"} {
		t.Run(filename, func(t *testing.T) { checkBasePath(t, filename) })
	}
}

func openTestReader(t *testing.T, client *mock.Client, key string) io.ReadCloser {
	t.Helper()
	reader, _, err := client.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	return reader
}

func closePartialReader(t *testing.T, reader io.ReadCloser) {
	t.Helper()
	if _, err := reader.Read(make([]byte, 2)); err != nil {
		t.Fatalf("partial read: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("read after close = %v", err)
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
	closePartialReader(t, openTestReader(t, client, "a.txt"))
	if got, _ := readFile(t, client, "a.txt"); got != "abcdef" {
		t.Fatalf("stored content = %q", got)
	}
	if err := client.Copy(context.Background(), "a.txt", "b.txt"); err != nil {
		t.Fatal(err)
	}
	if got, _ := readFile(t, client, "b.txt"); got != "abcdef" {
		t.Fatalf("copied content = %q", got)
	}
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
