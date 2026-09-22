package browser

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// TestLoginIntegration drives a real headless browser (via launchPage,
// rodPage and rodElement in rod.go) against a local fixture site
// (testdata/, fixturesite_test.go) instead of a real, external one, so it
// stays fast, deterministic, and has nothing else to depend on. It only
// runs when CARRIERPROXY_USERNAME and CARRIERPROXY_PASSWORD are set, so
// `go test ./...` stays fast and doesn't need a browser by default; see
// README.md to run it locally. When it does run, the fixture site is
// configured to accept exactly those credentials, so the test is still
// genuinely driven by them end to end, as asked for in README.md's
// "accept environment variables for the credentials".
func TestLoginIntegration(t *testing.T) {
	t.Parallel()

	username := os.Getenv("CARRIERPROXY_USERNAME")
	password := os.Getenv("CARRIERPROXY_PASSWORD")
	if username == "" || password == "" {
		t.Skip("set CARRIERPROXY_USERNAME and CARRIERPROXY_PASSWORD to run this test against a local fixture site")
	}
	requireBrowser(t)

	srv, site := newFixtureSite(t, username, password)
	fixture := integrationFixture{url: srv.URL, username: username, password: password, site: site}

	t.Run("valid credentials succeed", func(t *testing.T) {
		testLoginIntegrationValidCredentials(t, fixture)
	})

	t.Run("invalid credentials are rejected without retrying", func(t *testing.T) {
		testLoginIntegrationInvalidCredentials(t, fixture)
	})

	t.Run("a wrong selector times out with a classifiable error", func(t *testing.T) {
		testLoginIntegrationWrongSelector(t, fixture)
	})

	t.Run("an unreachable host fails without hanging", func(t *testing.T) {
		testLoginIntegrationUnreachableHost(t, fixture)
	})

	t.Run("Policies requires login, then lists real rows", func(t *testing.T) {
		testLoginIntegrationPolicies(t, fixture)
	})

	t.Run("DocumentDownload requires login, then fetches a real file", func(t *testing.T) {
		testLoginIntegrationDocumentDownload(t, fixture)
	})
}

type integrationFixture struct {
	url, username, password string
	site                    *fixtureSite
}

func testLoginIntegrationValidCredentials(t *testing.T, fixture integrationFixture) {
	client := NewClient(fixture.url + "/login")
	if err := client.Login(fixture.username, fixture.password); err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
}

func testLoginIntegrationInvalidCredentials(t *testing.T, fixture integrationFixture) {
	before := fixture.site.LoginAttempts()
	client := NewClient(fixture.url+"/login", WithRetries(2))
	err := client.Login("not-a-real-user", "not-a-real-password")
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if got := fixture.site.LoginAttempts() - before; got != 1 {
		t.Fatalf("expected exactly 1 POST /login for invalid credentials despite WithRetries(2), got %d", got)
	}
}

func testLoginIntegrationWrongSelector(t *testing.T, fixture integrationFixture) {
	client := NewClient(fixture.url+"/login",
		WithUsernameSelector("#this-field-does-not-exist"),
		WithTimeout(3*time.Second),
		WithRetries(0),
	)
	err := client.Login(fixture.username, fixture.password)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded in the chain, got %v", err)
	}
}

func testLoginIntegrationUnreachableHost(t *testing.T, fixture integrationFixture) {
	client := NewClient("http://carrierproxy-test.invalid/login", WithTimeout(15*time.Second), WithRetries(0))
	if err := client.Login(fixture.username, fixture.password); err == nil {
		t.Fatal("expected an error for an unreachable host")
	}
}

func testLoginIntegrationPolicies(t *testing.T, fixture integrationFixture) {
	client := NewClient(fixture.url+"/login", WithPoliciesURL(fixture.url+"/policies"))

	if _, err := client.Policies(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn before Login, got %v", err)
	}

	if err := client.Login(fixture.username, fixture.password); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	policies, err := client.Policies()
	if err != nil {
		t.Fatalf("expected Policies to succeed, got %v", err)
	}
	if !reflect.DeepEqual(policies, fixturePolicies) {
		t.Fatalf("got %+v, want %+v", policies, fixturePolicies)
	}
}

func testLoginIntegrationDocumentDownload(t *testing.T, fixture integrationFixture) {
	// The login page cannot see the /download cookie required by the
	// document endpoint. Cookie selection must use the document URL.
	client := NewClient(fixture.url+"/login", WithRetries(0), WithDocumentURL(func(key string) string {
		return fixture.url + "/download/" + key
	}))

	if _, err := client.DocumentDownload(fixtureDocumentKey); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn before Login, got %v", err)
	}

	if err := client.Login(fixture.username, fixture.password); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	rc, err := client.DocumentDownload(fixtureDocumentKey)
	if err != nil {
		t.Fatalf("expected DocumentDownload to succeed, got %v", err)
	}
	defer func() { _ = rc.Close() }()
	assertIntegrationDocumentBody(t, rc)
}

func assertIntegrationDocumentBody(t *testing.T, rc io.Reader) {
	t.Helper()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("expected to read the document body, got %v", err)
	}
	want, err := os.ReadFile("testdata/" + fixtureDocumentKey)
	if err != nil {
		t.Fatalf("read fixture document for comparison: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got document body %q, want %q", got, want)
	}
}
