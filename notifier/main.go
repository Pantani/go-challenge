package main

import (
	"context"
	"log"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func main() {
	mail := mockemail.NewClient()
	producer := NewProducer(mail)
	if err := producer.NotifyTopic(context.Background(), TopicDocumentUpload, DocumentUploadInput{
		Recipient: "user@example.com",
		Document:  "policy.pdf",
	}); err != nil {
		log.Fatal(err)
	}
	log.Printf("sent %d notification(s)", len(mail.SendLogs()))
}
