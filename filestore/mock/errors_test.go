package mock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

// TestCancelledContextShortCircuits confirms every operation checks its
// context before touching the bucket, returning the context's error and
// leaving the stored objects untouched.
func TestCancelledContextShortCircuits(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ops := map[string]func(*mock.Client) error{
		"Get":             func(c *mock.Client) error { _, _, err := c.Get(ctx, "a.txt"); return err },
		"Set":             func(c *mock.Client) error { return c.Set(ctx, "new.txt", []byte("x"), "text/plain") },
		"Purge":           func(c *mock.Client) error { return c.Purge(ctx, "a.txt") },
		"Move":            func(c *mock.Client) error { return c.Move(ctx, "a.txt", "b.txt") },
		"Copy":            func(c *mock.Client) error { return c.Copy(ctx, "a.txt", "b.txt") },
		"GetPresignedURL": func(c *mock.Client) error { _, err := c.GetPresignedURL(ctx, "a.txt", time.Minute); return err },
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := mock.NewClient(mock.Config{})
			if err := client.Set(context.Background(), "a.txt", []byte("hello"), "text/plain"); err != nil {
				t.Fatalf("seeding: %v", err)
			}

			if err := op(client); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}

			if got, _ := readFile(t, client, "a.txt"); got != "hello" {
				t.Fatalf("source changed to %q after a cancelled %s", got, name)
			}
			for _, name := range []string{"new.txt", "b.txt"} {
				if _, _, err := client.Get(context.Background(), name); !errors.Is(err, filestore.ErrNotFound) {
					t.Fatalf("%q should not exist after a cancelled op, got %v", name, err)
				}
			}
		})
	}
}

// TestMissingSourceIsNotFound covers the ErrNotFound contract for each
// operation that addresses an existing file.
func TestMissingSourceIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ops := map[string]func(*mock.Client) error{
		"Get":             func(c *mock.Client) error { _, _, err := c.Get(ctx, "missing.txt"); return err },
		"Move":            func(c *mock.Client) error { return c.Move(ctx, "missing.txt", "dst.txt") },
		"Copy":            func(c *mock.Client) error { return c.Copy(ctx, "missing.txt", "dst.txt") },
		"GetPresignedURL": func(c *mock.Client) error { _, err := c.GetPresignedURL(ctx, "missing.txt", time.Minute); return err },
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := mock.NewClient(mock.Config{})
			if err := op(client); !errors.Is(err, filestore.ErrNotFound) {
				t.Fatalf("got %v, want filestore.ErrNotFound", err)
			}
			if _, _, err := client.Get(ctx, "dst.txt"); !errors.Is(err, filestore.ErrNotFound) {
				t.Fatalf("destination should not be created, got %v", err)
			}
		})
	}
}

// TestMoveOntoExistingKeepsBoth confirms Move refuses to overwrite an
// existing destination and, having failed, leaves the source in place.
func TestMoveOntoExistingKeepsBoth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := mock.NewClient(mock.Config{})
	for name, body := range map[string]string{"src.txt": "src", "dst.txt": "dst"} {
		if err := client.Set(ctx, name, []byte(body), "text/plain"); err != nil {
			t.Fatalf("seeding %q: %v", name, err)
		}
	}

	if err := client.Move(ctx, "src.txt", "dst.txt"); !errors.Is(err, filestore.ErrFileExists) {
		t.Fatalf("got %v, want filestore.ErrFileExists", err)
	}
	if got, _ := readFile(t, client, "src.txt"); got != "src" {
		t.Fatalf("source changed to %q", got)
	}
	if got, _ := readFile(t, client, "dst.txt"); got != "dst" {
		t.Fatalf("destination changed to %q", got)
	}
}
