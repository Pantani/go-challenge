package filestore

import (
	"context"
	"io"
	"time"
)

type FileProvider interface {
	Get(ctx context.Context, filename string) (io.ReadCloser, string, error)
	Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error
	Purge(ctx context.Context, filename string) error
	Move(ctx context.Context, oldFilename, newFilename string) error
}

type PresignedFileProvider interface {
	FileProvider
	GetPresignedURL(ctx context.Context, filename string, expireAfter time.Duration) (string, error)
}
