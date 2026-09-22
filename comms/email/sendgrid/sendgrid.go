// Package sendgrid implements email.MailProvider using a local SendGrid-shaped
// stand-in. It does not deliver email over the network.
package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"

	// This is a stand-in for the 3rd party sendgrid library
	// In a real project this would be vendored as github.com/sendgrid/sendgrid-go
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid/mail"
)

// ErrNoRecipients is returned when a send is attempted without any To
// recipient.
var ErrNoRecipients = errors.New("sendgrid: at least one To recipient is required")

// Config holds the sender identity and credentials used to build a Client.
type Config struct {
	APIKey      string
	FromAddress string
	FromName    string
}

// mailClient is the subset of *mail.Client's behavior Client depends on. It
// exists so tests can substitute a fake instead of the real send call.
type mailClient interface {
	Send(v3mail *mail.V3Mail) error
}

// NewSvc builds a local stand-in client with the configured sender identity.
func NewSvc(cfg Config) *Client {
	return &Client{
		client:      mail.NewSendClient(cfg.APIKey),
		fromAddress: cfg.FromAddress,
		fromName:    cfg.FromName,
	}
}

// Client constructs mail for the bundled, non-networking stand-in.
type Client struct {
	client      mailClient
	fromAddress string
	fromName    string
}

var _ email.MailProvider = (*Client)(nil)

// Send delivers message, rendered with tpl, to the given to recipients.
func (c *Client) Send(to []string, message json.RawMessage, tpl email.TplID) error {
	return c.SendContext(context.Background(), to, message, tpl)
}

// SendWithCC behaves like Send but additionally copies the given cc
// recipients on the message.
func (c *Client) SendWithCC(to, cc []string, message json.RawMessage, tpl email.TplID) error {
	return c.SendWithCCContext(context.Background(), to, cc, message, tpl)
}

// SendContext checks cancellation before constructing the local stand-in mail.
func (c *Client) SendContext(ctx context.Context, to []string, message json.RawMessage, tpl email.TplID) error {
	return c.SendWithCCContext(ctx, to, nil, message, tpl)
}

// SendWithCCContext checks cancellation before the synchronous stand-in call.
// The bundled stand-in performs no network delivery.
func (c *Client) SendWithCCContext(ctx context.Context, to, cc []string, message json.RawMessage, tpl email.TplID) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("sendgrid: send: %w", err)
	}
	return c.send(to, cc, message, tpl)
}

// send builds the v3 mail message for to/cc and dispatches it through the
// underlying mail client. It refuses to build a message with no To
// recipient rather than hand the provider an undeliverable request.
func (c *Client) send(to, cc []string, message json.RawMessage, tpl email.TplID) error {

	if len(to) == 0 {
		return ErrNoRecipients
	}

	// create personalization
	p := mail.NewPersonalization().
		AddTos(to...).
		AddCCs(cc...)

	// create the email
	m := mail.NewV3Mail().
		SetMessage(message).
		SetTemplateID(string(tpl)).
		SetFromName(c.fromName).
		SetFromAddress(c.fromAddress).
		AddPersonalization(p)

	// send the email
	if err := c.client.Send(m); err != nil {
		return fmt.Errorf("sendgrid: send failed: %w", err)
	}

	return nil
}
