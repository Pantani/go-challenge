# Notification Topic Challenge

This challenge is a small extraction of Glovebox's notification producer. The producer turns typed business inputs into email delivery requests through topic builders.

## Business Problem

Glovebox needs to notify policyholders when a policy renewal is approaching. Add a new notification topic that sends a policy renewal reminder using the existing producer and email abstractions.

## Instructions

* [ ] Add the policy renewal input model and notification topic.
* [ ] Implement the topic builder using the patterns demonstrated by the existing builders.
* [ ] Register the new builder so `NotifyTopic` can find it.
* [ ] Add unit tests for the successful path and invalid input paths.
* [ ] Ensure the module compiles and all tests pass with `go test ./...`.

## Implementation

The policy renewal reminder is added as a new topic, following the same shape as the existing topic builders: one file per topic, defining a `Topic...` constant, an `...Input` struct, and a `...TopicBuilder` with `Topic()`/`BuildRequest()`, registered in `NewProducer`.

* `TopicPolicyRenewal` — the topic key.
* `PolicyRenewalInput{Recipient, PolicyNumber, RenewalDate}` — the typed input. All three fields are required; `RenewalDate` must be a non-zero `time.Time`.
* `policyRenewalTopicBuilder` — validates the input and builds the email `Request`, passing `policyNumber` and a formatted `renewalDate` (`YYYY-MM-DD`) as template variables.

### Usage

```go
producer := NewProducer(mailProvider)
err := producer.NotifyTopic(ctx, TopicPolicyRenewal, PolicyRenewalInput{
    Recipient:    "policyholder@example.com",
    PolicyNumber: "POL-123",
    RenewalDate:  time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
})
```

### Running the tests

```bash
go test ./... -v -cover
```

Every package in the module (`notifier`, `channels/email`, `channels/email/mockemail`, `channels/email/sendgrid`) is covered at 90% or higher.
