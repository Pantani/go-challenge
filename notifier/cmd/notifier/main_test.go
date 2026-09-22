package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email/mockemail"
)

func TestRun(t *testing.T) {
	mail := mockemail.NewClient()

	if err := run(context.Background(), mail); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// One example per registered topic.
	if got := len(mail.SendLogs()); got != 3 {
		t.Fatalf("expected 3 send logs, got %d", got)
	}
}

func TestRunSendError(t *testing.T) {
	mail := mockemail.NewClient()
	sendErr := errors.New("provider down")
	mail.SetSendError(sendErr)

	err := run(context.Background(), mail)
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected %v, got %v", sendErr, err)
	}
	if !mail.SendLogs().IsEmpty() {
		t.Fatal("expected no sends to be recorded after a failure")
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mail := mockemail.NewClient()

	err := run(ctx, mail)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !mail.SendLogs().IsEmpty() {
		t.Fatal("expected no sends once the context is cancelled")
	}
}
