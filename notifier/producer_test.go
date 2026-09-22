package notifier_test

import (
	"context"
	"errors"
	"strings"
	"testing"

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

func TestNotifyDocumentUpload(t *testing.T) {
	mail := mockemail.NewClient()
	producer := notifier.NewProducer(mail)

	err := producer.NotifyTopic(context.Background(), notifier.TopicDocumentUpload, notifier.DocumentUploadInput{
		Recipient: "user@example.com",
		Document:  "policy.pdf",
	})
	if err != nil {
		t.Fatalf("notify document upload: %v", err)
	}

	logs := mail.SendLogs()
	if logs.IsEmpty() {
		t.Fatal("expected send log")
	}
	last := logs.Last()
	assertEqual(t, "recipient", "user@example.com", last.To)
	assertEqual(t, "template", email.TplDocumentUpload, last.Tpl)
	assertEqual(t, "document var", "policy.pdf", last.Vars["document"])
}

func TestNotifyUnknownTopic(t *testing.T) {
	mail := mockemail.NewClient()

	err := notifier.NewProducer(mail).NotifyTopic(context.Background(), "unknown", nil)
	if !errors.Is(err, notifier.ErrTopicNotRegistered) {
		t.Fatalf("expected ErrTopicNotRegistered, got %v", err)
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected error to name the topic, got %q", err.Error())
	}
	if !mail.SendLogs().IsEmpty() {
		t.Fatal("expected no email to be sent for an unknown topic")
	}
}

// A bare err==nil check can't tell "the right validation fired" from "some
// other validation fired instead" -- it only proves *a* branch returned an
// error, not *which* branch. errors.Is against the specific sentinel (and
// checking that nothing was sent) ties each case to the exact check it
// claims to exercise, without coupling the test to message wording.
func TestNotifyDocumentUploadInvalidInput(t *testing.T) {
	testCases := map[string]struct {
		input   any
		wantErr error
	}{
		"wrong input type": {
			input:   "not-a-document-upload-input",
			wantErr: notifier.ErrInvalidDocumentUploadInput,
		},
		"nil input": {
			input:   nil,
			wantErr: notifier.ErrInvalidDocumentUploadInput,
		},
		"pointer instead of value": {
			input:   &notifier.DocumentUploadInput{Recipient: "user@example.com", Document: "policy.pdf"},
			wantErr: notifier.ErrInvalidDocumentUploadInput,
		},
		"missing recipient": {
			input:   notifier.DocumentUploadInput{Document: "policy.pdf"},
			wantErr: notifier.ErrDocumentUploadMissingRecipient,
		},
		"missing document": {
			input:   notifier.DocumentUploadInput{Recipient: "user@example.com"},
			wantErr: notifier.ErrDocumentUploadMissingDocument,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := notifier.NewProducer(mail).NotifyTopic(context.Background(), notifier.TopicDocumentUpload, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent, got %d log(s)", len(mail.SendLogs()))
			}
		})
	}
}

// The wrong-type error should say what was expected and what arrived, so a
// caller passing the wrong struct can tell at a glance.
func TestNotifyInvalidInputTypeMessage(t *testing.T) {
	err := notifier.NewProducer(mockemail.NewClient()).NotifyTopic(
		context.Background(), notifier.TopicDocumentUpload, notifier.OTPLoginInput{})
	if !errors.Is(err, notifier.ErrInvalidDocumentUploadInput) {
		t.Fatalf("expected ErrInvalidDocumentUploadInput, got %v", err)
	}
	for _, want := range []string{"notifier.DocumentUploadInput", "notifier.OTPLoginInput"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error %q to mention %q", err.Error(), want)
		}
	}
}

func TestNotifyInvalidRequest(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	validReq := notifier.Request{
		Topic:      notifier.TopicOTPLogin,
		Recipients: []string{"user@example.com"},
		Template:   email.TplOTPLogin,
	}

	testCases := map[string]struct {
		ctx     context.Context
		req     notifier.Request
		wantErr error
	}{
		"cancelled context": {
			ctx:     cancelled,
			req:     validReq,
			wantErr: context.Canceled,
		},
		"empty request": {
			ctx:     context.Background(),
			req:     notifier.Request{},
			wantErr: notifier.ErrMissingRecipients,
		},
		"blank first recipient": {
			ctx:     context.Background(),
			req:     notifier.Request{Topic: validReq.Topic, Recipients: []string{""}, Template: validReq.Template},
			wantErr: notifier.ErrMissingRecipients,
		},
		"blank later recipient": {
			ctx: context.Background(),
			req: notifier.Request{
				Topic:      validReq.Topic,
				Recipients: []string{"user@example.com", ""},
				Template:   validReq.Template,
			},
			wantErr: notifier.ErrMissingRecipients,
		},
		"missing template": {
			ctx:     context.Background(),
			req:     notifier.Request{Topic: validReq.Topic, Recipients: validReq.Recipients},
			wantErr: notifier.ErrMissingTemplate,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := notifier.NewProducer(mail).Notify(tc.ctx, tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatal("expected no email to be sent")
			}
		})
	}
}

func TestNotifyDeliversValidRequest(t *testing.T) {
	mail := mockemail.NewClient()

	err := notifier.NewProducer(mail).Notify(context.Background(), notifier.Request{
		Topic:      notifier.TopicOTPLogin,
		Recipients: []string{"a@example.com", "b@example.com"},
		Template:   email.TplOTPLogin,
		Vars:       map[string]any{"otpCode": "123456"},
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	logs := mail.SendLogs()
	assertEqual(t, "log count", 2, len(logs))
	assertEqual(t, "first recipient", "a@example.com", logs[0].To)
	assertEqual(t, "second recipient", "b@example.com", logs[1].To)
	assertEqual(t, "otp code", "123456", logs.Last().Vars["otpCode"])
}

func TestNotifyTopicCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mail := mockemail.NewClient()

	err := notifier.NewProducer(mail).NotifyTopic(ctx, notifier.TopicDocumentUpload, notifier.DocumentUploadInput{
		Recipient: "user@example.com",
		Document:  "policy.pdf",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !mail.SendLogs().IsEmpty() {
		t.Fatal("expected no email to be sent once the context is cancelled")
	}
}

func TestNotifyTopicPropagatesSendError(t *testing.T) {
	mail := mockemail.NewClient()
	sendErr := errors.New("provider down")
	mail.SetSendError(sendErr)

	err := notifier.NewProducer(mail).NotifyTopic(context.Background(), notifier.TopicDocumentUpload, notifier.DocumentUploadInput{
		Recipient: "user@example.com",
		Document:  "policy.pdf",
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected %v, got %v", sendErr, err)
	}
}

// Every registered topic must be reachable through NotifyTopic and must
// stamp its own key on the request the provider receives.
func TestNotifyTopicRegistersAllTopics(t *testing.T) {
	inputs := map[string]any{
		notifier.TopicDocumentUpload: notifier.DocumentUploadInput{Recipient: "user@example.com", Document: "policy.pdf"},
		notifier.TopicOTPLogin:       notifier.OTPLoginInput{Recipient: "user@example.com", OTPCode: "123456", ExpirationMins: 5},
		notifier.TopicPolicyRenewal:  validPolicyRenewalInput(),
	}
	wantTpl := map[string]email.TplID{
		notifier.TopicDocumentUpload: email.TplDocumentUpload,
		notifier.TopicOTPLogin:       email.TplOTPLogin,
		notifier.TopicPolicyRenewal:  email.TplPolicyRenewal,
	}

	for topic, input := range inputs {
		t.Run(topic, func(t *testing.T) {
			mail := mockemail.NewClient()

			if err := notifier.NewProducer(mail).NotifyTopic(context.Background(), topic, input); err != nil {
				t.Fatalf("notify %s: %v", topic, err)
			}
			assertEqual(t, "log count", 1, len(mail.SendLogs()))
			assertEqual(t, "template", wantTpl[topic], mail.SendLogs().Last().Tpl)
		})
	}
}
