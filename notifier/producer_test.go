package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestNotifyDocumentUpload(t *testing.T) {
	mail := mockemail.NewClient()
	producer := NewProducer(mail)

	err := producer.NotifyTopic(context.Background(), TopicDocumentUpload, DocumentUploadInput{
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
	if last.To != "user@example.com" {
		t.Fatalf("expected recipient %q, got %q", "user@example.com", last.To)
	}
	if last.Tpl != email.TplDocumentUpload {
		t.Fatalf("expected template %q, got %q", email.TplDocumentUpload, last.Tpl)
	}
	if last.Vars["document"] != "policy.pdf" {
		t.Fatalf("expected document var %q, got %v", "policy.pdf", last.Vars["document"])
	}
}

func TestNotifyUnknownTopic(t *testing.T) {
	mail := mockemail.NewClient()

	err := NewProducer(mail).NotifyTopic(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected unknown topic error")
	}
	if !errors.Is(err, ErrTopicNotRegistered) {
		t.Fatalf("expected ErrTopicNotRegistered, got %q", err.Error())
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
			wantErr: ErrInvalidDocumentUploadInput,
		},
		"missing recipient": {
			input:   DocumentUploadInput{Document: "policy.pdf"},
			wantErr: ErrDocumentUploadMissingRecipient,
		},
		"missing document": {
			input:   DocumentUploadInput{Recipient: "user@example.com"},
			wantErr: ErrDocumentUploadMissingDocument,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mail := mockemail.NewClient()

			err := NewProducer(mail).NotifyTopic(context.Background(), TopicDocumentUpload, tc.input)
			if err == nil {
				t.Fatalf("expected error for %s", name)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if !mail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent, got %d log(s)", len(mail.SendLogs()))
			}
		})
	}
}

func TestNotifyRequiresRecipient(t *testing.T) {
	mail := mockemail.NewClient()

	err := NewProducer(mail).Notify(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}
	if !errors.Is(err, ErrMissingRecipients) {
		t.Fatalf("expected ErrMissingRecipients, got %v", err)
	}
	if !mail.SendLogs().IsEmpty() {
		t.Fatal("expected no email to be sent")
	}
}
