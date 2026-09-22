package mockemail_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

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

func TestClientConcurrentUse(t *testing.T) {
	const workers, sendsPerWorker = 8, 50
	client := mockemail.NewClient()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			to := []string{fmt.Sprintf("user%d@example.com", w)}
			for i := 0; i < sendsPerWorker; i++ {
				if err := client.Send(to, email.TplOTPLogin, nil); err != nil {
					t.Errorf("send: %v", err)
				}
				_ = client.SendLogs()
			}
		}(w)
	}
	wg.Wait()

	assertEqual(t, "log count", workers*sendsPerWorker, len(client.SendLogs()))
}
