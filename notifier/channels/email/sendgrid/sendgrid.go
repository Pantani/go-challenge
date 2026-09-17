package sendgrid

import (
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
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

func (c *Client) Send(_ []string, _ email.TplID, _ map[string]any) error {
	return nil
}
