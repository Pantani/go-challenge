// Package mockemail is an in-memory email.MailProvider that records every
// send, for the demo binary and for tests.
package mockemail

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// Client synchronizes its internal log collection. Each successful send copies
// the top-level Vars map; nested mutable values remain caller-owned and require
// caller synchronization.
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

// Send validates and records a delivery using a background context.
func (c *Client) Send(to []string, tpl email.TplID, vars map[string]any) error {
	return c.SendContext(context.Background(), to, tpl, vars)
}

// SendContext checks cancellation before validation and again after acquiring
// the log mutex. Mutex acquisition is not interruptible, and cancellation
// racing with a recorded batch does not roll it back.
func (c *Client) SendContext(ctx context.Context, to []string, tpl email.TplID, vars map[string]any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mockemail send: %w", err)
	}
	if err := email.RequireRecipient(to); err != nil {
		return fmt.Errorf("mockemail send: %w", err)
	}
	if err := email.RequireTemplate(tpl); err != nil {
		return fmt.Errorf("mockemail send: %w", err)
	}
	return c.record(ctx, to, tpl, vars)
}

func (c *Client) record(ctx context.Context, to []string, tpl email.TplID, vars map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mockemail record: %w", err)
	}
	if c.sendErr != nil {
		return c.sendErr
	}
	for _, recipient := range to {
		c.sendLogs = append(c.sendLogs, SendLog{To: recipient, Tpl: tpl, Vars: maps.Clone(vars)})
	}
	return nil
}

// SendLogs returns a new log slice and a shallow copy of each Vars map.
// Nested maps, slices, and pointers are shared with the caller's original values.
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
