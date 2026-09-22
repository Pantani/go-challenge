package browser

import (
	"errors"
	"os"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
	"github.com/go-rod/rod/lib/launcher"
)

func TestLoginIntegration(t *testing.T) {
	username := os.Getenv("CARRIERPROXY_USERNAME")
	password := os.Getenv("CARRIERPROXY_PASSWORD")
	if username == "" || password == "" {
		t.Skip("set CARRIERPROXY_USERNAME and CARRIERPROXY_PASSWORD to run the browser fixture")
	}
	if _, ok := launcher.LookPath(); !ok {
		t.Skip("no local Chrome/Chromium found")
	}

	server := newFixtureSite(t, username, password)
	client := NewClient(server.URL + "/login")
	if err := client.Login(username, password); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := client.Login("wrong", password); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v", err)
	}
}
