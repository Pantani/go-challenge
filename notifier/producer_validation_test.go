package notifier_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestNotifyRejectsBlankRequestFields(t *testing.T) {
	tests := []struct {
		name         string
		req          notifier.Request
		want         error
		providerWant error
	}{
		{"recipient", notifier.Request{Recipients: []string{" \t"}, Template: email.TplOTPLogin}, notifier.ErrMissingRecipients, email.ErrMissingRecipient},
		{"template", notifier.Request{Recipients: []string{"u@example.com"}, Template: "\u2003"}, notifier.ErrMissingTemplate, email.ErrMissingTemplate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mail := mockemail.NewClient()
			err := notifier.NewProducer(mail).Notify(context.Background(), tc.req)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Notify() = %v; want %v", err, tc.want)
			}
			if !errors.Is(err, tc.providerWant) {
				t.Fatalf("Notify() = %v; want provider cause %v", err, tc.providerWant)
			}
			assertEqual(t, "send count", 0, len(mail.SendLogs()))
		})
	}
}

func TestNotifyMissingProvider(t *testing.T) {
	tests := map[string]email.MailProvider{"nil interface": nil, "typed nil": (*mockemail.Client)(nil)}
	for name, mail := range tests {
		t.Run(name, func(t *testing.T) {
			req := notifier.Request{Recipients: []string{"u@example.com"}, Template: email.TplOTPLogin}
			err := notifier.NewProducer(mail).Notify(context.Background(), req)
			if !errors.Is(err, notifier.ErrMissingMailProvider) {
				t.Fatalf("Notify() = %v", err)
			}
		})
	}
}

func TestNotifyTopicMissingProvider(t *testing.T) {
	err := notifier.NewProducer(nil).NotifyTopic(context.Background(), notifier.TopicDocumentUpload,
		notifier.DocumentUploadInput{Recipient: "u@example.com", Document: "p.pdf"})
	if !errors.Is(err, notifier.ErrMissingMailProvider) {
		t.Fatalf("NotifyTopic() = %v", err)
	}
}

func TestTopicsRejectBlankStrings(t *testing.T) {
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name, topic string
		input       any
		want        error
	}{
		{"document recipient", notifier.TopicDocumentUpload, notifier.DocumentUploadInput{Recipient: " \t", Document: "p.pdf"}, notifier.ErrDocumentUploadMissingRecipient},
		{"document name", notifier.TopicDocumentUpload, notifier.DocumentUploadInput{Recipient: "u@example.com", Document: "\n"}, notifier.ErrDocumentUploadMissingDocument},
		{"otp recipient", notifier.TopicOTPLogin, notifier.OTPLoginInput{Recipient: "\u2003", OTPCode: "123", ExpirationMins: 1}, notifier.ErrOTPLoginMissingRecipient},
		{"otp code", notifier.TopicOTPLogin, notifier.OTPLoginInput{Recipient: "u@example.com", OTPCode: " \t", ExpirationMins: 1}, notifier.ErrOTPLoginMissingCode},
		{"policy recipient", notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{Recipient: "\n", PolicyNumber: "P", RenewalDate: date}, notifier.ErrPolicyRenewalMissingRecipient},
		{"policy number", notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{Recipient: "u@example.com", PolicyNumber: "\u2003", RenewalDate: date}, notifier.ErrPolicyRenewalMissingPolicyNumber},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mail := mockemail.NewClient()
			err := notifier.NewProducer(mail).NotifyTopic(context.Background(), tc.topic, tc.input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("NotifyTopic() = %v; want %v", err, tc.want)
			}
			assertEqual(t, "send count", 0, len(mail.SendLogs()))
		})
	}
}

func TestTopicsPreserveNonblankValues(t *testing.T) {
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	const to = " user@example.com "
	tests := []struct {
		topic string
		input any
		vars  map[string]any
	}{
		{notifier.TopicDocumentUpload, notifier.DocumentUploadInput{Recipient: to, Document: " p.pdf "}, map[string]any{"document": " p.pdf "}},
		{notifier.TopicOTPLogin, notifier.OTPLoginInput{Recipient: to, OTPCode: " 123 ", ExpirationMins: 1}, map[string]any{"otpCode": " 123 ", "expirationMins": 1}},
		{notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{Recipient: to, PolicyNumber: " P ", RenewalDate: date}, map[string]any{"policyNumber": " P ", "renewalDate": "2026-10-01"}},
	}
	for _, tc := range tests {
		t.Run(tc.topic, func(t *testing.T) {
			mail := mockemail.NewClient()
			if err := notifier.NewProducer(mail).NotifyTopic(context.Background(), tc.topic, tc.input); err != nil {
				t.Fatal(err)
			}
			logs := mail.SendLogs()
			assertEqual(t, "send count", 1, len(logs))
			assertEqual(t, "recipient", to, logs[0].To)
			if !reflect.DeepEqual(logs[0].Vars, tc.vars) {
				t.Fatalf("Vars = %#v; want %#v", logs[0].Vars, tc.vars)
			}
		})
	}
}

func TestNotifyPreservesTemplate(t *testing.T) {
	mail := mockemail.NewClient()
	req := notifier.Request{Recipients: []string{" local "}, Template: " custom ", Vars: map[string]any{"n": 1}}
	if err := notifier.NewProducer(mail).Notify(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "send count", 1, len(mail.SendLogs()))
	assertEqual(t, "template", req.Template, mail.SendLogs()[0].Tpl)
}

func TestNotifyValidationPrecedence(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		ctx  context.Context
		req  notifier.Request
		want error
	}{
		{"cancellation before validation", canceled, notifier.Request{}, context.Canceled},
		{"recipient before provider", context.Background(), notifier.Request{}, notifier.ErrMissingRecipients},
		{"template before provider", context.Background(), notifier.Request{Recipients: []string{"u@example.com"}}, notifier.ErrMissingTemplate},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := notifier.NewProducer(nil).Notify(tc.ctx, tc.req)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Notify() = %v; want %v", err, tc.want)
			}
		})
	}
}
