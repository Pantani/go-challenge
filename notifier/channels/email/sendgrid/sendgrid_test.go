package sendgrid_test

import (
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/sendgrid"
)

func TestClientSend(t *testing.T) {
	client := sendgrid.NewSvc(sendgrid.Config{
		APIKey:      "key",
		FromAddress: "from@example.com",
		FromName:    "Glovebox",
	})

	err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, map[string]any{"otpCode": "123456"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
