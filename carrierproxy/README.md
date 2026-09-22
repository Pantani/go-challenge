# Carrierproxy Scraper Challenge

This challenge is to create a partial implementation of the `PolicyProvider` interface. Normally this interface would be used to scrape policies from a carrier website but in this case any website with a login form can be used as the target.

**Instructions:**

* [x] Implement the `PolicyProvider`'s `Login` method against a website of your choice using the [go-rod](https://github.com/go-rod/rod) scraping library.
* [x] Create test(s) with full coverage of your code that accept environment variables for the credentials (login & password).
* [x] Document your code and usage instructions.

## Layout

```
carrierproxy/
├── carrierproxy.go        # PolicyProvider interface (the contract)
├── policy.go              # Policy struct
├── errors.go              # sentinel errors shared by implementations
├── cmd/carrierproxy/      # demo binary: main() wires the real client into a testable run()
└── browser/               # go-rod implementation of PolicyProvider
    ├── client.go          # Client + NewClient
    ├── login.go           # Login
    ├── policies.go        # Policies
    ├── documents.go       # DocumentDownload
    ├── session.go         # re-auth, retries, credential storage (shared by all three)
    ├── options.go         # With* options and their defaults
    ├── rod.go             # the only file that talks to go-rod
    ├── http.go            # the only file that talks to net/http
    └── testdata/          # local fixture site for the integration test
```

## What's implemented

All three `PolicyProvider` methods, using go-rod:

* **`Login`** fills in the login form (clearing anything pre-filled first), submits it and classifies the result.
* **`Policies`** logs in again, opens a configured page and reads each row's first two cells as `CarrierID`/`PolicyNumber`.
* **`DocumentDownload`** logs in again, then downloads a configured URL over plain HTTP using the browser session's cookies.

`Policies` and `DocumentDownload` need their own URL options and a prior successful `Login`.

### Behavior

* **Retries.** Transient failures are retried `WithRetries` more times (default 2), `WithRetryDelay` apart (default 1s). `ErrInvalidCredentials`, `ErrMalformedResponse` and `ErrNotConfigured` are never retried, since another attempt would fail the same way; retrying bad credentials could also trip a lockout. The policy lives in `Client.withRetries` (`browser/session.go`).
* **Fresh attempts.** Each attempt, retries included, uses a new browser page and a new login, so nothing from a failed attempt carries over.
* **Cleanup.** Every attempt releases its browser before returning: it asks the browser to exit, kills it if that fails, and removes the temporary profile directory.
* **Concurrency.** A `Client` is safe for concurrent use; the only shared state is the remembered credentials, guarded by a mutex. `TestClientLoginConcurrentSafety` runs 20 concurrent logins under `-race`.
* **Errors** are wrapped with `%w`, so callers can use `errors.Is` with `carrierproxy.ErrInvalidCredentials`, `ErrNotLoggedIn`, `ErrNotConfigured`, or `context.DeadlineExceeded` (a selector didn't appear within `WithTimeout`).

## Configuration

`browser.NewClient` takes the login URL; nothing is hardcoded to one site. Selectors and timings have defaults matching the common `#username`/`#password` form, and each can be overridden:

```go
client := browser.NewClient(
    "https://your-target-site.example/login",
    browser.WithUsernameSelector("#email"),      // default: "#username"
    browser.WithPasswordSelector("#pwd"),         // default: "#password"
    browser.WithSubmitSelector(".login-button"),  // default: "button[type='submit']"
    browser.WithResultSelector("#login-banner"),  // default: "#flash"
    browser.WithSuccessClass("alert-success"),    // default: "success"
    browser.WithTimeout(45*time.Second),          // default: 30s
    browser.WithRetries(3),                       // default: 2
    browser.WithRetryDelay(2*time.Second),        // default: 1s

    // No default; Policies/DocumentDownload return ErrNotConfigured until set:
    browser.WithPoliciesURL("https://your-target-site.example/policies"),
    browser.WithPolicyRowSelector(".policy-row"), // default: "tr"
    browser.WithPolicyCellSelector(".cell"),      // default: "td"
    browser.WithDocumentURL(func(downloadKey string) string {
        return "https://your-target-site.example/documents/" + downloadKey
    }),
)
```

## Design

go-rod is isolated behind two small interfaces, `page` and `element`, in `browser/rod.go`. `rodPage`/`rodElement` adapt go-rod to them, and `launchPage` starts the browser. All the decision logic (what to click, how to read the result, which error to return) is written against those interfaces, so it is unit tested with fakes. `Client.newPage` and `Client.fetch` are function fields that tests replace.

`launchPage` uses a locally installed Chrome/Chromium when one exists and otherwise lets go-rod download one (cached under `~/.cache/rod`).

Functions stay within the repo's complexity limits (cyclomatic ≤ 6, cognitive ≤ 8), enforced in CI:

```bash
golangci-lint run --config ../.golangci-complexity.yml ./...
```

## Usage

The demo binary logs into [the-internet.herokuapp.com/login](https://the-internet.herokuapp.com/login), a public practice site with published test credentials:

```bash
CARRIERPROXY_USERNAME=tomsmith CARRIERPROXY_PASSWORD='SuperSecretPassword!' go run ./cmd/carrierproxy
```

Both variables are required. On success it prints `login to <url> succeeded as <username>`; on failure it logs the error and exits non-zero.

From your own code:

```go
client := browser.NewClient("https://your-target-site.example/login" /* , options */)
if err := client.Login(username, password); err != nil {
    // errors.Is(err, carrierproxy.ErrInvalidCredentials): the site rejected them.
}
policies, err := client.Policies()                   // needs WithPoliciesURL
doc, err := client.DocumentDownload("policy-42.pdf") // needs WithDocumentURL
```

## Testing

```bash
go test ./...
```

Needs no network. Tests that need a browser skip themselves when no local Chrome/Chromium is found. CI runs `go test ./... -race` and requires at least 90% coverage per module.

* **Unit tests** cover the login, policies, documents, session and options logic at 100% using fake pages, and `run()` using a fake `PolicyProvider`.
* **`browser/rod_test.go`** covers the go-rod adapter: launch-failure cleanup (no browser needed), and every `rodPage`/`rodElement` method against a real headless browser, including their timeout errors.
* **`cmd/carrierproxy/e2e_test.go`** runs the CLI's `run` through a real browser against a local login page.
* **`browser/integration_test.go`** runs only when `CARRIERPROXY_USERNAME` and `CARRIERPROXY_PASSWORD` are set (CI sets placeholders). It drives a real browser against a local fixture site (`browser/testdata/`) that accepts exactly those credentials:

  ```bash
  CARRIERPROXY_USERNAME=testuser CARRIERPROXY_PASSWORD=testpass go test ./... -run TestLoginIntegration
  ```

  It checks that valid credentials log in and invalid ones fail without retrying, that a missing selector times out with `context.DeadlineExceeded`, that an unreachable host fails, and that `Policies`/`DocumentDownload` return `ErrNotLoggedIn` before login and real data after it. It uses a local site rather than the public demo so the tests don't depend on a third-party site staying up.

Only `main()` itself is left uncovered.
