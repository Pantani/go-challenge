package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-rod/rod/lib/launcher"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// env returns a getenv func backed by vars, so tests don't touch the real
// process environment.
func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

// TestRunMissingCredentials confirms run rejects blank credentials before
// ever launching a browser, so it needs neither a browser nor a network.
func TestRunMissingCredentials(t *testing.T) {
	t.Parallel()

	err := run("http://127.0.0.1:0/login", env(nil))
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

// TestRunEndToEnd drives the CLI's run exactly as main does, through a real
// headless browser, against a local login page that accepts one
// username/password pair. It is skipped when no browser is installed.
func TestRunEndToEnd(t *testing.T) {
	t.Parallel()

	if _, ok := launcher.LookPath(); !ok {
		t.Skip("no local Chrome/Chromium found; skipping the end-to-end CLI test")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flash := ""
		if r.Method == http.MethodPost {
			if r.FormValue("username") == "e2e-user" && r.FormValue("password") == "e2e-pass" {
				flash = `<div id="flash" class="flash success">You logged into a secure area!</div>`
			} else {
				flash = `<div id="flash" class="flash error">Your username is invalid!</div>`
			}
		}
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><body>
<form method="post" action="/login">
  <input type="text" name="username" id="username">
  <input type="password" name="password" id="password">
  <button type="submit">Login</button>
</form>%s</body></html>`, flash)
	}))
	t.Cleanup(srv.Close)

	t.Run("valid credentials succeed", func(t *testing.T) {
		t.Parallel()

		vars := map[string]string{"CARRIERPROXY_USERNAME": "e2e-user", "CARRIERPROXY_PASSWORD": "e2e-pass"}
		if err := run(srv.URL+"/login", env(vars)); err != nil {
			t.Fatalf("expected login to succeed, got %v", err)
		}
	})

	t.Run("invalid credentials are reported", func(t *testing.T) {
		t.Parallel()

		vars := map[string]string{"CARRIERPROXY_USERNAME": "e2e-user", "CARRIERPROXY_PASSWORD": "wrong"}
		err := run(srv.URL+"/login", env(vars))
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
