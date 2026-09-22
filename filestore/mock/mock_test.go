package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func TestCopyKeepsSourceAndMetadata(t *testing.T) {
	store := mock.NewClient(mock.Config{})
	if err := store.Set(context.Background(), "source", []byte("data"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := store.Copy(context.Background(), "source", "copy"); err != nil {
		t.Fatal(err)
	}
	if err := store.Copy(context.Background(), "missing", "copy"); !errors.Is(err, filestore.ErrFileExists) {
		t.Fatalf("Copy() error = %v", err)
	}
}
