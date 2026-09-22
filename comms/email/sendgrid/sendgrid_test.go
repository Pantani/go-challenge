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
	err   error
	last  *mail.V3Mail
	calls int
}

func (f *fakeMailClient) Send(v3mail *mail.V3Mail) error {
	f.calls++
	f.last = v3mail
	return f.err
}

// assertMailHas fails the test unless fake's last captured message contains
// want under field (e.g. "tos", "ccs", "tplID"). mail.V3Mail is a vendored
// stand-in ("please do not modify") with unexported fields and no getters,
// so this reads them the same way %+v does: through fmt's built-in struct
// formatting, not unsafe or reflection of our own.
func assertMailHas(t *testing.T, fake *fakeMailClient, field, want string) {
	t.Helper()

	dump := fmt.Sprintf("%+v", fake.last)
	if !strings.Contains(dump, field+":"+want) {
		t.Fatalf("expected %s to be %s, got: %s", field, want, dump)
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

	sendErr := errors.New("send failed")

	testCases := map[string]struct {
		to        []string
		cc        []string
		clientErr error
		wantErr   error
		wantCalls int
	}{
		"pass": {
			to:        []string{"foo@bar.com"},
			wantCalls: 1,
		},
		"pass with cc": {
			to:        []string{"foo@bar.com"},
			cc:        []string{"cc@bar.com", "cc2@bar.com"},
			wantCalls: 1,
		},
		"pass multiple to": {
			to:        []string{"foo@bar.com", "baz@bar.com"},
			wantCalls: 1,
		},
		"fail send error": {
			to:        []string{"foo@bar.com"},
			clientErr: sendErr,
			wantErr:   sendErr,
			wantCalls: 1,
		},
		"fail nil to": {
			wantErr:   ErrNoRecipients,
			wantCalls: 0,
		},
		"fail empty to": {
			to:        []string{},
			cc:        []string{"cc@bar.com"},
			wantErr:   ErrNoRecipients,
			wantCalls: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeMailClient{err: tc.clientErr}
			c := &Client{client: fake, fromName: "Foo Bar", fromAddress: "foo@bar.com"}

			msg := json.RawMessage(`{"foo":"bar"}`)

			var err error
			tpl := email.TplAddPolicyVehicle
			if tc.cc == nil {
				err = c.Send(tc.to, msg, tpl)
			} else {
				tpl = email.TplAddPolicyCoverage
				err = c.SendWithCC(tc.to, tc.cc, msg, tpl)
			}

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v but got %v", tc.wantErr, err)
			}

			if fake.calls != tc.wantCalls {
				t.Fatalf("expected %d provider calls but got %d", tc.wantCalls, fake.calls)
			}

			if tc.wantCalls == 0 {
				return
			}

			assertMailHas(t, fake, "tos", fmt.Sprintf("%v", tc.to))
			assertMailHas(t, fake, "ccs", fmt.Sprintf("%v", tc.cc))
			assertMailHas(t, fake, "fromName", "Foo Bar")
			assertMailHas(t, fake, "fromAddress", "foo@bar.com")
			assertMailHas(t, fake, "tplID", string(tpl))
			assertMailHas(t, fake, "message", fmt.Sprintf("%v", []byte(msg)))
		})
	}
}

// TestClientSendWrapsProviderError pins that the provider's failure is
// wrapped with context but still reachable through errors.Is.
func TestClientSendWrapsProviderError(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("boom")
	c := &Client{client: &fakeMailClient{err: sendErr}}

	err := c.Send([]string{"foo@bar.com"}, json.RawMessage(`{}`), email.TplAddPolicyDriver)

	if !errors.Is(err, sendErr) {
		t.Fatalf("expected wrapped %v but got %v", sendErr, err)
	}
	if got := err.Error(); got != "sendgrid: send failed: boom" {
		t.Fatalf("unexpected error text %q", got)
	}
}
