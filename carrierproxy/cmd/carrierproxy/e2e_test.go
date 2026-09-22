package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-rod/rod/lib/launcher"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy/browser"
)

// TestRunEndToEnd drives run exactly as main does, with a real
// browser.Client and a real headless browser, against a local login page
// that accepts one username/password pair instead of the public demo. It
// is skipped when no browser is installed.
func TestRunEndToEnd(t *testing.T) {
	t.Parallel()

	if _, ok := launcher.LookPath(); !ok {
		t.Skip("no local Chrome/Chromium found; skipping the end-to-end CLI test")
	}

	srv := httptest.NewServer(http.HandlerFunc(serveE2ELogin))
	t.Cleanup(srv.Close)

	env := func(password string) func(string) string {
		vars := map[string]string{envUsername: "e2e-user", envPassword: password}
		return func(key string) string { return vars[key] }
	}

	t.Run("valid credentials succeed", func(t *testing.T) {
		t.Parallel()

		if err := run(env("e2e-pass"), browser.NewClient(srv.URL+"/login"), io.Discard); err != nil {
			t.Fatalf("expected login to succeed, got %v", err)
		}
	})

	t.Run("invalid credentials are reported", func(t *testing.T) {
		t.Parallel()

		err := run(env("wrong"), browser.NewClient(srv.URL+"/login"), io.Discard)
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}

func serveE2ELogin(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><body>
<form method="post" action="/login">
  <input type="text" name="username" id="username">
  <input type="password" name="password" id="password">
  <button type="submit">Login</button>
</form>%s</body></html>`, e2eFlash(r))
}

func e2eFlash(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	if r.FormValue("username") != "e2e-user" || r.FormValue("password") != "e2e-pass" {
		return `<div id="flash" class="flash error">Your username is invalid!</div>`
	}
	return `<div id="flash" class="flash success">You logged into a secure area!</div>`
}
