package browser

import (
	"errors"
	"fmt"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// credentials is the username/password Login last succeeded with, reused
// by Policies and DocumentDownload to re-authenticate.
type credentials struct {
	username, password string
}

// rememberCredentials stores creds for later reuse by Policies and
// DocumentDownload, replacing whatever was stored before. Safe for
// concurrent use.
func (c *Client) rememberCredentials(username, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.creds = &credentials{username: username, password: password}
}

// storedCredentials returns the credentials from the last successful
// Login, or carrierproxy.ErrNotLoggedIn if Login has not yet succeeded.
// Safe for concurrent use.
func (c *Client) storedCredentials() (credentials, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.creds == nil {
		return credentials{}, carrierproxy.ErrNotLoggedIn
	}
	return *c.creds, nil
}

// authenticatedPage opens a fresh browser page, submits the login form
// with username/password, and confirms the site accepted them. On success
// the caller owns the returned page and must call the returned func to
// release it; on failure the page is already released.
func (c *Client) authenticatedPage(username, password string) (page, func(), error) {
	if err := validateLoginOptions(c.opts); err != nil {
		return nil, nil, err
	}

	pg, closePage, err := c.newPage(c.opts.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("carrierproxy: open browser page: %w", err)
	}

	if err := fillLoginForm(pg, c.loginURL, c.opts, username, password); err != nil {
		closePage()
		return nil, nil, fmt.Errorf("carrierproxy: submit login form: %w", err)
	}

	if err := evaluateLoginResult(pg, c.opts); err != nil {
		closePage()
		return nil, nil, err
	}

	return pg, closePage, nil
}

// withRetries calls attempt, retrying up to opts.retries additional times
// if it returns a non-nil, non-final error (see isFinal), waiting
// opts.retryDelay between attempts.
func (c *Client) withRetries(attempt func() error) error {
	var err error
	for i := 0; i < attemptBudget(c.opts.retries); i++ {
		if i > 0 {
			c.sleep(c.opts.retryDelay)
		}
		if err = attempt(); err == nil || isFinal(err) {
			return err
		}
	}
	return err
}

// attemptWithRetries is withRetries for an attempt that also produces a
// value: it returns the value from the attempt that succeeded, or the zero
// value alongside the last error once retries are exhausted.
func attemptWithRetries[T any](c *Client, attempt func() (T, error)) (T, error) {
	var result T
	err := c.withRetries(func() error {
		var attemptErr error
		result, attemptErr = attempt()
		return attemptErr
	})
	return result, err
}

// finalErrors are the failures a retry would only repeat:
// carrierproxy.ErrInvalidCredentials means the same credentials would fail
// the same way, carrierproxy.ErrMalformedResponse means the target's
// response didn't match the expected shape, which retrying won't reshape,
// and carrierproxy.ErrNotConfigured means the Client itself is missing
// something no attempt can supply.
var finalErrors = []error{
	carrierproxy.ErrInvalidCredentials,
	carrierproxy.ErrMalformedResponse,
	carrierproxy.ErrNotConfigured,
}

// isFinal reports whether err is (or wraps) one of finalErrors, i.e. would
// just happen again on a retry.
func isFinal(err error) bool {
	for _, final := range finalErrors {
		if errors.Is(err, final) {
			return true
		}
	}
	return false
}

// attemptBudget returns how many attempts to make for a given retries
// count, always at least 1 even if retries is negative.
func attemptBudget(retries int) int {
	if retries < 0 {
		return 1
	}
	return retries + 1
}
