package notifier_test

import (
	"context"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

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
	if last.To != "user@example.com" {
		t.Fatalf("expected recipient %q, got %q", "user@example.com", last.To)
	}
	if last.Tpl != email.TplDocumentUpload {
		t.Fatalf("expected template %q, got %q", email.TplDocumentUpload, last.Tpl)
	}
}

func TestNotifyUnknownTopic(t *testing.T) {
	err := notifier.NewProducer(mockemail.NewClient()).NotifyTopic(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected unknown topic error")
	}
}

func TestNotifyDocumentUploadInvalidInput(t *testing.T) {
	tests := map[string]any{
		"wrong input type":  "not-a-document-upload-input",
		"missing recipient": notifier.DocumentUploadInput{Document: "policy.pdf"},
		"missing document":  notifier.DocumentUploadInput{Recipient: "user@example.com"},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			producer := notifier.NewProducer(mockemail.NewClient())
			if err := producer.NotifyTopic(context.Background(), notifier.TopicDocumentUpload, input); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestNotifyRequiresRecipient(t *testing.T) {
	err := notifier.NewProducer(mockemail.NewClient()).Notify(context.Background(), notifier.Request{})
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}
}
