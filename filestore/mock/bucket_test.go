package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func TestLegacyBucketIsSnapshot(t *testing.T) {
	seed := mock.NewMemoryFile([]byte("before"), "text/plain")
	objects := map[string]*mock.MemoryFile{"key": seed}
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Objects: objects}})
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	delete(objects, "key")
	got, ct := readFile(t, client, "key")
	if got != "before" || ct != "text/plain" {
		t.Fatalf("snapshot = (%q, %q)", got, ct)
	}
}

func TestLegacyClientsDoNotShareWrites(t *testing.T) {
	config := mock.Config{Bucket: mock.Bucket{Objects: map[string]*mock.MemoryFile{}}}
	first, second := mock.NewClient(config), mock.NewClient(config)
	if err := first.Set(context.Background(), "key", []byte("first"), ""); err != nil {
		t.Fatal(err)
	}
	_, _, err := second.Get(context.Background(), "key")
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("second client sees first client's write: %v", err)
	}
}

func TestSharedBucketOwnsSeed(t *testing.T) {
	seed := mock.NewMemoryFile([]byte("shared"), "text/plain")
	objects := map[string]*mock.MemoryFile{"key": seed}
	bucket := mock.NewSharedBucket(mock.Bucket{Objects: objects})
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	delete(objects, "key")
	client := mock.NewClientFromSharedBucket(bucket, "")
	got, contentType := readFile(t, client, "key")
	if got != "shared" || contentType != "text/plain" {
		t.Fatalf("shared seed = (%q, %q)", got, contentType)
	}
}

func TestSharedBucketZeroValue(t *testing.T) {
	bucket := &mock.SharedBucket{}
	first := mock.NewClientFromSharedBucket(bucket, "")
	second := mock.NewClientFromSharedBucket(bucket, "")
	if err := first.Set(context.Background(), "key", []byte("shared"), ""); err != nil {
		t.Fatal(err)
	}
	got, _ := readFile(t, second, "key")
	if got != "shared" {
		t.Fatalf("shared data = %q", got)
	}
}

func TestNilSeedEntryIsMissing(t *testing.T) {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Objects: map[string]*mock.MemoryFile{"nil": nil}}})
	_, _, err := client.Get(context.Background(), "nil")
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("nil seed error = %v", err)
	}
}

func TestSharedBucketNilCreatesPrivateState(t *testing.T) {
	first := mock.NewClientFromSharedBucket(nil, "")
	second := mock.NewClientFromSharedBucket(nil, "")
	if err := first.Set(context.Background(), "key", []byte("first"), ""); err != nil {
		t.Fatal(err)
	}
	_, _, err := second.Get(context.Background(), "key")
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("nil shared buckets share state: %v", err)
	}
}
