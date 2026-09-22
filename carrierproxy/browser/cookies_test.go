package browser

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

func mustCookieURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestHostOnlyCookieDoesNotReachSubdomain(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	setCookies(jar, []cookie{{name: "session", value: "secret", domain: "carrier.example", path: "/"}})
	if got := jar.Cookies(mustCookieURL(t, "https://carrier.example/doc")); len(got) != 1 {
		t.Fatalf("origin cookies = %v", got)
	}
	if got := jar.Cookies(mustCookieURL(t, "https://sub.carrier.example/doc")); len(got) != 0 {
		t.Fatalf("host-only cookie leaked: %v", got)
	}
}

type cookieScopeCase struct {
	name   string
	cookie cookie
	target string
	want   int
}

func assertCookieScope(t *testing.T, tc cookieScopeCase) {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	setCookies(jar, []cookie{tc.cookie})
	if got := len(jar.Cookies(mustCookieURL(t, tc.target))); got != tc.want {
		t.Fatalf("cookie count = %d, want %d", got, tc.want)
	}
}

func TestCookieScope(t *testing.T) {
	cases := []cookieScopeCase{
		{"domain child", cookie{name: "s", value: "x", domain: ".carrier.example", path: "/"}, "https://sub.carrier.example/doc", 1},
		{"domain sibling", cookie{name: "s", value: "x", domain: ".carrier.example", path: "/"}, "https://other.example/doc", 0},
		{"path inside", cookie{name: "s", value: "x", domain: "carrier.example", path: "/documents"}, "https://carrier.example/documents/42", 1},
		{"path prefix boundary", cookie{name: "s", value: "x", domain: "carrier.example", path: "/documents"}, "https://carrier.example/documents-other/42", 0},
		{"secure https", cookie{name: "s", value: "x", domain: "carrier.example", path: "/", secure: true}, "https://carrier.example/doc", 1},
		{"secure http", cookie{name: "s", value: "x", domain: "carrier.example", path: "/", secure: true}, "http://carrier.example/doc", 0},
		{"missing domain", cookie{name: "s", value: "x", path: "/"}, "https://carrier.example/doc", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCookieScope(t, tc)
		})
	}
}

type cookieTransport func(*http.Request) (*http.Response, error)

func (f cookieTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type redirectCase struct {
	name   string
	target string
	domain string
	path   string
	secure bool
	want   string
}

func assertRedirectCookie(t *testing.T, tc redirectCase) {
	t.Helper()
	var got string
	transport := cookieTransport(func(r *http.Request) (*http.Response, error) {
		response := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("policy")),
			Request:    r,
		}
		if r.URL.Path == "/documents/start" {
			response.StatusCode = http.StatusFound
			response.Header.Set("Location", tc.target)
			return response, nil
		}
		got = r.Header.Get("Cookie")
		return response, nil
	})
	body, err := fetchWithTransport(
		context.Background(),
		"https://carrier.example/documents/start",
		[]cookie{{name: "session", value: "secret", domain: tc.domain, path: tc.path, secure: tc.secure}},
		transport,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if got != tc.want {
		t.Fatalf("redirect Cookie = %q, want %q", got, tc.want)
	}
}

func TestCookieRedirects(t *testing.T) {
	cases := []redirectCase{
		{"same host", "https://carrier.example/documents/final", "carrier.example", "/", false, "session=secret"},
		{"host-only child", "https://sub.carrier.example/documents/final", "carrier.example", "/", false, ""},
		{"domain child", "https://sub.carrier.example/documents/final", ".carrier.example", "/", false, ""},
		{"other host", "https://other.example/documents/final", ".carrier.example", "/", false, ""},
		{"path outside", "https://carrier.example/elsewhere", "carrier.example", "/documents", false, ""},
		{"secure HTTPS", "https://carrier.example/documents/final", "carrier.example", "/", true, "session=secret"},
		{"secure HTTP", "http://carrier.example/documents/final", "carrier.example", "/", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertRedirectCookie(t, tc)
		})
	}
}

func TestCookieRedirectRetainsResponseUpdates(t *testing.T) {
	cookies := []cookie{
		{name: "session", value: "parent", domain: ".carrier.example", path: "/documents", secure: true},
		{name: "session", value: "host", domain: "sub.carrier.example", path: "/documents", secure: true},
	}
	assertCookieResponseUpdate(t, "https://sub.carrier.example/documents/start", cookies,
		"session=updated; Path=/documents; Secure", []string{"session=parent", "session=updated"})
}

func assertCookieResponseUpdate(t *testing.T, target string, cookies []cookie, header string, want []string) {
	t.Helper()
	requests := 0
	transport := cookieTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		response := &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader("policy")), Request: r,
		}
		if requests == 1 {
			assertCookieHeaderValues(t, r, []string{"session=host", "session=parent"})
			response.StatusCode = http.StatusFound
			response.Header.Set("Location", "/documents/final")
			response.Header.Set("Set-Cookie", header)
			return response, nil
		}
		assertCookieHeaderValues(t, r, want)
		return response, nil
	})
	body, err := fetchWithTransport(context.Background(), target, cookies, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if requests != 2 {
		t.Fatalf("requests = %d, want initial and redirect", requests)
	}
}

func TestHostBoundJarIPResponseIdentity(t *testing.T) {
	jar, err := newHostBoundJar("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	u := mustCookieURL(t, "https://127.0.0.1/documents/42")
	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "old", Path: "/"}})
	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "updated", Domain: "127.0.0.1", Path: "/"}})
	cookies := jar.Cookies(u)
	if len(cookies) != 1 {
		t.Fatalf("IP cookies = %v, want one host-only identity", cookies)
	}
	if cookies[0].Value != "updated" {
		t.Fatalf("IP session = %q, want updated", cookies[0].Value)
	}
}
