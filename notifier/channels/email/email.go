// Package email defines the MailProvider contract the notifier producer
// depends on, plus the template identifiers shared by its implementations.
package email

import (
	"errors"
	"fmt"
)

type TplID string

const (
	TplDocumentUpload TplID = "new-document"
	TplOTPLogin       TplID = "otp-login"
	TplPolicyRenewal  TplID = "policy-renewal" // used by the policy-renewal notification topic
)

// ErrMissingRecipient is returned (wrapped) by RequireRecipient when the
// recipient list is empty or contains a blank address.
var ErrMissingRecipient = errors.New("email requires recipient")

type MailProvider interface {
	Send(to []string, tpl TplID, vars map[string]any) error
}

// RequireRecipient reports ErrMissingRecipient when to is empty or when any
// entry in it is blank, so providers never attempt a send to "". Checking
// only the first entry would let a blank address later in the list through.
func RequireRecipient(to []string) error {
	if len(to) == 0 {
		return ErrMissingRecipient
	}
	for i, addr := range to {
		if addr == "" {
			return fmt.Errorf("%w: entry %d is blank", ErrMissingRecipient, i)
		}
	}
	return nil
}
