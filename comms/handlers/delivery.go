package handlers

import (
	"context"
	"encoding/json"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

type contextSender interface {
	SendContext(ctx context.Context, to []string, message json.RawMessage, tpl email.TplID) error
}

type contextCCSender interface {
	SendWithCCContext(ctx context.Context, to, cc []string, message json.RawMessage, tpl email.TplID) error
}

// deliver sends the envelope through svc, using SendWithCC only when there
// are CC recipients left after normalisation.
func (e envelope) deliver(ctx context.Context, svc email.MailProvider, tpl email.TplID) error {
	if len(e.cc) == 0 {
		return sendContext(ctx, svc, []string{e.to}, e.message, tpl)
	}
	return sendCCContext(ctx, svc, []string{e.to}, e.cc, e.message, tpl)
}

func sendContext(ctx context.Context, svc email.MailProvider, to []string, message json.RawMessage, tpl email.TplID) error {
	if sender, ok := svc.(contextSender); ok {
		return sender.SendContext(ctx, to, message, tpl)
	}
	return svc.Send(to, message, tpl)
}

func sendCCContext(ctx context.Context, svc email.MailProvider, to, cc []string, message json.RawMessage, tpl email.TplID) error {
	if sender, ok := svc.(contextCCSender); ok {
		return sender.SendWithCCContext(ctx, to, cc, message, tpl)
	}
	return svc.SendWithCC(to, cc, message, tpl)
}
