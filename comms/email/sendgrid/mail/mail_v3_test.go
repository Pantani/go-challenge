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

	checkEqual(t, "message", string(m.message), `{"foo":"bar"}`)
	checkEqual(t, "template", m.tplID, "tpl-1")
	checkEqual(t, "sender name", m.fromName, "Foo Bar")
	checkEqual(t, "sender address", m.fromAddress, "foo@bar.com")
	checkEqual(t, "personalization count", len(m.personalizations), 2)

	got := m.personalizations[0]
	checkEqual(t, "to", got.tos, []string{"a@bar.com", "b@bar.com", "c@bar.com"})
	checkEqual(t, "cc", got.ccs, []string{"cc@bar.com"})
	checkEqual(t, "bcc", got.bccs, []string{"bcc@bar.com"})
}

func checkEqual[T any](t *testing.T, label string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", label, got, want)
	}
}

// TestAddPersonalizationSnapshotsRecipients confirms AddPersonalization
// stores the personalization by value, so recipients added to the builder
// afterwards don't show up in an already added personalization. The copy
// is shallow (the recipient slices still share a backing array), so this
// only pins down the append behavior sendgrid.Client relies on, not deep
// isolation, which the vendored stand-in doesn't provide.
func TestAddPersonalizationSnapshotsRecipients(t *testing.T) {

	t.Parallel()

	p := NewPersonalization().AddTos("a@bar.com")
	m := NewV3Mail().AddPersonalization(p)
	p.AddTos("b@bar.com")

	if got := m.personalizations[0].tos; !reflect.DeepEqual(got, []string{"a@bar.com"}) {
		t.Fatalf("expected stored tos to be unaffected, got %v", got)
	}
}
