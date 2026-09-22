package mockemail_test

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
)

func TestClientRecordsOneLogPerRecipient(t *testing.T) {
	t.Parallel()

	c := mockemail.NewClient()

	if !c.SendLogs().IsEmpty() {
		t.Fatalf("expected a fresh client to have no sends")
	}

	if err := c.Send([]string{"a@bar.com", "b@bar.com"}, json.RawMessage(`{"k":1}`), email.TplAddPolicyVehicle); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := c.SendWithCC([]string{"c@bar.com"}, []string{"cc@bar.com"}, json.RawMessage(`{"k":2}`), email.TplAddPolicyCoverage); err != nil {
		t.Fatalf("SendWithCC: %v", err)
	}

	logs := c.SendLogs()
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs but got %d", len(logs))
	}

	if got := logs[0].ExtractTo(); got != "a@bar.com" {
		t.Fatalf("expected first to a@bar.com but got %q", got)
	}
	if got := logs[0].ExtractCC(); got != nil {
		t.Fatalf("expected nil cc on plain Send but got %v", got)
	}
	if got := logs[1].ExtractTo(); got != "b@bar.com" {
		t.Fatalf("expected second to b@bar.com but got %q", got)
	}

	last := logs.Last()
	if last.ExtractTo() != "c@bar.com" {
		t.Fatalf("expected last to c@bar.com but got %q", last.ExtractTo())
	}
	if !reflect.DeepEqual(last.ExtractCC(), []string{"cc@bar.com"}) {
		t.Fatalf("expected last cc [cc@bar.com] but got %v", last.ExtractCC())
	}
	if string(last.ExtractMessage()) != `{"k":2}` {
		t.Fatalf("expected last message {\"k\":2} but got %s", last.ExtractMessage())
	}
	if last.ExtractTplID() != email.TplAddPolicyCoverage {
		t.Fatalf("expected last tpl %v but got %v", email.TplAddPolicyCoverage, last.ExtractTplID())
	}
}

func TestLastOnEmptyLogsIsNil(t *testing.T) {
	t.Parallel()

	if got := mockemail.NewClient().SendLogs().Last(); got != nil {
		t.Fatalf("expected nil Last on empty logs but got %v", got)
	}
	if got := (mockemail.SendLogs)(nil).Last(); got != nil {
		t.Fatalf("expected nil Last on nil logs but got %v", got)
	}
}

func TestFlushSendLogs(t *testing.T) {
	t.Parallel()

	c := mockemail.NewClient()
	_ = c.Send([]string{"a@bar.com"}, json.RawMessage(`{}`), email.TplAddPolicyDriver)

	c.FlushSendLogs()

	if !c.SendLogs().IsEmpty() {
		t.Fatalf("expected no logs after flush but got %v", c.SendLogs())
	}
}

// TestSendLogsIsASnapshot proves callers cannot reach into the client's
// state through the returned slice, and that the client does not alias the
// caller's cc/message slices either.
func TestSendLogsIsASnapshot(t *testing.T) {
	t.Parallel()

	c := mockemail.NewClient()
	cc := []string{"cc@bar.com"}
	msg := json.RawMessage(`{"k":1}`)
	_ = c.SendWithCC([]string{"a@bar.com"}, cc, msg, email.TplAddPolicyCoverage)

	// mutate the caller-owned inputs after the send
	cc[0] = "mutated@bar.com"
	msg[0] = '['

	first := c.SendLogs()
	if got := first.Last().ExtractCC(); !reflect.DeepEqual(got, []string{"cc@bar.com"}) {
		t.Fatalf("client aliased caller's cc slice: %v", got)
	}
	if got := first.Last().ExtractMessage(); string(got) != `{"k":1}` {
		t.Fatalf("client aliased caller's message: %s", got)
	}

	// mutate what came out of the accessors
	first.Last().ExtractCC()[0] = "mutated@bar.com"
	first.Last().ExtractMessage()[0] = '['
	first[0] = mockemail.SendLog{}

	second := c.SendLogs()
	if len(second) != 1 || second.Last().ExtractTo() != "a@bar.com" {
		t.Fatalf("mutating the returned slice altered the client: %v", second)
	}
	if got := second.Last().ExtractCC(); !reflect.DeepEqual(got, []string{"cc@bar.com"}) {
		t.Fatalf("ExtractCC returned aliased storage: %v", got)
	}
	if got := second.Last().ExtractMessage(); string(got) != `{"k":1}` {
		t.Fatalf("ExtractMessage returned aliased storage: %s", got)
	}
}

// TestClientIsGoroutineSafe hammers the client from many goroutines; run
// under -race this fails if any access is unsynchronised.
func TestClientIsGoroutineSafe(t *testing.T) {
	t.Parallel()

	const workers, perWorker = 16, 50

	c := mockemail.NewClient()

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWorker {
				_ = c.Send([]string{"a@bar.com"}, json.RawMessage(`{}`), email.TplAddPolicyAddress)
				_ = c.SendLogs().IsEmpty()
			}
		}()
	}
	wg.Wait()

	if got := len(c.SendLogs()); got != workers*perWorker {
		t.Fatalf("expected %d logs but got %d", workers*perWorker, got)
	}
}
