package browser

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func stalledResponseServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush response: %v", err)
			return
		}
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchWithCookiesBodyCancellation(t *testing.T) {
	srv := stalledResponseServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body, err := fetchWithCookies(ctx, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	cancel()
	if _, err := io.ReadAll(body); !errors.Is(err, context.Canceled) {
		t.Fatalf("read error = %v, want context.Canceled", err)
	}
}

func TestFetchWithCookiesBodyDeadline(t *testing.T) {
	srv := stalledResponseServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	body, err := fetchWithCookies(ctx, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = body.Close() })
	if _, err := io.ReadAll(body); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("read error = %v, want context.DeadlineExceeded", err)
	}
}

func TestFetchWithCookies(t *testing.T) {
	t.Parallel()

	t.Run("attaches a cookie scoped to the request's own host", testFetchWithCookiesOwnHost)

	t.Run("does not leak a cookie scoped to a different host", testFetchWithCookiesOtherHost)

	t.Run("ignores a cookie with no domain rather than sending it everywhere", testFetchWithCookiesNoDomain)

	t.Run("a non-200 status is an error", testFetchWithCookiesNotFound)

	t.Run("a malformed URL is an error", testFetchWithCookiesMalformedURL)

	t.Run("an unreachable host is an error", testFetchWithCookiesUnreachableHost)
}

func testFetchWithCookiesOwnHost(t *testing.T) {
	t.Parallel()
	var gotCookies []*http.Cookie
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookies = r.Cookies()
		_, _ = w.Write([]byte("document contents"))
	}))
	defer srv.Close()
	host := serverHost(t, srv)

	rc, err := fetchWithCookies(context.Background(), srv.URL, []cookie{{name: "session", value: "abc123", domain: host, path: "/"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := rc.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	})

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(body) != "document contents" {
		t.Fatalf("got body %q, want %q", body, "document contents")
	}
	assertReceivedSessionCookie(t, gotCookies)
}

func assertReceivedSessionCookie(t *testing.T, gotCookies []*http.Cookie) {
	t.Helper()
	if len(gotCookies) != 1 || gotCookies[0].Name != "session" || gotCookies[0].Value != "abc123" {
		t.Fatalf("got cookies %+v, want session=abc123", gotCookies)
	}
}

func testFetchWithCookiesOtherHost(t *testing.T) {
	t.Parallel()
	var gotCookies []*http.Cookie
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookies = r.Cookies()
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := fetchWithCookies(context.Background(), srv.URL, []cookie{{name: "other-session", value: "leak-me-not", domain: "some-other-host.example", path: "/"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	})
	if len(gotCookies) != 0 {
		t.Fatalf("expected no cookies to be sent, got %+v", gotCookies)
	}
}

func testFetchWithCookiesNoDomain(t *testing.T) {
	t.Parallel()
	var gotCookies []*http.Cookie
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookies = r.Cookies()
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := fetchWithCookies(context.Background(), srv.URL, []cookie{{name: "no-domain", value: "x"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	})
	if len(gotCookies) != 0 {
		t.Fatalf("expected no cookies to be sent, got %+v", gotCookies)
	}
}

func testFetchWithCookiesNotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	defer srv.Close()

	if _, err := fetchWithCookies(context.Background(), srv.URL, nil); err == nil {
		t.Fatal("expected an error for a 404 response")
	}
}

func testFetchWithCookiesMalformedURL(t *testing.T) {
	t.Parallel()
	if _, err := fetchWithCookies(context.Background(), "://not-a-url", nil); err == nil {
		t.Fatal("expected an error for a malformed URL")
	}
}

func testFetchWithCookiesUnreachableHost(t *testing.T) {
	t.Parallel()
	if _, err := fetchWithCookies(context.Background(), "http://127.0.0.1:1", nil); err == nil {
		t.Fatal("expected an error for an unreachable host")
	}
}

// serverHost returns srv's bare hostname (no port), matching what a
// cookie's Domain attribute would hold for it.
func serverHost(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	return u.Hostname()
}
