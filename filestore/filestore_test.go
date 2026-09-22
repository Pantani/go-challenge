package filestore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func TestMockSetThenGet(t *testing.T) {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})
	ctx := context.Background()
	want := []byte("hello world")

	if err := client.Set(ctx, "greeting.txt", want, "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}

	rc, _, err := client.Get(ctx, "greeting.txt")
	if err != nil {
		t.Fatalf("getting file: %v", err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			t.Errorf("closing file: %v", err)
		}
	}()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

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

func TestMockSetGetRoundTrip(t *testing.T) {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "test"}})
	want := []byte("hello world")

	if err := client.Set(context.Background(), "greeting.txt", want, "text/plain"); err != nil {
		t.Fatalf("setting file: %v", err)
	}

	reader, _, err := client.Get(context.Background(), "greeting.txt")
	if err != nil {
		t.Fatalf("getting file: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
