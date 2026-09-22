package notifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func assertEqual(t *testing.T, label string, want, got any) {
	t.Helper()
	if want != got {
		t.Fatalf("expected %s %v, got %v", label, want, got)
	}
}

func TestNotifyPolicyRenewal(t *testing.T) {
	mail := mockemail.NewClient()
	producer := notifier.NewProducer(mail)

	renewalDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	err := producer.NotifyTopic(context.Background(), notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{
		Recipient:    "user@example.com",
		PolicyNumber: "POL-123",
		RenewalDate:  renewalDate,
	})
	if err != nil {
		t.Fatalf("notify policy renewal: %v", err)
	}

	logs := mail.SendLogs()
	if logs.IsEmpty() {
		t.Fatal("expected send log")
	}
	last := logs.Last()
	assertEqual(t, "recipient", "user@example.com", last.To)
	assertEqual(t, "template", email.TplPolicyRenewal, last.Tpl)
	assertEqual(t, "policy number", "POL-123", last.Vars["policyNumber"])
	assertEqual(t, "renewal date", "2026-10-01", last.Vars["renewalDate"])
}

func TestNotifyPolicyRenewalInvalidInput(t *testing.T) {
	validDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]any{
		"wrong input type":      "not-a-policy-renewal-input",
		"missing recipient":     notifier.PolicyRenewalInput{PolicyNumber: "POL-123", RenewalDate: validDate},
		"missing policy number": notifier.PolicyRenewalInput{Recipient: "user@example.com", RenewalDate: validDate},
		"missing renewal date":  notifier.PolicyRenewalInput{Recipient: "user@example.com", PolicyNumber: "POL-123"},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			producer := notifier.NewProducer(mockemail.NewClient())
			if err := producer.NotifyTopic(context.Background(), notifier.TopicPolicyRenewal, input); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}
