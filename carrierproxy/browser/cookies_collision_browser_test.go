package browser

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

func TestBrowserCookieIdentitySurvivesTransfer(t *testing.T) {
	requireBrowser(t)
	initial := "https://sub.carrier.example/documents/start"
	cookies := browserCollisionCookies(t, initial)
	cases := []struct {
		name, target string
		want         []string
	}{
		{"same host", "https://sub.carrier.example/documents/final", []string{"session=host", "session=parent"}},
		{"child", "https://deep.sub.carrier.example/documents/final", nil},
		{"parent", "https://carrier.example/documents/final", nil},
		{"sibling", "https://other.carrier.example/documents/final", nil},
		{"unrelated", "https://other.example/documents/final", nil},
		{"outside path", "https://sub.carrier.example/elsewhere", nil},
		{"insecure", "http://sub.carrier.example/documents/final", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCollisionRedirect(t, cookies, initial, tc.target, tc.want)
		})
	}
}

func browserCollisionCookies(t *testing.T, target string) []cookie {
	t.Helper()
	pg, _ := openCookieScopePage(t, newCookieScopeSite(t))
	err := pg.(rodPage).p.SetCookies([]*proto.NetworkCookieParam{
		{Name: "session", Value: "parent", Domain: ".carrier.example", Path: "/documents", Secure: true},
		{Name: "session", Value: "host", URL: target, Path: "/documents", Secure: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	logCookieBrowserVersion(t, pg, target)
	cookies, err := pg.Cookies(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 2 {
		t.Fatalf("browser target cookies = %#v, want both domain identities", cookies)
	}
	return cookies
}

func assertCollisionRedirect(t *testing.T, cookies []cookie, initial, target string, want []string) {
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
			response.Header.Set("Location", target)
			return response, nil
		}
		assertCookieHeaderValues(t, r, want)
		return response, nil
	})
	body, err := fetchWithTransport(context.Background(), initial, cookies, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if requests != 2 {
		t.Fatalf("requests = %d, want initial and redirect", requests)
	}
}

func TestBrowserCookieApexIdentitySurvivesTransfer(t *testing.T) {
	requireBrowser(t)
	initial := "https://carrier.example/documents/start"
	cookies := browserCollisionCookies(t, initial)
	cases := []struct {
		name, target string
		want         []string
	}{
		{"same host", "https://carrier.example/documents/final", []string{"session=host", "session=parent"}},
		{"child", "https://sub.carrier.example/documents/final", nil},
		{"parent", "https://example/documents/final", nil},
		{"sibling", "https://other.example/documents/final", nil},
		{"unrelated", "https://unrelated.invalid/documents/final", nil},
		{"outside path", "https://carrier.example/elsewhere", nil},
		{"insecure", "http://carrier.example/documents/final", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCollisionRedirect(t, cookies, initial, tc.target, tc.want)
		})
	}
}

func TestBrowserCookieApexResponseUpdates(t *testing.T) {
	requireBrowser(t)
	target := "https://carrier.example/documents/start"
	cookies := browserCollisionCookies(t, target)
	cases := []struct {
		name, header string
		want         []string
	}{
		{"host update", "session=updated; Path=/documents; Secure", []string{"session=parent", "session=updated"}},
		{"domain update", "session=updated; Domain=.carrier.example; Path=/documents; Secure", []string{"session=host", "session=updated"}},
		{"normalized domain update", "session=updated; Domain=CARRIER.EXAMPLE; Path=/documents; Secure", []string{"session=host", "session=updated"}},
		{"host expiry", "session=deleted; Path=/documents; Max-Age=0", []string{"session=parent"}},
		{"domain expiry", "session=deleted; Domain=carrier.example; Path=/documents; Max-Age=0", []string{"session=host"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCookieResponseUpdate(t, target, cookies, tc.header, tc.want)
		})
	}
}

func assertCookieHeaderValues(t *testing.T, r *http.Request, want []string) {
	t.Helper()
	cookies := r.Cookies()
	got := make([]string, 0, len(cookies))
	for _, c := range cookies {
		got = append(got, c.Name+"="+c.Value)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("cookies for %s = %v, want %v", r.URL, got, want)
	}
}
