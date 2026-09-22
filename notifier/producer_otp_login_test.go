package notifier_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestNotifyOTPLogin(t *testing.T) {
	mail := mockemail.NewClient()
	producer := notifier.NewProducer(mail)

	err := producer.NotifyTopic(context.Background(), notifier.TopicOTPLogin, notifier.OTPLoginInput{
		Recipient:      "user@example.com",
		OTPCode:        "123456",
		ExpirationMins: 5,
	})
	if err != nil {
		t.Fatalf("notify otp login: %v", err)
	}

	logs := mail.SendLogs()
	if logs.IsEmpty() {
		t.Fatal("expected send log")
	}
	last := logs.Last()
	assertEqual(t, "recipient", "user@example.com", last.To)
	assertEqual(t, "template", email.TplOTPLogin, last.Tpl)
	assertEqual(t, "otp code", "123456", last.Vars["otpCode"])
	// The template renders how long the code stays valid, so the validated
	// expiry must reach the provider too.
	assertEqual(t, "expiration minutes", 5, last.Vars["expirationMins"])
}

// A bare err==nil check can't tell "the right validation fired" from "some
// other validation fired instead" -- it only proves *a* branch returned an
// error, not *which* branch. errors.Is against the specific sentinel (and
// checking that nothing was sent) ties each case to the exact check it
// claims to exercise, without coupling the test to message wording.
func TestNotifyOTPLoginInvalidInput(t *testing.T) {
	testCases := map[string]struct {
		input   any
		wantErr error
	}{
		"wrong input type": {
			input:   "not-an-otp-login-input",
			wantErr: notifier.ErrInvalidOTPLoginInput,
		},
		"missing recipient": {
			input:   notifier.OTPLoginInput{OTPCode: "123456", ExpirationMins: 5},
			wantErr: notifier.ErrOTPLoginMissingRecipient,
		},
		"missing otp code": {
			input:   notifier.OTPLoginInput{Recipient: "user@example.com", ExpirationMins: 5},
			wantErr: notifier.ErrOTPLoginMissingCode,
		},
		"zero expiration": {
			input:   notifier.OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: 0},
			wantErr: notifier.ErrOTPLoginInvalidExpiration,
		},
		"negative expiration": {
			input:   notifier.OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: -1},
			wantErr: notifier.ErrOTPLoginInvalidExpiration,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := notifier.NewProducer(mail).NotifyTopic(context.Background(), notifier.TopicOTPLogin, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent, got %d log(s)", len(mail.SendLogs()))
			}
		})
	}
}

// TestNotifyOTPLoginExpirationBoundary pins the exact validity boundary:
// 1 is the smallest valid ExpirationMins. TestNotifyOTPLogin alone (which
// uses ExpirationMins:5) cannot detect a mutant that shifts the boundary,
// e.g. `<= 0` to `<= 1`.
func TestNotifyOTPLoginExpirationBoundary(t *testing.T) {
	mail := mockemail.NewClient()

	err := notifier.NewProducer(mail).NotifyTopic(context.Background(), notifier.TopicOTPLogin, notifier.OTPLoginInput{
		Recipient:      "user@example.com",
		OTPCode:        "123456",
		ExpirationMins: 1,
	})
	if err != nil {
		t.Fatalf("expected boundary value 1 to be valid: %v", err)
	}
	if mail.SendLogs().IsEmpty() {
		t.Fatal("expected send log for valid boundary expiration")
	}
}
