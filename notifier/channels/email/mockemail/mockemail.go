package mockemail

import "github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"

type Client struct {
	sendLogs SendLogs
}

func NewClient() *Client { return &Client{} }

func (c *Client) Send(to []string, tpl email.TplID, vars map[string]any) error {
	for _, recipient := range to {
		c.sendLogs = append(c.sendLogs, SendLog{To: recipient, Tpl: tpl, Vars: vars})
	}
	return nil
}

func (c *Client) SendLogs() SendLogs { return c.sendLogs }

func (c *Client) FlushSendLogs() { c.sendLogs = nil }
