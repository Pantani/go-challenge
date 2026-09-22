package notifier

import (
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// TopicDocumentUpload is the notification topic for a newly uploaded document.
const TopicDocumentUpload = "document-upload"

// DocumentUploadInput is the typed input for the document-upload topic.
// Both fields are required.
type DocumentUploadInput struct {
	Recipient string
	Document  string
}

var docUploadTopicBuilder = topicBuilder[DocumentUploadInput]{
	topic:           TopicDocumentUpload,
	tpl:             email.TplDocumentUpload,
	errInvalidInput: ErrInvalidDocumentUploadInput,
	build:           buildDocumentUpload,
}

func buildDocumentUpload(in DocumentUploadInput) (string, map[string]any, error) {
	if strings.TrimSpace(in.Recipient) == "" {
		return "", nil, ErrDocumentUploadMissingRecipient
	}
	if strings.TrimSpace(in.Document) == "" {
		return "", nil, ErrDocumentUploadMissingDocument
	}
	return in.Recipient, map[string]any{"document": in.Document}, nil
}
