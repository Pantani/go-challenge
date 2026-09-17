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
