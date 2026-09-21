package browser

import (
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

func TestClientDocumentDownload(t *testing.T) {
	t.Parallel()

	t.Run("requires WithDocumentURL", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		if _, err := c.DocumentDownload("key"); !errors.Is(err, carrierproxy.ErrNotConfigured) {
			t.Fatalf("expected ErrNotConfigured, got %v", err)
		}
	})

	t.Run("requires a prior successful Login", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		if _, err := c.DocumentDownload("key"); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected ErrNotLoggedIn, got %v", err)
		}
	})

	t.Run("rejects an invalid downloadKey", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		c.rememberCredentials("tomsmith", "SuperSecretPassword!")

		for _, key := range []string{"", "has/slash", `has\backslash`} {
			if _, err := c.DocumentDownload(key); err == nil {
				t.Fatalf("expected an error for downloadKey %q", key)
			}
		}
	})

	t.Run("downloads the document after a successful login", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(0))
		c.rememberCredentials("tomsmith", "SuperSecretPassword!")

		fp := successPage()
		fp.cookies = []cookie{{name: "session", value: "abc123"}}
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }

		var gotURL string
		var gotCookies []cookie
		wantBody := io.NopCloser(strings.NewReader("document contents"))
		c.fetch = func(url string, cookies []cookie, _ time.Duration) (io.ReadCloser, error) {
			gotURL, gotCookies = url, cookies
			return wantBody, nil
		}

		got, err := c.DocumentDownload("policy-42.pdf")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != wantBody {
			t.Fatalf("got a different ReadCloser than fetch returned")
		}
		if gotURL != "https://example.com/download/policy-42.pdf" {
			t.Fatalf("got URL %q, want %q", gotURL, "https://example.com/download/policy-42.pdf")
		}
		if !reflect.DeepEqual(gotCookies, fp.cookies) {
			t.Fatalf("got cookies %+v, want %+v", gotCookies, fp.cookies)
		}
	})

	t.Run("retries a transient failure", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL), WithRetries(1))
		c.sleep = func(time.Duration) {}
		c.rememberCredentials("tomsmith", "SuperSecretPassword!")

		fp := successPage()
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }

		calls := 0
		wantBody := io.NopCloser(strings.NewReader("ok"))
		c.fetch = func(string, []cookie, time.Duration) (io.ReadCloser, error) {
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
		if got != wantBody {
			t.Fatalf("got a different ReadCloser than fetch returned")
		}
		if calls != 2 {
			t.Fatalf("expected 2 attempts, got %d", calls)
		}
	})
}

func TestValidateDownloadKey(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		key     string
		wantErr bool
	}{
		"valid key":            {"policy-42.pdf", false},
		"empty key":            {"", true},
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

	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}

	t.Run("re-authentication fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		wantErr := errors.New("boom")
		c.newPage = func(time.Duration) (page, func(), error) { return nil, nil, wantErr }
		if _, err := c.fetchDocument(creds, "key"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("reading cookies fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		fp := successPage()
		wantErr := errors.New("boom")
		fp.cookiesErr = wantErr
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if _, err := c.fetchDocument(creds, "key"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("the fetch itself fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		fp := successPage()
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		wantErr := errors.New("boom")
		c.fetch = func(string, []cookie, time.Duration) (io.ReadCloser, error) { return nil, wantErr }
		if _, err := c.fetchDocument(creds, "key"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("releases the page once done", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithDocumentURL(testDocumentURL))
		fp := successPage()
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		c.fetch = func(string, []cookie, time.Duration) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("")), nil
		}
		if _, err := c.fetchDocument(creds, "key"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !closed {
			t.Fatal("expected the page to be released")
		}
	})
}
