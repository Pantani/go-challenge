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
// if it returns a non-nil error other than carrierproxy.ErrInvalidCredentials
// (which is never retried, since the same credentials would just fail
// again), waiting opts.retryDelay between attempts.
func (c *Client) withRetries(attempt func() error) error {
	var err error
	for i := 0; i < attemptBudget(c.opts.retries); i++ {
		if i > 0 {
			c.sleep(c.opts.retryDelay)
		}
		if err = attempt(); err == nil || errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			return err
		}
	}
	return err
}

// attemptBudget returns how many attempts to make for a given retries
// count, always at least 1 even if retries is negative.
func attemptBudget(retries int) int {
	if retries < 0 {
		return 1
	}
	return retries + 1
}
