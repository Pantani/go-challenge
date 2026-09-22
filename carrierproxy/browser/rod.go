package browser

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// page is the subset of *rod.Page that this package needs. It exists so
// the login/scraping flow can be unit tested against a fake, without a
// real browser.
type page interface {
	Navigate(url string) error
	WaitLoad() error
	Element(selector string) (element, error)
	Elements(selector string) ([]element, error)
	Cookies(targetURL string) ([]cookie, error)
}

// element is the subset of *rod.Element that this package needs.
type element interface {
	Input(value string) error
	Click() error
	Attribute(name string) (string, error)
	Text() (string, error)
	Elements(selector string) ([]element, error)
}

// cookie is a session cookie, decoupled from go-rod's proto types so the
// page interface stays fakeable. It keeps the scoping attributes a real
// browser would enforce (domain, path, secure) so a request to a
// different host than the one the cookie came from doesn't receive it;
// see fetchWithCookies in http.go.
type cookie struct {
	name   string
	value  string
	domain string
	path   string
	secure bool
}

// launchPage starts a headless browser, opens a blank page using ctx,
// and returns it as a page along with a func that releases the
// browser. It prefers a locally installed Chrome/Chromium and only falls
// back to go-rod's own download when none is found. Rod v0.116.2 does not
// propagate ctx through control-URL resolution, its browser-download lock, or
// all provisioning steps, so startup has no absolute wall-clock deadline.
func launchPage(ctx context.Context) (page, func(), error) {
	return launchPageWith(ctx, newLauncher(launcher.LookPath))
}

// newLauncher builds a headless launcher that uses the browser lookPath
// finds, if any. Without an explicit Bin, go-rod ignores any installed
// browser and always downloads its own Chromium snapshot, which is slow
// and, lacking Chrome's setuid sandbox helper, can't start on hosts that
// restrict unprivileged user namespaces (e.g. Ubuntu 24.04).
func newLauncher(lookPath func() (string, bool)) *launcher.Launcher {
	l := launcher.New().Headless(true)
	if bin, ok := lookPath(); ok {
		l = l.Bin(bin)
	}
	return l
}

// launchPageWith is launchPage against a caller-built launcher, so tests
// can point it at a binary that fails in a specific way and exercise the
// cleanup paths without a real browser.
func launchPageWith(ctx context.Context, l *launcher.Launcher) (page, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("carrierproxy: launch browser: %w", err)
	}
	controlURL, err := l.Context(ctx).Launch()
	if err != nil {
		// Launch can fail after it has already started the browser
		// process (e.g. it started but never reported its DevTools URL);
		// killAndCleanup only acts when that actually happened.
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("carrierproxy: launch browser: %w", err)
	}

	return connectPage(ctx, l, controlURL)
}

// connectPage carries the attempt context through the CDP connection and page.
func connectPage(ctx context.Context, l *launcher.Launcher, controlURL string) (page, func(), error) {
	b := rod.New().Context(ctx).ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		// Launch already started the browser process; Connect merely
		// failed to dial it, so unlike release (which first asks the
		// browser to close itself over CDP) the launcher itself has to
		// be the one to kill it, or it leaks.
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("carrierproxy: connect browser: %w", err)
	}

	rodPg, err := b.Page(proto.TargetCreateTarget{})
	if err != nil {
		release(b, l)
		return nil, nil, fmt.Errorf("carrierproxy: open page: %w", err)
	}

	return rodPage{p: rodPg.Context(ctx)}, func() { release(b, l) }, nil
}

// release shuts the browser down and frees everything a successful launch
// holds: it asks the browser to exit over CDP, falls back to killing the
// process if that request can't be delivered (a dead connection would
// otherwise leave Cleanup waiting forever for an exit that never comes),
// then asks Rod to wait for process exit and remove the temporary profile
// Launch created. Graceful CDP shutdown has an independent two-second context,
// but Rod's CDP Send performs a context-unaware websocket Write before it
// observes that deadline, so it is not an absolute socket-write bound. After
// Close reports failure or timeout, shutdown kills the launcher before cleanup.
// Rod's kill, process reaping, and profile cleanup are context-unaware, so final
// cleanup remains best-effort without an absolute deadline; profile removal is
// attempted, not an observable guarantee.
func release(b *rod.Browser, l *launcher.Launcher) {
	shutdownBrowser(func(ctx context.Context) error {
		return b.Context(ctx).Close()
	}, l.Kill, l.Cleanup)
}

// killAndCleanup kills the process l started, if any, then asks Rod to wait for
// process exit and remove its temporary profile. l.PID() is 0 until Launch
// actually starts a process, so a pre-start failure (the browser binary
// missing, say) leaves nothing to kill; calling Cleanup in that case would
// block forever waiting for an exit that will never come, so it's skipped too.
func killAndCleanup(l *launcher.Launcher) {
	if l.PID() == 0 {
		return
	}
	l.Kill()
	l.Cleanup()
}

// rodPage adapts *rod.Page to the page interface.
type rodPage struct{ p *rod.Page }

// Navigate loads url in the page.
func (r rodPage) Navigate(url string) error { return r.p.Navigate(url) }

// WaitLoad blocks until the page finishes loading.
func (r rodPage) WaitLoad() error { return r.p.WaitLoad() }

// Element retries until an element matching selector is found, then
// returns it adapted to the element interface.
func (r rodPage) Element(selector string) (element, error) {
	el, err := r.p.Element(selector)
	if err != nil {
		return nil, err
	}
	return rodElement{el: el}, nil
}

// Elements returns every element matching selector, adapted to the
// element interface.
func (r rodPage) Elements(selector string) ([]element, error) {
	els, err := r.p.Elements(selector)
	if err != nil {
		return nil, err
	}
	return wrapElements(els), nil
}

// Cookies returns the session cookies applicable to targetURL, which
// DocumentDownload attaches to its HTTP request to reuse the authenticated
// session. Chrome 153 did not distinguish host-only and domain cookies on
// localhost, so the download jar withholds cookies from other hostnames.
// Original domains are preserved here; the download jar encodes scope only in
// its internal keys to retain same-name/path cookies and standard global order.
// Original names, path and secure restrictions are retained on HTTP requests,
// but the host boundary can lose continuity on a cross-subdomain redirect.
func (r rodPage) Cookies(targetURL string) ([]cookie, error) {
	raw, err := r.p.Cookies([]string{targetURL})
	if err != nil {
		return nil, err
	}
	if _, err := url.Parse(targetURL); err != nil {
		return nil, fmt.Errorf("carrierproxy: parse cookie target: %w", err)
	}
	cookies := make([]cookie, len(raw))
	for i, c := range raw {
		cookies[i] = cookie{name: c.Name, value: c.Value, domain: c.Domain, path: c.Path, secure: c.Secure}
	}
	return cookies, nil
}

// rodElement adapts *rod.Element to the element interface.
type rodElement struct{ el *rod.Element }

// Input types value into the element, replacing any existing content:
// rod's Input appends to whatever is already there (a browser-remembered
// username, say), so the existing text is selected first and the typed
// value overwrites the selection.
func (r rodElement) Input(value string) error {
	if err := r.el.SelectAllText(); err != nil {
		return err
	}
	return r.el.Input(value)
}

// Click performs a single left click on the element.
func (r rodElement) Click() error { return r.el.Click(proto.InputMouseButtonLeft, 1) }

// Text returns the element's visible text content.
func (r rodElement) Text() (string, error) { return r.el.Text() }

// Elements returns every descendant of the element matching selector,
// adapted to the element interface. Policies uses this to read the cells
// of a row.
func (r rodElement) Elements(selector string) ([]element, error) {
	els, err := r.el.Elements(selector)
	if err != nil {
		return nil, err
	}
	return wrapElements(els), nil
}

// Attribute flattens *rod.Element's *string result; absent attributes become
// an empty string, and login evaluates whole class tokens.
func (r rodElement) Attribute(name string) (string, error) {
	val, err := r.el.Attribute(name)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

// wrapElements adapts a slice of *rod.Element to []element.
func wrapElements(els rod.Elements) []element {
	wrapped := make([]element, len(els))
	for i, el := range els {
		wrapped[i] = rodElement{el: el}
	}
	return wrapped
}
