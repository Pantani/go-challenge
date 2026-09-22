package s3

import (
	"context"
	"errors"
	"testing"
)

type interleavedAPI struct {
	*memAPI
	afterHead    func()
	beforeDelete func()
	deletedKeys  []string
}

func (a *interleavedAPI) HeadObject(ctx context.Context, bucket, key string) error {
	err := a.memAPI.HeadObject(ctx, bucket, key)
	if a.afterHead != nil {
		a.afterHead()
	}
	return err
}

func (a *interleavedAPI) DeleteObject(ctx context.Context, bucket, key string) error {
	a.deletedKeys = append(a.deletedKeys, key)
	if a.beforeDelete != nil {
		a.beforeDelete()
	}
	return a.memAPI.DeleteObject(ctx, bucket, key)
}

func TestCopyDestinationCheckIsBestEffort(t *testing.T) {
	backend := newMemAPI()
	backend.objects["source"] = "source data"
	api := &interleavedAPI{memAPI: backend}
	api.afterHead = func() { backend.objects["destination"] = "another writer" }
	client := NewClient(Config{Bucket: "bucket", API: api})
	if err := client.Copy(context.Background(), "source", "destination"); err != nil {
		t.Fatal(err)
	}
	if backend.objects["destination"] != "source data" {
		t.Fatalf("destination = %q", backend.objects["destination"])
	}
	if backend.objects["source"] != "source data" {
		t.Fatal("copy changed source")
	}
}

func TestMoveDeleteFailurePreservesConcurrentDestination(t *testing.T) {
	backend := newMemAPI()
	backend.objects["source"] = "source data"
	deleteErr := errors.New("delete unavailable")
	backend.errs["DeleteObject"] = deleteErr
	api := &interleavedAPI{memAPI: backend}
	api.beforeDelete = func() { backend.objects["destination"] = "new writer data" }
	client := NewClient(Config{Bucket: "bucket", API: api})
	if err := client.Move(context.Background(), "source", "destination"); !errors.Is(err, deleteErr) {
		t.Fatalf("Move = %v, want wrapped delete error", err)
	}
	if len(api.deletedKeys) != 1 {
		t.Fatalf("DeleteObject keys = %v, want only source", api.deletedKeys)
	}
	if api.deletedKeys[0] != "source" {
		t.Fatalf("DeleteObject key = %q, want source", api.deletedKeys[0])
	}
	if backend.objects["source"] != "source data" {
		t.Fatal("failed delete removed source")
	}
	if backend.objects["destination"] != "new writer data" {
		t.Fatal("failed move changed concurrent destination")
	}
}
