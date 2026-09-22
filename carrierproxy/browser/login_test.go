package browser

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func TestLoginRejectsInvalidCredentialsBeforeLaunching(t *testing.T) {
	client := NewClient(testLoginURL)
	client.newPage = func(time.Duration) (page, func(), error) {
		t.Fatal("Login() launched a page for invalid credentials")
		return nil, nil, nil
	}

	if err := client.Login("", "password"); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestLoginRejectsInvalidConfigurationBeforeLaunching(t *testing.T) {
	client := NewClient(testLoginURL, WithSuccessClass(""))
	client.newPage = func(time.Duration) (page, func(), error) {
		t.Fatal("Login() launched a page for invalid configuration")
		return nil, nil, nil
	}

	if err := client.Login("user", "password"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestLoginWrapsLaunchFailure(t *testing.T) {
	want := errors.New("browser unavailable")
	client := NewClient(testLoginURL)
	client.newPage = func(time.Duration) (page, func(), error) { return nil, nil, want }

	if err := client.Login("user", "password"); !errors.Is(err, want) {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestLoginClassifiesRejectedCredentialsAndClosesPage(t *testing.T) {
	fake := successPage()
	fake.elements["#flash"] = &fakeElement{attr: "flash error", text: "bad password"}
	closed := false
	client := NewClient(testLoginURL)
	client.newPage = func(time.Duration) (page, func(), error) {
		return fake, func() { closed = true }, nil
	}

	err := client.Login("user", "password")
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v", err)
	}
	if !strings.Contains(err.Error(), "bad password") {
		t.Fatalf("Login() error = %v", err)
	}
	if !closed {
		t.Fatal("Login() did not close the page after rejection")
	}
}

func TestFillLoginFormReturnsPageFailures(t *testing.T) {
	tests := map[string]func(*fakePage){
		"navigate": func(page *fakePage) { page.navigateErr = errors.New("navigate") },
		"wait":     func(page *fakePage) { page.waitLoadErr = errors.New("wait") },
		"username": func(page *fakePage) { delete(page.elements, "#username") },
		"password": func(page *fakePage) { delete(page.elements, "#password") },
		"submit":   func(page *fakePage) { delete(page.elements, "button[type='submit']") },
	}
	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			page := successPage()
			configure(page)
			if err := fillLoginForm(page, testLoginURL, newOptions(), "user", "password"); err == nil {
				t.Fatal("fillLoginForm() error = nil")
			}
		})
	}
}

func TestEvaluateLoginResult(t *testing.T) {
	tests := []struct {
		name  string
		class string
		text  string
		want  error
	}{
		{name: "success", class: "flash success"},
		{name: "substring is not success", class: "flash unsuccessful", text: "bad", want: carrierproxy.ErrInvalidCredentials},
		{name: "rejection", class: "flash error", text: "bad", want: carrierproxy.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := newFakePage()
			page.elements["#flash"] = &fakeElement{attr: tt.class, text: tt.text}
			err := evaluateLoginResult(page, newOptions())
			if !errors.Is(err, tt.want) {
				t.Fatalf("evaluateLoginResult() error = %v, want %v", err, tt.want)
			}
		})
	}
}
