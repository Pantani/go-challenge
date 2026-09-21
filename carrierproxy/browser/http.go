package browser

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// fetchWithCookies performs a plain HTTP GET against target, scoping
// cookies through a cookiejar so a request to a different host than the
// one a cookie came from doesn't receive it — the same rule a real
// browser enforces. It is the default for Client.fetch; tests swap it for
// a fake, the same way they swap newPage.
func fetchWithCookies(target string, cookies []cookie, timeout time.Duration) (io.ReadCloser, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: build cookie jar: %w", err)
	}
	setCookies(jar, cookies)

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: build download request: %w", err)
	}

	resp, err := (&http.Client{Timeout: timeout, Jar: jar}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: download %s: %w", target, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("carrierproxy: download %s: unexpected status %s", target, resp.Status)
	}
	return resp.Body, nil
}

// setCookies loads each cookie into jar under its own domain/path/secure
// attributes, so the http.Client built around jar only attaches it to a
// request whose host that domain actually matches, instead of to every
// request regardless of host.
func setCookies(jar http.CookieJar, cookies []cookie) {
	for _, c := range cookies {
		if c.domain == "" {
			continue
		}
		u := &url.URL{Scheme: "https", Host: strings.TrimPrefix(c.domain, ".")}
		jar.SetCookies(u, []*http.Cookie{{
			Name:   c.name,
			Value:  c.value,
			Domain: c.domain,
			Path:   c.path,
			Secure: c.secure,
		}})
	}
}
