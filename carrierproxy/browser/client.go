// Package browser implements carrierproxy.PolicyProvider for any site with
// a plain HTML login form, using go-rod to drive a real, headless browser.
// See README.md.
package browser

import (
	"io"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// Client implements the Login portion of carrierproxy.PolicyProvider against
// a real site using go-rod. The zero value is not valid; build one with
// NewClient.
type Client struct {
	loginURL string
	opts     options
	newPage  func(time.Duration) (page, func(), error)
}

var _ carrierproxy.PolicyProvider = (*Client)(nil)

// NewClient builds a Client that logs into loginURL, a page with a plain
// HTML login form. By default it targets the common "#username" /
// "#password" / button[type=submit] convention and reads success from a
// "#flash" element whose CSS class contains "success"; use the With*
// options (see options.go) to match a site whose form doesn't follow those
// defaults.
func NewClient(loginURL string, opts ...Option) *Client {
	o := newOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		loginURL: loginURL,
		opts:     o,
		newPage:  launchPage,
	}
}

// Policies is outside this challenge's partial implementation.
func (c *Client) Policies() ([]carrierproxy.Policy, error) {
	return nil, carrierproxy.ErrNotImplemented
}

// DocumentDownload is outside this challenge's partial implementation.
func (c *Client) DocumentDownload(string) (io.ReadCloser, error) {
	return nil, carrierproxy.ErrNotImplemented
}
