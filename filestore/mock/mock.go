// Package mock provides an in-memory filestore.PresignedFileProvider.
package mock

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

var _ filestore.PresignedFileProvider = (*Client)(nil)

// Bucket is the in-memory object store backing a Client.
type Bucket struct {
	// Name identifies the bucket; the mock does not otherwise use it.
	Name string
	// Objects holds the stored files by object key (see Client.key). A nil
	// map is allocated by NewClient.
	Objects map[string]*MemoryFile
}

// Config configures a mock Client.
type Config struct {
	Bucket   Bucket
	BasePath string
}

// Client is an in-memory file store. It is safe for concurrent use.
type Client struct {
	mu       sync.RWMutex
	bucket   Bucket
	basePath string
}

// NewClient returns a Client backed by config.Bucket, allocating the object
// map when the bucket has none.
func NewClient(config Config) *Client {
	if config.Bucket.Objects == nil {
		config.Bucket.Objects = map[string]*MemoryFile{}
	}
	return &Client{bucket: config.Bucket, basePath: config.BasePath}
}

// key maps a filename to its object key under the configured base path.
func (c *Client) key(filename string) string { return path.Join(c.basePath, filename) }

// lookup returns the stored object for filename, or an ErrNotFound error
// naming op. The caller must hold c.mu.
func (c *Client) lookup(op, filename string) (*MemoryFile, error) {
	file, ok := c.bucket.Objects[c.key(filename)]
	if !ok {
		return nil, opErr(op, filename, filestore.ErrNotFound)
	}
	return file, nil
}

// Get implements filestore.FileProvider. It returns an independent snapshot
// of the stored bytes, so closing the reader never disturbs the store.
func (c *Client) Get(ctx context.Context, filename string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
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
	c.mu.Lock()
	c.bucket.Objects[c.key(filename)] = NewMemoryFile(fileBytes, contentType)
	c.mu.Unlock()
	return nil
}

// Purge implements filestore.FileProvider.
func (c *Client) Purge(ctx context.Context, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.bucket.Objects, c.key(filename))
	c.mu.Unlock()
	return nil
}

// Move implements filestore.FileProvider. The rename is atomic with respect
// to other Client operations.
func (c *Client) Move(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.copyLocked("move", oldFilename, newFilename); err != nil {
		return err
	}
	delete(c.bucket.Objects, c.key(oldFilename))
	return nil
}

// Copy implements filestore.FileProvider.
func (c *Client) Copy(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.copyLocked("copy", oldFilename, newFilename)
}

// copyLocked duplicates src to dst as a new, independent object, enforcing
// the ErrFileExists / ErrNotFound contract with the destination checked
// first, in the same order as the s3 provider. The caller must hold c.mu for
// writing.
func (c *Client) copyLocked(op, src, dst string) error {
	if _, taken := c.bucket.Objects[c.key(dst)]; taken {
		return opErr(op, dst, filestore.ErrFileExists)
	}
	file, err := c.lookup(op, src)
	if err != nil {
		return err
	}
	c.bucket.Objects[c.key(dst)] = file.snapshot()
	return nil
}

// GetPresignedURL implements filestore.PresignedFileProvider. The returned
// URL is a placeholder and is not served by anything.
func (c *Client) GetPresignedURL(ctx context.Context, filename string, _ time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, err := c.lookup("presign", filename); err != nil {
		return "", err
	}
	return "http://not-a-real-presigned-url.example/" + c.key(filename), nil
}

// opErr wraps err with the failing operation and filename.
func opErr(op, filename string, err error) error {
	return fmt.Errorf("mock: %s %q: %w", op, filename, err)
}
