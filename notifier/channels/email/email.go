package email

import "fmt"

type TplID string

const (
	TplDocumentUpload TplID = "new-document"
	TplOTPLogin       TplID = "otp-login"
	TplPolicyRenewal  TplID = "policy-renewal"
)

type MailProvider interface {
	Send(to []string, tpl TplID, vars map[string]any) error
}

func RequireRecipient(to []string) error {
	if len(to) == 0 || to[0] == "" {
		return fmt.Errorf("email requires recipient")
	}
	return nil
}
