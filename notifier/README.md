# Notification Topic Challenge

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
