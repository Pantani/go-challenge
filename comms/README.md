# Comms API Challenge

The challenge is to solve an emerging business problem within an existing codebase.

**Context:**

The existing codebase represents a restful API for various comms operations. The API utilizes an email service inside the route handlers to send emails. In the `main.go` server application the email service uses a sendgrid implementation, but the handlers also have unit tests which utilize a mock implementation to ensure full coverage of the handler email functionality.

**Business Problem:**

The comms api needs to support a new operation for adding policy coverage. The new `add-policy-coverage` route should have the same data contract as the others, with the exception that the payload for the new route also needs to support `CC` in addition to `To` for the email recipients. Until this point only `To` recipients have been needed, so the current email service provider does not yet support `CC`.

**Instructions:** 

* [x] Create a handler for the new comms operation and implement the correct business logic inside of the handler.
* [x] Make whatever improvements are needed to support the new email `CC` field.
* [x] Create a unit test for the new handler to prove it meets business requirements.

## Layout

The existing package layout is unchanged — this challenge is additive, not a restructuring:

```
comms/
├── main.go                   # config from env, newMux (route table), newServer (timeouts), run/main
├── main_test.go               # integration tests: real HTTP requests against newMux; config + server tests
├── handlers/                 # one http.HandlerFunc per comms operation
│   ├── handlers.go             # AddPolicyVehicle/Driver/Address/Coverage + the shared sendHandler pipeline
│   ├── models.go                # one request struct per operation
│   └── handlers_test.go          # one table-driven test func per handler, sharing a common case set
└── email/                    # the MailProvider contract and its implementations
    ├── email.go                 # MailProvider interface, TplID, template constants
    ├── sendgrid/                 # real implementation, backed by the vendored mail stand-in
    └── mockemail/                 # goroutine-safe in-memory implementation used by handler tests
```

## What's implemented

`AddPolicyCoverage` (see [`handlers/handlers.go`](handlers/handlers.go)) is a new handler registered at `POST /api/comms/add-policy-coverage`. It runs through the same pipeline as the three existing handlers — method check, bounded JSON decode, validation, send, translate errors to HTTP status codes — with one difference: its request struct, `AddPolicyCoverageReq` (see [`handlers/models.go`](handlers/models.go)), adds an `email_cc` field alongside the shared `email_to`/`message` contract:

```go
type AddPolicyCoverageReq struct {
	EmailTo string          `json:"email_to"`
	EmailCC []string        `json:"email_cc"`
	Message json.RawMessage `json:"message"`
}
```

### Request contract

All four routes accept `POST` with a JSON body and respond with an empty `200 OK` once the email has been handed to the provider. Requests are rejected before anything reaches the provider when:

| Condition | Status | Body |
|---|---|---|
| Method is not `POST` (`Allow: POST` header is set) | `405` | `method not allowed` |
| Body is not valid JSON, or a field has the wrong type | `400` | `invalid payload` |
| Body contains anything after the first JSON value | `400` | `invalid payload: unexpected data after JSON value` |
| Body exceeds 1 MiB | `413` | `request body too large` |
| `email_to` is missing or blank | `400` | `email_to: is required` |
| `email_to` is not an email address | `400` | `email_to: is not a valid email address` |
| `email_to` carries a display name (`Foo <foo@bar.com>`) | `400` | `email_to: must be a bare email address` |
| an `email_cc` entry fails any of the above (`add-policy-coverage` only) | `400` | `email_cc[<index>]: <same reason>` |
| `message` is missing or `null` | `400` | `message: is required` |
| the email provider fails | `500` | `error sending email` (the provider's error is logged server-side, never echoed) |

Addresses are trimmed of surrounding whitespace before validation and delivery. `email_cc` may be omitted, `null` or `[]`.

### CC normalisation

Before delivery the CC list is normalised so the provider is never asked to copy the same mailbox twice: entries are de-duplicated case-insensitively (first occurrence wins, order preserved), and any entry equal to `email_to` is dropped. If nothing remains, the email is sent with no CC at all. This is a deliberate business rule — sendgrid rejects personalizations that repeat an address across `to`/`cc` — and it is covered by `TestAddPolicyCoverage`.

## Design

**`MailProvider` gained a method, not a changed signature.** The existing `Send(to []string, message json.RawMessage, tpl TplID) error` is untouched. `SendWithCC(to, cc []string, message json.RawMessage, tpl TplID) error` is the new, additive method (see [`email/email.go`](email/email.go)); the handler pipeline calls it only when CC recipients remain after normalisation, and `Send` otherwise. Both `sendgrid.Client` and `mockemail.Client` implement `Send` and `SendWithCC` against one shared private helper (`send`/`record`), so the CC-handling logic exists in exactly one place per implementation.

**The four handlers share one generic pipeline.** `sendHandler[T]` (see [`handlers/handlers.go`](handlers/handlers.go)) owns the method check, `http.MaxBytesReader`-bounded decode with trailing-data rejection, field validation, CC normalisation, delivery and error translation. Each exported handler is a three-line adapter that maps its own request struct onto a provider-agnostic `envelope{to, cc, message}`. Adding a fifth operation means adding a request struct and one adapter; the contract table above applies to it automatically.

**The sendgrid wrapper is unit-testable, including its error path, without touching the vendored stand-in.** `sendgrid.Client` holds an unexported `mailClient` interface — the one method (`Send(*mail.V3Mail) error`) `Client` actually depends on (see [`email/sendgrid/sendgrid.go`](email/sendgrid/sendgrid.go)). `*mail.Client` satisfies that interface automatically, so `mail_v3.go` (marked "please do not modify") needed no changes. [`email/sendgrid/sendgrid_test.go`](email/sendgrid/sendgrid_test.go) substitutes a fake that captures the generated `*mail.V3Mail` and can return an error the real stand-in never does; since that type also has no getters, the test reads its unexported fields the same way `%+v` does — `fmt`'s own struct formatting — rather than adding `unsafe` or touching the vendored type. `Client` refuses to build a message with no `To` recipient (`sendgrid.ErrNoRecipients`) and wraps provider failures with context while keeping them reachable through `errors.Is`.

**`mockemail` is goroutine-safe and hands out snapshots.** `Client` guards its log with a mutex; `SendLogs()` returns a copy, and `ExtractCC`/`ExtractMessage` return copies too, so tests (including parallel ones) cannot alias the client's state or each other's inputs. `SendLog` gained a `cc` field and `ExtractCC()` getter (see [`email/mockemail/sendlog.go`](email/mockemail/sendlog.go)) following the existing `Extract*` accessor pattern, and `SendLogs.Last()` returns `nil` on an empty log instead of panicking.

**`main.go` is configurable and defensive.** `loadConfig` reads the environment (see Usage) and falls back to the built-in defaults; `newMux(emailsvc)` builds the route table on a fresh `http.ServeMux` rather than the global default; `newServer` wraps it in an `http.Server` with read-header, read, write and idle timeouts. [`main_test.go`](main_test.go) drives `newMux` through `httptest.NewServer` with real HTTP requests, table-tests `loadConfig`, checks `newServer`'s timeouts, and covers `run` by handing it a port the test already holds so `ListenAndServe` fails immediately.

Every function in the module is kept small and single-purpose, within the repo's CI limits of cyclomatic complexity ≤ 6 and cognitive complexity ≤ 8 (`golangci-lint run --config ../.golangci-complexity.yml ./...`). The current maxima are cyclomatic 5 (`normalizeCC`, `envelope.validate`) and cognitive 6 (`sendHandler`):

```bash
go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -top 5 -ignore '_test.go' .
go run github.com/uudashr/gocognit/cmd/gocognit@latest -top 5 -ignore '_test.go' .
```

## Usage

```bash
go run .
```

The server listens on `:8090` by default. Every setting has a built-in default and can be overridden through the environment; an unset or empty variable keeps the default:

| Variable | Default | Purpose |
|---|---|---|
| `COMMS_ADDR` | `:8090` | listen address |
| `COMMS_SENDGRID_API_KEY` | `FOOBAR123XYZ` | sendgrid API key |
| `COMMS_EMAIL_FROM_NAME` | `Foo Bar` | sender display name |
| `COMMS_EMAIL_FROM_ADDRESS` | `foo@bar.com` | sender address |

```bash
COMMS_ADDR=:9000 COMMS_SENDGRID_API_KEY=SG.xxx go run .
```

```bash
curl -X POST localhost:8090/api/comms/add-policy-coverage \
  -H 'Content-Type: application/json' \
  -d '{"email_to":"policyholder@example.com","email_cc":["agent@example.com"],"message":{"policy_number":"P-123"}}'
```

The three pre-existing routes (`add-policy-vehicle`, `add-policy-driver`, `add-policy-address`) take the same body without `email_cc`.

## Testing

```bash
go test ./... -race -cover
```

**Coverage** (`go test -race -coverpkg=./... -coverprofile=cover.out ./... && go tool cover -func=cover.out`): **98.2%** of statements module-wide. Every handler, `newMux`, `newServer`, `loadConfig`, `run`, and everything in `email/sendgrid` and `email/mockemail` is at **100%**.

- `handlers/handlers_test.go` runs one shared table of cases (success, trimmed address, every row of the contract table above, and a provider failure via a small local `erroringMailProvider`, since `mockemail.Client` never fails on its own) against each of the four handlers, asserting on status, exact response body, the `Allow` header, and the recorded send. `TestAddPolicyCoverage` extends that table with the CC business rules: CC recorded on the send, per-index CC validation errors, trimming, case-insensitive de-duplication, dropping of the `To` address, and the fall-back to a plain send when the list empties. `TestSendErrorIsLoggedNotEchoed` pins that the provider's error text reaches the log and not the response.
- `email/mockemail/mockemail_test.go` covers per-recipient logging, `Last()` on an empty log, flushing, snapshot semantics of `SendLogs()`/`Extract*`, and a `-race`-checked concurrent hammer.
- `email/sendgrid/sendgrid_test.go` asserts recipients, sender, template and message land in the generated `V3Mail`, that zero `To` recipients are refused before the provider is called, and that provider errors are wrapped but still match with `errors.Is`.
- `main_test.go` covers routing, mux-level rejections and 404s, `loadConfig` defaults/overrides, `newServer` timeouts and `run`'s listen failure.

Two things are deliberately left uncovered:
- `main()`'s single statement (`log.Fatal(run(loadConfig(os.Getenv)))`) — everything it calls is covered.
- `mail.Personalization.AddBCCs` in the vendored `mail_v3.go` — BCC isn't part of this challenge's contract, and that file is marked "please do not modify".

`gofmt -l .`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./...` and the complexity lint above are all clean.
