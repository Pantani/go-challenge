package browser

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestDocumentOwnsAttemptUntilClose(t *testing.T) {
	c := NewClient(testLoginURL, WithRetries(0), WithDocumentURL(testDocumentURL))
	c.rememberCredentials("alice", "secret")
	var pageCtx, fetchCtx context.Context
	c.newPage = func(ctx context.Context) (page, func(), error) {
		pageCtx = ctx
		return successPage(), func() {}, nil
	}
	c.fetch = func(ctx context.Context, _ string, _ []cookie) (io.ReadCloser, error) {
		fetchCtx = ctx
		return io.NopCloser(strings.NewReader("policy")), nil
	}
	body, err := c.DocumentDownloadContext(context.Background(), "key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if pageCtx != fetchCtx {
		t.Fatal("download reset the attempt context")
	}
	requireAttemptDeadline(t, pageCtx)
	if err := fetchCtx.Err(); err != nil {
		t.Fatalf("body canceled before use: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(fetchCtx.Err(), context.Canceled) {
		t.Fatal("close did not cancel attempt")
	}
}

func TestLoginAttemptDeadlineStartsBeforeLaunch(t *testing.T) {
	c := NewClient(testLoginURL, WithRetries(0), WithTimeout(time.Millisecond))
	c.newPage = waitForAttemptDeadline(t)
	if err := c.LoginContext(context.Background(), "alice", "secret"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestPoliciesAttemptDeadlineStartsBeforeLaunch(t *testing.T) {
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(0), WithTimeout(time.Millisecond))
	c.rememberCredentials("alice", "secret")
	c.newPage = waitForAttemptDeadline(t)
	if _, err := c.PoliciesContext(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestDocumentAttemptDeadlineStartsBeforeLaunch(t *testing.T) {
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(0), WithTimeout(time.Millisecond))
	c.rememberCredentials("alice", "secret")
	c.newPage = waitForAttemptDeadline(t)
	if _, err := c.DocumentDownloadContext(context.Background(), "key"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestAttemptTimeoutRetriedWhileCallerAlive(t *testing.T) {
	c := NewClient(testLoginURL, WithRetries(1), WithRetryDelay(0), WithTimeout(time.Millisecond))
	calls := 0
	c.newPage = func(ctx context.Context) (page, func(), error) {
		calls++
		if calls == 1 {
			<-ctx.Done()
			return nil, nil, ctx.Err()
		}
		return successPage(), func() {}, nil
	}
	if err := c.LoginContext(context.Background(), "alice", "secret"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("attempts = %d, want 2", calls)
	}
}

func waitForAttemptDeadline(t *testing.T) func(context.Context) (page, func(), error) {
	t.Helper()
	return func(ctx context.Context) (page, func(), error) {
		requireAttemptDeadline(t, ctx)
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
}

func requireAttemptDeadline(t *testing.T, ctx context.Context) {
	t.Helper()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("attempt deadline missing before launch")
	}
}

func TestRetryDelayCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewClient(testLoginURL, WithRetries(2), WithRetryDelay(time.Hour))
	calls := 0
	c.wait = func(ctx context.Context, delay time.Duration) error {
		cancel()
		return waitRetry(ctx, delay)
	}
	err := c.withRetries(ctx, func(context.Context) error {
		calls++
		return errors.New("transient")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("attempts = %d", calls)
	}
}

func newCanceledClient(t *testing.T) (*Client, context.Context) {
	t.Helper()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithDocumentURL(testDocumentURL))
	c.rememberCredentials("alice", "secret")
	c.newPage = func(context.Context) (page, func(), error) {
		t.Error("canceled call launched a browser")
		return nil, nil, context.Canceled
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return c, ctx
}

func TestLoginCanceledBeforeLaunch(t *testing.T) {
	c, ctx := newCanceledClient(t)
	if err := c.LoginContext(ctx, "alice", "secret"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPoliciesCanceledBeforeLaunch(t *testing.T) {
	c, ctx := newCanceledClient(t)
	if _, err := c.PoliciesContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDocumentCanceledBeforeLaunch(t *testing.T) {
	c, ctx := newCanceledClient(t)
	body, err := c.DocumentDownloadContext(ctx, "key")
	if body != nil {
		_ = body.Close()
		t.Fatal("canceled call returned a body")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
