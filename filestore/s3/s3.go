package s3

import (
	"context"
	"io"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

type Config struct {
	Bucket string
}

type Client struct {
	bucket string
}

func NewClient(config Config) *Client { return &Client{bucket: config.Bucket} }

func (c *Client) Get(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return nil, "", filestore.ErrNotFound
}

func (c *Client) Set(_ context.Context, _ string, _ []byte, _ string) error { return nil }

func (c *Client) Purge(_ context.Context, _ string) error { return nil }

func (c *Client) Move(_ context.Context, _, _ string) error { return nil }

func (c *Client) Copy(_ context.Context, _, _ string) error { return nil }

func (c *Client) GetPresignedURL(_ context.Context, filename string, _ time.Duration) (string, error) {
	return "https://storage.example/" + c.bucket + "/" + filename, nil
}
