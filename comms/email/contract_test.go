package email_test

import (
	"encoding/json"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid"
)

// TestMailProviderContract runs the same behavioral expectations against
// every email.MailProvider implementation in this module, so the mock the
// handler tests rely on can't drift from the real provider: both must
// accept every template, with and without cc recipients.
func TestMailProviderContract(t *testing.T) {

	t.Parallel()

	providers := map[string]func() email.MailProvider{
		"sendgrid": func() email.MailProvider {
			return sendgrid.NewSvc(sendgrid.Config{APIKey: "key", FromName: "Foo Bar", FromAddress: "foo@bar.com"})
		},
		"mockemail": func() email.MailProvider { return mockemail.NewClient() },
	}

	tpls := []email.TplID{
		email.TplAddPolicyVehicle,
		email.TplAddPolicyDriver,
		email.TplAddPolicyAddress,
		email.TplAddPolicyCoverage,
	}

	for name, newProvider := range providers {
		t.Run(name, func(t *testing.T) {

			t.Parallel()

			for _, tpl := range tpls {
				p := newProvider()
				msg := json.RawMessage(`{"foo":"bar"}`)

				if err := p.Send([]string{"to@bar.com"}, msg, tpl); err != nil {
					t.Fatalf("Send(%v): unexpected error: %v", tpl, err)
				}
				if err := p.SendWithCC([]string{"to@bar.com"}, []string{"cc@bar.com"}, msg, tpl); err != nil {
					t.Fatalf("SendWithCC(%v): unexpected error: %v", tpl, err)
				}
				if err := p.SendWithCC([]string{"to@bar.com"}, nil, msg, tpl); err != nil {
					t.Fatalf("SendWithCC(%v) with no cc: unexpected error: %v", tpl, err)
				}
			}
		})
	}
}
