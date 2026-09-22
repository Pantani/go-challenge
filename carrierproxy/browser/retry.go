package browser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

func contextError(ctx context.Context, err error) error {
	if cause := ctx.Err(); cause != nil {
		return fmt.Errorf("carrierproxy: operation canceled: %w", cause)
	}
	return err
}

func (c *Client) beforeAttempt(ctx context.Context, index int) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("carrierproxy: before attempt: %w", err)
	}
	if index == 0 {
		return nil
	}
	return contextError(ctx, c.wait(ctx, c.opts.retryDelay))
}

// attemptBudget returns how many attempts to make for a given retries count.
// Negative retries still make one attempt. The largest int cannot be converted
// to an attempt count because adding one would overflow.
func attemptBudget(retries int) (int, error) {
	if retries < 0 {
		return 1, nil
	}
	if retries == int(^uint(0)>>1) {
		return 0, fmt.Errorf("%w: WithRetries overflows attempt count", carrierproxy.ErrNotConfigured)
	}
	return retries + 1, nil
}

// isFinal reports whether err is a failure a retry would only repeat.
func isFinal(err error) bool {
	finalErrors := [...]error{
		carrierproxy.ErrInvalidCredentials,
		carrierproxy.ErrMalformedResponse,
		carrierproxy.ErrNotConfigured,
	}
	for _, final := range finalErrors {
		if errors.Is(err, final) {
			return true
		}
	}
	return false
}
