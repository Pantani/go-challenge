package browser

import (
	"context"
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
func (c *Client) authenticatedPage(ctx context.Context, username, password string) (page, func(), error) {
	if err := validateLoginOptions(c.opts); err != nil {
		return nil, nil, err
	}

	pg, closePage, err := c.newPage(ctx)
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
func (c *Client) withRetries(ctx context.Context, attempt func(context.Context) error) error {
	budget, err := attemptBudget(c.opts.retries)
	if err != nil {
		return err
	}

	for i := 0; i < budget; i++ {
		if err = c.beforeAttempt(ctx, i); err != nil {
			return err
		}
		err = contextError(ctx, attempt(ctx))
		if err == nil || isFinal(err) {
			return err
		}
	}
	return err
}

// attemptWithRetries is withRetries for an attempt that also produces a
// value: it returns the last attempt's value alongside its error.
func attemptWithRetries[T any](ctx context.Context, c *Client, attempt func(context.Context) (T, error)) (T, error) {
	var result T
	err := c.withRetries(ctx, func(ctx context.Context) error {
		var attemptErr error
		result, attemptErr = attempt(ctx)
		return attemptErr
	})
	return result, err
}
