package browser

import (
	"fmt"
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// Login drives a real browser through the target site's login form and
// classifies the result. It returns nil on success,
// carrierproxy.ErrInvalidCredentials (wrapped with the site's own message)
// when the site rejects the credentials, or the last wrapped error for any
// other failure (browser launch, navigation, missing form elements, a
// timed-out wait — see WithTimeout). Non-credential failures are retried
// (see WithRetries/WithRetryDelay); an invalid-credentials result is
// never retried, since the same credentials would just fail again. On
// success, username/password are remembered so Policies and
// DocumentDownload can re-authenticate.
func (c *Client) Login(username, password string) error {
	if err := validateCredentials(username, password); err != nil {
		return err
	}

	err := c.withRetries(func() error { return c.loginOnce(username, password) })
	if err == nil {
		c.rememberCredentials(username, password)
	}
	return err
}

// loginOnce is a single login attempt: open a fresh page, submit the
// form, confirm the site accepted it, then release the page. It is the
// unit of work withRetries repeats.
func (c *Client) loginOnce(username, password string) error {
	_, closePage, err := c.authenticatedPage(username, password)
	if err != nil {
		return err
	}
	closePage()
	return nil
}

// validateCredentials rejects a blank username or password before a
// browser is ever launched.
func validateCredentials(username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("%w: username and password are required", carrierproxy.ErrInvalidCredentials)
	}
	return nil
}

// fillLoginForm navigates to loginURL and submits the given credentials
// through the form located by opts' selectors.
func fillLoginForm(pg page, loginURL string, opts options, username, password string) error {
	if err := pg.Navigate(loginURL); err != nil {
		return err
	}
	if err := pg.WaitLoad(); err != nil {
		return err
	}
	if err := fillField(pg, opts.usernameSelector, username); err != nil {
		return err
	}
	if err := fillField(pg, opts.passwordSelector, password); err != nil {
		return err
	}
	return clickSubmit(pg, opts.submitSelector)
}

// fillField locates the element matching selector and types value into it.
func fillField(pg page, selector, value string) error {
	el, err := pg.Element(selector)
	if err != nil {
		return err
	}
	return el.Input(value)
}

// clickSubmit locates the element matching selector and clicks it.
func clickSubmit(pg page, selector string) error {
	el, err := pg.Element(selector)
	if err != nil {
		return err
	}
	return el.Click()
}

// evaluateLoginResult reads the post-submit result banner and turns it into
// a Go error, using its CSS class to tell success from failure.
func evaluateLoginResult(pg page, opts options) error {
	el, err := pg.Element(opts.resultSelector)
	if err != nil {
		return fmt.Errorf("carrierproxy: locate result banner: %w", err)
	}

	class, err := el.Attribute("class")
	if err != nil {
		return fmt.Errorf("carrierproxy: read result banner: %w", err)
	}
	if strings.Contains(class, opts.successClass) {
		return nil
	}

	message, textErr := el.Text()
	if textErr != nil {
		message = ""
	}
	return fmt.Errorf("%w: %s", carrierproxy.ErrInvalidCredentials, strings.TrimSpace(message))
}
