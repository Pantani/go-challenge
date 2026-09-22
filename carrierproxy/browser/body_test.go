package browser

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type observedBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *observedBody) Close() error { b.closed = true; return b.closeErr }

func TestCanceledDocumentClosesSuccessfulResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewClient(testLoginURL, WithRetries(0), WithDocumentURL(testDocumentURL))
	c.rememberCredentials("alice", "secret")
	c.newPage = func(context.Context) (page, func(), error) { return successPage(), func() {}, nil }
	raw := &observedBody{Reader: strings.NewReader("policy")}
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) {
		cancel()
		return raw, nil
	}
	body, err := c.DocumentDownloadContext(ctx, "key")
	if body != nil {
		_ = body.Close()
		t.Error("returned body with cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if !raw.closed {
		t.Fatal("successful response leaked when cancellation won")
	}
}

func TestDocumentResultPreservesCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	raw := &observedBody{Reader: strings.NewReader("policy"), closeErr: closeErr}
	body, err := documentResult(raw, context.Canceled)
	if body != nil {
		t.Fatal("returned body with cancellation")
	}
	if !raw.closed {
		t.Fatal("did not close the response")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("lost close error: %v", err)
	}
}

func TestAttemptBodyEOFCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &attemptBody{ReadCloser: io.NopCloser(strings.NewReader("policy")), cancel: cancel}
	t.Cleanup(func() { _ = body.Close() })
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "policy" {
		t.Fatalf("body = %q", got)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("EOF did not cancel attempt")
	}
}

type readErrorBody struct{ err error }

func (b readErrorBody) Read([]byte) (int, error) { return 0, b.err }
func (b readErrorBody) Close() error             { return nil }

func TestAttemptBodyReadErrorCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wantErr := errors.New("read failed")
	body := &attemptBody{ReadCloser: readErrorBody{err: wantErr}, cancel: cancel}
	t.Cleanup(func() { _ = body.Close() })
	_, err := body.Read(make([]byte, 1))
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v", err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("read error did not cancel attempt")
	}
}
