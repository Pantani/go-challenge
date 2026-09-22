package mockemail_test

import (
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
	if !mockemail.NewClient().SendLogs().IsEmpty() {
		t.Fatal("expected no send logs before any send")
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
