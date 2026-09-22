package browser

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://example.com/login", WithTimeout(5*time.Second))
	if client.loginURL != "https://example.com/login" {
		t.Fatalf("loginURL = %q", client.loginURL)
	}
	if client.opts.timeout != 5*time.Second {
		t.Fatalf("timeout = %v", client.opts.timeout)
	}
	if client.newPage == nil {
		t.Fatal("newPage is nil")
	}
}

func TestClientUnsupportedOperations(t *testing.T) {
	client := NewClient(testLoginURL)

	if _, err := client.Policies(); !errors.Is(err, carrierproxy.ErrNotImplemented) {
		t.Fatalf("Policies() error = %v", err)
	}
	if body, err := client.DocumentDownload("policy.pdf"); body != nil || !errors.Is(err, carrierproxy.ErrNotImplemented) {
		t.Fatalf("DocumentDownload() = (%v, %v)", body, err)
	}
}

func TestClientLoginClosesPage(t *testing.T) {
	fake := successPage()
	closed := false
	client := NewClient(testLoginURL)
	client.newPage = func(time.Duration) (page, func(), error) {
		return fake, func() { closed = true }, nil
	}

	if err := client.Login("alice", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !closed {
		t.Fatal("Login() did not close the page")
	}
	want := []string{
		"navigate:" + testLoginURL, "wait-load",
		"element:#username", "input:#username=alice",
		"element:#password", "input:#password=secret",
		"element:button[type='submit']", "click:button[type='submit']",
		"wait-load", "element:#flash", "attribute:#flash=class",
	}
	if !reflect.DeepEqual(fake.events, want) {
		t.Fatalf("events = %#v, want %#v", fake.events, want)
	}
}
