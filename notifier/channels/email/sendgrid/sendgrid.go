// Package sendgrid is a local email provider stand-in. It validates requests
// and configuration but performs no network delivery.
package sendgrid

import (
	"errors"
	"fmt"
	"strings"

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

// Send validates a request. This stand-in performs no network delivery.
func (c *Client) Send(to []string, tpl email.TplID, _ map[string]any) error {
	if strings.TrimSpace(c.config.APIKey) == "" {
		return fmt.Errorf("sendgrid send: %w", ErrMissingAPIKey)
	}
	if err := email.RequireRecipient(to); err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	if err := email.RequireTemplate(tpl); err != nil {
		return fmt.Errorf("sendgrid send: %w: %w", ErrMissingTemplate, err)
	}
	return nil
}
