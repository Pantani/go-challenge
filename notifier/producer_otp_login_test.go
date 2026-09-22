package main

import (
	"context"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestNotifyOTPLogin(t *testing.T) {
	mail := mockemail.NewClient()
	producer := NewProducer(mail)

	err := producer.NotifyTopic(context.Background(), TopicOTPLogin, OTPLoginInput{
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
}

// A bare err==nil check can't tell "the right validation fired" from "some
// other validation fired instead" -- it only proves *a* branch returned an
// error, not *which* branch. Pinning the exact message (and that nothing
// was sent) ties each case to the specific check it claims to exercise.
func TestNotifyOTPLoginInvalidInput(t *testing.T) {
	testCases := map[string]struct {
		input     any
		wantError string
	}{
		"wrong input type": {
			input:     "not-an-otp-login-input",
			wantError: "invalid otp login input type",
		},
		"missing recipient": {
			input:     OTPLoginInput{OTPCode: "123456", ExpirationMins: 5},
			wantError: "otp login requires recipient",
		},
		"missing otp code": {
			input:     OTPLoginInput{Recipient: "user@example.com", ExpirationMins: 5},
			wantError: "otp login requires otp code",
		},
		"zero expiration": {
			input:     OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: 0},
			wantError: "otp login requires valid expiration minutes",
		},
		"negative expiration": {
			input:     OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: -1},
			wantError: "otp login requires valid expiration minutes",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := NewProducer(mail).NotifyTopic(context.Background(), TopicOTPLogin, tc.input)
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

// TestNotifyOTPLoginExpirationBoundary pins the exact validity boundary:
// 1 is the smallest valid ExpirationMins. TestNotifyOTPLogin alone (which
// uses ExpirationMins:5) cannot detect a mutant that shifts the boundary,
// e.g. `<= 0` to `<= 1`.
func TestNotifyOTPLoginExpirationBoundary(t *testing.T) {
	mail := mockemail.NewClient()

	err := NewProducer(mail).NotifyTopic(context.Background(), TopicOTPLogin, OTPLoginInput{
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
