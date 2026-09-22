package filestore_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func TestMockPurge(t *testing.T) {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})
	if err := client.Purge(context.Background(), "missing.txt"); err != nil {
		t.Fatalf("purging a missing file: %v", err)
	}
}

func TestFileProviderContract(t *testing.T) {
	var _ filestore.FileProvider = mock.NewClient(mock.Config{})
	_, _, err := mock.NewClient(mock.Config{}).Get(context.Background(), "missing.txt")
	if err == nil {
		t.Fatal("expected missing file error")
	}
	if errors.Is(err, filestore.ErrNotFound) {
		t.Fatal("starter mock intentionally does not preserve the sentinel error")
	}
}

type mockCopyCase struct {
	setup       map[string]string
	oldFilename string
	newFilename string
	wantErr     error
	wantContent string
}

func TestMockCopy(t *testing.T) {

	t.Parallel()

	testCases := map[string]mockCopyCase{
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
		t.Run(name, func(t *testing.T) { runMockCopyCase(t, tc) })
	}
}

func runMockCopyCase(t *testing.T, tc mockCopyCase) {
	t.Helper()

	client := newMockClientWithFiles(t, tc.setup)

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

	if got := readMockFile(t, client, tc.newFilename); got != tc.wantContent {
		t.Fatalf("expected copied content %q but got %q", tc.wantContent, got)
	}
	if got := readMockFile(t, client, tc.oldFilename); got != tc.wantContent {
		t.Fatalf("expected source content %q but got %q", tc.wantContent, got)
	}
}

func newMockClientWithFiles(t *testing.T, files map[string]string) *mock.Client {
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

func readMockFile(t *testing.T, client *mock.Client, filename string) string {
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
