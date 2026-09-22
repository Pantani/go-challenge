package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

type cookieWireStep struct {
	want    string
	updates []string
}

func TestCookieWireOrder(t *testing.T) {
	cases := []struct {
		name    string
		cookies []cookie
		want    string
	}{
		{"path specificity across scopes", []cookie{
			{name: "session", value: "root", domain: "carrier.example", path: "/"},
			{name: "session", value: "document", domain: ".carrier.example", path: "/documents"},
		}, "session=document; session=root"},
		{"domain created first", []cookie{
			{name: "session", value: "domain", domain: ".carrier.example", path: "/"},
			{name: "session", value: "host", domain: "carrier.example", path: "/"},
		}, "session=domain; session=host"},
		{"host created first", []cookie{
			{name: "session", value: "host", domain: "carrier.example", path: "/"},
			{name: "session", value: "domain", domain: ".carrier.example", path: "/"},
		}, "session=host; session=domain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCookieWireSteps(t, tc.cookies, []cookieWireStep{{want: tc.want}})
		})
	}
}

func TestCookieWireOrderAfterUpdates(t *testing.T) {
	cookies := []cookie{
		{name: "session", value: "domain", domain: ".carrier.example", path: "/documents"},
		{name: "session", value: "host", domain: "carrier.example", path: "/documents"},
	}
	steps := []cookieWireStep{
		{"session=domain; session=host", []string{"session=updated; Domain=CARRIER.EXAMPLE; Path=/documents"}},
		{"session=updated; session=host", []string{"session=deleted; Domain=.carrier.example; Path=/documents; Max-Age=0"}},
		{"session=host", []string{"session=recreated; Domain=carrier.example; Path=/documents"}},
		{"session=host; session=recreated", nil},
	}
	assertCookieWireSteps(t, cookies, steps)
}

func TestCookieNamesRoundTripInternalPrefixes(t *testing.T) {
	cookies := []cookie{
		{name: "H_session", value: "one", domain: ".carrier.example", path: "/"},
		{name: "D_session", value: "two", domain: "carrier.example", path: "/"},
		{name: "H_D_session", value: "three", domain: ".carrier.example", path: "/"},
		{name: "D_H_session", value: "four", domain: "carrier.example", path: "/"},
	}
	steps := []cookieWireStep{
		{"H_session=one; D_session=two; H_D_session=three; D_H_session=four", []string{"H_session=updated; Domain=.carrier.example; Path=/"}},
		{"H_session=updated; D_session=two; H_D_session=three; D_H_session=four", nil},
	}
	assertCookieWireSteps(t, cookies, steps)
}

func TestBrowserCookieWireOrder(t *testing.T) {
	requireBrowser(t)
	pg, _ := openCookieScopePage(t, newCookieScopeSite(t))
	target := "https://carrier.example/documents/start"
	err := pg.(rodPage).p.SetCookies([]*proto.NetworkCookieParam{
		{Name: "session", Value: "root", URL: target, Path: "/", Secure: true},
		{Name: "session", Value: "document", Domain: ".carrier.example", Path: "/documents", Secure: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	logCookieBrowserVersion(t, pg, target)
	cookies, err := pg.Cookies(target)
	if err != nil {
		t.Fatal(err)
	}
	assertCookieWireSteps(t, cookies, []cookieWireStep{{want: "session=document; session=root"}})
}

func TestHostBoundJarNameOwnership(t *testing.T) {
	jar, err := newHostBoundJar("carrier.example")
	if err != nil {
		t.Fatal(err)
	}
	u := mustCookieURL(t, "https://carrier.example/documents/start")
	c := &http.Cookie{Name: "H_session", Value: "value", Domain: ".carrier.example", Path: "/"}
	jar.SetCookies(u, []*http.Cookie{c})
	if c.Name != "H_session" {
		t.Fatalf("caller cookie name mutated to %q", c.Name)
	}
	got := jar.Cookies(u)
	if len(got) != 1 {
		t.Fatalf("cookies = %v, want one", got)
	}
	got[0].Name = "changed-by-caller"
	if name := jar.Cookies(u)[0].Name; name != "H_session" {
		t.Fatalf("stored cookie name mutated to %q", name)
	}
}

func assertCookieWireSteps(t *testing.T, cookies []cookie, steps []cookieWireStep) {
	t.Helper()
	requests := 0
	transport := cookieTransport(func(r *http.Request) (*http.Response, error) {
		if requests >= len(steps) {
			return nil, fmt.Errorf("unexpected request %d", requests+1)
		}
		step := steps[requests]
		requests++
		assertCookieHeaderOrder(t, r, step.want)
		response := &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Set-Cookie": step.updates},
			Body: io.NopCloser(strings.NewReader("policy")), Request: r,
		}
		if requests < len(steps) {
			response.StatusCode = http.StatusFound
			response.Header.Set("Location", "/documents/next")
		}
		return response, nil
	})
	body, err := fetchWithTransport(context.Background(), "https://carrier.example/documents/start", cookies, transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if requests != len(steps) {
		t.Fatalf("requests = %d, want %d", requests, len(steps))
	}
}

func assertCookieHeaderOrder(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Header.Get("Cookie"); got != want {
		t.Errorf("Cookie header = %q, want %q", got, want)
	}
	first, _, _ := strings.Cut(want, ";")
	name, value, _ := strings.Cut(first, "=")
	c, err := r.Cookie(name)
	if err != nil {
		t.Fatal(err)
	}
	if c.Value != value {
		t.Errorf("first %s cookie = %q, want %q", name, c.Value, value)
	}
}
