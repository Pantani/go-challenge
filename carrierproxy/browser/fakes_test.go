package browser

import "errors"

const testLoginURL = "https://example.com/login"

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

func (e *fakeElement) Input(value string) error {
	*e.events = append(*e.events, "input:"+e.selector+"="+value)
	return e.inputErr
}

func (e *fakeElement) Click() error {
	*e.events = append(*e.events, "click:"+e.selector)
	return e.clickErr
}

func (e *fakeElement) Attribute(name string) (string, error) {
	*e.events = append(*e.events, "attribute:"+e.selector+"="+name)
	return e.attr, e.attrErr
}

func (e *fakeElement) Text() (string, error) {
	*e.events = append(*e.events, "text:"+e.selector)
	return e.text, e.textErr
}

type fakePage struct {
	events      []string
	navigateErr error
	waitLoadErr error
	elements    map[string]*fakeElement
	elementErr  map[string]error
}

func newFakePage() *fakePage {
	return &fakePage{elements: map[string]*fakeElement{}, elementErr: map[string]error{}}
}

func (p *fakePage) Navigate(target string) error {
	p.events = append(p.events, "navigate:"+target)
	return p.navigateErr
}

func (p *fakePage) WaitLoad() error {
	p.events = append(p.events, "wait-load")
	return p.waitLoadErr
}

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

func successPage() *fakePage {
	opts := newOptions()
	page := newFakePage()
	page.elements[opts.usernameSelector] = &fakeElement{}
	page.elements[opts.passwordSelector] = &fakeElement{}
	page.elements[opts.submitSelector] = &fakeElement{}
	page.elements[opts.resultSelector] = &fakeElement{attr: "flash success", text: "You logged in"}
	return page
}
