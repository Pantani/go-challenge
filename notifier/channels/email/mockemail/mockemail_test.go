package mockemail_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

type cancelAfterCheckContext struct {
	context.Context
	cancel context.CancelFunc
}

func (c cancelAfterCheckContext) Err() error {
	err := c.Context.Err()
	c.cancel()
	return err
}

func TestClientSendContextRechecksCancellationBeforeRecording(t *testing.T) {
	for name, sendErr := range map[string]error{"success": nil, "configured failure": errors.New("configured failure")} {
		t.Run(name, func(t *testing.T) { checkCancellationBeforeRecording(t, sendErr) })
	}
}

func checkCancellationBeforeRecording(t *testing.T, sendErr error) {
	t.Helper()
	client := mockemail.NewClient()
	client.SetSendError(sendErr)
	provider, ok := any(client).(notifier.ContextMailProvider)
	if !ok {
		t.Fatal("mock lacks optional context interface")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checked := cancelAfterCheckContext{Context: ctx, cancel: cancel}
	err := provider.SendContext(checked, []string{"u@example.com"}, email.TplOTPLogin, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendContext() = %v; want cancellation before configured failure", err)
	}
	assertEqual(t, "log count", 0, len(client.SendLogs()))
}

func assertEqual(t *testing.T, label string, want, got any) {
	t.Helper()
	if want != got {
		t.Fatalf("expected %s %v, got %v", label, want, got)
	}
}

func TestClientSendLogsEmptyInitially(t *testing.T) {
	logs := mockemail.NewClient().SendLogs()
	if !logs.IsEmpty() {
		t.Fatal("expected no send logs before any send")
	}
	if logs.Last() != nil {
		t.Fatal("expected Last() to be nil on an empty log")
	}
}

func TestClientSendRecordsLog(t *testing.T) {
	client := mockemail.NewClient()

	err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, map[string]any{"otpCode": "123456"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	logs := client.SendLogs()
	if logs.IsEmpty() {
		t.Fatal("expected a send log")
	}
	last := logs.Last()
	assertEqual(t, "recipient", "user@example.com", last.To)
	assertEqual(t, "template", email.TplOTPLogin, last.Tpl)
	assertEqual(t, "otp code", "123456", last.Vars["otpCode"])
}

func TestClientSendRecordsOneLogPerRecipient(t *testing.T) {
	client := mockemail.NewClient()

	err := client.Send([]string{"a@example.com", "b@example.com"}, email.TplDocumentUpload, nil)
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	logs := client.SendLogs()
	assertEqual(t, "log count", 2, len(logs))
	assertEqual(t, "first recipient", "a@example.com", logs[0].To)
	assertEqual(t, "last recipient", "b@example.com", logs.Last().To)
}

func TestClientSendRejectsMissingRecipient(t *testing.T) {
	tests := map[string][]string{
		"empty":       {},
		"blank":       {""},
		"blank later": {"a@example.com", ""},
	}

	for name, to := range tests {
		t.Run(name, func(t *testing.T) {
			client := mockemail.NewClient()

			err := client.Send(to, email.TplOTPLogin, nil)
			if !errors.Is(err, email.ErrMissingRecipient) {
				t.Fatalf("expected ErrMissingRecipient, got %v", err)
			}
			if !client.SendLogs().IsEmpty() {
				t.Fatal("expected a rejected send not to be recorded")
			}
		})
	}
}

func TestClientSetSendError(t *testing.T) {
	client := mockemail.NewClient()
	sendErr := errors.New("boom")
	client.SetSendError(sendErr)

	err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, nil)
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected %v, got %v", sendErr, err)
	}
	if !client.SendLogs().IsEmpty() {
		t.Fatal("expected a failed send not to be recorded")
	}

	client.SetSendError(nil)
	if err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, nil); err != nil {
		t.Fatalf("expected send to succeed after clearing the error, got %v", err)
	}
	assertEqual(t, "log count", 1, len(client.SendLogs()))
}

func TestClientFlushSendLogs(t *testing.T) {
	client := mockemail.NewClient()
	if err := client.Send([]string{"user@example.com"}, email.TplOTPLogin, nil); err != nil {
		t.Fatalf("send: %v", err)
	}

	client.FlushSendLogs()
	if !client.SendLogs().IsEmpty() {
		t.Fatal("expected send logs to be empty after flush")
	}
}

// SendLogs must hand out a copy: mutating or holding on to the result must
// not observe later sends or flushes, and vice versa.
func TestClientSendLogsIsSnapshot(t *testing.T) {
	client := mockemail.NewClient()
	if err := client.Send([]string{"first@example.com"}, email.TplOTPLogin, nil); err != nil {
		t.Fatalf("send: %v", err)
	}

	snapshot := client.SendLogs()
	snapshot[0].To = "mutated@example.com"
	assertEqual(t, "recipient after external mutation", "first@example.com", client.SendLogs()[0].To)

	if err := client.Send([]string{"second@example.com"}, email.TplOTPLogin, nil); err != nil {
		t.Fatalf("send: %v", err)
	}
	client.FlushSendLogs()
	assertEqual(t, "snapshot length after later send and flush", 1, len(snapshot))
}

func TestClientSendLogsClonesVars(t *testing.T) {
	client := mockemail.NewClient()
	vars := map[string]any{"otpCode": "1234"}
	if err := client.Send([]string{"first@example.com"}, email.TplOTPLogin, vars); err != nil {
		t.Fatalf("send: %v", err)
	}

	vars["otpCode"] = "caller-mutated"
	assertEqual(t, "var after caller mutation", "1234", client.SendLogs()[0].Vars["otpCode"])

	client.SendLogs()[0].Vars["otpCode"] = "snapshot-mutated"
	assertEqual(t, "var after snapshot mutation", "1234", client.SendLogs()[0].Vars["otpCode"])
}

func TestClientSendLogsNestedValuesRemainCallerOwned(t *testing.T) {
	client := mockemail.NewClient()
	nested := map[string]string{"name": "before"}
	vars := map[string]any{"nested": nested, "scalar": "original"}
	if err := client.Send([]string{"a@example.com", "b@example.com"}, email.TplOTPLogin, vars); err != nil {
		t.Fatal(err)
	}
	nested["name"] = "after"
	logs := client.SendLogs()
	assertEqual(t, "send count", 2, len(logs))
	got, ok := logs[0].Vars["nested"].(map[string]string)
	if !ok {
		t.Fatalf("nested type = %T", logs[0].Vars["nested"])
	}
	assertEqual(t, "nested value", "after", got["name"])
	logs[0].Vars["scalar"] = "snapshot change"
	assertEqual(t, "second recipient scalar", "original", logs[1].Vars["scalar"])
	assertEqual(t, "stored scalar", "original", client.SendLogs()[0].Vars["scalar"])
}

func TestClientSendRejectsBlankTemplateWithoutLog(t *testing.T) {
	client := mockemail.NewClient()
	err := client.Send([]string{"u@example.com"}, " \t", nil)
	if !errors.Is(err, email.ErrMissingTemplate) {
		t.Fatalf("Send() = %v", err)
	}
	assertEqual(t, "send count", 0, len(client.SendLogs()))
}

func sendWorker(t *testing.T, client *mockemail.Client, worker, count int) {
	t.Helper()
	to := []string{fmt.Sprintf("user%d@example.com", worker)}
	for i := 0; i < count; i++ {
		if err := client.Send(to, email.TplOTPLogin, nil); err != nil {
			t.Errorf("send: %v", err)
		}
		_ = client.SendLogs()
	}
}

func startSendWorker(t *testing.T, wg *sync.WaitGroup, client *mockemail.Client, worker, count int) {
	t.Helper()
	wg.Add(1)
	go func() {
		defer wg.Done()
		sendWorker(t, client, worker, count)
	}()
}

func TestClientConcurrentUse(t *testing.T) {
	const workers, sendsPerWorker = 8, 50
	client := mockemail.NewClient()
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		startSendWorker(t, &wg, client, worker, sendsPerWorker)
	}
	wg.Wait()
	assertEqual(t, "log count", workers*sendsPerWorker, len(client.SendLogs()))
}
