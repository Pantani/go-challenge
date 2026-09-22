package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

// memAPI is a minimal single-bucket ObjectAPI used to drive the Client's
// error handling from inside the package. It records the bucket passed to
// every call and can make any method fail with a chosen error.
type memAPI struct {
	objects map[string]string
	errs    map[string]error
	buckets []string
}

func newMemAPI() *memAPI { return &memAPI{objects: map[string]string{}, errs: map[string]error{}} }

func (m *memAPI) call(method, bucket string) error {
	m.buckets = append(m.buckets, bucket)
	return m.errs[method]
}

func (m *memAPI) HeadObject(_ context.Context, bucket, key string) error {
	if err := m.call("HeadObject", bucket); err != nil {
		return err
	}
	if _, ok := m.objects[key]; !ok {
		return ErrNoSuchKey
	}
	return nil
}

func (m *memAPI) GetObject(_ context.Context, bucket, key string) (io.ReadCloser, string, error) {
	if err := m.call("GetObject", bucket); err != nil {
		return nil, "", err
	}
	data, ok := m.objects[key]
	if !ok {
		return nil, "", ErrNoSuchKey
	}
	return io.NopCloser(strings.NewReader(data)), "text/plain", nil
}

func (m *memAPI) PutObject(_ context.Context, bucket, key string, body io.Reader, _ string) error {
	if err := m.call("PutObject", bucket); err != nil {
		return err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.objects[key] = string(data)
	return nil
}

func (m *memAPI) CopyObject(_ context.Context, bucket, src, dst string) error {
	if err := m.call("CopyObject", bucket); err != nil {
		return err
	}
	data, ok := m.objects[src]
	if !ok {
		return ErrNoSuchKey
	}
	m.objects[dst] = data
	return nil
}

func (m *memAPI) DeleteObject(_ context.Context, bucket, key string) error {
	if err := m.call("DeleteObject", bucket); err != nil {
		return err
	}
	delete(m.objects, key)
	return nil
}

func (m *memAPI) PresignGetObject(_ context.Context, bucket, key string, _ time.Duration) (string, error) {
	if err := m.call("PresignGetObject", bucket); err != nil {
		return "", err
	}
	return "https://" + bucket + "/" + key, nil
}

var errBoom = errors.New("boom")

func TestAPIErrorsArePropagated(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		method string
		call   func(*Client) error
	}{
		{"Get", "GetObject", func(c *Client) error { _, _, err := c.Get(ctx, "a"); return err }},
		{"Set", "PutObject", func(c *Client) error { return c.Set(ctx, "a", []byte("x"), "") }},
		{"Purge", "DeleteObject", func(c *Client) error { return c.Purge(ctx, "a") }},
		{"Copy head", "HeadObject", func(c *Client) error { return c.Copy(ctx, "a", "b") }},
		{"Copy copy", "CopyObject", func(c *Client) error { return c.Copy(ctx, "a", "b") }},
		{"Move head", "HeadObject", func(c *Client) error { return c.Move(ctx, "a", "b") }},
		{"Move copy", "CopyObject", func(c *Client) error { return c.Move(ctx, "a", "b") }},
		{"Move delete", "DeleteObject", func(c *Client) error { return c.Move(ctx, "a", "b") }},
		{"Presign head", "HeadObject", func(c *Client) error { _, err := c.GetPresignedURL(ctx, "a", time.Minute); return err }},
		{"Presign sign", "PresignGetObject", func(c *Client) error { _, err := c.GetPresignedURL(ctx, "a", time.Minute); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMemAPI()
			api.objects["a"] = "data"
			api.errs[tt.method] = errBoom
			client := NewClient(Config{Bucket: "my-bucket", API: api})

			assertAdapterError(t, tt.call(client))
			assertAPIBuckets(t, api.buckets)
		})
	}
}

func assertAdapterError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want wrapped errBoom", err)
	}
	if errors.Is(err, filestore.ErrNotFound) || errors.Is(err, filestore.ErrFileExists) {
		t.Fatalf("unrelated error mapped to sentinel: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "s3: ") {
		t.Fatalf("missing operation context: %v", err)
	}
}

func assertAPIBuckets(t *testing.T, buckets []string) {
	t.Helper()
	if len(buckets) == 0 {
		t.Fatal("no adapter calls observed")
	}
	for _, bucket := range buckets {
		if bucket != "my-bucket" {
			t.Fatalf("adapter bucket = %q", bucket)
		}
	}
}

func TestMoveDeleteFailureLeavesCopy(t *testing.T) {
	api := newMemAPI()
	api.objects["a"] = "data"
	api.errs["DeleteObject"] = errBoom
	client := NewClient(Config{Bucket: "b", API: api})

	if err := client.Move(context.Background(), "a", "b"); !errors.Is(err, errBoom) {
		t.Fatalf("error = %v", err)
	}
	if api.objects["a"] != "data" || api.objects["b"] != "data" {
		t.Fatalf("objects = %v; expected both a and b after a failed source delete", api.objects)
	}
}

func TestWrappedNoSuchKeyMapsToNotFound(t *testing.T) {
	api := newMemAPI()
	api.errs["GetObject"] = errors.Join(errors.New("adapter"), ErrNoSuchKey)
	client := NewClient(Config{Bucket: "b", API: api})

	_, _, err := client.Get(context.Background(), "a")
	if !errors.Is(err, filestore.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if want := `s3: get "a": file not found`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestDefaultAPIIsAnEmptyBucket(t *testing.T) {
	ctx := context.Background()
	client := NewClient(Config{Bucket: "b"})

	if err := client.Set(ctx, "a", []byte("x"), "text/plain"); !errors.Is(err, ErrNoAPI) {
		t.Fatalf("Set error = %v, want ErrNoAPI", err)
	}
	if err := client.Purge(ctx, "a"); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	notFound := []struct {
		name string
		call func() error
	}{
		{"Get", func() error { _, _, err := client.Get(ctx, "a"); return err }},
		{"Copy", func() error { return client.Copy(ctx, "a", "b") }},
		{"Move", func() error { return client.Move(ctx, "a", "b") }},
		{"GetPresignedURL", func() error { _, err := client.GetPresignedURL(ctx, "a", time.Minute); return err }},
	}
	for _, tt := range notFound {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, filestore.ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

// TestEmptyAPI pins the fallback's behaviour method by method; through the
// Client, PresignGetObject is unreachable because HeadObject fails first.
func TestEmptyAPI(t *testing.T) {
	ctx := context.Background()
	var api ObjectAPI = emptyAPI{}

	missing := map[string]func() error{
		"HeadObject": func() error { return api.HeadObject(ctx, "b", "k") },
		"GetObject":  func() error { _, _, err := api.GetObject(ctx, "b", "k"); return err },
		"CopyObject": func() error { return api.CopyObject(ctx, "b", "k", "j") },
		"PresignGetObject": func() error {
			_, err := api.PresignGetObject(ctx, "b", "k", time.Minute)
			return err
		},
	}
	for name, call := range missing {
		if err := call(); !errors.Is(err, ErrNoSuchKey) {
			t.Errorf("%s error = %v, want ErrNoSuchKey", name, err)
		}
	}
	if err := api.PutObject(ctx, "b", "k", strings.NewReader("x"), ""); !errors.Is(err, ErrNoAPI) {
		t.Errorf("PutObject error = %v, want ErrNoAPI", err)
	}
	if err := api.DeleteObject(ctx, "b", "k"); err != nil {
		t.Errorf("DeleteObject error = %v", err)
	}
}
