package s3

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

// TestCancelledContextShortCircuits confirms every operation checks its
// context before calling the ObjectAPI at all.
func TestCancelledContextShortCircuits(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ops := map[string]func(*Client) error{
		"Get":             func(c *Client) error { _, _, err := c.Get(ctx, "a.txt"); return err },
		"Set":             func(c *Client) error { return c.Set(ctx, "a.txt", []byte("x"), "text/plain") },
		"Purge":           func(c *Client) error { return c.Purge(ctx, "a.txt") },
		"Move":            func(c *Client) error { return c.Move(ctx, "a.txt", "b.txt") },
		"Copy":            func(c *Client) error { return c.Copy(ctx, "a.txt", "b.txt") },
		"GetPresignedURL": func(c *Client) error { _, err := c.GetPresignedURL(ctx, "a.txt", time.Minute); return err },
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api := newMemAPI()
			if err := op(NewClient(Config{Bucket: "b", API: api})); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
			if len(api.buckets) != 0 {
				t.Fatalf("expected no ObjectAPI calls, got %d", len(api.buckets))
			}
		})
	}
}

// TestMissingKeyIsNotFound covers the ErrNoSuchKey -> filestore.ErrNotFound
// mapping on each operation that addresses an existing key.
func TestMissingKeyIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ops := map[string]func(*Client) error{
		"Get":             func(c *Client) error { _, _, err := c.Get(ctx, "missing.txt"); return err },
		"Move":            func(c *Client) error { return c.Move(ctx, "missing.txt", "dst.txt") },
		"Copy":            func(c *Client) error { return c.Copy(ctx, "missing.txt", "dst.txt") },
		"GetPresignedURL": func(c *Client) error { _, err := c.GetPresignedURL(ctx, "missing.txt", time.Minute); return err },
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := op(NewClient(Config{Bucket: "b", API: newMemAPI()})); !errors.Is(err, filestore.ErrNotFound) {
				t.Fatalf("got %v, want filestore.ErrNotFound", err)
			}
		})
	}
}

// TestDestinationProbeFailure confirms a HeadObject failure other than
// ErrNoSuchKey aborts Copy/Move before CopyObject is attempted.
func TestDestinationProbeFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ops := map[string]func(*Client) error{
		"Move": func(c *Client) error { return c.Move(ctx, "src.txt", "dst.txt") },
		"Copy": func(c *Client) error { return c.Copy(ctx, "src.txt", "dst.txt") },
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api := newMemAPI()
			api.objects["src.txt"] = "data"
			api.errs["HeadObject"] = errBoom

			if err := op(NewClient(Config{Bucket: "b", API: api})); !errors.Is(err, errBoom) {
				t.Fatalf("got %v, want errBoom", err)
			}
			if _, ok := api.objects["dst.txt"]; ok {
				t.Fatal("destination must not be written when the probe fails")
			}
			if _, ok := api.objects["src.txt"]; !ok {
				t.Fatal("source must be left in place when the probe fails")
			}
		})
	}
}
