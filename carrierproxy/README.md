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
* **`DocumentDownload`** logs in again, then downloads a configured URL with Go's HTTP client using cookies from the browser session.

`Policies` and `DocumentDownload` need their own URL options and a prior successful `Login`.

### Behavior

* **Retries.** Transient failures are retried `WithRetries` more times (default 2), `WithRetryDelay` apart (default 1s). `ErrInvalidCredentials`, `ErrMalformedResponse` and `ErrNotConfigured` are never retried, since another attempt would fail the same way; retrying bad credentials could also trip a lockout. The policy lives in `Client.withRetries` (`browser/session.go`).
* **Fresh attempts.** Each attempt, retries included, uses a new browser page and a new login, so nothing from a failed attempt carries over.
* **Context-aware calls.** `LoginContext`, `PoliciesContext`, and `DocumentDownloadContext` accept caller cancellation. The original `PolicyProvider` methods remain available and use `context.Background()`. A canceled caller stops retries and interrupts retry delays. Each attempt starts its configured timeout before browser launch and shares it with browser operations and HTTP download/body reads. A successful document body keeps that attempt context until EOF, a read error, `Close`, or its deadline. The caller must always close the returned body.
* **Cleanup.** Each attempt releases its browser before returning its result. CDP shutdown uses an independent two-second context; a failure or an observed timeout triggers launcher kill before process-exit cleanup. Rod v0.116.2 performs the websocket write for a CDP command before observing the context deadline, so that context is not an absolute bound on the socket write. Rod also includes context-unaware launch URL resolution, a browser-download lock, process waiting, and filesystem cleanup, so `WithTimeout` is not an absolute wall-clock bound for those dependency operations. Profile removal is attempted by Rod and is not an observable guarantee of successful deletion.
* **Document URL builders.** The `WithDocumentURL` callback receives a `url.PathEscape`-encoded single path segment after the client rejects empty, dot, dot-dot, slash, and backslash keys. Append that segment directly to a trusted document path without decoding or encoding it again. The callback is application code: it chooses the target origin and can return an unrelated URL. The client does not guarantee that an arbitrary callback preserves an origin or path boundary.
* **Cookie scope.** Cookie extraction uses the document target URL. Transfers preserve original domain identities and encode host-only/domain scope only in the standard jar's internal keys, so same-name/path cookies remain distinct even on the exact same hostname. Original cookie names are restored before sending; there is no public name change. One standard jar handles normalized domains, path, Secure, expiry, and global ordering by path specificity then creation order; response `Set-Cookie` updates retain the matching scope. Chrome 153 did not distinguish host-only and domain cookies on `localhost`, so the approved fallback returns cookies only when the request hostname exactly matches the original document target hostname. Same-host redirects retain applicable cookies, but cross-subdomain redirects can lose session continuity because cookies are withheld from every other hostname.
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
    browser.WithDocumentURL(func(escapedDownloadKey string) string {
        return "https://your-target-site.example/documents/" + escapedDownloadKey
    }),
)
```

## Design

go-rod is isolated behind two small interfaces, `page` and `element`, in `browser/rod.go`. `rodPage`/`rodElement` adapt go-rod to them, and `launchPage` starts the browser. All the decision logic (what to click, how to read the result, which error to return) is written against those interfaces, so it is unit tested with fakes. `Client.newPage` and `Client.fetch` are function fields that tests replace.

`launchPage` uses a locally installed Chrome/Chromium when one exists and otherwise lets go-rod download one (cached under `~/.cache/rod`).

The shared lint configuration enforces the repository's complexity limits for all production and test functions: cyclomatic ≤ 6 and cognitive ≤ 10.

Run the shared lint check from the carrierproxy module directory:

```bash
golangci-lint run --config ../.golangci-complexity.yml ./...
```

## Usage

The demo binary logs into [the-internet.herokuapp.com/login](https://the-internet.herokuapp.com/login), a public practice site with published test credentials:

```bash
CARRIERPROXY_USERNAME=tomsmith CARRIERPROXY_PASSWORD='SuperSecretPassword!' go run ./cmd/carrierproxy
```

Both variables are required. On success it prints `login to <url> succeeded as <username>`; on failure it logs the error and exits non-zero.

From your own code, use these imports:

```go
import (
    "context"
    "io"
    "time"

    "github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy/browser"
)
```

The following snippet is the body of an error-returning consumer function; its caller supplies `username`, `password`, and `destination io.Writer`:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()
client := browser.NewClient("https://your-target-site.example/login",
    browser.WithDocumentURL(func(escapedDownloadKey string) string {
        return "https://your-target-site.example/documents/" + escapedDownloadKey
    }),
)
if err := client.LoginContext(ctx, username, password); err != nil {
    return err
}
body, err := client.DocumentDownloadContext(ctx, "policy-42.pdf")
if err != nil {
    return err
}
defer func() { _ = body.Close() }()
_, err = io.Copy(destination, body)
return err
```

## Testing

```bash
go test ./...
```

Needs no network. Tests that need a browser skip themselves when no local Chrome/Chromium is found. CI runs `go test ./... -race` and requires at least 90% coverage per module.

* **Unit tests** cover login, policies, documents, session and options behavior using fake pages, and `run()` using a fake `PolicyProvider`.
* **`browser/rod_test.go`** covers the go-rod adapter: launch-failure cleanup (no browser needed), and every `rodPage`/`rodElement` method against a real headless browser, including their timeout errors.
* **`cmd/carrierproxy/e2e_test.go`** runs the CLI's `run` through a real browser against a local login page.
* **`browser/integration_test.go`** runs only when `CARRIERPROXY_USERNAME` and `CARRIERPROXY_PASSWORD` are set (CI sets placeholders). It drives a real browser against a local fixture site (`browser/testdata/`) that accepts exactly those credentials:

  ```bash
  CARRIERPROXY_USERNAME=testuser CARRIERPROXY_PASSWORD=testpass go test ./... -run TestLoginIntegration
  ```

  It checks that valid credentials log in and invalid ones fail without retrying, that a missing selector times out with `context.DeadlineExceeded`, that an unreachable host fails, and that `Policies`/`DocumentDownload` return `ErrNotLoggedIn` before login and real data after it. It uses a local site rather than the public demo so the tests don't depend on a third-party site staying up.
