// Package mockemail is an in-memory email.MailProvider that records every
// send, for the demo binary and for tests.
package mockemail

import (
	"maps"
	"sync"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// Client records every successful Send. It is safe for concurrent use.
type Client struct {
	mu       sync.Mutex
	sendLogs SendLogs
	sendErr  error
}

// NewClient returns an empty, ready-to-use Client.
func NewClient() *Client { return &Client{} }

// SetSendError makes every subsequent Send fail with err until it is reset
// with nil. Failed sends are not recorded. It lets tests exercise provider
// failure paths without a real provider.
func (c *Client) SetSendError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendErr = err
}

// Send validates the recipients the same way a real provider would, then
// records one SendLog per recipient.
func (c *Client) Send(to []string, tpl email.TplID, vars map[string]any) error {
	if err := email.RequireRecipient(to); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return c.sendErr
	}
	for _, recipient := range to {
		c.sendLogs = append(c.sendLogs, SendLog{To: recipient, Tpl: tpl, Vars: maps.Clone(vars)})
	}
	return nil
}

// SendLogs returns a snapshot copy of the recorded sends. Later sends or a
// FlushSendLogs do not affect a snapshot already handed out.
func (c *Client) SendLogs() SendLogs {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(SendLogs, len(c.sendLogs))
	for i, l := range c.sendLogs {
		out[i] = l
		out[i].Vars = maps.Clone(l.Vars)
	}
	return out
}

// FlushSendLogs discards every recorded send.
func (c *Client) FlushSendLogs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendLogs = nil
}
