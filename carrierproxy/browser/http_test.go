package browser

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchWithCookies(t *testing.T) {
	t.Parallel()

	t.Run("attaches cookies and returns the body", func(t *testing.T) {
		t.Parallel()
		var gotCookies []*http.Cookie
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotCookies = r.Cookies()
			_, _ = w.Write([]byte("document contents"))
		}))
		defer srv.Close()

		rc, err := fetchWithCookies(srv.URL, []cookie{{name: "session", value: "abc123"}}, 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = rc.Close() }()

		body, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		if string(body) != "document contents" {
			t.Fatalf("got body %q, want %q", body, "document contents")
		}
		if len(gotCookies) != 1 || gotCookies[0].Name != "session" || gotCookies[0].Value != "abc123" {
			t.Fatalf("got cookies %+v, want session=abc123", gotCookies)
		}
	})

	t.Run("a non-200 status is an error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
		defer srv.Close()

		if _, err := fetchWithCookies(srv.URL, nil, 5*time.Second); err == nil {
			t.Fatal("expected an error for a 404 response")
		}
	})

	t.Run("a malformed URL is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := fetchWithCookies("://not-a-url", nil, time.Second); err == nil {
			t.Fatal("expected an error for a malformed URL")
		}
	})

	t.Run("an unreachable host is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := fetchWithCookies("http://127.0.0.1:1", nil, 2*time.Second); err == nil {
			t.Fatal("expected an error for an unreachable host")
		}
	})
}
