// Package s3 implements filestore.PresignedFileProvider on top of an Amazon
// S3-shaped object API.
//
// The Client does not depend on an AWS SDK directly. It talks to an
// ObjectAPI, a narrow interface shaped after the S3 object operations it
// needs, so a real bucket can be plugged in through a thin adapter while
// tests use an in-memory implementation (see the s3test package).
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

var _ filestore.PresignedFileProvider = (*Client)(nil)

// ErrNoSuchKey is the error an ObjectAPI returns when the requested key does
// not exist in the bucket (S3's NoSuchKey / 404 NotFound). Adapters over a
// real SDK must translate the SDK's error into (an error wrapping)
// ErrNoSuchKey; the Client maps it onto filestore.ErrNotFound.
var ErrNoSuchKey = errors.New("no such key")

// ErrNoAPI is returned by write operations when no ObjectAPI was configured,
// so a missing or mis-wired adapter surfaces at the first Set instead of
// silently discarding data.
var ErrNoAPI = errors.New("no object API configured")

// ObjectAPI is the subset of the S3 object API the Client relies on.
//
// Every method operates on a single bucket. Methods that address an existing
// key must return (an error wrapping) ErrNoSuchKey when that key is absent,
// except DeleteObject, which, like S3, succeeds for missing keys.
type ObjectAPI interface {
	// HeadObject checks that key exists.
	HeadObject(ctx context.Context, bucket, key string) error
	// GetObject returns the object's body and content type.
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, string, error)
	// PutObject stores body under key, replacing any existing object.
	PutObject(ctx context.Context, bucket, key string, body io.Reader, contentType string) error
	// CopyObject duplicates srcKey to dstKey within the bucket, replacing
	// any existing destination object. Metadata such as the content type
	// is copied along with the body.
	CopyObject(ctx context.Context, bucket, srcKey, dstKey string) error
	// DeleteObject removes key. Deleting a missing key is not an error.
	DeleteObject(ctx context.Context, bucket, key string) error
	// PresignGetObject returns a URL granting read access to key until
	// expireAfter has elapsed.
	PresignGetObject(ctx context.Context, bucket, key string, expireAfter time.Duration) (string, error)
}

// Config configures a Client.
type Config struct {
	Bucket string
	// API is the object store to talk to. When nil the Client falls back
	// to a stand-in that behaves like an empty bucket which discards
	// writes, so the module can be wired up without credentials.
	API ObjectAPI
}

// Client stores files as objects in a single S3 bucket.
type Client struct {
	bucket string
	api    ObjectAPI
}

// NewClient returns a Client for config.Bucket.
func NewClient(config Config) *Client {
	api := config.API
	if api == nil {
		api = emptyAPI{}
	}
	return &Client{bucket: config.Bucket, api: api}
}

// Get implements filestore.FileProvider.
func (c *Client) Get(ctx context.Context, filename string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	body, contentType, err := c.api.GetObject(ctx, c.bucket, filename)
	if err != nil {
		return nil, "", opErr("get", filename, err)
	}
	return body, contentType, nil
}

// Set implements filestore.FileProvider.
func (c *Client) Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return opErr("set", filename, c.api.PutObject(ctx, c.bucket, filename, bytes.NewReader(fileBytes), contentType))
}

// Purge implements filestore.FileProvider.
func (c *Client) Purge(ctx context.Context, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return opErr("purge", filename, c.api.DeleteObject(ctx, c.bucket, filename))
}

// Move implements filestore.FileProvider. S3 has no rename, so Move is a
// Copy followed by deleting the source. Should the delete fail the copy is
// left in place and the error reported.
func (c *Client) Move(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.copy(ctx, "move", oldFilename, newFilename); err != nil {
		return err
	}
	return opErr("move", oldFilename, c.api.DeleteObject(ctx, c.bucket, oldFilename))
}

// Copy implements filestore.FileProvider.
func (c *Client) Copy(ctx context.Context, oldFilename, newFilename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.copy(ctx, "copy", oldFilename, newFilename)
}

// copy performs a conflict-checked CopyObject. S3's CopyObject overwrites
// silently, so the destination is probed with HeadObject first; a missing
// source then surfaces from CopyObject itself. The two calls are not atomic:
// a concurrent writer can still win the race between them.
func (c *Client) copy(ctx context.Context, op, src, dst string) error {
	switch err := c.api.HeadObject(ctx, c.bucket, dst); {
	case err == nil:
		return opErr(op, dst, filestore.ErrFileExists)
	case !errors.Is(err, ErrNoSuchKey):
		return opErr(op, dst, err)
	}
	return opErr(op, src, c.api.CopyObject(ctx, c.bucket, src, dst))
}

// GetPresignedURL implements filestore.PresignedFileProvider. Presigning is
// a purely local operation in S3, so the key is probed first to honour the
// ErrNotFound contract shared with the other providers.
func (c *Client) GetPresignedURL(ctx context.Context, filename string, expireAfter time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := c.api.HeadObject(ctx, c.bucket, filename); err != nil {
		return "", opErr("presign", filename, err)
	}
	url, err := c.api.PresignGetObject(ctx, c.bucket, filename, expireAfter)
	return url, opErr("presign", filename, err)
}

// opErr wraps a non-nil ObjectAPI error with the failing operation and key,
// translating ErrNoSuchKey into the provider-neutral filestore.ErrNotFound.
// A nil err is passed through unchanged.
func opErr(op, filename string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNoSuchKey) {
		err = filestore.ErrNotFound
	}
	return fmt.Errorf("s3: %s %q: %w", op, filename, err)
}

// emptyAPI is the ObjectAPI used when none is configured: a bucket that is
// always empty and rejects writes with ErrNoAPI.
type emptyAPI struct{}

func (emptyAPI) HeadObject(context.Context, string, string) error { return ErrNoSuchKey }

func (emptyAPI) GetObject(context.Context, string, string) (io.ReadCloser, string, error) {
	return nil, "", ErrNoSuchKey
}

func (emptyAPI) PutObject(context.Context, string, string, io.Reader, string) error {
	return ErrNoAPI
}

func (emptyAPI) CopyObject(context.Context, string, string, string) error { return ErrNoSuchKey }

func (emptyAPI) DeleteObject(context.Context, string, string) error { return nil }

func (emptyAPI) PresignGetObject(context.Context, string, string, time.Duration) (string, error) {
	return "", ErrNoSuchKey
}
