// Package mockemail provides an in-memory email.MailProvider that records
// every delivery so tests can assert on what would have been sent.
package mockemail

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

// Client is a goroutine-safe in-memory email.MailProvider that records
// every send in SendLogs instead of delivering anything.
type Client struct {
	mu       sync.Mutex
	sendLogs SendLogs
}

var _ email.MailProvider = (*Client)(nil)

// NewClient returns a Client with no recorded sends.
func NewClient() *Client {
	return &Client{
		sendLogs: SendLogs{},
	}
}

// Send records a delivery of message to the given to recipients.
func (c *Client) Send(to []string, message json.RawMessage, tplID email.TplID) error {
	return c.SendContext(context.Background(), to, message, tplID)
}

// SendWithCC behaves like Send but additionally records the given cc
// recipients on the logged send.
func (c *Client) SendWithCC(to, cc []string, message json.RawMessage, tplID email.TplID) error {
	return c.SendWithCCContext(context.Background(), to, cc, message, tplID)
}

// SendContext checks cancellation before recording a delivery.
func (c *Client) SendContext(ctx context.Context, to []string, message json.RawMessage, tplID email.TplID) error {
	return c.SendWithCCContext(ctx, to, nil, message, tplID)
}

// SendWithCCContext checks cancellation before the synchronous recording operation.
func (c *Client) SendWithCCContext(ctx context.Context, to, cc []string, message json.RawMessage, tplID email.TplID) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mockemail: record: %w", err)
	}
	return c.record(to, cc, message, tplID)
}

// record appends one SendLog per to recipient, each carrying its own copy
// of the cc list, message and template for that send.
func (c *Client) record(to, cc []string, message json.RawMessage, tplID email.TplID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range to {
		c.sendLogs = append(c.sendLogs, SendLog{
			to:      v,
			cc:      slices.Clone(cc),
			message: slices.Clone(message),
			tplID:   tplID,
		})
	}

	return nil
}

// SendLogs returns a snapshot of every send recorded so far. Mutating the
// returned slice does not affect the client.
func (c *Client) SendLogs() SendLogs {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.sendLogs)
}

// FlushSendLogs clears all recorded sends.
func (c *Client) FlushSendLogs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendLogs = SendLogs{}
}
