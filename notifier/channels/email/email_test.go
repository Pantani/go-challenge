package email_test

import (
	"errors"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

func TestRequireRecipient(t *testing.T) {
	tests := map[string][]string{
		"single":   {"user@example.com"},
		"multiple": {"a@example.com", "b@example.com"},
	}

	for name, to := range tests {
		t.Run(name, func(t *testing.T) {
			if err := email.RequireRecipient(to); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestRequireRecipientMissing(t *testing.T) {
	tests := map[string][]string{
		"nil slice":         nil,
		"empty slice":       {},
		"blank first entry": {""},
		"blank later entry": {"a@example.com", ""},
		"blank middle":      {"a@example.com", "", "c@example.com"},
		"whitespace later":  {"a@example.com", " \t\n"},
	}

	for name, to := range tests {
		t.Run(name, func(t *testing.T) {
			err := email.RequireRecipient(to)
			if !errors.Is(err, email.ErrMissingRecipient) {
				t.Fatalf("expected ErrMissingRecipient for %s, got %v", name, err)
			}
		})
	}
}
