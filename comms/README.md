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
├── main.go                   # newMux (route table) + main (wires sendgrid, listens)
├── main_test.go               # integration test: real HTTP requests against newMux
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

**The sendgrid wrapper is now unit-testable, including its error path, without touching the vendored stand-in.** `sendgrid.Client` used to hold a concrete `*mail.Client`. It now holds an unexported `mailClient` interface — the one method (`Send(*mail.V3Mail) error`) `Client` actually depends on (see [`email/sendgrid/sendgrid.go`](email/sendgrid/sendgrid.go)). `*mail.Client` satisfies that interface automatically, so `mail_v3.go` (marked "please do not modify") needed no changes. [`email/sendgrid/sendgrid_test.go`](email/sendgrid/sendgrid_test.go) substitutes a fake that captures the generated `*mail.V3Mail` and can return an error the real stand-in never does; since that type also has no getters (same "do not modify" constraint), the test reads its unexported `To`/`CC` fields the same way `%+v` does — `fmt`'s own struct formatting — rather than adding `unsafe` or touching the vendored type. This means the tests assert recipients actually land in the right field, not just that the call succeeds.

**`mockemail` gained a `cc` field and `ExtractCC()` getter** on `SendLog` (see [`email/mockemail/sendlog.go`](email/mockemail/sendlog.go)), following the existing `Extract*` accessor pattern, so handler tests can assert on recorded CC recipients the same way they already assert on `To`, `Message` and the template ID.

**`main.go`'s route table was extracted into `newMux(emailsvc) *http.ServeMux`**, replacing registration on the global `http.DefaultServeMux`. `main()` now just builds the real `sendgrid` client and calls `newMux` + `ListenAndServe`; [`main_test.go`](main_test.go) builds the same mux with `mockemail` instead and drives it through `httptest.NewServer` with real HTTP requests — an integration test covering the actual routing, which no test exercised before (the handler tests call handler funcs directly, bypassing routing entirely).

**Two of the three pre-existing handlers were missing a `return`** after their method-not-allowed check (`AddPolicyDriver`, `AddPolicyAddress` — `AddPolicyVehicle` had it right). Harmless in practice (the first `http.Error` call wins), but real: execution fell through to decode the body anyway. Fixed to match `AddPolicyVehicle`'s and `AddPolicyCoverage`'s pattern while extending those handlers' tests.

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

**Coverage** (`go test ./... -coverpkg=./... -coverprofile=cover.out && go tool cover -func=cover.out`): **92.9%** of statements module-wide. Every handler (`AddPolicyVehicle`, `AddPolicyDriver`, `AddPolicyAddress`, `AddPolicyCoverage`), `newMux`, and everything in `email/sendgrid` and `email/mockemail` is at **100%**, including the sendgrid error-return path (see Design, above). Each handler's table test covers all four branches: success, invalid method, invalid JSON payload, and an email-service failure (via a small local `erroringMailProvider` fake, since `mockemail.Client` never fails on its own) — `TestAddPolicyCoverage` additionally asserts `CC` on the recorded send.

Two things are deliberately left uncovered, both untestable without either a real network listener or touching the vendored stand-in:
- `main()`'s own body (building the real `sendgrid` client and calling `ListenAndServe`) — a thin wiring entrypoint; `newMux`, where the actual routing logic lives, is what `main_test.go` covers instead.
- `mail.Personalization.AddBCCs` in the vendored `mail_v3.go` — BCC isn't part of this challenge's contract, and that file is marked "please do not modify".

`golangci-lint run ./...` (gocyclo, gocognit, staticcheck and friends) and `go vet ./...` are both clean.
