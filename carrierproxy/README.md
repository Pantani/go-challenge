# Carrierproxy Scraper Challenge

This challenge is to create a partial implementation of the `PolicyProvider` interface. Normally this interface would be used to scrape policies from a carrier website but in this case any website with a login form can be used as the target.

**Instructions:**

* [x] Implement the `PolicyProvider`'s `Login` method against a website of your choice using the [go-rod](https://github.com/go-rod/rod) scraping library.
* [x] Create test(s) with full coverage of your code that accept environment variables for the credentials (login & password).
* [x] Document your code and usage instructions.

## Layout

```
carrierproxy/
├── cmd/carrierproxy/main.go   # demo binary (package main)
├── carrierproxy.go            # PolicyProvider interface (the contract)
├── policy.go                  # Policy struct
├── errors.go                  # sentinel errors any implementation can return
└── browser/                   # the go-rod-backed implementation
    ├── client.go                # Client (implements PolicyProvider) + NewClient
    ├── login.go                  # Login
    ├── policies.go                 # Policies
    ├── documents.go                  # DocumentDownload
    ├── session.go                      # shared by all three: re-auth, retry, credential storage
    ├── options.go                       # functional options (With*) + their defaults
    ├── rod.go                            # the only file that talks to go-rod
    ├── http.go                            # the only file that talks to net/http
    ├── testdata/                          # fixture site used by TestLoginIntegration
    │   ├── login.html
    │   ├── policies.html
    │   └── policy-42.txt
    └── *_test.go
```

The root `carrierproxy` package is just the contract — the interface, the shared `Policy` type, and sentinel errors any implementation can return — with no implementation code of its own. `browser` is one concrete implementation of that contract. `cmd/carrierproxy` holds only the demo entrypoint.

`browser` isn't named `rod` because that's the import name of the `go-rod/rod` dependency itself — a same-named subpackage would compile fine here (no file imports both at once) but would be confusing to read.

## What's implemented

All three `PolicyProvider` methods are implemented against a real site using [go-rod](https://github.com/go-rod/rod):

* **`Login`** ([`browser/login.go`](browser/login.go)) drives a headless browser through the target's login form and classifies the result.
* **`Policies`** ([`browser/policies.go`](browser/policies.go)) re-authenticates, navigates to a configured page, and reads each matched row's first two cells as `CarrierID`/`PolicyNumber`.
* **`DocumentDownload`** ([`browser/documents.go`](browser/documents.go)) re-authenticates, then does a plain HTTP GET for a configured URL, attaching the browser session's cookies so the download is authenticated too.

`Policies` and `DocumentDownload` need their own configuration (there's no way to guess a "policies page" or "document URL" shape for an arbitrary site) and require a prior successful `Login` — see [Reusable, not hardcoded to one site](#reusable-not-hardcoded-to-one-site) below. `browser.Client` satisfies the full `PolicyProvider` interface, checked at compile time.

## Idempotent and resilient

* **Retries.** All three methods retry non-credential failures (a slow-to-render form, a dropped connection, a browser that's slow to start) up to `WithRetries` additional times (default 2, so 3 attempts total), waiting `WithRetryDelay` between them (default 1s). `carrierproxy.ErrInvalidCredentials` is **never** retried — the same credentials would just fail again, and retrying them against a real site risks tripping a lockout. This lives in one place, [`Client.withRetries`](browser/session.go), that every method's single-attempt logic is written against, so the policy only needs testing once (`browser/session_test.go`) — see [`TestClientWithRetries`](browser/session_test.go).
* **Stateless attempts.** Every attempt — including every retry — starts from a fresh browser page and a fresh login; nothing from a failed attempt is reused. That's what makes retrying safe: there's no partially-broken session to carry forward, and calling `Login`, `Policies` or `DocumentDownload` again after a failure behaves exactly like calling it the first time.
* **Guaranteed cleanup.** The page opened by an attempt is always released before that attempt returns (`defer`/explicit close on every path, including panics unwinding through a `defer`), whether it succeeds, fails, or is abandoned partway through — so a failed attempt never leaks a browser process into the next one.
* **Safe for concurrent and repeated use.** A `*Client` has no per-call mutable state except the remembered credentials, which are guarded by a mutex (`browser/session.go`); `TestClientLoginConcurrentSafety` (`browser/login_test.go`) drives 20 concurrent `Login` calls on one `Client` under `-race`. Calling `Login` again later (e.g. to refresh an expired session) simply replaces the remembered credentials — nothing needs to be reset by hand.
* **Classifiable errors.** Errors are wrapped (`%w`), not stringified, so a caller can tell failure modes apart: `errors.Is(err, carrierproxy.ErrInvalidCredentials)` (bad login), `errors.Is(err, carrierproxy.ErrNotLoggedIn)` (`Policies`/`DocumentDownload` called before a successful `Login`), `errors.Is(err, carrierproxy.ErrNotConfigured)` (missing `WithPoliciesURL`/`WithDocumentURL`), or `errors.Is(err, context.DeadlineExceeded)` (a selector never appeared within `WithTimeout` — go-rod surfaces this directly, see `TestLoginIntegration/a_wrong_selector_times_out_with_a_classifiable_error`).

## Reusable, not hardcoded to one site

`browser.NewClient` takes the login URL as a required argument — there is no built-in default site baked into the library. The demo site is a choice made by `cmd/carrierproxy` (the example program), not by the `browser` package itself:

```go
client := browser.NewClient("https://your-target-site.example/login")
```

Everything about *how* that site's form is found, and how `Policies`/`DocumentDownload` reach their own pages, is overridable through functional options (`browser/options.go`), each with a sensible default where one makes sense:

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

    // Policies and DocumentDownload have no site-agnostic default and are
    // disabled (return carrierproxy.ErrNotConfigured) until set:
    browser.WithPoliciesURL("https://your-target-site.example/policies"),
    browser.WithPolicyRowSelector(".policy-row"), // default: "tr"
    browser.WithPolicyCellSelector(".cell"),      // default: "td"
    browser.WithDocumentURL(func(downloadKey string) string {
        return "https://your-target-site.example/documents/" + downloadKey
    }),
)
```

Only pass the options a given site actually needs; everything else keeps its default. This is the same pattern used by [`ignite/cli`'s `xast` package](https://github.com/ignite/cli/blob/main/ignite/pkg/xast/global.go): an unexported `options` struct with defaults from `newOptions()`, an exported `Option func(*options)` type, and one `WithX` constructor per configurable field, applied in a loop inside `NewClient`.

## Design

The go-rod-specific code is isolated behind two small interfaces in [`browser/rod.go`](browser/rod.go):

```go
type page interface {
    Navigate(url string) error
    WaitLoad() error
    Element(selector string) (element, error)
    Elements(selector string) ([]element, error)
    Cookies() ([]cookie, error)
}

type element interface {
    Input(value string) error
    Click() error
    Attribute(name string) (string, error)
    Text() (string, error)
    Elements(selector string) ([]element, error)
}
```

`rodPage`/`rodElement` adapt `*rod.Page`/`*rod.Element` to these interfaces, and `launchPage` is the only function that launches a browser and connects go-rod. Everything else — `Login` (`login.go`), `Policies` (`policies.go`), `DocumentDownload` (`documents.go`), and the re-auth/retry machinery they share (`session.go`) — is written against `page`/`element`, so the whole decision-making flow (what to click, how to read a result, which rows to keep, which error to return) is testable without a real browser. `Client.newPage` is a function field defaulting to `launchPage`, and `Client.fetch` (the plain-HTTP download, `http.go`) defaults to `fetchWithCookies`; tests swap both for fakes.

Every function is kept small and single-purpose on purpose: the deepest orchestration sits at cyclomatic complexity 5 and cognitive complexity 4 (package average 2.3/3.8), checked with [`gocyclo`](https://github.com/fzipp/gocyclo) and [`gocognit`](https://github.com/uudashr/gocognit):

```bash
go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -avg -ignore '_test.go' .
go run github.com/uudashr/gocognit/cmd/gocognit@latest -avg -ignore '_test.go' .
```

## Usage

Run the demo binary, which calls `Login` against [the-internet.herokuapp.com/login](https://the-internet.herokuapp.com/login) — a public site built for browser-automation practice, published with its own test credentials — with credentials from the environment:

```bash
CARRIERPROXY_USERNAME=tomsmith CARRIERPROXY_PASSWORD='SuperSecretPassword!' go run ./cmd/carrierproxy
```

(The test suite below targets a local fixture instead, not this site — see [Testing](#testing).)

Or use `browser.Client` from your own code, against any site with a login form:

```go
import "github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy/browser"

client := browser.NewClient("https://your-target-site.example/login" /* , With* options as needed */)
if err := client.Login(username, password); err != nil {
    // errors.Is(err, carrierproxy.ErrInvalidCredentials) means the site rejected the
    // credentials; anything else is a browser/navigation/element failure.
}

policies, err := client.Policies()                 // needs WithPoliciesURL
doc, err := client.DocumentDownload("policy-42.pdf") // needs WithDocumentURL
```

Chrome/Chromium isn't required to be pre-installed: go-rod downloads a matching revision on first run if it can't find one (cached under `~/.cache/rod`).

## Testing

```bash
go test ./...
```

This runs every test except the real-browser integration test, which is skipped unless credentials are supplied (see below). No network access is needed, and no browser is required either: the tests that do need one (see below) skip themselves when `launcher.LookPath()` finds no local Chrome/Chromium, so the default run never downloads a browser.

**Coverage** (`go test ./... -coverprofile=cover.out && go tool cover -func=cover.out`): every function in `browser/login.go`, `browser/policies.go`, `browser/documents.go`, `browser/session.go`, `browser/options.go`, `browser/client.go` and `browser/http.go` (the actual challenge logic) is at **100%**, and the module is at ~97% overall when a local Chrome is present. CI enforces a 90% minimum per module (`.github/workflows/ci.yml`). The go-rod adapter and the CLI are covered by:

* `browser/rod_test.go` — `TestLaunchPageWithFailedLaunch` points `launchPageWith` at a missing binary and at one that exits without a DevTools URL, covering both launch-failure cleanup paths with no browser at all. `TestRodAdapter` and `TestRodAdapterErrors` drive every `rodPage`/`rodElement` method against a real headless Chrome and a local page, including the error each one returns once the page's timeout expires.
* `cmd/carrierproxy/main_test.go` — `run` takes the login URL and a `getenv` func, so `TestRunEndToEnd` runs the CLI's code path end to end through a real browser against a local login page, and `TestRunMissingCredentials` covers the blank-credentials path without one. Only `main()` itself stays uncovered.

To fulfil "test(s) ... that accept environment variables for the credentials", set `CARRIERPROXY_USERNAME`/`CARRIERPROXY_PASSWORD` to also run the integration test (CI sets placeholder values for this):

```bash
CARRIERPROXY_USERNAME=testuser CARRIERPROXY_PASSWORD=testpass go test ./... -v -run TestLoginIntegration
```

This still launches a real, headless Chrome via go-rod (downloaded automatically on first run if none is found — see [Usage](#usage)) — but instead of a real external site, it drives that browser against **a local fixture site** (`browser/testdata/`, served in-process over `httptest.NewServer` by `browser/fixturesite_test.go`): a login form, a session-cookie-gated policies table, and a session-cookie-gated document download endpoint. This is deliberately *not* the same public site `cmd/carrierproxy` targets — a test suite that depends on a third-party site's uptime, markup staying stable, and (for `/download`) a shared, other-people's-test-runs-included file listing is exactly the kind of flakiness worth avoiding. The fixture site is configured to accept whichever username/password the test was given, so it's still genuinely exercised end to end through the environment variables, just without leaving the machine:

* Valid credentials succeed; invalid ones are rejected (and, with `WithRetries(2)` set, confirmed *not* retried).
* A selector that never appears times out and the error unwraps to `context.DeadlineExceeded`.
* An unreachable host (`.invalid`, [reserved by RFC 2606](https://www.rfc-editor.org/rfc/rfc2606) for exactly this) fails promptly rather than hanging.
* `Policies` and `DocumentDownload` each return `carrierproxy.ErrNotLoggedIn` before `Login`, then succeed for real after it — proving the auth-gating itself, which a public, ungated practice page couldn't have.

(The default credentials above are placeholders the fixture site is told to expect for that one run — not real secrets, and not tied to any specific value; any username/password pair works as long as both sides of the test are told the same one, which is exactly what happens here.)
