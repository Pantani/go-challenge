# Comms API Challenge

The challenge is to solve an emerging business problem within an existing codebase.

**Context:**

The API exposes four comms operations and injects an email provider into each handler. The executable uses the bundled SendGrid-shaped local stand-in, which builds mail values but performs no network delivery. Tests also use an in-memory recording provider.

**Business Problem:**

The comms api needs to support a new operation for adding policy coverage. The new `add-policy-coverage` route should have the same data contract as the others, with the exception that the payload for the new route also needs to support `CC` in addition to `To` for the email recipients. Until this point only `To` recipients have been needed, so the current email service provider does not yet support `CC`.

**Instructions:** 

* [x] Create a handler for the new comms operation and implement the correct business logic inside of the handler.
* [x] Make whatever improvements are needed to support the new email `CC` field.
* [x] Create a unit test for the new handler to prove it meets business requirements.

## Layout

```
comms/
├── cmd/comms/
│   ├── main.go                # server binary (package main): config from env, newMux (route table),
│   │                          # newServer (timeouts), run/main
│   └── main_test.go            # integration tests: real HTTP requests against newMux; config + server tests
├── handlers/                  # one http.HandlerFunc per comms operation
│   ├── handlers.go              # AddPolicyVehicle/Driver/Address/Coverage + the shared sendHandler pipeline
│   ├── delivery.go               # optional context capabilities and legacy fallback
│   ├── models.go                 # one request struct per operation
│   └── *_test.go                 # HTTP contracts, byte boundaries, JSON policy and context dispatch
└── email/                     # the MailProvider contract and its implementations
    ├── email.go                  # MailProvider interface, TplID, template constants
    ├── sendgrid/                  # local SendGrid-shaped adapter; no network delivery
    └── mockemail/                  # goroutine-safe in-memory implementation used by handler tests
```

`cmd/comms` holds only the server entrypoint, matching the `carrierproxy`/`filestore`/`notifier` convention of keeping the binary separate from the implementation packages. `handlers` and `email` keep their own packages, each with a single responsibility — the HTTP layer and the delivery mechanism it depends on, respectively.

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

All four routes accept `POST` with a JSON body and respond with an empty `200 OK` when the selected provider returns `nil`. Method, payload and validation failures are rejected before calling the provider; provider failures return `500`:

| Condition | Status | Body |
|---|---|---|
| Method is not `POST` (`Allow: POST` header is set) | `405` | `method not allowed` |
| Body is not valid JSON, or a field has the wrong type | `400` | `invalid payload` |
| A second JSON value or trailing non-whitespace data | `400` | `invalid payload: unexpected data after JSON value` |
| Body exceeds 1 MiB | `413` | `request body too large` |
| `email_to` is missing or blank | `400` | `email_to: is required` |
| `email_to` is not an email address | `400` | `email_to: is not a valid email address` |
| `email_to` carries a display name (`Foo <foo@bar.com>`) | `400` | `email_to: must be a bare email address` |
| an `email_cc` entry fails any of the above (`add-policy-coverage` only) | `400` | `email_cc[<index>]: <same reason>` |
| `message` is missing or `null` | `400` | `message: is required` |
| the email provider fails | `500` | `error sending email` (the provider's error is logged server-side, never echoed) |

The limit is 1,048,576 bytes for the entire request body, including trailing whitespace. Bodies of exactly that size are accepted when otherwise valid; a body one byte larger receives 413 and does not reach the provider.

The decoder accepts unknown object fields and does not require a Content-Type header. It accepts exactly one JSON value followed by optional JSON whitespace. The request must decode into the route's request struct and pass field validation; message may contain any non-null JSON value, including a scalar or array. Duplicate fields follow encoding/json's normal processing order; for repeated scalar email_to fields, the last value is used. No additional strict-JSON policy is imposed.

An empty 200 response means the selected provider returned nil. In the bundled executable that is local acceptance only: the SendGrid stand-in performs no network delivery, credential authentication, or external delivery confirmation.

Addresses are trimmed of surrounding whitespace before validation and delivery. `email_cc` may be omitted, `null` or `[]`.

### CC normalisation

Before delivery the CC list is normalised so the provider is never asked to copy the same mailbox twice: entries are de-duplicated case-insensitively (first occurrence wins, order preserved), and any entry equal to `email_to` is dropped. If nothing remains, the email is sent with no CC at all. This is a deliberate business rule — sendgrid rejects personalizations that repeat an address across `to`/`cc` — and it is covered by `TestAddPolicyCoverage`.

## Design

MailProvider retains both existing methods unchanged. The handler separately discovers optional SendContext(context.Context, []string, json.RawMessage, email.TplID) error and SendWithCCContext(context.Context, []string, []string, json.RawMessage, email.TplID) error capabilities. It forwards the request context to the method selected after CC normalization. A provider may implement either capability independently; when the selected capability is absent, the handler uses the corresponding legacy method.

The repository providers check cancellation before their synchronous local operation. Their legacy methods use context.Background(). Third-party providers exposing only the legacy contract remain non-cancellable once called. The recording provider does not promise to interrupt mutex acquisition or roll back a recorded delivery, and the bundled SendGrid stand-in has no network operation to interrupt. A context-capable third-party adapter must propagate the supplied context to its own I/O.

**The four handlers share one generic pipeline.** `sendHandler[T]` (see [`handlers/handlers.go`](handlers/handlers.go)) owns the method check, `http.MaxBytesReader`-bounded decode with trailing-data rejection, field validation, CC normalisation, delivery and error translation. Each exported handler is a three-line adapter that maps its own request struct onto a provider-agnostic `envelope{to, cc, message}`. Adding a fifth operation means adding a request struct and one adapter; the contract table above applies to it automatically.

**The sendgrid wrapper is unit-testable, including its error path, without touching the vendored stand-in.** `sendgrid.Client` holds an unexported `mailClient` interface — the one method (`Send(*mail.V3Mail) error`) `Client` actually depends on (see [`email/sendgrid/sendgrid.go`](email/sendgrid/sendgrid.go)). `*mail.Client` satisfies that interface automatically, so `mail_v3.go` (marked "please do not modify") needed no changes. [`email/sendgrid/sendgrid_test.go`](email/sendgrid/sendgrid_test.go) substitutes a fake that captures the generated `*mail.V3Mail` and can return an error the bundled stand-in never does. Tests compare the entire constructed mail value against an independently built expected value using reflect.DeepEqual, including unexported fields. The upstream stand-in builder tests independently verify those builders. No stand-in changes or unsafe field access are required. `Client` refuses to build a message with no `To` recipient (`sendgrid.ErrNoRecipients`) and wraps provider failures with context while keeping them reachable through `errors.Is`.

**`mockemail` is goroutine-safe and hands out snapshots.** `Client` guards its log with a mutex; `SendLogs()` returns a copy, and `ExtractCC`/`ExtractMessage` return copies too, so tests (including parallel ones) cannot alias the client's state or each other's inputs. `SendLog` gained a `cc` field and `ExtractCC()` getter (see [`email/mockemail/sendlog.go`](email/mockemail/sendlog.go)) following the existing `Extract*` accessor pattern, and `SendLogs.Last()` returns `nil` on an empty log instead of panicking.

**`cmd/comms/main.go` is configurable and defensive.** `loadConfig` reads the environment (see Usage) and falls back to the built-in defaults; `newMux(emailsvc)` builds the route table on a fresh `http.ServeMux` rather than the global default; `newServer` wraps it in an `http.Server` with read-header, read, write and idle timeouts. [`cmd/comms/main_test.go`](cmd/comms/main_test.go) drives `newMux` through `httptest.NewServer` with real HTTP requests, table-tests `loadConfig`, checks `newServer`'s timeouts, and covers `run` by handing it a port the test already holds so `ListenAndServe` fails immediately.

The root complexity configuration enforces cyclomatic complexity ≤ 6 and cognitive complexity ≤ 10 for every production function, test and helper. To run the configured check from the repository root:

```bash
make complexity-comms
```

## Usage

From the `comms` module directory:

```bash
go run ./cmd/comms
```

The server listens on `:8090` by default. Every setting has a built-in default and can be overridden through the environment; an unset or empty variable keeps the default:

| Variable | Default | Purpose |
|---|---|---|
| `COMMS_ADDR` | `:8090` | listen address |
| `COMMS_SENDGRID_API_KEY` | `FOOBAR123XYZ` | stand-in API key value; no credential authentication |
| `COMMS_EMAIL_FROM_NAME` | `Foo Bar` | sender display name |
| `COMMS_EMAIL_FROM_ADDRESS` | `foo@bar.com` | sender address |

```bash
COMMS_ADDR=:9000 COMMS_SENDGRID_API_KEY=SG.xxx go run ./cmd/comms
```

```bash
curl -X POST localhost:8090/api/comms/add-policy-coverage \
  -H 'Content-Type: application/json' \
  -d '{"email_to":"policyholder@example.com","email_cc":["agent@example.com"],"message":{"policy_number":"P-123"}}'
```

The three pre-existing routes (`add-policy-vehicle`, `add-policy-driver`, `add-policy-address`) take the same body without `email_cc`.

## Testing

CI requires at least 90% statement coverage per module. Run the commands below to measure the current checkout.

From the repository root:

```bash
make test-comms
make race-comms
make cover-comms
make check-comms
```

From the `comms` module directory, use `go test -race ./...`. From the repository root, `make check-comms` includes the same race-enabled 90% aggregate statement coverage gate used by CI; `make coverage-check-comms` runs just that gate. `make cover-comms` generates reports without enforcing the threshold. See the [root README](../README.md#verification) for the shared verification workflow.

- `handlers/handlers_test.go` runs one shared table of cases against all four handlers: success, trimmed addresses, every row of the contract table above, and a provider failure. `TestAddPolicyCoverage` adds the CC rules: CC recorded on the send, per-index validation errors, trimming, case-insensitive de-duplication, dropping the `To` address, and falling back to a plain send when the list ends up empty. `TestSendErrorIsLoggedNotEchoed` checks that provider errors are logged but not returned to the client.
- `handlers/contract_test.go` pins the exact body-byte limits on all four routes and the accepted JSON and Content-Type policies.
- `handlers/delivery_test.go` verifies request-context propagation, optional capabilities selected independently after CC normalization, legacy fallback and cancellation.
- `email/contract_test.go` runs the same `MailProvider` expectations against both `sendgrid` and `mockemail`, checking their shared contract.
- `email/mockemail/mockemail_test.go` covers per-recipient logging, `Last()` on an empty log, flushing, snapshot semantics and concurrent use.
- `email/sendgrid/sendgrid_test.go` checks recipients, sender, template and message in the generated `V3Mail`, rejection of zero `To` recipients, and error wrapping.
- The provider `context_test.go` files verify cancellation before local recording or stand-in dispatch and preserve `errors.Is` classification.
- `email/sendgrid/mail/mail_v3_test.go` pins the behavior of the vendored stand-in that `sendgrid.Client` relies on, without modifying it.
- `cmd/comms/main_test.go` covers routing, rejections and 404s, `loadConfig`, `newServer`'s timeouts and `run`'s listen failure.
