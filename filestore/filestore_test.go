package filestore_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3/s3test"
)

// providers lists every implementation the contract suite runs against, so
// the error contract is verified identically for each of them.
var providers = []struct {
	name string
	new  func() filestore.PresignedFileProvider
}{
	{name: "mock", new: func() filestore.PresignedFileProvider {
		return mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}, BasePath: "base"})
	}},
	{name: "s3", new: func() filestore.PresignedFileProvider {
		return s3.NewClient(s3.Config{Bucket: "test", API: &s3test.MemoryAPI{}})
	}},
}

func forEachProvider(t *testing.T, fn func(t *testing.T, store filestore.PresignedFileProvider)) {
	t.Helper()
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			fn(t, p.new())
		})
	}
}

// readFile fetches filename and returns its contents and content type.
func readFile(t *testing.T, store filestore.FileProvider, filename string) (string, string) {
	t.Helper()
	body, contentType, err := store.Get(context.Background(), filename)
	if err != nil {
		t.Fatalf("Get(%q): %v", filename, err)
	}
	data, readErr := io.ReadAll(body)
	closeErr := body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatalf("reading and closing %q: %v", filename, err)
	}
	return string(data), contentType
}

func assertFile(t *testing.T, store filestore.FileProvider, filename, wantData, wantContentType string) {
	t.Helper()
	data, contentType := readFile(t, store, filename)
	if data != wantData {
		t.Errorf("%q data = %q, want %q", filename, data, wantData)
	}
	if contentType != wantContentType {
		t.Errorf("%q content type = %q, want %q", filename, contentType, wantContentType)
	}
}

func assertMissing(t *testing.T, store filestore.FileProvider, filename string) {
	t.Helper()
	body, _, err := store.Get(context.Background(), filename)
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("Get(%q) error = %v, want ErrNotFound", filename, err)
	}
	if body != nil {
		t.Errorf("Get(%q) returned a body alongside the error", filename)
	}
}

func mustSet(t *testing.T, store filestore.FileProvider, filename, data, contentType string) {
	t.Helper()
	if err := store.Set(context.Background(), filename, []byte(data), contentType); err != nil {
		t.Fatalf("Set(%q): %v", filename, err)
	}
}

func mustPurge(t *testing.T, store filestore.FileProvider, filename string) {
	t.Helper()
	if err := store.Purge(context.Background(), filename); err != nil {
		t.Fatalf("Purge(%q): %v", filename, err)
	}
}

func TestGetSet(t *testing.T) {
	forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
		assertMissing(t, store, "missing.txt")

		mustSet(t, store, "a.txt", "first", "text/plain")
		assertFile(t, store, "a.txt", "first", "text/plain")

		// Reading a file must not consume it.
		assertFile(t, store, "a.txt", "first", "text/plain")

		// Set replaces existing content and content type.
		mustSet(t, store, "a.txt", "second", "application/json")
		assertFile(t, store, "a.txt", "second", "application/json")

		// The store keeps its own copy of the bytes.
		payload := []byte("mutable")
		if err := store.Set(context.Background(), "b.txt", payload, "text/plain"); err != nil {
			t.Fatal(err)
		}
		payload[0] = 'X'
		assertFile(t, store, "b.txt", "mutable", "text/plain")
	})
}

func TestPurge(t *testing.T) {
	forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
		mustPurge(t, store, "missing.txt")
		mustSet(t, store, "a.txt", "data", "text/plain")
		mustPurge(t, store, "a.txt")
		assertMissing(t, store, "a.txt")
	})
}

// transferCase drives both Copy and Move: every case starts with src.txt
// and taken.txt present.
type transferCase struct {
	name    string
	src     string
	dst     string
	wantErr error
}

var transferCases = []transferCase{
	{name: "success", src: "src.txt", dst: "dst.txt"},
	{name: "missing source", src: "missing.txt", dst: "dst.txt", wantErr: filestore.ErrNotFound},
	{name: "destination conflict", src: "src.txt", dst: "taken.txt", wantErr: filestore.ErrFileExists},
	{name: "onto itself", src: "src.txt", dst: "src.txt", wantErr: filestore.ErrFileExists},
	{name: "missing source and taken destination", src: "missing.txt", dst: "taken.txt", wantErr: filestore.ErrFileExists},
}

func seedTransfer(t *testing.T, store filestore.FileProvider) {
	t.Helper()
	mustSet(t, store, "src.txt", "source", "text/plain")
	mustSet(t, store, "taken.txt", "taken", "image/png")
}

func TestCopy(t *testing.T) {
	for _, tc := range transferCases {
		t.Run(tc.name, func(t *testing.T) {
			forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
				checkCopyCase(t, store, tc)
			})
		})
	}
}

func TestMove(t *testing.T) {
	for _, tc := range transferCases {
		t.Run(tc.name, func(t *testing.T) {
			forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
				checkMoveCase(t, store, tc)
			})
		})
	}
}

func assertTransferFailure(t *testing.T, store filestore.FileProvider, tc transferCase) {
	t.Helper()
	assertFile(t, store, "src.txt", "source", "text/plain")
	if tc.dst == "dst.txt" {
		assertMissing(t, store, tc.dst)
	}
}

func checkCopyCase(t *testing.T, store filestore.PresignedFileProvider, tc transferCase) {
	t.Helper()
	seedTransfer(t, store)
	err := store.Copy(context.Background(), tc.src, tc.dst)
	if !errors.Is(err, tc.wantErr) {
		t.Fatalf("Copy error = %v, want %v", err, tc.wantErr)
	}
	// Pre-existing files are never disturbed.
	assertFile(t, store, "src.txt", "source", "text/plain")
	assertFile(t, store, "taken.txt", "taken", "image/png")
	if tc.wantErr != nil {
		assertTransferFailure(t, store, tc)
		return
	}
	assertFile(t, store, tc.dst, "source", "text/plain")
	// The copy is independent of the source.
	mustSet(t, store, tc.src, "changed", "text/plain")
	assertFile(t, store, tc.dst, "source", "text/plain")
	mustPurge(t, store, tc.src)
	assertFile(t, store, tc.dst, "source", "text/plain")
}

func checkMoveCase(t *testing.T, store filestore.PresignedFileProvider, tc transferCase) {
	t.Helper()
	seedTransfer(t, store)
	err := store.Move(context.Background(), tc.src, tc.dst)
	if !errors.Is(err, tc.wantErr) {
		t.Fatalf("Move error = %v, want %v", err, tc.wantErr)
	}
	assertFile(t, store, "taken.txt", "taken", "image/png")
	if tc.wantErr != nil {
		assertTransferFailure(t, store, tc)
		return
	}
	assertMissing(t, store, tc.src)
	assertFile(t, store, tc.dst, "source", "text/plain")
}

func TestGetPresignedURL(t *testing.T) {
	forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
		_, err := store.GetPresignedURL(context.Background(), "missing.txt", time.Minute)
		if !errors.Is(err, filestore.ErrNotFound) {
			t.Fatalf("presigning a missing file: error = %v, want ErrNotFound", err)
		}
		mustSet(t, store, "a.txt", "data", "text/plain")
		url, err := store.GetPresignedURL(context.Background(), "a.txt", time.Minute)
		if err != nil {
			t.Fatalf("GetPresignedURL: %v", err)
		}
		if !strings.HasPrefix(url, "http") || !strings.Contains(url, "a.txt") {
			t.Errorf("unexpected presigned URL %q", url)
		}
	})
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ops := []struct {
		name string
		call func(filestore.PresignedFileProvider) error
	}{
		{"Get", func(s filestore.PresignedFileProvider) error { _, _, err := s.Get(ctx, "a.txt"); return err }},
		{"Set", func(s filestore.PresignedFileProvider) error { return s.Set(ctx, "a.txt", nil, "") }},
		{"Purge", func(s filestore.PresignedFileProvider) error { return s.Purge(ctx, "a.txt") }},
		{"Move", func(s filestore.PresignedFileProvider) error { return s.Move(ctx, "a.txt", "b.txt") }},
		{"Copy", func(s filestore.PresignedFileProvider) error { return s.Copy(ctx, "a.txt", "b.txt") }},
		{"GetPresignedURL", func(s filestore.PresignedFileProvider) error {
			_, err := s.GetPresignedURL(ctx, "a.txt", time.Minute)
			return err
		}},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
				mustSet(t, store, "a.txt", "data", "text/plain")
				if err := op.call(store); !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
				// Nothing happened.
				assertFile(t, store, "a.txt", "data", "text/plain")
				assertMissing(t, store, "b.txt")
			})
		})
	}
}

// TestErrorsCarryContext checks that wrapped sentinels still name the
// operation and file, which is what makes them useful in logs.
func TestErrorsCarryContext(t *testing.T) {
	forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
		_, _, err := store.Get(context.Background(), "missing.txt")
		if err == nil || !strings.Contains(err.Error(), `"missing.txt"`) || !strings.Contains(err.Error(), "file not found") {
			t.Fatalf("error %q should name the file and the sentinel", err)
		}
	})
}
