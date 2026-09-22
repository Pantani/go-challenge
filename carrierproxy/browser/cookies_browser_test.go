package browser

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

func newCookieScopeSite(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "host_session", Value: "h", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "domain_session", Value: "d", Domain: "localhost", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "document_session", Value: "p", Path: "/documents"})
		_, _ = io.WriteString(w, "<html><body>cookie fixture</body></html>")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func cookieByName(t *testing.T, cookies []cookie, name string) cookie {
	t.Helper()
	for _, c := range cookies {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("cookie %q absent from %#v", name, cookies)
	return cookie{}
}

func openCookieScopePage(t *testing.T, srv *httptest.Server) (page, string) {
	t.Helper()
	u := mustCookieURL(t, srv.URL)
	u.Host = net.JoinHostPort("localhost", u.Port())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pg, release, err := launchPage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(release)
	if err := pg.Navigate(u.String() + "/login"); err != nil {
		t.Fatal(err)
	}
	if err := pg.WaitLoad(); err != nil {
		t.Fatal(err)
	}
	return pg, u.String()
}

func logCookieBrowserVersion(t *testing.T, pg page, target string) {
	t.Helper()
	version, err := (proto.BrowserGetVersion{}).Call(pg.(rodPage).p)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := pg.(rodPage).p.Cookies([]string{target})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range raw {
		t.Logf("browser = %s; raw CDP cookie %s: domain=%q path=%q", version.Product, c.Name, c.Domain, c.Path)
	}
}

func TestBrowserCookieDomainRepresentation(t *testing.T) {
	requireBrowser(t)
	pg, base := openCookieScopePage(t, newCookieScopeSite(t))
	cookies, err := pg.Cookies(base + "/documents/42")
	if err != nil {
		t.Fatal(err)
	}
	logCookieBrowserVersion(t, pg, base+"/documents/42")
	if got := cookieByName(t, cookies, "host_session").domain; got != "localhost" {
		t.Fatalf("transferred host-only cookie domain = %q", got)
	}
	// Chromium does not reliably distinguish both kinds on localhost. The
	// download jar must therefore restrict both to the document hostname.
	if got := cookieByName(t, cookies, "domain_session").domain; got != "localhost" {
		t.Fatalf("transferred domain cookie domain = %q", got)
	}
	if got := cookieByName(t, cookies, "document_session").path; got != "/documents" {
		t.Fatalf("document cookie path = %q", got)
	}
}

func TestBrowserTransferredDomainCookie(t *testing.T) {
	requireBrowser(t)
	pg, _ := openCookieScopePage(t, newCookieScopeSite(t))
	// Seed browser state directly: these reserved hosts never receive I/O.
	err := pg.(rodPage).p.SetCookies([]*proto.NetworkCookieParam{{
		Name: "session", Value: "secret", Domain: ".carrier.example", Path: "/documents", Secure: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	target := "https://carrier.example/documents/start"
	logCookieBrowserVersion(t, pg, target)
	cookies, err := pg.Cookies(target)
	if err != nil {
		t.Fatal(err)
	}
	c := cookieByName(t, cookies, "session")
	assertTransferredCookieRedirects(t, c)
	t.Run("target is child of raw domain", func(t *testing.T) {
		assertParentDomainNarrowing(t, pg)
	})
}

func assertParentDomainNarrowing(t *testing.T, pg page) {
	t.Helper()
	target := "https://sub.carrier.example/documents/42"
	cookies, err := pg.Cookies(target)
	if err != nil {
		t.Fatal(err)
	}
	c := cookieByName(t, cookies, "session")
	bound, err := newHostBoundJar("sub.carrier.example")
	if err != nil {
		t.Fatal(err)
	}
	setCookies(bound, []cookie{c})
	cases := []cookieScopeCase{
		{"target", c, target, 1},
		{"parent", c, "https://carrier.example/documents/42", 0},
		{"sibling", c, "https://other.carrier.example/documents/42", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(bound.Cookies(mustCookieURL(t, tc.target))); got != tc.want {
				t.Fatalf("cookie count = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBrowserCookieInvalidTarget(t *testing.T) {
	requireBrowser(t)
	pg, _ := openCookieScopePage(t, newCookieScopeSite(t))
	_, err := pg.Cookies("%")
	var parseError *url.Error
	if !errors.As(err, &parseError) {
		t.Fatalf("expected wrapped URL parse error, got %v", err)
	}
}

func assertTransferredCookieRedirects(t *testing.T, c cookie) {
	t.Helper()
	cases := []redirectCase{
		{"same host", "https://carrier.example/documents/final", c.domain, c.path, c.secure, "session=secret"},
		{"child host", "https://sub.carrier.example/documents/final", c.domain, c.path, c.secure, ""},
		{"other host", "https://other.example/documents/final", c.domain, c.path, c.secure, ""},
		{"outside path", "https://carrier.example/elsewhere", c.domain, c.path, c.secure, ""},
		{"insecure", "http://carrier.example/documents/final", c.domain, c.path, c.secure, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertRedirectCookie(t, tc)
		})
	}
}
