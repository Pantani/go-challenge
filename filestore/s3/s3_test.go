package s3_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3"
)

func TestClientGet(t *testing.T) {
	t.Parallel()

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	_, _, err := client.Get(context.Background(), "source.txt")
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("expected %v but got %v", filestore.ErrNotFound, err)
	}
}

func TestClientSet(t *testing.T) {
	t.Parallel()

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	if err := client.Set(context.Background(), "source.txt", []byte("hello world"), "text/plain"); err != nil {
		t.Fatalf("unexpected error setting file: %v", err)
	}
}

func TestClientPurge(t *testing.T) {
	t.Parallel()

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	if err := client.Purge(context.Background(), "source.txt"); err != nil {
		t.Fatalf("unexpected error purging file: %v", err)
	}
}

func TestClientMove(t *testing.T) {
	t.Parallel()

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	if err := client.Move(context.Background(), "source.txt", "destination.txt"); err != nil {
		t.Fatalf("unexpected error moving file: %v", err)
	}
}

func TestClientCopy(t *testing.T) {

	t.Parallel()

	var _ filestore.PresignedFileProvider = (*s3.Client)(nil)

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	if err := client.Copy(context.Background(), "source.txt", "destination.txt"); err != nil {
		t.Fatalf("unexpected error copying file: %v", err)
	}
}

func TestClientGetPresignedURL(t *testing.T) {
	t.Parallel()

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	url, err := client.GetPresignedURL(context.Background(), "source.txt", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error getting presigned url: %v", err)
	}
	if url == "" {
		t.Fatal("expected a non-empty presigned url")
	}
}
