package mock_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

// The provider-neutral behavior of Copy is covered by the contract suite in
// the parent package; these tests pin down mock-specific details.

func TestCopyKeepsSourceAndMetadata(t *testing.T) {
	ctx := context.Background()
	store := mock.NewClient(mock.Config{BasePath: "base"})
	if err := store.Set(ctx, "source", []byte("data"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := store.Copy(ctx, "source", "copy"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "copy"} {
		assertObject(t, store, name, "data", "text/plain")
	}
}

func TestCopyReportsConflictBeforeMissingSource(t *testing.T) {
	ctx := context.Background()
	store := mock.NewClient(mock.Config{})
	if err := store.Set(ctx, "taken", []byte("data"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := store.Copy(ctx, "missing", "taken"); !errors.Is(err, filestore.ErrFileExists) {
		t.Fatalf("Copy() error = %v, want ErrFileExists", err)
	}
}

func assertObject(t *testing.T, store filestore.FileProvider, name, wantData, wantType string) {
	t.Helper()
	body, contentType, err := store.Get(context.Background(), name)
	if err != nil {
		t.Fatalf("Get(%q): %v", name, err)
	}
	data, err := io.ReadAll(body)
	if err := errors.Join(err, body.Close()); err != nil {
		t.Fatalf("read %q: %v", name, err)
	}
	if string(data) != wantData || contentType != wantType {
		t.Fatalf("Get(%q) = (%q, %q), want (%q, %q)", name, data, contentType, wantData, wantType)
	}
}
