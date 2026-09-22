package mockemail

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

func TestNewClientStartsEmpty(t *testing.T) {

	t.Parallel()

	c := NewClient()
	if !c.SendLogs().IsEmpty() {
		t.Fatalf("expected no send logs, got %d", len(c.SendLogs()))
	}
}

func TestSendRecordsOneLogPerRecipient(t *testing.T) {

	t.Parallel()

	c := NewClient()
	msg := json.RawMessage(`{"foo":"bar"}`)

	if err := c.Send([]string{"a@bar.com", "b@bar.com"}, msg, email.TplAddPolicyVehicle); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := c.SendLogs()
	if len(logs) != 2 {
		t.Fatalf("expected 2 send logs, got %d", len(logs))
	}

	for i, want := range []string{"a@bar.com", "b@bar.com"} {
		if got := logs[i].ExtractTo(); got != want {
			t.Fatalf("log %d: expected to %q, got %q", i, want, got)
		}
		if got := logs[i].ExtractCC(); got != nil {
			t.Fatalf("log %d: expected no cc, got %v", i, got)
		}
		if got := logs[i].ExtractTplID(); got != email.TplAddPolicyVehicle {
			t.Fatalf("log %d: expected tpl %v, got %v", i, email.TplAddPolicyVehicle, got)
		}
		if got := logs[i].ExtractMessage(); string(got) != string(msg) {
			t.Fatalf("log %d: expected message %s, got %s", i, msg, got)
		}
	}

	if got := logs.Last().ExtractTo(); got != "b@bar.com" {
		t.Fatalf("expected last recipient b@bar.com, got %q", got)
	}
}

func TestSendWithCCRecordsCCOnEveryLog(t *testing.T) {

	t.Parallel()

	c := NewClient()
	cc := []string{"cc1@bar.com", "cc2@bar.com"}

	if err := c.SendWithCC([]string{"a@bar.com", "b@bar.com"}, cc, json.RawMessage(`{}`), email.TplAddPolicyCoverage); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, l := range c.SendLogs() {
		if got := l.ExtractCC(); !reflect.DeepEqual(got, cc) {
			t.Fatalf("log %d: expected cc %v, got %v", i, cc, got)
		}
		if got := l.ExtractTplID(); got != email.TplAddPolicyCoverage {
			t.Fatalf("log %d: expected tpl %v, got %v", i, email.TplAddPolicyCoverage, got)
		}
	}
}

func TestSendWithNoRecipientsRecordsNothing(t *testing.T) {

	t.Parallel()

	c := NewClient()
	if err := c.Send(nil, json.RawMessage(`{}`), email.TplAddPolicyDriver); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.SendLogs().IsEmpty() {
		t.Fatalf("expected no send logs, got %d", len(c.SendLogs()))
	}
}

func TestFlushSendLogs(t *testing.T) {

	t.Parallel()

	c := NewClient()
	if err := c.Send([]string{"a@bar.com"}, json.RawMessage(`{}`), email.TplAddPolicyAddress); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SendLogs().IsEmpty() {
		t.Fatal("expected a send log before flushing")
	}

	c.FlushSendLogs()

	if !c.SendLogs().IsEmpty() {
		t.Fatalf("expected no send logs after flushing, got %d", len(c.SendLogs()))
	}
}
