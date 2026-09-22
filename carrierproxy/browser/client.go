// Package browser implements carrierproxy.PolicyProvider for any site with
// a plain HTML login form, using go-rod to drive a real, headless browser.
// See README.md.
package browser

import (
	"io"
	"sync"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// Client implements carrierproxy.PolicyProvider against a real site using
// go-rod (see login.go, policies.go, documents.go and rod.go). Policies
// and DocumentDownload re-authenticate using the credentials from the
// last successful Login, and need their own With* configuration (a
// PolicyProvider consumer never sees that difference); see README.md.
// The zero value is not valid; build one with NewClient.
type Client struct {
	loginURL string
	opts     options
	newPage  func(time.Duration) (page, func(), error)
	sleep    func(time.Duration)
	fetch    func(url string, cookies []cookie, timeout time.Duration) (io.ReadCloser, error)

	mu    sync.Mutex
	creds *credentials
}

var _ carrierproxy.PolicyProvider = (*Client)(nil)

// NewClient builds a Client that logs into loginURL, a page with a plain
// HTML login form. By default it targets the common "#username" /
// "#password" / button[type=submit] convention and reads success from a
// "#flash" element whose CSS class contains "success"; use the With*
// options (see options.go) to match a site whose form doesn't follow
// those defaults, and to enable Policies/DocumentDownload. A Client is
// safe for concurrent use.
func NewClient(loginURL string, opts ...Option) *Client {
	o := newOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		loginURL: loginURL,
		opts:     o,
		newPage:  launchPage,
		sleep:    time.Sleep,
		fetch:    fetchWithCookies,
	}
}
