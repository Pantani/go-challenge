package browser

import (
	"fmt"
	"time"

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
	Cookies() ([]cookie, error)
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

// launchPage starts a headless browser, opens a blank page on it bounded
// by timeout, and returns it as a page along with a func that releases the
// browser. It is the only function in this package that talks to go-rod
// directly.
func launchPage(timeout time.Duration) (page, func(), error) {
	l := launcher.New().Headless(true)
	controlURL, err := l.Launch()
	if err != nil {
		// Launch can fail after it has already started the browser
		// process (e.g. it started but never reported its DevTools URL);
		// killAndCleanup only acts when that actually happened.
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("carrierproxy: launch browser: %w", err)
	}

	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		// Launch already started the browser process; Connect merely
		// failed to dial it, so unlike release() below (which needs a
		// live CDP connection to ask the browser to close itself) the
		// launcher itself has to be the one to kill it, or it leaks.
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("carrierproxy: connect browser: %w", err)
	}

	// b.Close() only asks the browser to exit; it leaves the temporary
	// profile directory Launch created behind. l.Cleanup() waits for the
	// browser to actually exit and removes that directory, so every
	// launch this function returns successfully from must be paired with
	// a release() call, on both the success and page-open-failure paths.
	release := func() {
		_ = b.Close()
		l.Cleanup()
	}

	rodPg, err := b.Page(proto.TargetCreateTarget{})
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("carrierproxy: open page: %w", err)
	}

	return rodPage{p: rodPg.Timeout(timeout)}, release, nil
}

// killAndCleanup kills the process l started, if any, and removes its
// temporary profile directory. l.PID() is 0 until Launch actually starts
// a process, so a pre-start failure (the browser binary missing, say)
// leaves nothing to kill; calling Cleanup in that case would block
// forever waiting for an exit that will never come, so it's skipped too.
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

// Cookies returns the session cookies for the page's current URL, which
// DocumentDownload attaches to its plain HTTP request to reuse the
// authenticated session.
func (r rodPage) Cookies() ([]cookie, error) {
	raw, err := r.p.Cookies(nil)
	if err != nil {
		return nil, err
	}
	cookies := make([]cookie, len(raw))
	for i, c := range raw {
		cookies[i] = cookie{name: c.Name, value: c.Value, domain: c.Domain, path: c.Path, secure: c.Secure}
	}
	return cookies, nil
}

// rodElement adapts *rod.Element to the element interface.
type rodElement struct{ el *rod.Element }

// Input types value into the element, replacing any existing content.
func (r rodElement) Input(value string) error { return r.el.Input(value) }

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

// Attribute flattens *rod.Element's *string result (nil when the attribute
// is absent) into a plain string, since callers only care about substrings.
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
