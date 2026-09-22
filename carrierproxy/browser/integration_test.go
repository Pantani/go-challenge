package browser

import (
	"errors"
	"os"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
	"github.com/go-rod/rod/lib/launcher"
)

// TestLoginIntegration drives a real headless browser. It needs
// CARRIERPROXY_USERNAME and CARRIERPROXY_PASSWORD; with CARRIERPROXY_LOGIN_URL
// it targets that site, otherwise a local fixture that accepts exactly those
// credentials.
func TestLoginIntegration(t *testing.T) {
	username := os.Getenv("CARRIERPROXY_USERNAME")
	password := os.Getenv("CARRIERPROXY_PASSWORD")
	if username == "" || password == "" {
		t.Skip("set CARRIERPROXY_USERNAME and CARRIERPROXY_PASSWORD to run the browser test")
	}
	if _, ok := launcher.LookPath(); !ok {
		t.Skip("no local Chrome/Chromium found")
	}

	client := NewClient(integrationLoginURL(t, username, password))
	if err := client.Login(username, password); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := client.Login("wrong-"+username, password); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v", err)
	}
}

// integrationLoginURL returns CARRIERPROXY_LOGIN_URL when set, otherwise the
// login page of a fixture site that accepts username/password.
func integrationLoginURL(t *testing.T, username, password string) string {
	t.Helper()
	if url := os.Getenv("CARRIERPROXY_LOGIN_URL"); url != "" {
		return url
	}
	return newFixtureSite(t, username, password).URL + "/login"
}
