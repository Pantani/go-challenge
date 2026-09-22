package filestore_test

import (
	"context"
	"errors"
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
