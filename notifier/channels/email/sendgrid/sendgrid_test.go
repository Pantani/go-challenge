package sendgrid_test

import (
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/sendgrid"
)

var validConfig = sendgrid.Config{
	APIKey:      "key",
	FromAddress: "from@example.com",
	FromName:    "Glovebox",
}

func TestClientSend(t *testing.T) {
	client := sendgrid.NewSvc(validConfig)

	err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, map[string]any{"otpCode": "123456"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClientSendInvalid(t *testing.T) {
	tests := map[string]struct {
		config  sendgrid.Config
		to      []string
		tpl     email.TplID
		wantErr error
	}{
		"missing api key": {
			config:  sendgrid.Config{FromAddress: "from@example.com"},
			to:      []string{"user@example.com"},
			tpl:     email.TplOTPLogin,
			wantErr: sendgrid.ErrMissingAPIKey,
		},
		"no recipients": {
			config:  validConfig,
			to:      nil,
			tpl:     email.TplOTPLogin,
			wantErr: email.ErrMissingRecipient,
		},
		"blank later recipient": {
			config:  validConfig,
			to:      []string{"user@example.com", ""},
			tpl:     email.TplOTPLogin,
			wantErr: email.ErrMissingRecipient,
		},
		"missing template": {
			config:  validConfig,
			to:      []string{"user@example.com"},
			tpl:     "",
			wantErr: sendgrid.ErrMissingTemplate,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := sendgrid.NewSvc(tc.config).Send(tc.to, tc.tpl, nil)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}
