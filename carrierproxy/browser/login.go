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
// timed-out wait — see WithTimeout).
func (c *Client) Login(username, password string) error {
	if err := validateCredentials(username, password); err != nil {
		return err
	}
	if err := validateLoginOptions(c.opts); err != nil {
		return err
	}
	pg, closePage, err := c.newPage(c.opts.timeout)
	if err != nil {
		return fmt.Errorf("carrierproxy: launch browser: %w", err)
	}
	defer closePage()
	if err := fillLoginForm(pg, c.loginURL, c.opts, username, password); err != nil {
		return fmt.Errorf("carrierproxy: submit login form: %w", err)
	}
	return evaluateLoginResult(pg, c.opts)
}

// validateLoginOptions rejects options that no login attempt could
// succeed with, before a browser is ever launched: an empty success class
// can never match a banner, so every attempt would be misreported as
// rejected credentials.
func validateLoginOptions(opts options) error {
	if opts.successClass == "" {
		return fmt.Errorf("%w: WithSuccessClass must not be empty", carrierproxy.ErrNotConfigured)
	}
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
	if err := clickSubmit(pg, opts.submitSelector); err != nil {
		return err
	}
	return pg.WaitLoad()
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
// a Go error, using its CSS class to tell success from failure. opts must
// have passed validateLoginOptions.
func evaluateLoginResult(pg page, opts options) error {
	el, err := pg.Element(opts.resultSelector)
	if err != nil {
		return fmt.Errorf("carrierproxy: locate result banner: %w", err)
	}

	class, err := el.Attribute("class")
	if err != nil {
		return fmt.Errorf("carrierproxy: read result banner: %w", err)
	}
	if hasClassToken(class, opts.successClass) {
		return nil
	}

	message, textErr := el.Text()
	if textErr != nil {
		message = ""
	}
	return fmt.Errorf("%w: %s", carrierproxy.ErrInvalidCredentials, strings.TrimSpace(message))
}

// hasClassToken reports whether classAttr (a space-separated CSS class
// list, as a browser reports a multi-class attribute) contains token as a
// complete class, not merely a substring — "success" must not match
// "unsuccessful".
func hasClassToken(classAttr, token string) bool {
	for _, c := range strings.Fields(classAttr) {
		if c == token {
			return true
		}
	}
	return false
}
