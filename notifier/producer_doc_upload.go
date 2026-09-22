package notifier

import (
	"context"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

const TopicDocumentUpload = "document-upload"

type DocumentUploadInput struct {
	Recipient string
	Document  string
}

type docUploadTopicBuilder struct{}

func (docUploadTopicBuilder) Topic() string { return TopicDocumentUpload }

func (docUploadTopicBuilder) BuildRequest(_ context.Context, input any) (Request, error) {

	typedInput, ok := input.(DocumentUploadInput)
	if !ok {
		return Request{}, ErrInvalidDocumentUploadInput
	}
	if typedInput.Recipient == "" {
		return Request{}, ErrDocumentUploadMissingRecipient
	}
	if typedInput.Document == "" {
		return Request{}, ErrDocumentUploadMissingDocument
	}
	return Request{
		Topic:      TopicDocumentUpload,
		Recipients: []string{typedInput.Recipient},
		Template:   email.TplDocumentUpload,
		Vars:       map[string]any{"document": typedInput.Document},
	}, nil
}
