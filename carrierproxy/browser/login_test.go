package browser

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		username, password string
		wantErr            bool
	}{
		"valid":            {"user", "pass", false},
		"missing username": {"", "pass", true},
		"missing password": {"user", "", true},
		"missing both":     {"", "", true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := validateCredentials(tc.username, tc.password)
			if tc.wantErr && !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
				t.Fatalf("expected ErrInvalidCredentials, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestFillField(t *testing.T) {
	t.Parallel()

	t.Run("input succeeds", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		p.elements["#field"] = &fakeElement{}
		if err := fillField(p, "#field", "value"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("element lookup fails", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elementErr["#field"] = wantErr
		if err := fillField(p, "#field", "value"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("input fails", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elements["#field"] = &fakeElement{inputErr: wantErr}
		if err := fillField(p, "#field", "value"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestClickSubmit(t *testing.T) {
	t.Parallel()

	t.Run("click succeeds", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		p.elements["#submit"] = &fakeElement{}
		if err := clickSubmit(p, "#submit"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("element lookup fails", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elementErr["#submit"] = wantErr
		if err := clickSubmit(p, "#submit"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("click fails", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elements["#submit"] = &fakeElement{clickErr: wantErr}
		if err := clickSubmit(p, "#submit"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestFillLoginForm(t *testing.T) {
	t.Parallel()

	opts := newOptions()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("navigate fails", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		p.navigateErr = errors.New("boom")
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); !errors.Is(err, p.navigateErr) {
			t.Fatalf("expected navigate error, got %v", err)
		}
	})

	t.Run("wait load fails", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		p.waitLoadErr = errors.New("boom")
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); !errors.Is(err, p.waitLoadErr) {
			t.Fatalf("expected wait load error, got %v", err)
		}
	})

	t.Run("username field missing", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		delete(p.elements, opts.usernameSelector)
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("password field missing", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		delete(p.elements, opts.passwordSelector)
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("submit missing", func(t *testing.T) {
		t.Parallel()
		p := successPage()
		delete(p.elements, opts.submitSelector)
		if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEvaluateLoginResult(t *testing.T) {
	t.Parallel()

	opts := newOptions()

	t.Run("success class", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		p.elements[opts.resultSelector] = &fakeElement{attr: "flash success"}
		if err := evaluateLoginResult(p, opts); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("error class carries message", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		p.elements[opts.resultSelector] = &fakeElement{attr: "flash error", text: "Your username is invalid!"}
		err := evaluateLoginResult(p, opts)
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if !strings.Contains(err.Error(), "Your username is invalid!") {
			t.Fatalf("expected message in error, got %v", err)
		}
	})

	t.Run("error class with unreadable text still fails", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		p.elements[opts.resultSelector] = &fakeElement{attr: "flash error", textErr: errors.New("boom")}
		err := evaluateLoginResult(p, opts)
		if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("banner missing", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elementErr[opts.resultSelector] = wantErr
		if err := evaluateLoginResult(p, opts); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("class attribute unreadable", func(t *testing.T) {
		t.Parallel()
		p := newFakePage()
		wantErr := errors.New("boom")
		p.elements[opts.resultSelector] = &fakeElement{attrErr: wantErr}
		if err := evaluateLoginResult(p, opts); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestClientLogin(t *testing.T) {
	t.Parallel()

	t.Run("rejects blank credentials before opening a page", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		c.newPage = func(time.Duration) (page, func(), error) {
			t.Fatal("newPage should not be called for invalid credentials")
			return nil, nil, nil
		}
		if err := c.Login("", "pass"); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("wraps browser launch failure", func(t *testing.T) {
		t.Parallel()
		// WithRetries(0): this checks one attempt's error handling, not
		// the retry mechanism itself (see TestClientWithRetries).
		c := NewClient(testLoginURL, WithRetries(0))
		wantErr := errors.New("boom")
		c.newPage = func(time.Duration) (page, func(), error) { return nil, nil, wantErr }
		if err := c.Login("user", "pass"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("closes the page even on failure", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithRetries(0))
		fp := successPage()
		delete(fp.elements, newOptions().submitSelector)
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		if err := c.Login("user", "pass"); err == nil {
			t.Fatal("expected error")
		}
		if !closed {
			t.Fatal("expected page to be closed")
		}
	})

	t.Run("succeeds against a valid form and flash, and remembers the credentials", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		fp := successPage()
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if err := c.Login("tomsmith", "SuperSecretPassword!"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		creds, err := c.storedCredentials()
		if err != nil {
			t.Fatalf("expected credentials to be remembered, got %v", err)
		}
		if creds.username != "tomsmith" || creds.password != "SuperSecretPassword!" {
			t.Fatalf("got %+v, want tomsmith/SuperSecretPassword!", creds)
		}
	})

	t.Run("surfaces invalid credentials from the target site without remembering them", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		fp := successPage()
		fp.elements[newOptions().resultSelector] = &fakeElement{attr: "flash error", text: "Your password is invalid!"}
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if err := c.Login("tomsmith", "wrong"); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if _, err := c.storedCredentials(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected rejected credentials not to be remembered, got %v", err)
		}
	})

	t.Run("honors WithUsernameSelector and other options end to end", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL,
			WithUsernameSelector("#u"),
			WithPasswordSelector("#p"),
			WithSubmitSelector("#go"),
			WithResultSelector("#result"),
			WithSuccessClass("ok"),
		)
		fp := newFakePage()
		fp.elements["#u"] = &fakeElement{}
		fp.elements["#p"] = &fakeElement{}
		fp.elements["#go"] = &fakeElement{}
		fp.elements["#result"] = &fakeElement{attr: "banner ok"}
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if err := c.Login("user", "pass"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestClientLoginConcurrentSafety(t *testing.T) {
	t.Parallel()

	c := NewClient(testLoginURL)
	c.newPage = func(time.Duration) (page, func(), error) { return successPage(), func() {}, nil }

	const n = 20
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() { errs <- c.Login("tomsmith", "SuperSecretPassword!") }()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("unexpected error from concurrent Login: %v", err)
		}
	}
}
