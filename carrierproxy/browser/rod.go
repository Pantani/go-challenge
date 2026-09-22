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

func launchPage(timeout time.Duration) (page, func(), error) {
	return launchPageWith(newLauncher(launcher.LookPath), timeout)
}

func newLauncher(lookPath func() (string, bool)) *launcher.Launcher {
	l := launcher.New().Headless(true)
	if bin, ok := lookPath(); ok {
		l = l.Bin(bin)
	}
	return l
}

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

func release(b *rod.Browser, l *launcher.Launcher, cancel context.CancelFunc) {
	defer cancel()
	if err := b.Close(); err != nil {
		l.Kill()
	}
	l.Cleanup()
}

func killAndCleanup(l *launcher.Launcher) {
	if l.PID() == 0 {
		return
	}
	l.Kill()
	l.Cleanup()
}

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

type elementAdapter struct{ element *rod.Element }

func (e elementAdapter) Input(value string) error {
	if err := e.element.SelectAllText(); err != nil {
		return err
	}
	return e.element.Input(value)
}

func (e elementAdapter) Click() error          { return e.element.Click(proto.InputMouseButtonLeft, 1) }
func (e elementAdapter) Text() (string, error) { return e.element.Text() }

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
