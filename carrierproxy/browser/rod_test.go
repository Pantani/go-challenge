package browser

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

// TestLaunchPageWithFailedLaunch covers launchPageWith's launch-failure
// cleanup without a real browser, by pointing the launcher at binaries that
// fail in the two ways that matter to killAndCleanup: one that never
// starts (no process to kill) and one that starts but exits without ever
// reporting a DevTools URL (a process to kill and a profile to remove).
func TestLaunchPageWithFailedLaunch(t *testing.T) {
	t.Parallel()

	exitsImmediately, err := exec.LookPath("true")
	if err != nil {
		t.Fatalf("look up the true binary: %v", err)
	}

	tests := map[string]string{
		"binary missing":                      "/nonexistent/carrierproxy-test-chrome",
		"binary exits without a DevTools URL": exitsImmediately,
	}

	for name, bin := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				pg, release, err := launchPageWith(launcher.New().Headless(true).Bin(bin), time.Second)
				if err == nil || !strings.Contains(err.Error(), "launch browser") {
					t.Errorf("expected a launch browser error, got %v", err)
				}
				if pg != nil || release != nil {
					t.Errorf("expected no page and no release func on failure, got %v, %v", pg, release != nil)
				}
			}()

			// Generous on purpose: go-rod serializes launches (and browser
			// downloads) across processes on a shared lock port, so a
			// concurrent browser test can delay this one without it hanging.
			select {
			case <-done:
			case <-time.After(2 * time.Minute):
				t.Fatal("launchPageWith hung cleaning up after a failed launch")
			}
		})
	}
}

// TestNewLauncher checks launchPage uses a locally installed browser when
// there is one, and otherwise leaves Bin unset so go-rod downloads its own.
func TestNewLauncher(t *testing.T) {
	t.Parallel()

	found := newLauncher(func() (string, bool) { return "/opt/chrome/chrome", true })
	if got := found.Get(flags.Bin); got != "/opt/chrome/chrome" {
		t.Fatalf("expected Bin to be the local browser, got %q", got)
	}
	if !found.Has(flags.Headless) {
		t.Fatal("expected the launcher to be headless")
	}

	notFound := newLauncher(func() (string, bool) { return "", false })
	if got := notFound.Get(flags.Bin); got != launcher.New().Get(flags.Bin) {
		t.Fatalf("expected go-rod's default Bin when no browser is found, got %q", got)
	}
	if !notFound.Has(flags.Headless) {
		t.Fatal("expected the launcher to be headless")
	}
}

// adapterPage is served to the go-rod adapter test: a form field, a button
// whose click handler writes to #out, an element with and without a given
// attribute, and a small table to exercise element-scoped Elements.
const adapterPage = `<!DOCTYPE html>
<html><body>
  <input id="name" type="text">
  <button id="btn" onclick="document.getElementById('out').textContent = 'clicked:' + document.getElementById('name').value">Go</button>
  <div id="out"></div>
  <a id="link" href="/elsewhere" data-kind="policy">Link</a>
  <table>
    <tr class="row"><td>a</td><td>b</td></tr>
    <tr class="row"><td>c</td><td>d</td></tr>
  </table>
</body></html>`

// newAdapterSite serves adapterPage and sets a session cookie on it.
func newAdapterSite(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		_, _ = w.Write([]byte(adapterPage))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// requireBrowser skips t unless a local Chrome/Chromium is installed, so the
// default test run never downloads a browser.
func requireBrowser(t *testing.T) {
	t.Helper()
	if _, ok := launcher.LookPath(); !ok {
		t.Skip("no local Chrome/Chromium found; skipping the go-rod adapter test")
	}
}

// TestRodAdapter drives launchPage, rodPage and rodElement against a real
// headless browser and a local page, checking each adapter method maps
// go-rod's behavior onto the page/element interfaces the rest of the
// package is unit tested against. It is skipped when no browser is
// installed.
func TestRodAdapter(t *testing.T) {
	t.Parallel()
	requireBrowser(t)

	srv := newAdapterSite(t)

	pg, release, err := launchPage(30 * time.Second)
	if err != nil {
		t.Fatalf("launch page: %v", err)
	}
	t.Cleanup(release)

	if err := pg.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if err := pg.WaitLoad(); err != nil {
		t.Fatalf("wait load: %v", err)
	}

	t.Run("Input, Click and Text", func(t *testing.T) {
		if err := mustElement(t, pg, "#name").Input("hello"); err != nil {
			t.Fatalf("input: %v", err)
		}
		if err := mustElement(t, pg, "#btn").Click(); err != nil {
			t.Fatalf("click: %v", err)
		}
		got, err := mustElement(t, pg, "#out").Text()
		if err != nil {
			t.Fatalf("text: %v", err)
		}
		if got != "clicked:hello" {
			t.Fatalf("got text %q, want %q", got, "clicked:hello")
		}
	})

	t.Run("Attribute present and absent", func(t *testing.T) {
		link := mustElement(t, pg, "#link")
		if got, err := link.Attribute("data-kind"); err != nil || got != "policy" {
			t.Fatalf("got (%q, %v), want (%q, nil)", got, err, "policy")
		}
		if got, err := link.Attribute("data-missing"); err != nil || got != "" {
			t.Fatalf("got (%q, %v) for a missing attribute, want (\"\", nil)", got, err)
		}
	})

	t.Run("page and element Elements", func(t *testing.T) {
		rows, err := pg.Elements("tr.row")
		if err != nil {
			t.Fatalf("elements: %v", err)
		}
		var got []string
		for _, row := range rows {
			cells, err := row.Elements("td")
			if err != nil {
				t.Fatalf("row elements: %v", err)
			}
			for _, cell := range cells {
				text, err := cell.Text()
				if err != nil {
					t.Fatalf("cell text: %v", err)
				}
				got = append(got, text)
			}
		}
		if strings.Join(got, ",") != "a,b,c,d" {
			t.Fatalf("got cells %v, want [a b c d]", got)
		}

		none, err := pg.Elements(".does-not-exist")
		if err != nil || len(none) != 0 {
			t.Fatalf("got (%d elements, %v), want (0, nil)", len(none), err)
		}
	})

	t.Run("Cookies keep their scoping attributes", func(t *testing.T) {
		cookies, err := pg.Cookies()
		if err != nil {
			t.Fatalf("cookies: %v", err)
		}
		for _, c := range cookies {
			if c.name == "session" {
				if c.value != "abc" || c.domain != "127.0.0.1" || c.path != "/" || c.secure {
					t.Fatalf("unexpected session cookie %+v", c)
				}
				return
			}
		}
		t.Fatalf("session cookie not found in %+v", cookies)
	})
}

// TestRodAdapterErrors checks every adapter method surfaces go-rod's error
// once the page's timeout (set by launchPage) has expired, which is how a
// slow or broken site shows up in practice. It is skipped when no browser
// is installed.
func TestRodAdapterErrors(t *testing.T) {
	t.Parallel()
	requireBrowser(t)

	srv := newAdapterSite(t)

	pg, release, err := launchPage(5 * time.Second)
	if err != nil {
		t.Fatalf("launch page: %v", err)
	}
	t.Cleanup(release)

	if err := pg.Navigate(srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	link := mustElement(t, pg, "#link")

	// Element retries until its selector matches, so a missing one blocks
	// until the page's timeout expires.
	if _, err := pg.Element("#does-not-exist"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded for a missing element, got %v", err)
	}

	checks := map[string]func() error{
		"page Navigate": func() error { return pg.Navigate(srv.URL) },
		"page WaitLoad": pg.WaitLoad,
		"page Elements": func() error { _, err := pg.Elements("a"); return err },
		"page Cookies":  func() error { _, err := pg.Cookies(); return err },
		"element Input": func() error { return link.Input("x") },
		"element Click": link.Click,
		"element Text":  func() error { _, err := link.Text(); return err },
		"element Attribute": func() error {
			_, err := link.Attribute("href")
			return err
		},
		"element Elements": func() error { _, err := link.Elements("span"); return err },
	}
	for name, call := range checks {
		if err := call(); err == nil {
			t.Errorf("%s: expected an error after the page timeout expired", name)
		}
	}
}

// mustElement returns the element matching selector on pg, failing t if
// there is none.
func mustElement(t *testing.T, pg page, selector string) element {
	t.Helper()
	el, err := pg.Element(selector)
	if err != nil {
		t.Fatalf("element %q: %v", selector, err)
	}
	return el
}
