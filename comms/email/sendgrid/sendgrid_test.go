package sendgrid

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid/mail"
)

// fakeMailClient is a mailClient double that captures the last message it
// was asked to send and returns a configured error, letting tests exercise
// Client's request-building and error propagation without a real
// (always-succeeding) sendgrid stand-in.
type fakeMailClient struct {
	err  error
	last *mail.V3Mail
}

func (f *fakeMailClient) Send(v3mail *mail.V3Mail) error {
	f.last = v3mail
	return f.err
}

// assertPersonalizationHas fails the test unless fake's last captured
// message has exactly want under field ("tos" or "ccs"). mail.V3Mail is a
// vendored stand-in ("please do not modify") with unexported fields and no
// getters, so this reads them the same way %+v does: through fmt's
// built-in struct formatting, not unsafe or reflection of our own.
func assertPersonalizationHas(t *testing.T, fake *fakeMailClient, field, want string) {
	t.Helper()

	dump := fmt.Sprintf("%+v", fake.last)
	if !strings.Contains(dump, field+":["+want+"]") {
		t.Fatalf("expected personalization %s to be [%s], got: %s", field, want, dump)
	}
}

func TestNewSvc(t *testing.T) {

	t.Parallel()

	svc := NewSvc(Config{
		APIKey:      "key",
		FromName:    "Foo Bar",
		FromAddress: "foo@bar.com",
	})

	if svc.fromName != "Foo Bar" {
		t.Fatalf("expected fromName %q but got %q", "Foo Bar", svc.fromName)
	}

	if svc.fromAddress != "foo@bar.com" {
		t.Fatalf("expected fromAddress %q but got %q", "foo@bar.com", svc.fromAddress)
	}

	if err := svc.Send([]string{"foo@bar.com"}, json.RawMessage(`{}`), email.TplAddPolicyVehicle); err != nil {
		t.Fatalf("expected the real stand-in client to succeed, got: %v", err)
	}
}

func TestClientSend(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		clientErr error
	}{
		"pass":            {},
		"fail send error": {clientErr: errors.New("send failed")},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			fake := &fakeMailClient{err: tc.clientErr}
			c := &Client{client: fake, fromName: "Foo Bar", fromAddress: "foo@bar.com"}

			err := c.Send([]string{"foo@bar.com"}, json.RawMessage(`{"foo":"bar"}`), email.TplAddPolicyVehicle)

			if err != tc.clientErr {
				t.Fatalf("expected error %v but got %v", tc.clientErr, err)
			}

			assertPersonalizationHas(t, fake, "tos", "foo@bar.com")
			assertPersonalizationHas(t, fake, "ccs", "")
		})
	}
}

func TestClientSendWithCC(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		clientErr error
	}{
		"pass":            {},
		"fail send error": {clientErr: errors.New("send failed")},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			fake := &fakeMailClient{err: tc.clientErr}
			c := &Client{client: fake, fromName: "Foo Bar", fromAddress: "foo@bar.com"}

			err := c.SendWithCC([]string{"foo@bar.com"}, []string{"cc@bar.com"}, json.RawMessage(`{"foo":"bar"}`), email.TplAddPolicyCoverage)

			if err != tc.clientErr {
				t.Fatalf("expected error %v but got %v", tc.clientErr, err)
			}

			assertPersonalizationHas(t, fake, "tos", "foo@bar.com")
			assertPersonalizationHas(t, fake, "ccs", "cc@bar.com")
		})
	}
}
