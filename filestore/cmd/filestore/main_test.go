package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

var errBoom = errors.New("boom")

// faultyStore wraps a working provider and makes one step of the demo fail.
type faultyStore struct {
	filestore.FileProvider
	failOn string
}

func (f faultyStore) Set(ctx context.Context, name string, data []byte, ct string) error {
	if f.failOn == "set" {
		return errBoom
	}
	return f.FileProvider.Set(ctx, name, data, ct)
}

func (f faultyStore) Copy(ctx context.Context, src, dst string) error {
	if f.failOn == "copy" {
		return errBoom
	}
	return f.FileProvider.Copy(ctx, src, dst)
}

func (f faultyStore) Get(ctx context.Context, name string) (io.ReadCloser, string, error) {
	switch f.failOn {
	case "get":
		return nil, "", errBoom
	case "read":
		return io.NopCloser(iotest.ErrReader(errBoom)), "text/plain", nil
	}
	return f.FileProvider.Get(ctx, name)
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	store := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "local"}})
	if err := run(ctx, store); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, contentType, err := store.Get(ctx, "example-copy.txt")
	if err != nil {
		t.Fatalf("the demo did not leave a copy behind: %v", err)
	}
	data, _ := io.ReadAll(body)
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello, filestore" || contentType != "text/plain" {
		t.Fatalf("copy = (%q, %q), want (%q, %q)", data, contentType, "hello, filestore", "text/plain")
	}
}

func TestRunFailures(t *testing.T) {
	for _, step := range []string{"set", "copy", "get", "read"} {
		t.Run(step, func(t *testing.T) {
			store := faultyStore{FileProvider: mock.NewClient(mock.Config{}), failOn: step}
			if err := run(context.Background(), store); !errors.Is(err, errBoom) {
				t.Fatalf("error = %v, want errBoom", err)
			}
		})
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := run(ctx, mock.NewClient(mock.Config{}))
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
