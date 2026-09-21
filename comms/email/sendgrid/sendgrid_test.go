package sendgrid

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid/mail"
)

// fakeMailClient is a mailClient double that returns a configured error,
// letting tests exercise Client's error propagation without a real
// (always-succeeding) sendgrid stand-in.
type fakeMailClient struct {
	err error
}

func (f *fakeMailClient) Send(*mail.V3Mail) error {
	return f.err
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

			c := &Client{client: &fakeMailClient{err: tc.clientErr}, fromName: "Foo Bar", fromAddress: "foo@bar.com"}

			err := c.Send([]string{"foo@bar.com"}, json.RawMessage(`{"foo":"bar"}`), email.TplAddPolicyVehicle)

			if err != tc.clientErr {
				t.Fatalf("expected error %v but got %v", tc.clientErr, err)
			}
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

			c := &Client{client: &fakeMailClient{err: tc.clientErr}, fromName: "Foo Bar", fromAddress: "foo@bar.com"}

			err := c.SendWithCC([]string{"foo@bar.com"}, []string{"cc@bar.com"}, json.RawMessage(`{"foo":"bar"}`), email.TplAddPolicyCoverage)

			if err != tc.clientErr {
				t.Fatalf("expected error %v but got %v", tc.clientErr, err)
			}
		})
	}
}
