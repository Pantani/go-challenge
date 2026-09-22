package mockemail_test

import (
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestClientRecordsAndValidatesSend(t *testing.T) {
	client := mockemail.NewClient()
	if err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, map[string]any{"code": "123"}); err != nil {
		t.Fatal(err)
	}
	if got := client.SendLogs(); len(got) != 1 || got[0].To != "user@example.com" {
		t.Fatalf("logs = %#v", got)
	}
	if err := client.Send(nil, email.TplOTPLogin, nil); !errors.Is(err, email.ErrMissingRecipient) {
		t.Fatalf("Send() error = %v", err)
	}
}
