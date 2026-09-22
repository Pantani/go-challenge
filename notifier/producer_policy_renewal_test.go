package notifier_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func validPolicyRenewalInput() notifier.PolicyRenewalInput {
	return notifier.PolicyRenewalInput{
		Recipient:    "user@example.com",
		PolicyNumber: "POL-123",
		RenewalDate:  time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestNotifyPolicyRenewal(t *testing.T) {
	mail := mockemail.NewClient()
	producer := notifier.NewProducer(mail)

	err := producer.NotifyTopic(context.Background(), notifier.TopicPolicyRenewal, validPolicyRenewalInput())
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
		input   any
		wantErr error
	}{
		"wrong input type": {
			input:   "not-a-policy-renewal-input",
			wantErr: notifier.ErrInvalidPolicyRenewalInput,
		},
		"missing recipient": {
			input:   notifier.PolicyRenewalInput{PolicyNumber: "POL-123", RenewalDate: validDate},
			wantErr: notifier.ErrPolicyRenewalMissingRecipient,
		},
		"missing policy number": {
			input:   notifier.PolicyRenewalInput{Recipient: "user@example.com", RenewalDate: validDate},
			wantErr: notifier.ErrPolicyRenewalMissingPolicyNumber,
		},
		"missing renewal date": {
			input:   notifier.PolicyRenewalInput{Recipient: "user@example.com", PolicyNumber: "POL-123"},
			wantErr: notifier.ErrPolicyRenewalMissingRenewalDate,
		},
	}

	// A bare err==nil check can't tell "the right validation fired" from
	// "some other validation fired instead" -- it only proves *a* branch
	// returned an error, not *which* branch. errors.Is against the specific
	// sentinel (and checking that nothing was sent) ties each case to the
	// exact check it claims to exercise, without coupling the test to
	// message wording.
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := notifier.NewProducer(mail).NotifyTopic(context.Background(), notifier.TopicPolicyRenewal, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent, got %d log(s)", len(mail.SendLogs()))
			}
		})
	}
}
