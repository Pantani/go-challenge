package notifier

import (
	"context"
	"reflect"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

// ContextMailProvider is the optional cancellation-aware delivery contract
// consumed by Producer. MailProvider remains the required legacy contract.
type ContextMailProvider interface {
	SendContext(ctx context.Context, to []string, tpl email.TplID, vars map[string]any) error
}

func (p *Producer) send(ctx context.Context, req Request) error {
	if provider, ok := p.email.(ContextMailProvider); ok {
		return provider.SendContext(ctx, req.Recipients, req.Template, req.Vars)
	}
	return p.email.Send(req.Recipients, req.Template, req.Vars)
}

func missingMailProvider(provider email.MailProvider) bool {
	if provider == nil {
		return true
	}
	value := reflect.ValueOf(provider)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
