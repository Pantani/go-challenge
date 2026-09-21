package browser

import (
	"fmt"
	"io"
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// DocumentDownload fetches the document identified by downloadKey,
// re-authenticating first and reusing that session's cookies for the
// download. It requires WithDocumentURL and returns
// carrierproxy.ErrNotConfigured without it, and carrierproxy.ErrNotLoggedIn
// if Login has not yet succeeded. Like Login, transient failures are
// retried (see WithRetries).
func (c *Client) DocumentDownload(downloadKey string) (io.ReadCloser, error) {
	if c.opts.documentURL == nil {
		return nil, fmt.Errorf("%w: DocumentDownload needs WithDocumentURL", carrierproxy.ErrNotConfigured)
	}

	creds, err := c.storedCredentials()
	if err != nil {
		return nil, err
	}

	if err := validateDownloadKey(downloadKey); err != nil {
		return nil, err
	}

	var doc io.ReadCloser
	err = c.withRetries(func() error {
		var attemptErr error
		doc, attemptErr = c.fetchDocument(creds, downloadKey)
		return attemptErr
	})
	return doc, err
}

// validateDownloadKey rejects a key that is empty or could escape the URL
// WithDocumentURL builds from it.
func validateDownloadKey(downloadKey string) error {
	if downloadKey == "" || strings.ContainsAny(downloadKey, "/\\") {
		return fmt.Errorf("carrierproxy: invalid downloadKey %q", downloadKey)
	}
	return nil
}

// fetchDocument is a single DocumentDownload attempt: re-authenticate,
// then fetch the WithDocumentURL for downloadKey using that session's
// cookies. It is the unit of work withRetries repeats.
func (c *Client) fetchDocument(creds credentials, downloadKey string) (io.ReadCloser, error) {
	pg, closePage, err := c.authenticatedPage(creds.username, creds.password)
	if err != nil {
		return nil, err
	}
	defer closePage()

	cookies, err := pg.Cookies()
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: read session cookies: %w", err)
	}

	return c.fetch(c.opts.documentURL(downloadKey), cookies, c.opts.timeout)
}
