package mockemail_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
)

func TestContextCancellationDoesNotRecord(t *testing.T) {
	for _, withCC := range []bool{false, true} {
		t.Run(fmt.Sprint(withCC), func(t *testing.T) {
			verifyCanceledRecording(t, withCC)
		})
	}
}

func verifyCanceledRecording(t *testing.T, withCC bool) {
	t.Helper()
	c := mockemail.NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sendRecording(ctx, c, withCC)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if len(c.SendLogs()) != 0 {
		t.Fatal("canceled send recorded a delivery")
	}
}

func sendRecording(ctx context.Context, c *mockemail.Client, withCC bool) error {
	if withCC {
		return c.SendWithCCContext(ctx, []string{"to@example.com"}, []string{"cc@example.com"}, json.RawMessage(`{}`), email.TplAddPolicyCoverage)
	}
	return c.SendContext(ctx, []string{"to@example.com"}, json.RawMessage(`{}`), email.TplAddPolicyVehicle)
}

func TestContextExpiredDoesNotRecord(t *testing.T) {
	c := mockemail.NewClient()
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	err := c.SendContext(ctx, nil, nil, email.TplAddPolicyVehicle)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
	if len(c.SendLogs()) != 0 {
		t.Fatal("expired send recorded a delivery")
	}
}
