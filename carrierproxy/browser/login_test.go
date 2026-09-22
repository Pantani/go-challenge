package browser

import (
	"context"
	"errors"
	"reflect"
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

func TestValidateLoginOptions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		successClass string
		wantErr      error
	}{
		"the default success class is fine":       {newOptions().successClass, nil},
		"any non-empty class is fine":             {"ok", nil},
		"an empty class is a configuration error": {"", carrierproxy.ErrNotConfigured},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := newOptions()
			opts.successClass = tc.successClass
			if err := validateLoginOptions(opts); !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
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

	t.Run("happy path", testFillLoginFormHappyPath)

	t.Run("navigate fails", testFillLoginFormNavigateFails)

	t.Run("wait load fails", testFillLoginFormWaitLoadFails)

	t.Run("username field missing", testFillLoginFormUsernameMissing)

	t.Run("password field missing", testFillLoginFormPasswordMissing)

	t.Run("submit missing", testFillLoginFormSubmitMissing)
}

func testFillLoginFormHappyPath(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testFillLoginFormNavigateFails(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	p.navigateErr = errors.New("boom")
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); !errors.Is(err, p.navigateErr) {
		t.Fatalf("expected navigate error, got %v", err)
	}
}

func testFillLoginFormWaitLoadFails(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	p.waitLoadErr = errors.New("boom")
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); !errors.Is(err, p.waitLoadErr) {
		t.Fatalf("expected wait load error, got %v", err)
	}
}

func testFillLoginFormUsernameMissing(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	delete(p.elements, opts.usernameSelector)
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
		t.Fatal("expected error")
	}
}

func testFillLoginFormPasswordMissing(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	delete(p.elements, opts.passwordSelector)
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
		t.Fatal("expected error")
	}
}

func testFillLoginFormSubmitMissing(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := successPage()
	delete(p.elements, opts.submitSelector)
	if err := fillLoginForm(p, testLoginURL, opts, "user", "pass"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginProtocol(t *testing.T) {
	p := successPage()
	if err := fillLoginForm(p, testLoginURL, newOptions(), "alice", "secret"); err != nil {
		t.Fatal(err)
	}
	if err := evaluateLoginResult(p, newOptions()); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"navigate:" + testLoginURL, "wait-load",
		"element:#username", "input:#username=alice",
		"element:#password", "input:#password=secret",
		"element:button[type='submit']", "click:button[type='submit']",
		"element:#flash", "attribute:#flash=class",
	}
	if !reflect.DeepEqual(p.events, want) {
		t.Fatalf("protocol = %#v, want %#v", p.events, want)
	}
}

func TestEvaluateLoginResult(t *testing.T) {
	t.Parallel()

	t.Run("success class", testEvaluateLoginResultSuccessClass)

	t.Run("a class that merely contains the success token as a substring is not a match", testEvaluateLoginResultSubstringRejected)

	t.Run("error class carries message", testEvaluateLoginResultErrorMessage)

	t.Run("error class with unreadable text still fails", testEvaluateLoginResultUnreadableErrorText)

	t.Run("banner missing", testEvaluateLoginResultBannerMissing)

	t.Run("class attribute unreadable", testEvaluateLoginResultUnreadableClass)
}

func testEvaluateLoginResultSuccessClass(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := newFakePage()
	p.elements[opts.resultSelector] = &fakeElement{attr: "flash success"}
	if err := evaluateLoginResult(p, opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func testEvaluateLoginResultSubstringRejected(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := newFakePage()
	p.elements[opts.resultSelector] = &fakeElement{attr: "flash unsuccessful", text: "Your login was unsuccessful"}
	err := evaluateLoginResult(p, opts)
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for a merely-substring class match, got %v", err)
	}
}

func testEvaluateLoginResultErrorMessage(t *testing.T) {
	opts := newOptions()
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
}

func testEvaluateLoginResultUnreadableErrorText(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := newFakePage()
	p.elements[opts.resultSelector] = &fakeElement{attr: "flash error", textErr: errors.New("boom")}
	err := evaluateLoginResult(p, opts)
	if !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func testEvaluateLoginResultBannerMissing(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := newFakePage()
	wantErr := errors.New("boom")
	p.elementErr[opts.resultSelector] = wantErr
	if err := evaluateLoginResult(p, opts); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testEvaluateLoginResultUnreadableClass(t *testing.T) {
	opts := newOptions()
	t.Parallel()
	p := newFakePage()
	wantErr := errors.New("boom")
	p.elements[opts.resultSelector] = &fakeElement{attrErr: wantErr}
	if err := evaluateLoginResult(p, opts); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestHasClassToken(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		classAttr, token string
		want             bool
	}{
		"exact single class":       {"success", "success", true},
		"one of several classes":   {"flash success", "success", true},
		"substring is not a match": {"unsuccessful", "success", false},
		"prefix is not a match":    {"success-banner", "success", false},
		"no match at all":          {"flash error", "success", false},
		"empty class attribute":    {"", "success", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := hasClassToken(tc.classAttr, tc.token); got != tc.want {
				t.Fatalf("hasClassToken(%q, %q) = %v, want %v", tc.classAttr, tc.token, got, tc.want)
			}
		})
	}
}

func TestClientLogin(t *testing.T) {
	t.Parallel()

	t.Run("rejects blank credentials before opening a page", testClientLoginBlankCredentials)

	t.Run("wraps browser launch failure", testClientLoginLaunchFailure)

	t.Run("closes the page even on failure", testClientLoginFailureClosesPage)

	t.Run("succeeds against a valid form and flash, and remembers the credentials", testClientLoginSuccessRemembersCredentials)

	t.Run("surfaces invalid credentials from the target site without remembering them", testClientLoginInvalidCredentials)

	t.Run("a misconfigured client fails before opening a page and is not retried", testClientLoginMisconfigured)

	t.Run("retries a transient launch failure, then succeeds and remembers the credentials", testClientLoginTransientLaunch)

	t.Run("exhausts retries, returns the last error and remembers nothing", testClientLoginRetriesExhausted)

	t.Run("honors WithUsernameSelector and other options end to end", testClientLoginCustomSelectors)
}

func testClientLoginBlankCredentials(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	c.newPage = func(context.Context) (page, func(), error) {
		t.Fatal("newPage should not be called for invalid credentials")
		return nil, nil, nil
	}
	if err := c.Login("", "pass"); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func testClientLoginLaunchFailure(t *testing.T) {
	t.Parallel()
	// WithRetries(0): this checks one attempt's error handling, not
	// the retry mechanism itself (see TestClientWithRetries).
	c := NewClient(testLoginURL, WithRetries(0))
	wantErr := errors.New("boom")
	c.newPage = func(context.Context) (page, func(), error) { return nil, nil, wantErr }
	if err := c.Login("user", "pass"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testClientLoginFailureClosesPage(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(0))
	fp := successPage()
	delete(fp.elements, newOptions().submitSelector)
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	if err := c.Login("user", "pass"); err == nil {
		t.Fatal("expected error")
	}
	if !closed {
		t.Fatal("expected page to be closed")
	}
}

func testClientLoginSuccessRemembersCredentials(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	fp := successPage()
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
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
}

func testClientLoginInvalidCredentials(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	fp := successPage()
	fp.elements[newOptions().resultSelector] = &fakeElement{attr: "flash error", text: "Your password is invalid!"}
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if err := c.Login("tomsmith", "wrong"); !errors.Is(err, carrierproxy.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := c.storedCredentials(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected rejected credentials not to be remembered, got %v", err)
	}
}

func testClientLoginMisconfigured(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithSuccessClass(""), WithRetries(3))
	c.wait = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep: a configuration error must not be retried")
		return nil
	}
	c.newPage = func(context.Context) (page, func(), error) {
		t.Fatal("newPage should not be called for a misconfigured client")
		return nil, nil, nil
	}
	if err := c.Login("user", "pass"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func testClientLoginTransientLaunch(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(1))
	c.wait = func(context.Context, time.Duration) error { return nil }
	launches := 0
	c.newPage = func(context.Context) (page, func(), error) {
		launches++
		if launches == 1 {
			return nil, nil, errors.New("browser slow to start")
		}
		return successPage(), func() {}, nil
	}
	if err := c.Login("tomsmith", "SuperSecretPassword!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if launches != 2 {
		t.Fatalf("expected 2 launches, got %d", launches)
	}
	if _, err := c.storedCredentials(); err != nil {
		t.Fatalf("expected credentials to be remembered after the retried success, got %v", err)
	}
}

func testClientLoginRetriesExhausted(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithRetries(2))
	c.wait = func(context.Context, time.Duration) error { return nil }
	wantErr := errors.New("persistent")
	launches := 0
	c.newPage = func(context.Context) (page, func(), error) { launches++; return nil, nil, wantErr }
	if err := c.Login("user", "pass"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if launches != 3 {
		t.Fatalf("expected 3 launches (1 + 2 retries), got %d", launches)
	}
	if _, err := c.storedCredentials(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected nothing remembered after a failed Login, got %v", err)
	}
}

func testClientLoginCustomSelectors(t *testing.T) {
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
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if err := c.Login("user", "pass"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientLoginConcurrentSafety(t *testing.T) {
	t.Parallel()

	c := NewClient(testLoginURL)
	c.newPage = func(context.Context) (page, func(), error) { return successPage(), func() {}, nil }

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
