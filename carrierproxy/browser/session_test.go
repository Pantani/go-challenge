package browser

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func TestRetryBudgetOverflow(t *testing.T) {
	c := NewClient(testLoginURL, WithRetries(int(^uint(0)>>1)))
	calls := 0
	c.newPage = func(context.Context) (page, func(), error) {
		calls++
		return successPage(), func() {}, nil
	}

	err := c.Login("alice", "secret")
	if !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
	if calls != 0 {
		t.Fatalf("launches = %d, want 0", calls)
	}
}

func TestAttemptBudget(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		retries int
		want    int
	}{
		"zero retries means 1 attempt":                 {0, 1},
		"positive retries add to 1 attempt":            {3, 4},
		"negative retries still mean 1 attempt":        {-5, 1},
		"largest valid retries adds to largest budget": {int(^uint(0)>>1) - 1, int(^uint(0) >> 1)},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := attemptBudget(tc.retries)
			if err != nil {
				t.Fatalf("attemptBudget(%d) error = %v, want nil", tc.retries, err)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAttemptBudgetRejectsOverflow(t *testing.T) {
	_, err := attemptBudget(int(^uint(0) >> 1))
	if !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}

func TestIsFinal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		err  error
		want bool
	}{
		"invalid credentials":                      {carrierproxy.ErrInvalidCredentials, true},
		"malformed response":                       {carrierproxy.ErrMalformedResponse, true},
		"wrapped invalid credentials":              {fmt.Errorf("submit login form: %w", carrierproxy.ErrInvalidCredentials), true},
		"not configured":                           {carrierproxy.ErrNotConfigured, true},
		"wrapped not configured":                   {fmt.Errorf("x: %w", carrierproxy.ErrNotConfigured), true},
		"nil":                                      {nil, false},
		"an unrelated error":                       {errors.New("boom"), false},
		"not logged in is retryable at this level": {carrierproxy.ErrNotLoggedIn, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := isFinal(tc.err); got != tc.want {
				t.Fatalf("isFinal(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestClientWithRetries(t *testing.T) {
	t.Parallel()

	t.Run("succeeds on the first attempt without sleeping", testClientWithRetriesFirstAttempt)

	t.Run("retries a transient failure then succeeds", testClientWithRetriesTransientFailure)

	t.Run("never retries invalid credentials", testClientWithRetriesInvalidCredentials)

	t.Run("never retries a malformed response", testClientWithRetriesMalformedResponse)

	t.Run("exhausts retries and returns the last error", testClientWithRetriesExhausted)

	t.Run("WithRetries(0) makes exactly one attempt", testClientWithRetriesZeroRetries)

	t.Run("negative WithRetries still makes one attempt", testClientWithRetriesNegativeRetries)
}

func testClientWithRetriesFirstAttempt(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(3))
	c.wait = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep when the first attempt succeeds")
		return nil
	}
	calls := 0
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func testClientWithRetriesTransientFailure(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(2))
	var slept []time.Duration
	c.wait = func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil }
	calls := 0
	err := c.withRetries(context.Background(), func(context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if len(slept) != 1 || slept[0] != c.opts.retryDelay {
		t.Fatalf("expected exactly 1 sleep of %v, got %v", c.opts.retryDelay, slept)
	}
}

func testClientWithRetriesInvalidCredentials(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(3))
	c.wait = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep: invalid credentials must not be retried")
		return nil
	}
	calls := 0
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return carrierproxy.ErrInvalidCredentials })
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func testClientWithRetriesMalformedResponse(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(3))
	c.wait = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep: a malformed response must not be retried")
		return nil
	}
	calls := 0
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return carrierproxy.ErrMalformedResponse })
	if !errors.Is(err, carrierproxy.ErrMalformedResponse) {
		t.Fatalf("expected ErrMalformedResponse, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func testClientWithRetriesExhausted(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(2))
	c.wait = func(context.Context, time.Duration) error { return nil }
	calls := 0
	wantErr := errors.New("persistent")
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func testClientWithRetriesZeroRetries(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(0))
	calls := 0
	wantErr := errors.New("boom")
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func testClientWithRetriesNegativeRetries(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(-5))
	calls := 0
	err := c.withRetries(context.Background(), func(context.Context) error { calls++; return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestCredentialsStorage(t *testing.T) {
	t.Parallel()

	t.Run("not logged in yet", testCredentialsStorageMissing)

	t.Run("returns what was remembered", testCredentialsStorageRemembered)

	t.Run("a later call replaces earlier credentials", testCredentialsStorageReplaced)
}

func testCredentialsStorageMissing(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	if _, err := c.storedCredentials(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func testCredentialsStorageRemembered(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	c.rememberCredentials("user", "pass")
	got, err := c.storedCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.username != "user" || got.password != "pass" {
		t.Fatalf("got %+v, want user/pass", got)
	}
}

func testCredentialsStorageReplaced(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	c.rememberCredentials("first", "pass1")
	c.rememberCredentials("second", "pass2")
	got, err := c.storedCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.username != "second" {
		t.Fatalf("got %q, want %q", got.username, "second")
	}
}

func TestAuthenticatedPage(t *testing.T) {
	t.Parallel()

	t.Run("succeeds and returns an open page", testAuthenticatedPageSuccess)

	t.Run("releases the page when the site rejects the credentials", testAuthenticatedPageRejectedCredentials)

	t.Run("wraps a browser launch failure", testAuthenticatedPageLaunchFailure)

	t.Run("releases the page when the form cannot be submitted", testAuthenticatedPageFormFailure)

	t.Run("rejects a misconfigured client before opening a page", testAuthenticatedPageMisconfigured)
}

func testAuthenticatedPageSuccess(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	fp := successPage()
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	pg, closePage, err := c.authenticatedPage(context.Background(), "tomsmith", "SuperSecretPassword!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg == nil {
		t.Fatal("expected a non-nil page")
	}
	closePage()
	if !closed {
		t.Fatal("expected closePage to release the page")
	}
}

func testAuthenticatedPageRejectedCredentials(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	fp := successPage()
	fp.elements[newOptions().resultSelector] = &fakeElement{attr: "flash error", text: "Your password is invalid!"}
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	_, _, err := c.authenticatedPage(context.Background(), "tomsmith", "wrong")
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if !closed {
		t.Fatal("expected the page to be released on failure")
	}
}

func testAuthenticatedPageLaunchFailure(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	wantErr := errors.New("boom")
	c.newPage = func(context.Context) (page, func(), error) { return nil, nil, wantErr }
	_, _, err := c.authenticatedPage(context.Background(), "user", "pass")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testAuthenticatedPageFormFailure(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	fp := successPage()
	fp.navigateErr = errors.New("boom")
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	if _, _, err := c.authenticatedPage(context.Background(), "user", "pass"); !errors.Is(err, fp.navigateErr) {
		t.Fatalf("expected %v, got %v", fp.navigateErr, err)
	}
	if !closed {
		t.Fatal("expected the page to be released on failure")
	}
}

func testAuthenticatedPageMisconfigured(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithSuccessClass(""))
	c.newPage = func(context.Context) (page, func(), error) {
		t.Fatal("newPage should not be called for a misconfigured client")
		return nil, nil, nil
	}
	if _, _, err := c.authenticatedPage(context.Background(), "user", "pass"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestAttemptWithRetries(t *testing.T) {
	t.Parallel()

	t.Run("returns the value from the attempt that succeeded", testAttemptWithRetriesSuccessValue)

	t.Run("returns the last attempt value with the last error once exhausted", testAttemptWithRetriesLastValue)
}

func testAttemptWithRetriesSuccessValue(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(2))
	c.wait = func(context.Context, time.Duration) error { return nil }
	calls := 0
	got, err := attemptWithRetries(context.Background(), c, func(context.Context) (int, error) {
		calls++
		if calls < 2 {
			return -1, errors.New("transient")
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 || calls != 2 {
		t.Fatalf("got %d after %d calls, want 42 after 2", got, calls)
	}
}

func testAttemptWithRetriesLastValue(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(1))
	c.wait = func(context.Context, time.Duration) error { return nil }
	wantErr := errors.New("persistent")
	got, err := attemptWithRetries(context.Background(), c, func(context.Context) (string, error) { return "partial", wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != "partial" {
		// The last attempt's value is passed through untouched; callers
		// return nil/zero themselves on error, so this documents rather
		// than guarantees the behaviour.
		t.Fatalf("got %q, want the last attempt's value", got)
	}
}
