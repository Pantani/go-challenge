package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

func TestContextCancellationSkipsDispatch(t *testing.T) {
	for _, withCC := range []bool{false, true} {
		t.Run(fmt.Sprint(withCC), func(t *testing.T) { verifyCanceledDispatch(t, withCC) })
	}
}

func verifyCanceledDispatch(t *testing.T, withCC bool) {
	t.Helper()
	fake := &fakeMailClient{}
	c := &Client{client: fake}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var err error
	if withCC {
		err = c.SendWithCCContext(ctx, []string{"to@example.com"}, []string{"cc@example.com"}, json.RawMessage(`{}`), email.TplAddPolicyCoverage)
	} else {
		err = c.SendContext(ctx, []string{"to@example.com"}, json.RawMessage(`{}`), email.TplAddPolicyVehicle)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if fake.calls != 0 {
		t.Fatalf("canceled dispatch called stand-in %d times", fake.calls)
	}
}

func TestContextDeadlinePrecedesRecipientValidation(t *testing.T) {
	fake := &fakeMailClient{}
	c := &Client{client: fake}
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	err := c.SendWithCCContext(ctx, nil, nil, nil, email.TplAddPolicyVehicle)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	if fake.calls != 0 {
		t.Fatal("expired dispatch reached stand-in")
	}
}
