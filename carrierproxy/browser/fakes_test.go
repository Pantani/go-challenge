package browser

import "errors"

const testLoginURL = "https://example.com/login"

// fakeElement is a scriptable element used to drive the login/scraping
// logic without a real browser.
type fakeElement struct {
	inputErr error
	clickErr error
	attr     string
	attrErr  error
	text     string
	textErr  error

	subElements    map[string][]*fakeElement
	subElementsErr map[string]error
}

// Input reports inputErr, recording nothing else about the call.
func (e *fakeElement) Input(string) error { return e.inputErr }

// Click reports clickErr, recording nothing else about the call.
func (e *fakeElement) Click() error { return e.clickErr }

// Attribute returns the scripted attr/attrErr pair, ignoring name.
func (e *fakeElement) Attribute(string) (string, error) { return e.attr, e.attrErr }

// Text returns the scripted text/textErr pair.
func (e *fakeElement) Text() (string, error) { return e.text, e.textErr }

// Elements returns the scripted sub-elements or error registered for
// selector, used by Policies to read a row's cells.
func (e *fakeElement) Elements(selector string) ([]element, error) {
	if err, ok := e.subElementsErr[selector]; ok {
		return nil, err
	}
	return wrapFakeElements(e.subElements[selector]), nil
}

// fakePage is a scriptable page keyed by selector, used to drive the
// login/scraping logic without a real browser.
type fakePage struct {
	navigateErr      error
	navigateErrByURL map[string]error
	waitLoadErr      error
	waitLoadErrs     []error // per-call overrides, in order, consumed before waitLoadErr
	waitLoadCalls    int
	elements         map[string]*fakeElement
	elementErr       map[string]error

	elementLists    map[string][]*fakeElement
	elementListsErr map[string]error

	cookies    []cookie
	cookiesErr error
}

// newFakePage returns an empty fakePage ready to be wired up with
// elements and/or errors by the caller.
func newFakePage() *fakePage {
	return &fakePage{
		elements:         map[string]*fakeElement{},
		elementErr:       map[string]error{},
		elementLists:     map[string][]*fakeElement{},
		elementListsErr:  map[string]error{},
		navigateErrByURL: map[string]error{},
	}
}

// Navigate reports the error registered for url in navigateErrByURL, or
// navigateErr if url has no specific entry.
func (p *fakePage) Navigate(url string) error {
	if err, ok := p.navigateErrByURL[url]; ok {
		return err
	}
	return p.navigateErr
}

// WaitLoad reports the next entry in waitLoadErrs if any remain
// (allowing, e.g., a first call to succeed and a second to fail), or
// waitLoadErr once they run out.
func (p *fakePage) WaitLoad() error {
	if p.waitLoadCalls < len(p.waitLoadErrs) {
		err := p.waitLoadErrs[p.waitLoadCalls]
		p.waitLoadCalls++
		return err
	}
	p.waitLoadCalls++
	return p.waitLoadErr
}

// Element returns the scripted error or element registered for selector,
// or an error if neither was registered.
func (p *fakePage) Element(selector string) (element, error) {
	if err, ok := p.elementErr[selector]; ok {
		return nil, err
	}
	el, ok := p.elements[selector]
	if !ok {
		return nil, errors.New("fakePage: no element registered for " + selector)
	}
	return el, nil
}

// Elements returns the scripted list of elements or error registered for
// selector, used by Policies to list rows.
func (p *fakePage) Elements(selector string) ([]element, error) {
	if err, ok := p.elementListsErr[selector]; ok {
		return nil, err
	}
	return wrapFakeElements(p.elementLists[selector]), nil
}

// Cookies returns the scripted cookies/cookiesErr pair.
func (p *fakePage) Cookies() ([]cookie, error) { return p.cookies, p.cookiesErr }

// wrapFakeElements adapts a slice of *fakeElement to []element.
func wrapFakeElements(els []*fakeElement) []element {
	wrapped := make([]element, len(els))
	for i, el := range els {
		wrapped[i] = el
	}
	return wrapped
}

// successPage returns a fakePage wired up so a full login flow succeeds
// against the default options.
func successPage() *fakePage {
	o := newOptions()
	p := newFakePage()
	p.elements[o.usernameSelector] = &fakeElement{}
	p.elements[o.passwordSelector] = &fakeElement{}
	p.elements[o.submitSelector] = &fakeElement{}
	p.elements[o.resultSelector] = &fakeElement{attr: "flash success", text: "You logged into a secure area!"}
	return p
}

// fakeRow builds a fakeElement standing in for one table row, whose cells
// (matched by cellSelector) hold cellTexts in order.
func fakeRow(cellSelector string, cellTexts ...string) *fakeElement {
	cells := make([]*fakeElement, len(cellTexts))
	for i, text := range cellTexts {
		cells[i] = &fakeElement{text: text}
	}
	return &fakeElement{subElements: map[string][]*fakeElement{cellSelector: cells}}
}
