// Package s3test provides an in-memory s3.ObjectAPI for exercising the s3
// Client without a real bucket.
package s3test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3"
)

var _ s3.ObjectAPI = (*MemoryAPI)(nil)

type object struct {
	data        []byte
	contentType string
}

// MemoryAPI is an s3.ObjectAPI that keeps objects in memory, keyed by
// bucket and key. The zero value is ready to use and safe for concurrent
// use.
type MemoryAPI struct {
	mu      sync.RWMutex
	objects map[string]object
}

func objectKey(bucket, key string) string { return bucket + "/" + key }

// get returns the object stored under bucket/key, or an ErrNoSuchKey error.
func (m *MemoryAPI) get(bucket, key string) (object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[objectKey(bucket, key)]
	if !ok {
		return object{}, fmt.Errorf("%s/%s: %w", bucket, key, s3.ErrNoSuchKey)
	}
	return obj, nil
}

// put stores obj under bucket/key, replacing any existing object.
func (m *MemoryAPI) put(bucket, key string, obj object) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.objects == nil {
		m.objects = map[string]object{}
	}
	m.objects[objectKey(bucket, key)] = obj
}

// HeadObject implements s3.ObjectAPI.
func (m *MemoryAPI) HeadObject(ctx context.Context, bucket, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := m.get(bucket, key)
	return err
}

// GetObject implements s3.ObjectAPI.
func (m *MemoryAPI) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	obj, err := m.get(bucket, key)
	if err != nil {
		return nil, "", err
	}
	return io.NopCloser(bytes.NewReader(obj.data)), obj.contentType, nil
}

// PutObject implements s3.ObjectAPI.
func (m *MemoryAPI) PutObject(ctx context.Context, bucket, key string, body io.Reader, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.put(bucket, key, object{data: data, contentType: contentType})
	return nil
}

// CopyObject implements s3.ObjectAPI. Like S3 it overwrites the destination.
func (m *MemoryAPI) CopyObject(ctx context.Context, bucket, srcKey, dstKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	obj, err := m.get(bucket, srcKey)
	if err != nil {
		return err
	}
	m.put(bucket, dstKey, obj)
	return nil
}

// DeleteObject implements s3.ObjectAPI.
func (m *MemoryAPI) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.objects, objectKey(bucket, key))
	m.mu.Unlock()
	return nil
}

// PresignGetObject implements s3.ObjectAPI. Like S3 it does not check that
// the key exists.
func (m *MemoryAPI) PresignGetObject(ctx context.Context, bucket, key string, expireAfter time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s.s3.example/%s?X-Amz-Expires=%d", bucket, key, int64(expireAfter.Seconds())), nil
}
