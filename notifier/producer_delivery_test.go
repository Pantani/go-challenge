package notifier_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier"
	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

type blockingMail struct{ started chan context.Context }

func (m *blockingMail) Send([]string, email.TplID, map[string]any) error {
	return errors.New("legacy Send was selected")
}

func (m *blockingMail) SendContext(ctx context.Context, _ []string, _ email.TplID, _ map[string]any) error {
	m.started <- ctx
	<-ctx.Done()
	return ctx.Err()
}

func deliveryRequest() notifier.Request {
	return notifier.Request{Recipients: []string{"u@example.com"}, Template: email.TplOTPLogin, Vars: map[string]any{"otpCode": "123"}}
}

func awaitStarted(t *testing.T, started <-chan context.Context, done <-chan error) context.Context {
	t.Helper()
	select {
	case ctx := <-started:
		return ctx
	case err := <-done:
		t.Fatalf("delivery ended before context dispatch: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("delivery did not start")
	}
	return nil
}

func awaitDelivery(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("delivery did not finish")
	}
	return nil
}

func TestNotifyCancelsStartedDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mail := &blockingMail{started: make(chan context.Context, 1)}
	done := make(chan error, 1)
	go func() { done <- notifier.NewProducer(mail).Notify(ctx, deliveryRequest()) }()
	passed := awaitStarted(t, mail.started, done)
	if passed != ctx {
		t.Fatal("provider did not receive caller context")
	}
	cancel()
	if err := awaitDelivery(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Notify() = %v", err)
	}
}

type legacyMail struct {
	calls   chan notifier.Request
	sendErr error
}

func (m legacyMail) Send(to []string, tpl email.TplID, vars map[string]any) error {
	m.calls <- notifier.Request{Recipients: to, Template: tpl, Vars: vars}
	return m.sendErr
}

func TestNotifyLegacyCompatibility(t *testing.T) {
	calls := make(chan notifier.Request, 1)
	wantErr := errors.New("legacy provider failure")
	mail := legacyMail{calls: calls, sendErr: wantErr}
	req := deliveryRequest()
	err := notifier.NewProducer(mail).Notify(context.Background(), req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Notify() = %v; want %v", err, wantErr)
	}
	if len(calls) != 1 {
		t.Fatalf("legacy calls = %d; want 1", len(calls))
	}
	if got := <-calls; !reflect.DeepEqual(got, req) {
		t.Fatalf("legacy payload = %#v; want %#v", got, req)
	}
}

type contextSpy struct {
	legacyMail
	callsContext chan notifier.Request
}

func (m contextSpy) SendContext(_ context.Context, to []string, tpl email.TplID, vars map[string]any) error {
	m.callsContext <- notifier.Request{Recipients: to, Template: tpl, Vars: vars}
	return m.sendErr
}

func TestNotifyContextPayload(t *testing.T) {
	mail := contextSpy{legacyMail: legacyMail{calls: make(chan notifier.Request, 1)}, callsContext: make(chan notifier.Request, 1)}
	req := deliveryRequest()
	if err := notifier.NewProducer(mail).Notify(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(mail.callsContext) != 1 {
		t.Fatalf("context calls = %d; want 1", len(mail.callsContext))
	}
	if got := <-mail.callsContext; !reflect.DeepEqual(got, req) {
		t.Fatalf("context payload = %#v; want %#v", got, req)
	}
	if len(mail.calls) != 0 {
		t.Fatal("legacy Send was also called")
	}
}
