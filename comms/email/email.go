// Package email defines the mail provider contract used by the comms API
// handlers to send templated emails.
package email

import "encoding/json"

// TplID identifies which email template a MailProvider should render.
type TplID string

const (
	TplAddPolicyVehicle  TplID = "add-policy-vehicle"
	TplAddPolicyDriver   TplID = "add-policy-driver"
	TplAddPolicyAddress  TplID = "add-policy-address"
	TplAddPolicyCoverage TplID = "add-policy-coverage"
)

// MailProvider sends templated emails on behalf of the comms API.
type MailProvider interface {
	// Send delivers message, rendered with tpl, to the given to recipients.
	Send(to []string, message json.RawMessage, tpl TplID) error
	// SendWithCC behaves like Send but additionally copies the given cc
	// recipients on the message.
	SendWithCC(to, cc []string, message json.RawMessage, tpl TplID) error
}
