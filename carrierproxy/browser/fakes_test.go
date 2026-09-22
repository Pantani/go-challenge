package browser

import "errors"

// testLoginURL is the login URL the fake-driven tests use.
const testLoginURL = "https://example.com/login"

// fakeElement is a scriptable element used to drive the login logic
// without a real browser. Every call is appended to the owning page's events.
type fakeElement struct {
	events   *[]string
	selector string
	inputErr error
	clickErr error
	attr     string
	attrErr  error
	text     string
	textErr  error
}

// Input reports inputErr after recording the supplied value.
func (e *fakeElement) Input(value string) error {
	*e.events = append(*e.events, "input:"+e.selector+"="+value)
	return e.inputErr
}

// Click reports clickErr after recording the click.
func (e *fakeElement) Click() error {
	*e.events = append(*e.events, "click:"+e.selector)
	return e.clickErr
}

// Attribute returns the scripted attr/attrErr pair after recording name.
func (e *fakeElement) Attribute(name string) (string, error) {
	*e.events = append(*e.events, "attribute:"+e.selector+"="+name)
	return e.attr, e.attrErr
}

// Text returns the scripted text/textErr pair after recording the lookup.
func (e *fakeElement) Text() (string, error) {
	*e.events = append(*e.events, "text:"+e.selector)
	return e.text, e.textErr
}

// fakePage is a scriptable page keyed by selector, used to drive the login
// logic without a real browser. events records every call in order.
type fakePage struct {
	events      []string
	navigateErr error
	waitLoadErr error
	elements    map[string]*fakeElement
	elementErr  map[string]error
}

// newFakePage returns an empty fakePage ready to be wired up with
// elements and/or errors by the caller.
func newFakePage() *fakePage {
	return &fakePage{elements: map[string]*fakeElement{}, elementErr: map[string]error{}}
}

// Navigate records the target and reports navigateErr.
func (p *fakePage) Navigate(target string) error {
	p.events = append(p.events, "navigate:"+target)
	return p.navigateErr
}

// WaitLoad records the call and reports waitLoadErr.
func (p *fakePage) WaitLoad() error {
	p.events = append(p.events, "wait-load")
	return p.waitLoadErr
}

// Element returns the scripted error or element registered for selector,
// or an error if neither was registered.
func (p *fakePage) Element(selector string) (element, error) {
	p.events = append(p.events, "element:"+selector)
	if err, ok := p.elementErr[selector]; ok {
		return nil, err
	}
	el, ok := p.elements[selector]
	if !ok {
		return nil, errors.New("fake page: no element registered for " + selector)
	}
	el.events = &p.events
	el.selector = selector
	return el, nil
}

// successPage returns a fakePage wired up so a full login flow succeeds
// against the default options.
func successPage() *fakePage {
	opts := newOptions()
	page := newFakePage()
	page.elements[opts.usernameSelector] = &fakeElement{}
	page.elements[opts.passwordSelector] = &fakeElement{}
	page.elements[opts.submitSelector] = &fakeElement{}
	page.elements[opts.resultSelector] = &fakeElement{attr: "flash success", text: "You logged in"}
	return page
}
