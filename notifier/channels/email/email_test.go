package email_test

import (
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

func TestRequireRecipient(t *testing.T) {
	if err := email.RequireRecipient([]string{"user@example.com"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequireRecipientMissing(t *testing.T) {
	tests := map[string][]string{
		"empty slice":       {},
		"blank first entry": {""},
	}

	for name, to := range tests {
		t.Run(name, func(t *testing.T) {
			if err := email.RequireRecipient(to); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}
