package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/sendgrid"
)

var (
	_ email.MailProvider           = (*mockemail.Client)(nil)
	_ email.MailProvider           = (*sendgrid.Client)(nil)
	_ notifier.ContextMailProvider = (*mockemail.Client)(nil)
	_ notifier.ContextMailProvider = (*sendgrid.Client)(nil)
)

func TestProviderContextContract(t *testing.T) {
	for name, factory := range providerFactories() {
		t.Run(name, func(t *testing.T) { checkProviderContext(t, factory()) })
	}
}

func checkProviderContext(t *testing.T, provider email.MailProvider) {
	t.Helper()
	contextual, ok := provider.(notifier.ContextMailProvider)
	if !ok {
		t.Fatal("provider does not implement ContextMailProvider")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := contextual.SendContext(ctx, []string{"u@example.com"}, email.TplOTPLogin, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendContext() = %v", err)
	}
	if err := contextual.SendContext(context.Background(), []string{"u@example.com"}, email.TplOTPLogin, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMockCanceledSendHasNoLog(t *testing.T) {
	mail := mockemail.NewClient()
	provider, ok := any(mail).(notifier.ContextMailProvider)
	if !ok {
		t.Fatal("mock lacks optional context interface")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := provider.SendContext(ctx, nil, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("SendContext() = %v", err)
	}
	if len(mail.SendLogs()) != 0 {
		t.Fatal("canceled send was recorded")
	}
}

func TestSendGridCancellationPrecedesValidation(t *testing.T) {
	client := sendgrid.NewSvc(sendgrid.Config{})
	provider, ok := any(client).(notifier.ContextMailProvider)
	if !ok {
		t.Fatal("sendgrid lacks optional context interface")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := provider.SendContext(ctx, nil, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("SendContext() = %v; want cancellation before validation", err)
	}
}

func providerFactories() map[string]func() email.MailProvider {
	return map[string]func() email.MailProvider{
		"mock": func() email.MailProvider { return mockemail.NewClient() },
		"sendgrid": func() email.MailProvider {
			return sendgrid.NewSvc(sendgrid.Config{APIKey: " key "})
		},
	}
}

func TestProviderMinimumContract(t *testing.T) {
	for name, factory := range providerFactories() {
		t.Run(name, func(t *testing.T) { checkProviderValidation(t, factory) })
	}
}

func checkProviderValidation(t *testing.T, factory func() email.MailProvider) {
	t.Helper()
	tests := []struct {
		name string
		to   []string
		tpl  email.TplID
		want error
	}{
		{"nil recipients", nil, email.TplOTPLogin, email.ErrMissingRecipient},
		{"empty recipients", []string{}, email.TplOTPLogin, email.ErrMissingRecipient},
		{"blank first", []string{" \t\n"}, email.TplOTPLogin, email.ErrMissingRecipient},
		{"blank later", []string{"user@example.com", "\u2003"}, email.TplOTPLogin, email.ErrMissingRecipient},
		{"empty template", []string{"user@example.com"}, "", email.ErrMissingTemplate},
		{"blank template", []string{"user@example.com"}, " \t\n", email.ErrMissingTemplate},
		{"minimum only", []string{" local-recipient "}, " custom-template ", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := factory().Send(tc.to, tc.tpl, nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Send() = %v; want %v", err, tc.want)
			}
		})
	}
}

func TestSendGridRejectsBlankAPIKey(t *testing.T) {
	client := sendgrid.NewSvc(sendgrid.Config{APIKey: " \t\n"})
	err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, nil)
	if !errors.Is(err, sendgrid.ErrMissingAPIKey) {
		t.Fatalf("Send() = %v; want missing API key", err)
	}
}

func TestSendGridRetainsTemplateSentinel(t *testing.T) {
	client := sendgrid.NewSvc(sendgrid.Config{APIKey: "key"})
	err := client.Send([]string{"user@example.com"}, " \t", nil)
	if !errors.Is(err, sendgrid.ErrMissingTemplate) {
		t.Fatalf("Send() = %v; want provider template sentinel", err)
	}
}
