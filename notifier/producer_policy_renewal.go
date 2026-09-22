package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

const TopicPolicyRenewal = "policy-renewal"

type PolicyRenewalInput struct {
	Recipient    string
	PolicyNumber string
	RenewalDate  time.Time
}

type policyRenewalTopicBuilder struct{}

func (policyRenewalTopicBuilder) Topic() string { return TopicPolicyRenewal }

func (policyRenewalTopicBuilder) BuildRequest(_ context.Context, input any) (Request, error) {

	typedInput, ok := input.(PolicyRenewalInput)
	if !ok {
		return Request{}, fmt.Errorf("invalid policy renewal input type")
	}
	if typedInput.Recipient == "" {
		return Request{}, fmt.Errorf("policy renewal requires recipient")
	}
	if typedInput.PolicyNumber == "" {
		return Request{}, fmt.Errorf("policy renewal requires policy number")
	}
	if typedInput.RenewalDate.IsZero() {
		return Request{}, fmt.Errorf("policy renewal requires renewal date")
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
