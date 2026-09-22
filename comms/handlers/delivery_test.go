package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

type contextSendSpy struct {
	deliverySpy
	contexts []context.Context
}

func (p *contextSendSpy) SendContext(ctx context.Context, to []string, msg json.RawMessage, tpl email.TplID) error {
	p.contexts = append(p.contexts, ctx)
	return p.Send(to, msg, tpl)
}

type contextCCSpy struct {
	deliverySpy
	contexts []context.Context
}

func (p *contextCCSpy) SendWithCCContext(ctx context.Context, to, cc []string, msg json.RawMessage, tpl email.TplID) error {
	p.contexts = append(p.contexts, ctx)
	return p.SendWithCC(to, cc, msg, tpl)
}

func TestIndependentContextCapabilities(t *testing.T) {
	t.Run("send-only capability", func(t *testing.T) {
		p := &contextSendSpy{}
		exerciseContextSelection(t, p, &p.deliverySpy, &p.contexts, validBody, false, 1)
	})
	t.Run("cc-only capability", func(t *testing.T) {
		p := &contextCCSpy{}
		exerciseContextSelection(t, p, &p.deliverySpy, &p.contexts, coverageBody, true, 1)
	})
	t.Run("send-only falls back for cc", func(t *testing.T) {
		p := &contextSendSpy{}
		exerciseContextSelection(t, p, &p.deliverySpy, &p.contexts, coverageBody, true, 0)
	})
	t.Run("cc-only falls back for send", func(t *testing.T) {
		p := &contextCCSpy{}
		exerciseContextSelection(t, p, &p.deliverySpy, &p.contexts, validBody, false, 0)
	})
}

const coverageBody = `{"email_to":"foo@bar.com","email_cc":["cc@bar.com"],"message":{"foo":"bar"}}`

func exerciseContextSelection(t *testing.T, p email.MailProvider, spy *deliverySpy, contexts *[]context.Context, body string, withCC bool, wantContexts int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()
	handlers.AddPolicyCoverage(p)(w, req)
	assertEqual(t, "status", w.Code, http.StatusOK)
	assertEqual(t, "body", w.Body.String(), "")
	assertEqual(t, "context calls", len(*contexts), wantContexts)
	if wantContexts != 0 {
		assertEqual(t, "same context", (*contexts)[0] == ctx, true)
	}
	tc := handlerCase{expectStatus: http.StatusOK, expectTo: "foo@bar.com", expectMsg: `{"foo":"bar"}`}
	if withCC {
		tc.expectCC = []string{"cc@bar.com"}
	}
	assertHandlerDelivery(t, spy, email.TplAddPolicyCoverage, tc)
}

type blockingProvider struct {
	deliverySpy
	entered  chan context.Context
	returned chan error
	release  chan struct{}
}

func (p *blockingProvider) block(ctx context.Context) error {
	p.entered <- ctx
	select {
	case <-ctx.Done():
		err := ctx.Err()
		p.returned <- err
		return err
	case <-p.release:
		return context.Canceled
	}
}

func (p *blockingProvider) SendContext(ctx context.Context, _ []string, _ json.RawMessage, _ email.TplID) error {
	return p.block(ctx)
}

func (p *blockingProvider) SendWithCCContext(ctx context.Context, _, _ []string, _ json.RawMessage, _ email.TplID) error {
	return p.block(ctx)
}

func awaitValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case value := <-ch:
		return value
	case <-timer.C:
		t.Fatal("timed out waiting for provider/handler handshake")
		var zero T
		return zero
	}
}

func TestRequestCancellationDuringDelivery(t *testing.T) {
	for name, body := range map[string]string{"send": validBody, "cc": coverageBody} {
		t.Run(name, func(t *testing.T) { exerciseCancellation(t, body) })
	}
}

func exerciseCancellation(t *testing.T, body string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &blockingProvider{entered: make(chan context.Context, 1), returned: make(chan error, 1), release: make(chan struct{})}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	t.Cleanup(func() {
		close(p.release)
		awaitValue(t, done)
	})
	go func() {
		defer close(done)
		handlers.AddPolicyCoverage(p)(w, req)
	}()
	received := awaitValue(t, p.entered)
	assertEqual(t, "request context", received == ctx, true)
	cancel()
	err := awaitValue(t, p.returned)
	assertEqual(t, "cancellation cause", errors.Is(err, context.Canceled), true)
	awaitValue(t, done)
	assertEqual(t, "legacy calls", len(p.send)+len(p.sendCC), 0)
	assertHandlerResponse(t, w, handlerCase{expectStatus: http.StatusInternalServerError, expectBody: "error sending email\n"})
}
