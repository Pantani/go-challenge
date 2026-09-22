package main

import (
	"context"
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
}

func TestNotifyUnknownTopic(t *testing.T) {
	err := NewProducer(mockemail.NewClient()).NotifyTopic(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected unknown topic error")
	}
}

func TestNotifyDocumentUploadInvalidInput(t *testing.T) {
	tests := map[string]any{
		"wrong input type":  "not-a-document-upload-input",
		"missing recipient": DocumentUploadInput{Document: "policy.pdf"},
		"missing document":  DocumentUploadInput{Recipient: "user@example.com"},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			producer := NewProducer(mockemail.NewClient())
			if err := producer.NotifyTopic(context.Background(), TopicDocumentUpload, input); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestNotifyRequiresRecipient(t *testing.T) {
	err := NewProducer(mockemail.NewClient()).Notify(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}
}
