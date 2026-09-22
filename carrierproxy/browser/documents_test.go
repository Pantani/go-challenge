package browser

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func testDocumentURL(downloadKey string) string {
	return "https://example.com/download/" + downloadKey
}

func TestDocumentCookiesUseDocumentTarget(t *testing.T) {
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(0))
	p := successPage()
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("policy")), nil
	}
	body, err := c.fetchDocumentOnPage(context.Background(), p, "a report?.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	want := []string{"https://example.com/download/a%20report%3F.pdf"}
	if !reflect.DeepEqual(p.cookieTargets, want) {
		t.Fatalf("cookie targets = %v", p.cookieTargets)
	}
}

func TestClientDocumentDownload(t *testing.T) {
	t.Parallel()

	t.Run("requires WithDocumentURL", testClientDocumentDownloadMissingURL)

	t.Run("requires a prior successful Login", testClientDocumentDownloadMissingLogin)

	t.Run("rejects an invalid downloadKey without opening a page", testClientDocumentDownloadInvalidKey)

	t.Run("percent-encodes the downloadKey before building the URL", testClientDocumentDownloadEncodedKey)

	t.Run("downloads the document after a successful login", testClientDocumentDownloadSuccess)

	t.Run("exhausts retries and returns the last error with no document", testClientDocumentDownloadRetriesExhausted)

	t.Run("retries a transient failure", testClientDocumentDownloadTransientFailure)
}

func testClientDocumentDownloadMissingURL(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	if _, err := c.DocumentDownload("key"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func testClientDocumentDownloadMissingLogin(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	if _, err := c.DocumentDownload("key"); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func testClientDocumentDownloadInvalidKey(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")
	c.newPage = func(context.Context) (page, func(), error) {
		t.Fatal("newPage should not be called for an invalid downloadKey")
		return nil, nil, nil
	}

	for _, key := range []string{"", ".", "..", "has/slash", `has\backslash`} {
		if _, err := c.DocumentDownload(key); err == nil {
			t.Fatalf("expected an error for downloadKey %q", key)
		}
	}
}

func testClientDocumentDownloadEncodedKey(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(0))
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")
	fp := successPage()
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }

	var gotURL string
	c.fetch = func(_ context.Context, url string, _ []cookie) (io.ReadCloser, error) {
		gotURL = url
		return io.NopCloser(strings.NewReader("")), nil
	}

	body, err := c.DocumentDownload("a report?.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := body.Close(); err != nil {
			t.Errorf("close document: %v", err)
		}
	})
	want := "https://example.com/download/a%20report%3F.pdf"
	if gotURL != want {
		t.Fatalf("got URL %q, want %q", gotURL, want)
	}
}

func testClientDocumentDownloadSuccess(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(0))
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")

	fp := successPage()
	fp.cookies = []cookie{{name: "session", value: "abc123"}}
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }

	var gotURL string
	var gotCookies []cookie
	wantBody := &observedBody{Reader: strings.NewReader("document contents")}
	c.fetch = func(_ context.Context, url string, cookies []cookie) (io.ReadCloser, error) {
		gotURL, gotCookies = url, cookies
		return wantBody, nil
	}

	got, err := c.DocumentDownload("policy-42.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := got.Close(); err != nil {
			t.Errorf("close document: %v", err)
		}
	})
	assertDocumentBody(t, got, wantBody, "document contents")
	if gotURL != "https://example.com/download/policy-42.pdf" {
		t.Fatalf("got URL %q, want %q", gotURL, "https://example.com/download/policy-42.pdf")
	}
	if !reflect.DeepEqual(gotCookies, fp.cookies) {
		t.Fatalf("got cookies %+v, want %+v", gotCookies, fp.cookies)
	}
}

func testClientDocumentDownloadRetriesExhausted(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(1))
	c.wait = func(context.Context, time.Duration) error { return nil }
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")
	fp := successPage()
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	wantErr := errors.New("persistent")
	calls := 0
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) { calls++; return nil, wantErr }

	got, err := c.DocumentDownload("key")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != nil {
		t.Fatalf("expected no document on failure, got %v", got)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func testClientDocumentDownloadTransientFailure(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(1))
	c.wait = func(context.Context, time.Duration) error { return nil }
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")

	fp := successPage()
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }

	calls := 0
	wantBody := &observedBody{Reader: strings.NewReader("ok")}
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("transient")
		}
		return wantBody, nil
	}

	got, err := c.DocumentDownload("key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := got.Close(); err != nil {
			t.Errorf("close document: %v", err)
		}
	})
	assertDocumentBody(t, got, wantBody, "ok")
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func assertDocumentBody(t *testing.T, body io.ReadCloser, raw *observedBody, want string) {
	t.Helper()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if !raw.closed {
		t.Fatal("underlying response was not closed")
	}
}

func TestValidateDownloadKey(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		key     string
		wantErr bool
	}{
		"valid key":            {"policy-42.pdf", false},
		"empty key":            {"", true},
		"exactly a dot":        {".", true},
		"exactly two dots":     {"..", true},
		"forward slash":        {"a/b", true},
		"backslash":            {`a\b`, true},
		"dots without a slash": {"..policy", false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := validateDownloadKey(tc.key)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFetchDocument(t *testing.T) {
	t.Parallel()

	t.Run("re-authentication fails", testFetchDocumentAuthenticationFailure)

	t.Run("reading cookies fails", testFetchDocumentCookieFailure)

	t.Run("the fetch itself fails", testFetchDocumentFetchFailure)

	t.Run("releases the page once done", testFetchDocumentReleasesPage)
}

func testFetchDocumentAuthenticationFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	wantErr := errors.New("boom")
	c.newPage = func(context.Context) (page, func(), error) { return nil, nil, wantErr }
	if _, err := c.fetchDocument(context.Background(), creds, "key"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testFetchDocumentCookieFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	fp := successPage()
	wantErr := errors.New("boom")
	fp.cookiesErr = wantErr
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if _, err := c.fetchDocument(context.Background(), creds, "key"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testFetchDocumentFetchFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	fp := successPage()
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	wantErr := errors.New("boom")
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) { return nil, wantErr }
	if _, err := c.fetchDocument(context.Background(), creds, "key"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testFetchDocumentReleasesPage(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
	fp := successPage()
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	c.fetch = func(context.Context, string, []cookie) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}
	body, err := c.fetchDocument(context.Background(), creds, "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := body.Close(); err != nil {
			t.Errorf("close document: %v", err)
		}
	})
	if !closed {
		t.Fatal("expected the page to be released")
	}
}
