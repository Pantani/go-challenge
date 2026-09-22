// Command notifier demonstrates sending one notification per topic through
// notifier.Producer using the in-memory mock email client. See
// ../../README.md for the challenge this module implements.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func main() {
	mail := mockemail.NewClient()
	if err := run(context.Background(), mail); err != nil {
		log.Fatal(err)
	}
	log.Printf("sent %d notification(s)", len(mail.SendLogs()))
}

// run sends one example notification per registered topic through mail,
// stopping at the first failure.
func run(ctx context.Context, mail email.MailProvider) error {
	producer := notifier.NewProducer(mail)
	examples := []struct {
		topic string
		input any
	}{
		{notifier.TopicDocumentUpload, notifier.DocumentUploadInput{
			Recipient: "user@example.com",
			Document:  "policy.pdf",
		}},
		{notifier.TopicOTPLogin, notifier.OTPLoginInput{
			Recipient:      "user@example.com",
			OTPCode:        "123456",
			ExpirationMins: 5,
		}},
		{notifier.TopicPolicyRenewal, notifier.PolicyRenewalInput{
			Recipient:    "user@example.com",
			PolicyNumber: "POL-12345",
			RenewalDate:  time.Now().AddDate(0, 1, 0),
		}},
	}
	for _, ex := range examples {
		if err := producer.NotifyTopic(ctx, ex.topic, ex.input); err != nil {
			return err
		}
	}
	return nil
}
