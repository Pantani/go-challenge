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

func TestNotifyOTPLoginInvalidInput(t *testing.T) {
	tests := map[string]any{
		"wrong input type":        "not-an-otp-login-input",
		"missing recipient":       OTPLoginInput{OTPCode: "123456", ExpirationMins: 5},
		"missing otp code":        OTPLoginInput{Recipient: "user@example.com", ExpirationMins: 5},
		"non-positive expiration": OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: 0},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			producer := NewProducer(mockemail.NewClient())
			if err := producer.NotifyTopic(context.Background(), TopicOTPLogin, input); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}
