# Comms API Challenge

`AddPolicyCoverage` accepts the existing policy-operation payload plus
`email_cc` recipients. It validates a POST JSON request, forwards `email_to`
as `To` and `email_cc` as `CC`, and uses `TplAddPolicyCoverage`.

The mock provider records CC recipients for handler tests. The SendGrid-shaped
provider adds the same recipients to its personalization object.

```sh
cd comms
go test -race ./...
go vet ./...
```
