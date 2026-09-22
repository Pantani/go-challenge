package notifier

import (
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// TopicPolicyRenewal is the notification topic for policy renewal reminders.
const TopicPolicyRenewal = "policy-renewal"

// PolicyRenewalInput is the typed input for the policy-renewal topic.
// Recipient, PolicyNumber, and RenewalDate are all required.
type PolicyRenewalInput struct {
	Recipient    string
	PolicyNumber string
	RenewalDate  time.Time
}

var policyRenewalTopicBuilder = topicBuilder[PolicyRenewalInput]{
	topic:           TopicPolicyRenewal,
	tpl:             email.TplPolicyRenewal,
	errInvalidInput: ErrInvalidPolicyRenewalInput,
	build:           buildPolicyRenewal,
}

// buildPolicyRenewal validates a PolicyRenewalInput and returns the template
// variables: policyNumber and renewalDate formatted as YYYY-MM-DD.
func buildPolicyRenewal(in PolicyRenewalInput) (string, map[string]any, error) {
	if in.Recipient == "" {
		return "", nil, ErrPolicyRenewalMissingRecipient
	}
	if in.PolicyNumber == "" {
		return "", nil, ErrPolicyRenewalMissingPolicyNumber
	}
	if in.RenewalDate.IsZero() {
		return "", nil, ErrPolicyRenewalMissingRenewalDate
	}
	return in.Recipient, map[string]any{
		"policyNumber": in.PolicyNumber,
		"renewalDate":  in.RenewalDate.Format("2006-01-02"),
	}, nil
}
