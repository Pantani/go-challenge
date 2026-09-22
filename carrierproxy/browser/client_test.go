package browser

import (
	"reflect"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("uses defaults with no options", testNewClientDefaults)

	t.Run("applies given options", testNewClientOptions)
}

func testNewClientDefaults(t *testing.T) {
	t.Parallel()
	c := NewClient("https://example.com/login")
	if c.loginURL != "https://example.com/login" {
		t.Fatalf("got loginURL %q, want %q", c.loginURL, "https://example.com/login")
	}
	if !reflect.DeepEqual(c.opts, newOptions()) {
		t.Fatalf("got opts %+v, want defaults %+v", c.opts, newOptions())
	}
	if c.newPage == nil {
		t.Fatal("expected newPage to be wired to a browser launcher")
	}
	if c.wait == nil {
		t.Fatal("expected wait to be wired to a cancellable delay func")
	}
	if c.fetch == nil {
		t.Fatal("expected fetch to be wired to a real HTTP fetch func")
	}
}

func testNewClientOptions(t *testing.T) {
	t.Parallel()
	c := NewClient("https://example.com/login", WithTimeout(5*time.Second), WithUsernameSelector("#u"))
	if c.opts.timeout != 5*time.Second {
		t.Fatalf("got timeout %v, want 5s", c.opts.timeout)
	}
	if c.opts.usernameSelector != "#u" {
		t.Fatalf("got usernameSelector %q, want #u", c.opts.usernameSelector)
	}
}
