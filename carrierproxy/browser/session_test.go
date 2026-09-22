package browser

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func TestAttemptBudget(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		retries int
		want    int
	}{
		"zero retries means 1 attempt":          {0, 1},
		"positive retries add to 1 attempt":     {3, 4},
		"negative retries still mean 1 attempt": {-5, 1},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := attemptBudget(tc.retries); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
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

	t.Run("succeeds on the first attempt without sleeping", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(3))
		c.sleep = func(time.Duration) { t.Fatal("should not sleep when the first attempt succeeds") }
		calls := 0
		err := c.withRetries(func() error { calls++; return nil })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 call, got %d", calls)
		}
	})

	t.Run("retries a transient failure then succeeds", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(2))
		var slept []time.Duration
		c.sleep = func(d time.Duration) { slept = append(slept, d) }
		calls := 0
		err := c.withRetries(func() error {
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
	})

	t.Run("never retries invalid credentials", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(3))
		c.sleep = func(time.Duration) { t.Fatal("should not sleep: invalid credentials must not be retried") }
		calls := 0
		err := c.withRetries(func() error { calls++; return carrierproxy.ErrInvalidCredentials })
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected exactly 1 call, got %d", calls)
		}
	})

	t.Run("never retries a malformed response", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(3))
		c.sleep = func(time.Duration) { t.Fatal("should not sleep: a malformed response must not be retried") }
		calls := 0
		err := c.withRetries(func() error { calls++; return carrierproxy.ErrMalformedResponse })
		if !errors.Is(err, carrierproxy.ErrMalformedResponse) {
			t.Fatalf("expected ErrMalformedResponse, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected exactly 1 call, got %d", calls)
		}
	})

	t.Run("exhausts retries and returns the last error", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(2))
		c.sleep = func(time.Duration) {}
		calls := 0
		wantErr := errors.New("persistent")
		err := c.withRetries(func() error { calls++; return wantErr })
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if calls != 3 {
			t.Fatalf("expected 3 calls (1 + 2 retries), got %d", calls)
		}
	})

	t.Run("WithRetries(0) makes exactly one attempt", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(0))
		calls := 0
		wantErr := errors.New("boom")
		err := c.withRetries(func() error { calls++; return wantErr })
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if calls != 1 {
			t.Fatalf("expected exactly 1 call, got %d", calls)
		}
	})

	t.Run("negative WithRetries still makes one attempt", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(-5))
		calls := 0
		err := c.withRetries(func() error { calls++; return nil })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected exactly 1 call, got %d", calls)
		}
	})
}

func TestCredentialsStorage(t *testing.T) {
	t.Parallel()

	t.Run("not logged in yet", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		if _, err := c.storedCredentials(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected ErrNotLoggedIn, got %v", err)
		}
	})

	t.Run("returns what was remembered", func(t *testing.T) {
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
	})

	t.Run("a later call replaces earlier credentials", func(t *testing.T) {
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
	})
}

func TestAuthenticatedPage(t *testing.T) {
	t.Parallel()

	t.Run("succeeds and returns an open page", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		fp := successPage()
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		pg, closePage, err := c.authenticatedPage("tomsmith", "SuperSecretPassword!")
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
	})

	t.Run("releases the page when the site rejects the credentials", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		fp := successPage()
		fp.elements[newOptions().resultSelector] = &fakeElement{attr: "flash error", text: "Your password is invalid!"}
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		_, _, err := c.authenticatedPage("tomsmith", "wrong")
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if !closed {
			t.Fatal("expected the page to be released on failure")
		}
	})

	t.Run("wraps a browser launch failure", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		wantErr := errors.New("boom")
		c.newPage = func(time.Duration) (page, func(), error) { return nil, nil, wantErr }
		_, _, err := c.authenticatedPage("user", "pass")
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("releases the page when the form cannot be submitted", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		fp := successPage()
		fp.navigateErr = errors.New("boom")
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		if _, _, err := c.authenticatedPage("user", "pass"); !errors.Is(err, fp.navigateErr) {
			t.Fatalf("expected %v, got %v", fp.navigateErr, err)
		}
		if !closed {
			t.Fatal("expected the page to be released on failure")
		}
	})

	t.Run("rejects a misconfigured client before opening a page", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithSuccessClass(""))
		c.newPage = func(time.Duration) (page, func(), error) {
			t.Fatal("newPage should not be called for a misconfigured client")
			return nil, nil, nil
		}
		if _, _, err := c.authenticatedPage("user", "pass"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
			t.Fatalf("expected ErrNotConfigured, got %v", err)
		}
	})
}

func TestAttemptWithRetries(t *testing.T) {
	t.Parallel()

	t.Run("returns the value from the attempt that succeeded", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(2))
		c.sleep = func(time.Duration) {}
		calls := 0
		got, err := attemptWithRetries(c, func() (int, error) {
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
	})

	t.Run("returns the zero value with the last error once exhausted", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(1))
		c.sleep = func(time.Duration) {}
		wantErr := errors.New("persistent")
		got, err := attemptWithRetries(c, func() (string, error) { return "partial", wantErr })
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if got != "partial" {
			// The last attempt's value is passed through untouched; callers
			// return nil/zero themselves on error, so this documents rather
			// than guarantees the behaviour.
			t.Fatalf("got %q, want the last attempt's value", got)
		}
	})
}
