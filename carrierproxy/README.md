# Carrierproxy Scraper Challenge

This module implements the `Login` portion of `carrierproxy.PolicyProvider`
with [go-rod](https://github.com/go-rod/rod). It drives a plain HTML login
form, fills configurable selectors, submits the form, and reads a result
banner whose CSS class identifies success.

`Policies` and `DocumentDownload` remain on the original interface but return
`carrierproxy.ErrNotImplemented`: this is deliberately a partial
implementation, matching the challenge requirement.

## Usage

```go
client := browser.NewClient("https://carrier.example/login")
err := client.Login("username", "password")
```

Use `WithUsernameSelector`, `WithPasswordSelector`, `WithSubmitSelector`,
`WithResultSelector`, `WithSuccessClass`, and `WithTimeout` for sites that do
not use the defaults. Invalid credentials return an error wrapping
`carrierproxy.ErrInvalidCredentials`.

## Tests

```sh
cd carrierproxy
go test -race ./...
```

The real-browser fixture test reads `CARRIERPROXY_USERNAME` and
`CARRIERPROXY_PASSWORD`; it runs only when both variables and a local
Chrome/Chromium installation are available:

```sh
CARRIERPROXY_USERNAME=ci-user CARRIERPROXY_PASSWORD=ci-pass go test -race ./...
```
