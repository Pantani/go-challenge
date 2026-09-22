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
├── cmd/notifier/main.go              # demo binary (package main)
├── producer.go                        # Producer, ProducerProvider, TopicRequestBuilder, Request
├── producer_doc_upload.go              # TopicDocumentUpload + its topic builder
├── producer_otp_login.go                # TopicOTPLogin + its topic builder
├── producer_policy_renewal.go            # TopicPolicyRenewal + its topic builder
├── producer_test.go                       # Producer + doc-upload tests (package notifier_test)
├── producer_otp_login_test.go              # otp-login tests
├── producer_policy_renewal_test.go          # policy-renewal tests
└── channels/
    └── email/                 # the MailProvider contract and its implementations
        ├── email.go             # MailProvider interface, TplID, template constants
        ├── sendgrid/             # real implementation
        └── mockemail/             # in-memory implementation used by the demo and tests
```

The root `notifier` package holds the producer's orchestration logic (`Producer`, routing by topic) and one file per notification topic builder. `channels/email` is the delivery mechanism the producer depends on, following the same interface-plus-implementations shape as `comms/email`. `cmd/notifier` holds only the demo entrypoint.

## Implementation

The policy renewal reminder is added as a new topic, following the same shape as the existing topic builders: one file per topic, defining a `Topic...` constant, an `...Input` struct, and a `...TopicBuilder` with `Topic()`/`BuildRequest()`, registered in `NewProducer`.

* `TopicPolicyRenewal` — the topic key.
* `PolicyRenewalInput{Recipient, PolicyNumber, RenewalDate}` — the typed input. All three fields are required; `RenewalDate` must be a non-zero `time.Time`.
* `policyRenewalTopicBuilder` — validates the input and builds the email `Request`, passing `policyNumber` and a formatted `renewalDate` (`YYYY-MM-DD`) as template variables.

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

### Running the tests

```bash
go test ./... -v -cover
```

`notifier`, `channels/email`, `channels/email/mockemail`, and `channels/email/sendgrid` are all at **100%** statement coverage. `cmd/notifier` sits at 62.5%: `TestRun` exercises the demo end to end, and only the `main()` wrapper's `log.Fatal` path — reached solely by a `run()` failure that can't happen against the in-memory mock — is left uncovered, the same pattern `filestore/cmd/filestore` uses.
