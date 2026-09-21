package browser

import (
	"context"
	"errors"
	"io"
	"os"
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

	site := newFixtureSite(t, username, password)

	t.Run("valid credentials succeed", func(t *testing.T) {
		client := NewClient(site.URL + "/login")
		if err := client.Login(username, password); err != nil {
			t.Fatalf("expected login to succeed, got %v", err)
		}
	})

	t.Run("invalid credentials are rejected without retrying", func(t *testing.T) {
		client := NewClient(site.URL+"/login", WithRetries(2))
		err := client.Login("not-a-real-user", "not-a-real-password")
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("a wrong selector times out with a classifiable error", func(t *testing.T) {
		client := NewClient(site.URL+"/login",
			WithUsernameSelector("#this-field-does-not-exist"),
			WithTimeout(3*time.Second),
			WithRetries(0),
		)
		err := client.Login(username, password)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded in the chain, got %v", err)
		}
	})

	t.Run("an unreachable host fails without hanging", func(t *testing.T) {
		client := NewClient("http://carrierproxy-test.invalid/login", WithTimeout(15*time.Second), WithRetries(0))
		if err := client.Login(username, password); err == nil {
			t.Fatal("expected an error for an unreachable host")
		}
	})

	t.Run("Policies requires login, then lists real rows", func(t *testing.T) {
		client := NewClient(site.URL+"/login", WithPoliciesURL(site.URL+"/policies"))

		if _, err := client.Policies(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected ErrNotLoggedIn before Login, got %v", err)
		}

		if err := client.Login(username, password); err != nil {
			t.Fatalf("login failed: %v", err)
		}
		policies, err := client.Policies()
		if err != nil {
			t.Fatalf("expected Policies to succeed, got %v", err)
		}
		if len(policies) != len(fixturePolicies) || policies[0] != fixturePolicies[0] {
			t.Fatalf("got %+v, want %+v", policies, fixturePolicies)
		}
	})

	t.Run("DocumentDownload requires login, then fetches a real file", func(t *testing.T) {
		client := NewClient(site.URL+"/login", WithDocumentURL(func(key string) string {
			return site.URL + "/download/" + key
		}))

		if _, err := client.DocumentDownload(fixtureDocumentKey); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected ErrNotLoggedIn before Login, got %v", err)
		}

		if err := client.Login(username, password); err != nil {
			t.Fatalf("login failed: %v", err)
		}
		rc, err := client.DocumentDownload(fixtureDocumentKey)
		if err != nil {
			t.Fatalf("expected DocumentDownload to succeed, got %v", err)
		}
		defer func() { _ = rc.Close() }()

		body, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("expected to read the document body, got %v", err)
		}
		if len(body) == 0 {
			t.Fatal("expected a non-empty document body")
		}
	})
}
