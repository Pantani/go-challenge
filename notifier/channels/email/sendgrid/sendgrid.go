// Package sendgrid is the SendGrid-backed email.MailProvider. The outbound
// call itself is stubbed out in this challenge; request validation is real.
package sendgrid

import (
	"errors"
	"fmt"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// Sentinel errors returned (wrapped) by Client.Send.
var (
	ErrMissingAPIKey   = errors.New("sendgrid: api key is required")
	ErrMissingTemplate = errors.New("sendgrid: template is required")
)

type Config struct {
	APIKey      string
	FromAddress string
	FromName    string
}

type Client struct {
	config Config
}

func NewSvc(config Config) *Client { return &Client{config: config} }

// Send validates the request the same way the real provider would before the
// network call: an API key must be configured, every recipient must be
// non-blank, and a template must be named.
func (c *Client) Send(to []string, tpl email.TplID, _ map[string]any) error {
	if c.config.APIKey == "" {
		return ErrMissingAPIKey
	}
	if err := email.RequireRecipient(to); err != nil {
		return fmt.Errorf("sendgrid: %w", err)
	}
	if tpl == "" {
		return ErrMissingTemplate
	}
	return nil
}
