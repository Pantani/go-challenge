package notifier

import (
	"context"
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

type policyRenewalTopicBuilder struct{}

// Topic returns the notification topic key this builder handles.
func (policyRenewalTopicBuilder) Topic() string { return TopicPolicyRenewal }

// BuildRequest validates a PolicyRenewalInput and builds the email Request
// for a policy renewal reminder.
func (policyRenewalTopicBuilder) BuildRequest(_ context.Context, input any) (Request, error) {

	typedInput, ok := input.(PolicyRenewalInput)
	if !ok {
		return Request{}, ErrInvalidPolicyRenewalInput
	}
	if typedInput.Recipient == "" {
		return Request{}, ErrPolicyRenewalMissingRecipient
	}
	if typedInput.PolicyNumber == "" {
		return Request{}, ErrPolicyRenewalMissingPolicyNumber
	}
	if typedInput.RenewalDate.IsZero() {
		return Request{}, ErrPolicyRenewalMissingRenewalDate
	}
	return Request{
		Topic:      TopicPolicyRenewal,
		Recipients: []string{typedInput.Recipient},
		Template:   email.TplPolicyRenewal,
		Vars: map[string]any{
			"policyNumber": typedInput.PolicyNumber,
			"renewalDate":  typedInput.RenewalDate.Format("2006-01-02"),
		},
	}, nil
}
