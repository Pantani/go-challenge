package main

import (
	"context"
	"testing"
	"time"

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
	producer := NewProducer(mail)

	renewalDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	err := producer.NotifyTopic(context.Background(), TopicPolicyRenewal, PolicyRenewalInput{
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

	testCases := map[string]struct {
		input     any
		wantError string
	}{
		"wrong input type": {
			input:     "not-a-policy-renewal-input",
			wantError: "invalid policy renewal input type",
		},
		"missing recipient": {
			input:     PolicyRenewalInput{PolicyNumber: "POL-123", RenewalDate: validDate},
			wantError: "policy renewal requires recipient",
		},
		"missing policy number": {
			input:     PolicyRenewalInput{Recipient: "user@example.com", RenewalDate: validDate},
			wantError: "policy renewal requires policy number",
		},
		"missing renewal date": {
			input:     PolicyRenewalInput{Recipient: "user@example.com", PolicyNumber: "POL-123"},
			wantError: "policy renewal requires renewal date",
		},
	}

	// A bare err==nil check can't tell "the right validation fired" from
	// "some other validation fired instead" -- it only proves *a* branch
	// returned an error, not *which* branch. Pinning the exact message
	// (and that nothing was sent) is what actually ties each case to the
	// specific check it claims to exercise.
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := NewProducer(mail).NotifyTopic(context.Background(), TopicPolicyRenewal, tc.input)
			if err == nil {
				t.Fatalf("expected error for %s", name)
			}
			if err.Error() != tc.wantError {
				t.Fatalf("expected error %q, got %q", tc.wantError, err.Error())
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent, got %d log(s)", len(mail.SendLogs()))
			}
		})
	}
}
