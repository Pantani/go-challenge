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

func putTestObject(t *testing.T, api *s3test.MemoryAPI, key, data, contentType string) {
	t.Helper()
	if err := api.PutObject(context.Background(), "b", key, strings.NewReader(data), contentType); err != nil {
		t.Fatal(err)
	}
}

func assertMemoryObject(t *testing.T, api *s3test.MemoryAPI, key, wantData, wantType string) {
	t.Helper()
	body, contentType, err := api.GetObject(context.Background(), "b", key)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(body)
	closeErr := body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatal(err)
	}
	if string(data) != wantData || contentType != wantType {
		t.Fatalf("object = (%q, %q), want (%q, %q)", data, contentType, wantData, wantType)
	}
}

func assertMemoryHead(t *testing.T, api *s3test.MemoryAPI, bucket, key string, want error) {
	t.Helper()
	if err := api.HeadObject(context.Background(), bucket, key); !errors.Is(err, want) {
		t.Fatalf("HeadObject(%q, %q) = %v, want %v", bucket, key, err, want)
	}
}

func TestMemoryAPIKeysAreScopedToBucket(t *testing.T) {
	api := &s3test.MemoryAPI{}
	assertMemoryHead(t, api, "b", "k", s3.ErrNoSuchKey)
	putTestObject(t, api, "k", "data", "text/plain")
	assertMemoryHead(t, api, "b", "k", nil)
	assertMemoryHead(t, api, "other", "k", s3.ErrNoSuchKey)
}

func TestMemoryAPICopyOverwritesMetadata(t *testing.T) {
	api := &s3test.MemoryAPI{}
	putTestObject(t, api, "k", "data", "text/plain")
	putTestObject(t, api, "j", "old", "image/png")
	if err := api.CopyObject(context.Background(), "b", "k", "j"); err != nil {
		t.Fatal(err)
	}
	assertMemoryObject(t, api, "k", "data", "text/plain")
	assertMemoryObject(t, api, "j", "data", "text/plain")
}

func TestMemoryAPIDeleteIsIdempotent(t *testing.T) {
	api := &s3test.MemoryAPI{}
	putTestObject(t, api, "k", "data", "text/plain")
	for range 2 {
		if err := api.DeleteObject(context.Background(), "b", "k"); err != nil {
			t.Fatal(err)
		}
	}
	assertMemoryHead(t, api, "b", "k", s3.ErrNoSuchKey)
	if err := api.CopyObject(context.Background(), "b", "k", "j"); !errors.Is(err, s3.ErrNoSuchKey) {
		t.Fatalf("CopyObject missing source = %v", err)
	}
}

func TestMemoryAPIPresignDuration(t *testing.T) {
	api := &s3test.MemoryAPI{}
	url, err := api.PresignGetObject(context.Background(), "b", "j", 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://b.s3.example/j?X-Amz-Expires=90"; url != want {
		t.Fatalf("URL = %q, want %q", url, want)
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
