# Carrierproxy Scraper Challenge

This challenge is to create a partial implementation of the `PolicyProvider` interface. Normally this interface would be used to scrape policies from a carrier website but in this case any website with a login form can be used as the target.

**Instructions:**

* [x] Implement the `PolicyProvider`'s `Login` method against a website of your choice using the [go-rod](https://github.com/go-rod/rod) scraping library.
* [ ] Create test(s) with full coverage of your code that accept environment variables for the credentials (login & password). 
* [x] Document your code and usage instructions.

## Implementation notes

This module implements the `Login` method of `carrierproxy.PolicyProvider`
with [go-rod](https://github.com/go-rod/rod). It drives a headless browser
through a plain HTML login form: fills the username and password fields,
submits, and reads a result banner whose CSS class tells success from
rejection.

`Policies` and `DocumentDownload` stay on the interface but return
`carrierproxy.ErrNotImplemented`; the challenge asks for a partial
implementation.

## Usage

The default selectors match
[the-internet.herokuapp.com/login](https://the-internet.herokuapp.com/login),
a public practice site with published test credentials:

```sh
cd carrierproxy
CARRIERPROXY_USERNAME=tomsmith CARRIERPROXY_PASSWORD='SuperSecretPassword!' go run ./cmd/carrierproxy
```

Set `CARRIERPROXY_LOGIN_URL` to target another site. From Go:

```go
client := browser.NewClient("https://carrier.example/login",
    browser.WithUsernameSelector("#email"), // default "#username"
    browser.WithSuccessClass("alert-ok"),   // default "success"
)
err := client.Login(username, password)
// errors.Is(err, carrierproxy.ErrInvalidCredentials) when the site rejects them.
```

Other options: `WithPasswordSelector`, `WithSubmitSelector`,
`WithResultSelector` and `WithTimeout`. Every error is wrapped, so callers
compare with `errors.Is`.

## Design

`browser/rod.go` is the only file that touches go-rod. It adapts a rod page
to two small interfaces, `page` and `element`, and the login flow in
`browser/login.go` is written against those, so it is unit-tested with fakes
and no browser.

## Tests

```sh
cd carrierproxy
go test -race ./...
```

Unit tests need no browser. The real-browser test reads the credentials from
`CARRIERPROXY_USERNAME` and `CARRIERPROXY_PASSWORD` and skips when they are
unset or no local Chrome/Chromium is found. By default it logs into a local
fixture site (`browser/testdata/login.html`) that accepts exactly those
credentials, so CI needs no external account:

```sh
CARRIERPROXY_USERNAME=ci-user CARRIERPROXY_PASSWORD=ci-pass go test -race ./...
```

With `CARRIERPROXY_LOGIN_URL` set it runs against that site instead:

```sh
CARRIERPROXY_LOGIN_URL=https://the-internet.herokuapp.com/login \
CARRIERPROXY_USERNAME=tomsmith CARRIERPROXY_PASSWORD='SuperSecretPassword!' \
go test -race -run TestLoginIntegration ./browser/
```

## Validation status

The unit tests and real-browser integration test pass against the local fixture
with credentials supplied through environment variables. `go vet ./...` and
`go build ./...` also pass.

Measured statement coverage with the integration test enabled is 88.0% overall:
89.9% for `browser` and 71.4% for `cmd/carrierproxy`. The full-coverage requirement
remains incomplete. No external website was exercised in this validation.
