package mock_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func newClientWithFiles(t *testing.T, files map[string]string) *mock.Client {
	t.Helper()

	objects := map[string]*mock.MemoryFile{}
	for name, content := range files {
		file := &mock.MemoryFile{}
		if _, err := file.WriteString(content); err != nil {
			t.Fatalf("seeding file %q: %v", name, err)
		}
		objects[name] = file
	}

	return mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test", Objects: objects}})
}

func readFile(t *testing.T, client *mock.Client, filename string) string {
	t.Helper()

	rc, _, err := client.Get(context.Background(), filename)
	if err != nil {
		t.Fatalf("getting %q: %v", filename, err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			t.Errorf("closing %q: %v", filename, err)
		}
	}()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading %q: %v", filename, err)
	}
	return string(content)
}

func TestClientGet(t *testing.T) {
	t.Parallel()

	client := newClientWithFiles(t, map[string]string{"greeting.txt": "hello world"})

	if got := readFile(t, client, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q but got %q", "hello world", got)
	}
	if _, _, err := client.Get(context.Background(), "missing.txt"); err == nil {
		t.Fatal("expected error getting a missing file")
	}
}

func TestClientGetDoesNotConsumeStoredFile(t *testing.T) {
	t.Parallel()

	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})
	if err := client.Set(context.Background(), "greeting.txt", []byte("hello world"), "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}

	if got := readFile(t, client, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q on first get but got %q", "hello world", got)
	}
	if got := readFile(t, client, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q on second get but got %q", "hello world", got)
	}
}

func TestClientSet(t *testing.T) {
	t.Parallel()

	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})

	if err := client.Set(context.Background(), "greeting.txt", []byte("hello world"), "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}
	if got := readFile(t, client, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q but got %q", "hello world", got)
	}
}

// TestClientBasePath proves BasePath is actually applied as a key prefix
// rather than ignored: every other test in this file uses an empty
// BasePath, under which the composed key equals the raw filename, so none
// of them can tell a client that honors BasePath apart from one that drops
// it. Two clients share the same underlying object map here specifically
// so the prefix can be observed from outside the scoped client.
func TestClientBasePath(t *testing.T) {
	t.Parallel()

	objects := map[string]*mock.MemoryFile{}
	scoped := mock.NewClient(mock.Config{BasePath: "tenant-a", Bucket: mock.Bucket{Objects: objects}})
	unscoped := mock.NewClient(mock.Config{Bucket: mock.Bucket{Objects: objects}})

	if err := scoped.Set(context.Background(), "greeting.txt", []byte("hello world"), "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}
	if got := readFile(t, scoped, "greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q but got %q", "hello world", got)
	}

	if _, _, err := unscoped.Get(context.Background(), "greeting.txt"); err == nil {
		t.Fatal("expected a file set under a base path not to be visible without it")
	}
	if got := readFile(t, unscoped, "tenant-a/greeting.txt"); got != "hello world" {
		t.Fatalf("expected content %q at the composed key but got %q", "hello world", got)
	}
}

func TestClientPurge(t *testing.T) {
	t.Parallel()

	client := newClientWithFiles(t, map[string]string{"greeting.txt": "hello world"})

	if err := client.Purge(context.Background(), "greeting.txt"); err != nil {
		t.Fatalf("purging file: %v", err)
	}
	if _, _, err := client.Get(context.Background(), "greeting.txt"); err == nil {
		t.Fatal("expected error getting a purged file")
	}
}

func TestClientGetPresignedURL(t *testing.T) {
	t.Parallel()

	client := newClientWithFiles(t, map[string]string{"greeting.txt": "hello world"})

	url, err := client.GetPresignedURL(context.Background(), "greeting.txt", time.Minute)
	if err != nil {
		t.Fatalf("getting presigned url: %v", err)
	}
	if url == "" {
		t.Fatal("expected a non-empty presigned url")
	}
	if _, err := client.GetPresignedURL(context.Background(), "missing.txt", time.Minute); err == nil {
		t.Fatal("expected error for a missing file")
	}
}

func TestMemoryFileReadAfterClose(t *testing.T) {
	t.Parallel()

	file := &mock.MemoryFile{}
	if _, err := file.WriteString("hello world"); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("closing file: %v", err)
	}
	if _, err := file.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected error reading a closed file")
	}
}

type moveCase struct {
	setup       map[string]string
	oldFilename string
	newFilename string
	wantErr     bool
}

func TestClientMove(t *testing.T) {
	t.Parallel()

	testCases := map[string]moveCase{
		"success": {
			setup:       map[string]string{"source.txt": "hello world"},
			oldFilename: "source.txt",
			newFilename: "destination.txt",
		},
		"missing source": {
			oldFilename: "missing.txt",
			newFilename: "destination.txt",
			wantErr:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) { runMoveCase(t, tc) })
	}
}

func runMoveCase(t *testing.T, tc moveCase) {
	t.Helper()

	client := newClientWithFiles(t, tc.setup)

	err := client.Move(context.Background(), tc.oldFilename, tc.newFilename)
	if tc.wantErr {
		if err == nil {
			t.Fatal("expected error moving file")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error moving file: %v", err)
	}

	if got := readFile(t, client, tc.newFilename); got != "hello world" {
		t.Fatalf("expected content %q but got %q", "hello world", got)
	}
	if _, _, err := client.Get(context.Background(), tc.oldFilename); err == nil {
		t.Fatal("expected old key to be gone after move")
	}
}

type copyCase struct {
	setup       map[string]string
	oldFilename string
	newFilename string
	wantErr     error
	wantContent string
}

func TestClientCopy(t *testing.T) {
	t.Parallel()

	testCases := map[string]copyCase{
		"success": {
			setup:       map[string]string{"source.txt": "hello world"},
			oldFilename: "source.txt",
			newFilename: "destination.txt",
			wantContent: "hello world",
		},
		"missing source": {
			oldFilename: "missing.txt",
			newFilename: "destination.txt",
			wantErr:     filestore.ErrNotFound,
		},
		"destination conflict": {
			setup: map[string]string{
				"source.txt":      "hello world",
				"destination.txt": "existing content",
			},
			oldFilename: "source.txt",
			newFilename: "destination.txt",
			wantErr:     filestore.ErrFileExists,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) { runCopyCase(t, tc) })
	}
}

func runCopyCase(t *testing.T, tc copyCase) {
	t.Helper()

	client := newClientWithFiles(t, tc.setup)

	err := client.Copy(context.Background(), tc.oldFilename, tc.newFilename)
	if tc.wantErr != nil {
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("expected error %v but got %v", tc.wantErr, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error copying file: %v", err)
	}

	if got := readFile(t, client, tc.newFilename); got != tc.wantContent {
		t.Fatalf("expected copied content %q but got %q", tc.wantContent, got)
	}
	if got := readFile(t, client, tc.oldFilename); got != tc.wantContent {
		t.Fatalf("expected source content %q but got %q", tc.wantContent, got)
	}
}

func TestClientCopyAfterSourceReadAndClose(t *testing.T) {
	t.Parallel()

	client := newClientWithFiles(t, map[string]string{"source.txt": "abcdef"})

	rc, _, err := client.Get(context.Background(), "source.txt")
	if err != nil {
		t.Fatalf("getting source file: %v", err)
	}
	if _, err := rc.Read(make([]byte, 2)); err != nil {
		t.Fatalf("partially reading source file: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("closing source file: %v", err)
	}

	if err := client.Copy(context.Background(), "source.txt", "destination.txt"); err != nil {
		t.Fatalf("unexpected error copying file: %v", err)
	}
	if got := readFile(t, client, "destination.txt"); got != "abcdef" {
		t.Fatalf("expected copied content %q but got %q", "abcdef", got)
	}
}
