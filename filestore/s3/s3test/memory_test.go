package s3test_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3/s3test"
)

func TestMemoryAPIObjectLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &s3test.MemoryAPI{}

	if err := api.HeadObject(ctx, "b", "k"); !errors.Is(err, s3.ErrNoSuchKey) {
		t.Fatalf("HeadObject on empty store: error = %v, want ErrNoSuchKey", err)
	}
	if err := api.PutObject(ctx, "b", "k", strings.NewReader("data"), "text/plain"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if err := api.HeadObject(ctx, "b", "k"); err != nil {
		t.Fatalf("HeadObject: %v", err)
	}
	// Objects are scoped per bucket.
	if err := api.HeadObject(ctx, "other", "k"); !errors.Is(err, s3.ErrNoSuchKey) {
		t.Fatalf("HeadObject in other bucket: error = %v, want ErrNoSuchKey", err)
	}

	// CopyObject copies the content type and, like S3, overwrites.
	if err := api.PutObject(ctx, "b", "j", strings.NewReader("old"), "image/png"); err != nil {
		t.Fatal(err)
	}
	if err := api.CopyObject(ctx, "b", "k", "j"); err != nil {
		t.Fatalf("CopyObject: %v", err)
	}
	body, contentType, err := api.GetObject(ctx, "b", "j")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	data, _ := io.ReadAll(body)
	if string(data) != "data" || contentType != "text/plain" {
		t.Fatalf("GetObject = (%q, %q), want (%q, %q)", data, contentType, "data", "text/plain")
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}

	if err := api.DeleteObject(ctx, "b", "k"); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if err := api.DeleteObject(ctx, "b", "k"); err != nil {
		t.Fatalf("DeleteObject of a missing key: %v", err)
	}
	if err := api.CopyObject(ctx, "b", "k", "j"); !errors.Is(err, s3.ErrNoSuchKey) {
		t.Fatalf("CopyObject from missing key: error = %v, want ErrNoSuchKey", err)
	}

	url, err := api.PresignGetObject(ctx, "b", "j", 90*time.Second)
	if err != nil {
		t.Fatalf("PresignGetObject: %v", err)
	}
	if want := "https://b.s3.example/j?X-Amz-Expires=90"; url != want {
		t.Fatalf("PresignGetObject = %q, want %q", url, want)
	}
}

func TestMemoryAPIPutObjectReadError(t *testing.T) {
	t.Parallel()

	api := &s3test.MemoryAPI{}
	err := api.PutObject(context.Background(), "b", "k", iotest.ErrReader(io.ErrUnexpectedEOF), "")
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want the reader's error", err)
	}
	if err := api.HeadObject(context.Background(), "b", "k"); !errors.Is(err, s3.ErrNoSuchKey) {
		t.Fatalf("object stored despite failed read: error = %v", err)
	}
}

func TestMemoryAPICancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	api := &s3test.MemoryAPI{}

	calls := map[string]func() error{
		"HeadObject":   func() error { return api.HeadObject(ctx, "b", "k") },
		"GetObject":    func() error { _, _, err := api.GetObject(ctx, "b", "k"); return err },
		"PutObject":    func() error { return api.PutObject(ctx, "b", "k", strings.NewReader("x"), "") },
		"CopyObject":   func() error { return api.CopyObject(ctx, "b", "k", "j") },
		"DeleteObject": func() error { return api.DeleteObject(ctx, "b", "k") },
		"PresignGetObject": func() error {
			_, err := api.PresignGetObject(ctx, "b", "k", time.Minute)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}
