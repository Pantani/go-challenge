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

	checkEqual(t, "fresh logs", c.SendLogs().IsEmpty(), true)

	if err := c.Send([]string{"a@bar.com", "b@bar.com"}, json.RawMessage(`{"k":1}`), email.TplAddPolicyVehicle); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := c.SendWithCC([]string{"c@bar.com"}, []string{"cc@bar.com"}, json.RawMessage(`{"k":2}`), email.TplAddPolicyCoverage); err != nil {
		t.Fatalf("SendWithCC: %v", err)
	}

	logs := c.SendLogs()
	checkEqual(t, "log count", len(logs), 3)
	checkEqual(t, "first recipient", logs[0].ExtractTo(), "a@bar.com")
	checkEqual(t, "first CC", logs[0].ExtractCC(), []string(nil))
	checkEqual(t, "second recipient", logs[1].ExtractTo(), "b@bar.com")

	last := logs.Last()
	checkEqual(t, "last recipient", last.ExtractTo(), "c@bar.com")
	checkEqual(t, "last CC", last.ExtractCC(), []string{"cc@bar.com"})
	checkEqual(t, "last message", string(last.ExtractMessage()), `{"k":2}`)
	checkEqual(t, "last template", last.ExtractTplID(), email.TplAddPolicyCoverage)
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
	checkEqual(t, "input CC snapshot", first.Last().ExtractCC(), []string{"cc@bar.com"})
	checkEqual(t, "input message snapshot", string(first.Last().ExtractMessage()), `{"k":1}`)

	// mutate what came out of the accessors
	first.Last().ExtractCC()[0] = "mutated@bar.com"
	first.Last().ExtractMessage()[0] = '['
	first[0] = mockemail.SendLog{}

	second := c.SendLogs()
	checkEqual(t, "snapshot count", len(second), 1)
	checkEqual(t, "snapshot recipient", second.Last().ExtractTo(), "a@bar.com")
	checkEqual(t, "accessor CC snapshot", second.Last().ExtractCC(), []string{"cc@bar.com"})
	checkEqual(t, "accessor message snapshot", string(second.Last().ExtractMessage()), `{"k":1}`)
}

func checkEqual[T any](t *testing.T, label string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", label, got, want)
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
