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

type Bucket struct {
	Name    string
	Objects map[string]*MemoryFile
}

type Config struct {
	Bucket   Bucket
	BasePath string
}

type Client struct {
	mu       sync.RWMutex
	bucket   Bucket
	basePath string
}

func NewClient(config Config) *Client {
	if config.Bucket.Objects == nil {
		config.Bucket.Objects = map[string]*MemoryFile{}
	}
	return &Client{bucket: config.Bucket, basePath: config.BasePath}
}

func (c *Client) key(filename string) string { return path.Join(c.basePath, filename) }

func (c *Client) lookup(op, filename string) (*MemoryFile, error) {
	file, ok := c.bucket.Objects[c.key(filename)]
	if !ok {
		return nil, opErr(op, filename, filestore.ErrNotFound)
	}
	return file, nil
}

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

func (c *Client) Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.bucket.Objects[c.key(filename)] = NewMemoryFile(fileBytes, contentType)
	c.mu.Unlock()
	return nil
}

func (c *Client) Purge(ctx context.Context, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.bucket.Objects, c.key(filename))
	c.mu.Unlock()
	return nil
}

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

func (c *Client) Copy(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.copyLocked("copy", oldFilename, newFilename)
}

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

func opErr(op, filename string, err error) error {
	return fmt.Errorf("mock: %s %q: %w", op, filename, err)
}
