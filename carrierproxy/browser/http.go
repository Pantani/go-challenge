package browser

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// fetchWithCookies performs a plain HTTP GET against target. The standard jar
// preserves encoded scope identities, global cookie order, and path/secure rules;
// a host boundary withholds cookies from redirects to any other hostname.
// It is the default for Client.fetch; tests swap it for a fake, the same way
// they swap newPage.
func fetchWithCookies(ctx context.Context, target string, cookies []cookie) (io.ReadCloser, error) {
	return fetchWithTransport(ctx, target, cookies, nil)
}

func fetchWithTransport(ctx context.Context, target string, cookies []cookie, transport http.RoundTripper) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: build download request: %w", err)
	}

	bound, err := newHostBoundJar(req.URL.Hostname())
	if err != nil {
		return nil, err
	}
	setCookies(bound, cookies)
	resp, err := (&http.Client{Jar: bound, Transport: transport}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: download %s: %w", target, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("carrierproxy: download %s: unexpected status %s", target, resp.Status)
	}
	return resp.Body, nil
}

// hostBoundJar encodes scope into internal names so one standard jar preserves
// host-only/domain identities and global path/creation ordering. Names are
// restored before HTTP serialization; caller cookie values are never mutated.
// Outgoing cookies are limited to the original document hostname.
type hostBoundJar struct {
	jar  *cookiejar.Jar
	host string
}

func newHostBoundJar(host string) (*hostBoundJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: build cookie jar: %w", err)
	}
	return &hostBoundJar{jar: jar, host: host}, nil
}

func (j *hostBoundJar) Cookies(u *url.URL) []*http.Cookie {
	if u.Hostname() != j.host {
		return nil
	}
	cookies := j.jar.Cookies(u)
	for _, c := range cookies {
		c.Name = decodeCookieName(c.Name)
	}
	return cookies
}

func (j *hostBoundJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	encoded := make([]*http.Cookie, len(cookies))
	for i, c := range cookies {
		encoded[i] = encodeCookieScope(u, c)
	}
	j.jar.SetCookies(u, encoded)
}

const (
	hostOnlyCookiePrefix = "H_"
	domainCookiePrefix   = "D_"
)

func encodeCookieScope(u *url.URL, c *http.Cookie) *http.Cookie {
	encoded := *c
	prefix := domainCookiePrefix
	// Like net/http/cookiejar, explicit IP domains remain host-only.
	if c.Domain == "" || net.ParseIP(u.Hostname()) != nil {
		prefix = hostOnlyCookiePrefix
	}
	encoded.Name = prefix + c.Name
	return &encoded
}

func decodeCookieName(name string) string {
	if strings.HasPrefix(name, hostOnlyCookiePrefix) || strings.HasPrefix(name, domainCookiePrefix) {
		// Both internal prefixes are two bytes; strip exactly one layer.
		return name[2:]
	}
	return name
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
		jar.SetCookies(u, []*http.Cookie{jarCookie(c)})
	}
}

func jarCookie(c cookie) *http.Cookie {
	out := &http.Cookie{Name: c.name, Value: c.value, Path: c.path, Secure: c.secure}
	if strings.HasPrefix(c.domain, ".") {
		out.Domain = c.domain
	}
	return out
}
