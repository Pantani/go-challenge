package mock

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"time"
)

type Bucket struct {
	Name    string
	Objects map[string]*MemoryFile
}

type Config struct {
	Bucket   Bucket
	BasePath string
}

type Client struct {
	bucket   Bucket
	basePath string
}

func NewClient(config Config) *Client {
	if config.Bucket.Objects == nil {
		config.Bucket.Objects = map[string]*MemoryFile{}
	}
	return &Client{bucket: config.Bucket, basePath: config.BasePath}
}

func (c *Client) key(filename string) string { return filepath.Join(c.basePath, filename) }

func (c *Client) Get(_ context.Context, filename string) (io.ReadCloser, string, error) {
	file, ok := c.bucket.Objects[c.key(filename)]
	if !ok {
		return nil, "", errors.New("not found")
	}
	return file, "application/octet-stream", nil
}

func (c *Client) Set(_ context.Context, filename string, fileBytes []byte, _ string) error {
	file := &MemoryFile{}
	if _, err := io.Copy(file, bytes.NewReader(fileBytes)); err != nil {
		return err
	}
	c.bucket.Objects[c.key(filename)] = file
	return nil
}

func (c *Client) Purge(_ context.Context, filename string) error {
	delete(c.bucket.Objects, c.key(filename))
	return nil
}

func (c *Client) Move(_ context.Context, oldFilename, newFilename string) error {
	file, ok := c.bucket.Objects[c.key(oldFilename)]
	if !ok {
		return errors.New("not found")
	}
	c.bucket.Objects[c.key(newFilename)] = file
	delete(c.bucket.Objects, c.key(oldFilename))
	return nil
}

func (c *Client) GetPresignedURL(_ context.Context, filename string, _ time.Duration) (string, error) {
	if _, ok := c.bucket.Objects[c.key(filename)]; !ok {
		return "", errors.New("not found")
	}
	return "http://not-a-real-presigned-url.example/" + c.key(filename), nil
}
