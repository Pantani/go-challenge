package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// page is the subset of *rod.Page used by Login. It keeps the login flow
// independently testable without a real browser.
type page interface {
	Navigate(url string) error
	WaitLoad() error
	Element(selector string) (element, error)
}

// element is the subset of *rod.Element used by Login.
type element interface {
	Input(value string) error
	Click() error
	Attribute(name string) (string, error)
	Text() (string, error)
}

// launchPage starts a headless browser, opens a blank page and returns it as
// a page along with a func that releases the browser. It prefers a locally
// installed Chrome/Chromium and only falls back to go-rod's own download when
// none is found. timeout bounds the whole Login attempt.
func launchPage(timeout time.Duration) (page, func(), error) {
	return launchPageWith(newLauncher(launcher.LookPath), timeout)
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
func launchPageWith(l *launcher.Launcher, timeout time.Duration) (page, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	controlURL, err := l.Context(ctx).Launch()
	if err != nil {
		cancel()
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("launch browser: %w", err)
	}

	return connectPage(ctx, cancel, l, controlURL)
}

// connectPage connects to the launched browser at controlURL and opens a
// blank page, carrying the attempt context through both. On failure it
// releases whatever was already started.
func connectPage(ctx context.Context, cancel context.CancelFunc, l *launcher.Launcher, controlURL string) (page, func(), error) {
	b := rod.New().Context(ctx).ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		cancel()
		killAndCleanup(l)
		return nil, nil, fmt.Errorf("connect browser: %w", err)
	}

	rodPage, err := b.Page(proto.TargetCreateTarget{})
	if err != nil {
		release(b, l, cancel)
		return nil, nil, fmt.Errorf("open page: %w", err)
	}

	return pageAdapter{page: rodPage.Context(ctx)}, func() { release(b, l, cancel) }, nil
}

// release shuts the browser down and frees everything a successful launch
// holds: it asks the browser to exit over CDP, falls back to killing the
// process if that request can't be delivered, then lets Rod wait for
// process exit and remove the temporary profile Launch created.
func release(b *rod.Browser, l *launcher.Launcher, cancel context.CancelFunc) {
	defer cancel()
	if err := b.Close(); err != nil {
		l.Kill()
	}
	l.Cleanup()
}

// killAndCleanup kills the process l started, if any, then asks Rod to wait
// for process exit and remove its temporary profile. l.PID() is 0 until
// Launch actually starts a process, so a pre-start failure (the browser
// binary missing, say) leaves nothing to kill; calling Cleanup in that case
// would block forever waiting for an exit that never comes, so it's skipped.
func killAndCleanup(l *launcher.Launcher) {
	if l.PID() == 0 {
		return
	}
	l.Kill()
	l.Cleanup()
}

// pageAdapter adapts *rod.Page to the page interface.
type pageAdapter struct{ page *rod.Page }

func (p pageAdapter) Navigate(target string) error { return p.page.Navigate(target) }
func (p pageAdapter) WaitLoad() error              { return p.page.WaitLoad() }

func (p pageAdapter) Element(selector string) (element, error) {
	el, err := p.page.Element(selector)
	if err != nil {
		return nil, err
	}
	return elementAdapter{element: el}, nil
}

// elementAdapter adapts *rod.Element to the element interface.
type elementAdapter struct{ element *rod.Element }

// Input clears any pre-filled text before typing value, so a remembered
// username is replaced rather than appended to.
func (e elementAdapter) Input(value string) error {
	if err := e.element.SelectAllText(); err != nil {
		return err
	}
	return e.element.Input(value)
}

func (e elementAdapter) Click() error          { return e.element.Click(proto.InputMouseButtonLeft, 1) }
func (e elementAdapter) Text() (string, error) { return e.element.Text() }

// Attribute returns the named attribute, or "" when the element lacks it.
func (e elementAdapter) Attribute(name string) (string, error) {
	value, err := e.element.Attribute(name)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}
	return *value, nil
}
