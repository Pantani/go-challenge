package sendgrid

import (
	"encoding/json"
	"errors"
	"reflect"
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

type clientSendCase struct {
	to        []string
	cc        []string
	clientErr error
	wantErr   error
	wantCalls int
}

func assertGeneratedMail(t *testing.T, got *mail.V3Mail, to, cc []string, msg json.RawMessage, tpl email.TplID) {
	t.Helper()
	want := mail.NewV3Mail().
		SetMessage(msg).
		SetTemplateID(string(tpl)).
		SetFromName("Foo Bar").
		SetFromAddress("foo@bar.com").
		AddPersonalization(mail.NewPersonalization().AddTos(to...).AddCCs(cc...))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("generated mail: got %+v, want %+v", got, want)
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

	testCases := map[string]clientSendCase{
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
		"pass recipient prefixes remain distinct": {
			to:        []string{"foo@bar.com", "foo@bar.com.example"},
			cc:        []string{"cc@bar.com", "cc@bar.com.example"},
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
			runClientSendCase(t, tc)
		})
	}
}

func runClientSendCase(t *testing.T, tc clientSendCase) {
	t.Helper()
	fake := &fakeMailClient{err: tc.clientErr}
	c := &Client{client: fake, fromName: "Foo Bar", fromAddress: "foo@bar.com"}
	msg := json.RawMessage(`{"foo":"bar"}`)
	tpl := email.TplAddPolicyVehicle
	var err error
	if tc.cc == nil {
		err = c.Send(tc.to, msg, tpl)
	} else {
		tpl = email.TplAddPolicyCoverage
		err = c.SendWithCC(tc.to, tc.cc, msg, tpl)
	}
	checkEqual(t, "error cause", errors.Is(err, tc.wantErr), true)
	checkEqual(t, "provider calls", fake.calls, tc.wantCalls)
	if tc.wantCalls == 0 {
		return
	}
	assertGeneratedMail(t, fake.last, tc.to, tc.cc, msg, tpl)
}

func checkEqual[T any](t *testing.T, label string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", label, got, want)
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
