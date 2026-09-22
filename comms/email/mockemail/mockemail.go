package mockemail

import (
	"encoding/json"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

// Client is an in-memory email.MailProvider that records every send in
// SendLogs instead of delivering anything, for use in handler tests.
type Client struct {
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
	return c.record(to, nil, message, tplID)
}

// SendWithCC behaves like Send but additionally records the given cc
// recipients on the logged send.
func (c *Client) SendWithCC(to, cc []string, message json.RawMessage, tplID email.TplID) error {
	return c.record(to, cc, message, tplID)
}

// record appends one SendLog per to recipient, each carrying the full cc
// list, message and template for that send.
func (c *Client) record(to, cc []string, message json.RawMessage, tplID email.TplID) error {

	for _, v := range to {
		c.sendLogs = append(c.sendLogs, SendLog{
			to:      v,
			cc:      cc,
			message: message,
			tplID:   tplID,
		})
	}

	return nil
}

// SendLogs returns every send recorded so far.
func (c *Client) SendLogs() SendLogs {
	return c.sendLogs
}

// FlushSendLogs clears all recorded sends.
func (c *Client) FlushSendLogs() {
	c.sendLogs = SendLogs{}
}
