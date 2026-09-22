// Command notifier demonstrates sending a document-upload notification
// through notifier.Producer using the in-memory mock email client. See
// ../../README.md for the challenge this module implements.
package main

import (
	"context"
	"log"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	mail := mockemail.NewClient()
	producer := notifier.NewProducer(mail)
	if err := producer.NotifyTopic(context.Background(), notifier.TopicDocumentUpload, notifier.DocumentUploadInput{
		Recipient: "user@example.com",
		Document:  "policy.pdf",
	}); err != nil {
		return err
	}
	log.Printf("sent %d notification(s)", len(mail.SendLogs()))
	return nil
}
