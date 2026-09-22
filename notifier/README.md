# Notification Topic Challenge

This challenge is a small extraction of Glovebox's notification producer. The producer turns typed business inputs into email delivery requests through topic builders.

## Business Problem

Glovebox needs to notify policyholders when a policy renewal is approaching. Add a new notification topic that sends a policy renewal reminder using the existing producer and email abstractions.

## Instructions

* [x] Add the policy renewal input model and notification topic.
* [x] Implement the topic builder using the patterns demonstrated by the existing builders.
* [x] Register the new builder so `NotifyTopic` can find it.
* [x] Add unit tests for the successful path and invalid input paths.
* [x] Ensure the module compiles and all tests pass with `go test ./...`.

## Layout

```
notifier/
├── cmd/notifier/
│   ├── main.go                        # demo binary (package main): main() + testable run()
│   └── main_test.go
├── producer.go                        # Producer, ProducerProvider, TopicRequestBuilder, Request,
│                                      # defaultTopicBuilders() and the generic topicBuilder[T] skeleton
├── producer_delivery.go               # optional ContextMailProvider, delivery dispatch and nil guard
├── errors.go                          # sentinel errors (errors.Is-able) for the producer and each topic
├── producer_doc_upload.go             # TopicDocumentUpload + its topic builder
├── producer_otp_login.go              # TopicOTPLogin + its topic builder
├── producer_policy_renewal.go         # TopicPolicyRenewal + its topic builder
├── producer_test.go                   # Producer + doc-upload tests (package notifier_test)
├── producer_validation_test.go        # nonblank validation, missing providers and original values
├── producer_delivery_test.go          # context dispatch, cancellation and legacy compatibility
├── producer_otp_login_test.go         # otp-login tests
├── producer_policy_renewal_test.go    # policy-renewal tests
└── channels/
    └── email/                         # the MailProvider contract and its implementations
        ├── email.go                   # MailProvider, TplID, constants, RequireRecipient and RequireTemplate
        ├── provider_contract_test.go  # shared provider validation and cancellation contracts
        ├── sendgrid/                  # local stand-in (validation only; no network delivery)
        └── mockemail/                 # thread-safe in-memory implementation used by the demo and tests
```

The root `notifier` package holds the producer's orchestration logic (`Producer`, routing by topic) and one file per notification topic builder. `channels/email` is the delivery mechanism the producer depends on, following the same interface-plus-implementations shape as `comms/email`. `cmd/notifier` holds only the demo entrypoint.

## Implementation

The policy renewal reminder is added as a new topic, following the same shape as the existing topic builders: one file per topic, defining a `Topic...` constant, an `...Input` struct, and a `...TopicBuilder` value built from the shared generic `topicBuilder[T]` in `producer.go`, which handles the input type assertion and `Request` assembly so each topic only supplies its validation and template variables. Every builder is listed once in `defaultTopicBuilders()`, which `NewProducer` uses to populate the topic map; adding a topic means adding one file and one line there.

* `TopicPolicyRenewal` — the topic key.
* `PolicyRenewalInput{Recipient, PolicyNumber, RenewalDate}` — the typed input. All three fields are required; `RenewalDate` must be a non-zero `time.Time`.
* `policyRenewalTopicBuilder` — validates the input and builds the email `Request`, passing `policyNumber` and a formatted `renewalDate` (`YYYY-MM-DD`) as template variables.

### Validation and delivery contracts

Required recipient, template, document, OTP code, policy number, and SendGrid API-key strings reject values whose `strings.TrimSpace` result is empty. Nonblank values reach the provider unchanged. This is minimum presence validation, not RFC email parsing or verification of deliverability. Sender display-name/address configuration is not newly required.

Validation errors remain classifiable with `errors.Is`. `Producer.Notify` validates the request before checking provider configuration and reports `ErrMissingMailProvider` for nil-interface and typed-nil providers. Pre-canceled contexts are rejected before request validation. The producer retains `ErrMissingRecipients` and `ErrMissingTemplate`; shared provider checks expose `email.ErrMissingRecipient` and `email.ErrMissingTemplate`. SendGrid template failures also retain `sendgrid.ErrMissingTemplate` compatibility.

`email.MailProvider` retains `Send(to, tpl, vars)`. Providers may additionally implement `notifier.ContextMailProvider.SendContext(ctx, to, tpl, vars)`. The producer passes the caller's context to that optional method. Legacy providers remain usable, but an already-running legacy `Send` cannot be interrupted. The bundled mock checks cancellation before validation and again after acquiring its log mutex; mutex acquisition itself is not interruptible, and cancellation racing with a recorded batch does not roll it back. The SendGrid stand-in checks cancellation at entry and performs validation only, with no network delivery.

The mock copies the top-level `Vars` map when recording and when returning logs. Nested maps, slices, and pointers remain caller-owned and shared; synchronize their use and do not treat the log as a deep immutable snapshot. Built-in topic builders currently emit scalar strings and integers only. No reflection-based deep clone is provided.

`ProducerProvider` remains available as an injection contract for callers. `TopicRequestBuilder` describes the topic-construction boundary; registration remains internal to `NewProducer`, with no public dynamic registration API. Existing exported abstractions are retained for source compatibility.

### Usage

```go
import "github.com/gloveboxhq/glovebox-go-code-challenge/notifier"

producer := notifier.NewProducer(mailProvider)
err := producer.NotifyTopic(ctx, notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{
    Recipient:    "policyholder@example.com",
    PolicyNumber: "POL-123",
    RenewalDate:  time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
})
```

### Running tests and the demo

Run these commands from the repository root. The subshells return the caller to the repository root:

```bash
(cd notifier && go test -race ./...)
(cd notifier && go run ./cmd/notifier)
make check-notifier
make cover-notifier
make docker-check-notifier
```

Use Go 1.22.3 or newer for direct module commands. The root Makefile provides per-module and Docker alternatives. CI enforces the repository's 90% minimum per module; generated local coverage reports are measurements, not permanent package guarantees. The demo records one example per registered topic in memory.
