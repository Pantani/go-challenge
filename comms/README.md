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
├── main.go                   # wires the sendgrid client to the routes
├── handlers/                 # one http.HandlerFunc per comms operation
│   ├── handlers.go             # AddPolicyVehicle, AddPolicyDriver, AddPolicyAddress, AddPolicyCoverage
│   ├── models.go                # one request struct per operation
│   └── handlers_test.go          # one table-driven test func per handler
└── email/                    # the MailProvider contract and its implementations
    ├── email.go                 # MailProvider interface, TplID, template constants
    ├── sendgrid/                 # real implementation, backed by the vendored mail stand-in
    └── mockemail/                 # in-memory implementation used by handler tests
```

## What's implemented

`AddPolicyCoverage` (see [`handlers/handlers.go`](handlers/handlers.go)) is a new handler registered at `POST /api/comms/add-policy-coverage`. It follows the exact shape of the three existing handlers — method check, JSON decode, send, translate errors to HTTP status codes — with one difference: its request struct, `AddPolicyCoverageReq` (see [`handlers/models.go`](handlers/models.go)), adds an `email_cc` field alongside the shared `email_to`/`message` contract:

```go
type AddPolicyCoverageReq struct {
	EmailTo string          `json:"email_to"`
	EmailCC []string        `json:"email_cc"`
	Message json.RawMessage `json:"message"`
}
```

## Design

**`MailProvider` gained a method, not a changed signature.** The existing `Send(to []string, message json.RawMessage, tpl TplID) error` is untouched, so the three pre-existing handlers and their tests needed no changes. `SendWithCC(to, cc []string, message json.RawMessage, tpl TplID) error` is the new, additive method that only `AddPolicyCoverage` calls (see [`email/email.go`](email/email.go)). Both `sendgrid.Client` and `mockemail.Client` implement `Send` and `SendWithCC` against one shared private helper (`send`/`record`), so the CC-handling logic exists in exactly one place per implementation.

**The sendgrid wrapper is now unit-testable, including its error path, without touching the vendored stand-in.** `sendgrid.Client` used to hold a concrete `*mail.Client`. It now holds an unexported `mailClient` interface — the one method (`Send(*mail.V3Mail) error`) `Client` actually depends on (see [`email/sendgrid/sendgrid.go`](email/sendgrid/sendgrid.go)). `*mail.Client` satisfies that interface automatically, so `mail_v3.go` (marked "please do not modify") needed no changes; [`email/sendgrid/sendgrid_test.go`](email/sendgrid/sendgrid_test.go) substitutes a fake that can return an error, which the real stand-in never does.

**`mockemail` gained a `cc` field and `ExtractCC()` getter** on `SendLog` (see [`email/mockemail/sendlog.go`](email/mockemail/sendlog.go)), following the existing `Extract*` accessor pattern, so handler tests can assert on recorded CC recipients the same way they already assert on `To`, `Message` and the template ID.

Every function touched by this change is kept small and single-purpose, checked with [`gocyclo`](https://github.com/fzipp/gocyclo) and [`gocognit`](https://github.com/uudashr/gocognit). The highest in the module — `AddPolicyCoverage`, tied with its three siblings — sits at cyclomatic complexity 4 and cognitive complexity 6:

```bash
go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -top 5 -ignore '_test.go' .
go run github.com/uudashr/gocognit/cmd/gocognit@latest -top 5 -ignore '_test.go' .
```

## Usage

```bash
go run .
```

```bash
curl -X POST localhost:8090/api/comms/add-policy-coverage \
  -H 'Content-Type: application/json' \
  -d '{"email_to":"policyholder@example.com","email_cc":["agent@example.com"],"message":{"policy_number":"P-123"}}'
```

## Testing

```bash
go test ./...
```

**Coverage** (`go test ./... -coverpkg=./... -coverprofile=cover.out && go tool cover -func=cover.out`): every function touched by this change — `AddPolicyCoverage`, all of `email/sendgrid/sendgrid.go`, and all of `email/mockemail` (`mockemail.go` and `sendlog.go`) — is at **100%**, including the sendgrid error-return path (see Design, above). `TestAddPolicyCoverage` in [`handlers/handlers_test.go`](handlers/handlers_test.go) covers all four branches: success (asserting `To`, `CC`, `Message` and template on the recorded send), invalid method, invalid JSON payload, and an email-service failure (via a small local `erroringMailProvider` fake, since `mockemail.Client` never fails on its own).

The three pre-existing handlers (`AddPolicyVehicle`, `AddPolicyDriver`, `AddPolicyAddress`) and `main.go` are untouched by this change and left at their existing coverage — `main.go` is a thin wiring entrypoint.

`golangci-lint run ./...` (gocyclo, gocognit, staticcheck and friends) and `go vet ./...` are both clean.
