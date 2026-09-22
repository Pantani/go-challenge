package browser

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// DocumentDownload fetches the document identified by downloadKey,
// re-authenticating first and reusing that session's cookies for the
// download. It requires WithDocumentURL and returns
// carrierproxy.ErrNotConfigured without it, and carrierproxy.ErrNotLoggedIn
// if Login has not yet succeeded. Like Login, transient failures are
// retried (see WithRetries). The caller must close the returned body.
func (c *Client) DocumentDownload(downloadKey string) (io.ReadCloser, error) {
	return c.DocumentDownloadContext(context.Background(), downloadKey)
}

// DocumentDownloadContext is DocumentDownload with caller cancellation. Its
// attempt timeout starts before browser launch and remains active while the
// returned body is read. The caller must always close the returned body.
func (c *Client) DocumentDownloadContext(ctx context.Context, downloadKey string) (io.ReadCloser, error) {
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

	return documentResult(attemptWithRetries(ctx, c, func(ctx context.Context) (io.ReadCloser, error) {
		return c.fetchDocument(ctx, creds, downloadKey)
	}))
}

// validateDownloadKey rejects a key that is empty, ".", "..", or contains a
// slash or backslash before it is escaped as one path segment. WithDocumentURL
// remains responsible for appending that segment to a trusted path unchanged.
func validateDownloadKey(downloadKey string) error {
	if downloadKey == "" || downloadKey == "." || downloadKey == ".." || strings.ContainsAny(downloadKey, "/\\") {
		return fmt.Errorf("carrierproxy: invalid downloadKey %q", downloadKey)
	}
	return nil
}

// fetchDocument is a single DocumentDownload attempt: re-authenticate,
// then fetch the WithDocumentURL for downloadKey using that session's
// cookies. It is the unit of work withRetries repeats.
func (c *Client) fetchDocument(ctx context.Context, creds credentials, downloadKey string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, c.opts.timeout)
	pg, closePage, err := c.authenticatedPage(ctx, creds.username, creds.password)
	if err != nil {
		cancel()
		return nil, err
	}
	defer closePage()
	body, err := c.fetchDocumentOnPage(ctx, pg, downloadKey)
	if err != nil {
		cancel()
		return nil, err
	}
	return &attemptBody{ReadCloser: body, cancel: cancel}, nil
}

func (c *Client) fetchDocumentOnPage(ctx context.Context, pg page, downloadKey string) (io.ReadCloser, error) {
	// The builder receives one escaped segment. It is application code and may
	// choose any origin or URL layout, so the client does not claim that an
	// arbitrary builder preserves an origin or path boundary.
	target := c.opts.documentURL(url.PathEscape(downloadKey))
	cookies, err := pg.Cookies(target)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: read session cookies: %w", err)
	}
	return c.fetch(ctx, target, cookies)
}
