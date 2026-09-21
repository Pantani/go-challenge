package browser

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchWithCookies performs a plain HTTP GET against url, attaching
// cookies so the request reuses a session already established by the
// browser. It is the default for Client.fetch; tests swap it for a fake,
// the same way they swap newPage.
func fetchWithCookies(url string, cookies []cookie, timeout time.Duration) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: build download request: %w", err)
	}
	for _, c := range cookies {
		req.AddCookie(&http.Cookie{Name: c.name, Value: c.value})
	}

	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: download %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("carrierproxy: download %s: unexpected status %s", url, resp.Status)
	}
	return resp.Body, nil
}
