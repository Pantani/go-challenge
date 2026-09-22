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

## Implementation notes

The producer supports `policy-renewal` through `PolicyRenewalInput`. The topic
is registered by `NewProducer`, delivers with `TplPolicyRenewal`, and supplies
the policy number and ISO renewal date to the existing email abstraction.

Recipient, policy number, and renewal date are required. Empty or whitespace-
only recipient and policy-number values are rejected before email delivery.

```sh
cd notifier
go test -race ./...
go vet ./...
```
