package mail

import (
	"encoding/json"
	"reflect"
	"testing"
)

// These tests only pin down the behavior of the vendored stand-in that
// sendgrid.Client relies on; the stand-in itself is not modified.

func TestNewSendClient(t *testing.T) {

	t.Parallel()

	c := NewSendClient("key-123")
	if c.apiKey != "key-123" {
		t.Fatalf("expected apiKey %q, got %q", "key-123", c.apiKey)
	}
	if err := c.Send(NewV3Mail()); err != nil {
		t.Fatalf("expected Send to succeed, got %v", err)
	}
}

func TestV3MailBuilder(t *testing.T) {

	t.Parallel()

	p := NewPersonalization().
		AddTos("a@bar.com").
		AddTos("b@bar.com", "c@bar.com").
		AddCCs("cc@bar.com").
		AddBCCs("bcc@bar.com")

	m := NewV3Mail().
		SetMessage(json.RawMessage(`{"foo":"bar"}`)).
		SetTemplateID("tpl-1").
		SetFromName("Foo Bar").
		SetFromAddress("foo@bar.com").
		AddPersonalization(p).
		AddPersonalization(NewPersonalization())

	if string(m.message) != `{"foo":"bar"}` {
		t.Fatalf("unexpected message %s", m.message)
	}
	if m.tplID != "tpl-1" || m.fromName != "Foo Bar" || m.fromAddress != "foo@bar.com" {
		t.Fatalf("unexpected sender/template fields: %+v", m)
	}
	if len(m.personalizations) != 2 {
		t.Fatalf("expected 2 personalizations, got %d", len(m.personalizations))
	}

	got := m.personalizations[0]
	if want := []string{"a@bar.com", "b@bar.com", "c@bar.com"}; !reflect.DeepEqual(got.tos, want) {
		t.Fatalf("expected tos %v, got %v", want, got.tos)
	}
	if want := []string{"cc@bar.com"}; !reflect.DeepEqual(got.ccs, want) {
		t.Fatalf("expected ccs %v, got %v", want, got.ccs)
	}
	if want := []string{"bcc@bar.com"}; !reflect.DeepEqual(got.bccs, want) {
		t.Fatalf("expected bccs %v, got %v", want, got.bccs)
	}
}

// TestAddPersonalizationCopies confirms AddPersonalization stores a copy,
// so mutating the builder afterwards doesn't change an already added
// personalization.
func TestAddPersonalizationCopies(t *testing.T) {

	t.Parallel()

	p := NewPersonalization().AddTos("a@bar.com")
	m := NewV3Mail().AddPersonalization(p)
	p.AddTos("b@bar.com")

	if got := m.personalizations[0].tos; !reflect.DeepEqual(got, []string{"a@bar.com"}) {
		t.Fatalf("expected stored tos to be unaffected, got %v", got)
	}
}
