// Package mock provides an in-memory filestore.PresignedFileProvider for
// tests and local development.
package mock

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

var _ filestore.PresignedFileProvider = (*Client)(nil)

// Config configures a mock Client.
type Config struct {
	Bucket Bucket
	// BasePath is prefixed to every filename to form the object key.
	BasePath string
}

// Client is an in-memory file store. It is safe for concurrent use.
type Client struct {
	bucket   *SharedBucket
	basePath string
}

// NewClient snapshots config.Bucket into private client state.
func NewClient(config Config) *Client {
	return NewClientFromSharedBucket(NewSharedBucket(config.Bucket), config.BasePath)
}

// NewClientFromSharedBucket attaches to shared; nil creates private state.
func NewClientFromSharedBucket(shared *SharedBucket, basePath string) *Client {
	if shared == nil {
		shared = NewSharedBucket(Bucket{})
	}
	shared.initialize()
	return &Client{bucket: shared, basePath: basePath}
}

// key maps a filename to its object key. A non-empty base path confines keys
// by an exact byte prefix; object names are opaque and are never normalized.
func (c *Client) key(filename string) string {
	if c.basePath == "" {
		return filename
	}
	return c.basePath + "/" + filename
}

// lookup returns the stored object for filename. The caller must hold c.bucket.mu.
func (c *Client) lookup(op, filename string) (*MemoryFile, error) {
	file, ok := c.bucket.objects[c.key(filename)]
	if !ok {
		return nil, opErr(op, filename, filestore.ErrNotFound)
	}
	return file, nil
}

// Get implements filestore.FileProvider.
func (c *Client) Get(ctx context.Context, filename string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	c.bucket.mu.RLock()
	defer c.bucket.mu.RUnlock()
	file, err := c.lookup("get", filename)
	if err != nil {
		return nil, "", err
	}
	return file.snapshot(), file.contentType(), nil
}

// Set implements filestore.FileProvider.
func (c *Client) Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file := NewMemoryFile(fileBytes, contentType)
	c.bucket.mu.Lock()
	c.bucket.objects[c.key(filename)] = file
	c.bucket.mu.Unlock()
	return nil
}

// Purge implements filestore.FileProvider.
func (c *Client) Purge(ctx context.Context, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.bucket.mu.Lock()
	delete(c.bucket.objects, c.key(filename))
	c.bucket.mu.Unlock()
	return nil
}

// Move implements filestore.FileProvider. The rename is atomic with respect
// to other Client operations.
func (c *Client) Move(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.bucket.mu.Lock()
	defer c.bucket.mu.Unlock()
	if err := c.copyLocked("move", oldFilename, newFilename); err != nil {
		return err
	}
	delete(c.bucket.objects, c.key(oldFilename))
	return nil
}

// Copy implements filestore.FileProvider.
func (c *Client) Copy(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.bucket.mu.Lock()
	defer c.bucket.mu.Unlock()
	return c.copyLocked("copy", oldFilename, newFilename)
}

// copyLocked duplicates src to dst as a new, independent object, enforcing
// the ErrFileExists / ErrNotFound contract with the destination checked
// first, in the same order as the s3 provider. The caller must hold c.bucket.mu for
// writing.
func (c *Client) copyLocked(op, src, dst string) error {
	if _, taken := c.bucket.objects[c.key(dst)]; taken {
		return opErr(op, dst, filestore.ErrFileExists)
	}
	file, err := c.lookup(op, src)
	if err != nil {
		return err
	}
	c.bucket.objects[c.key(dst)] = file.snapshot()
	return nil
}

// GetPresignedURL implements filestore.PresignedFileProvider. The returned
// URL is a placeholder and is not served by anything.
func (c *Client) GetPresignedURL(ctx context.Context, filename string, _ time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	c.bucket.mu.RLock()
	defer c.bucket.mu.RUnlock()
	if _, err := c.lookup("presign", filename); err != nil {
		return "", err
	}
	return "http://not-a-real-presigned-url.example/" + c.key(filename), nil
}

func opErr(op, filename string, err error) error {
	return fmt.Errorf("mock: %s %q: %w", op, filename, err)
}
