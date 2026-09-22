package browser

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestShutdownHasIndependentDeadline(t *testing.T) {
	var events []string
	shutdownBrowser(func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("shutdown has no deadline")
		}
		if err := ctx.Err(); err != nil {
			t.Errorf("shutdown already canceled: %v", err)
		}
		events = append(events, "close")
		return context.DeadlineExceeded
	}, func() { events = append(events, "kill") }, func() { events = append(events, "cleanup") })
	if !reflect.DeepEqual(events, []string{"close", "kill", "cleanup"}) {
		t.Fatalf("shutdown order = %v", events)
	}
}

func TestShutdownSuccessDoesNotKill(t *testing.T) {
	var events []string
	shutdownBrowser(func(context.Context) error {
		events = append(events, "close")
		return nil
	}, func() { t.Error("successful shutdown killed browser") },
		func() { events = append(events, "cleanup") })
	if !reflect.DeepEqual(events, []string{"close", "cleanup"}) {
		t.Fatal(events)
	}
}

func TestShutdownTimeoutKillsBeforeCleanup(t *testing.T) {
	var events []string
	shutdownBrowser(func(ctx context.Context) error {
		events = append(events, "close")
		guard := time.NewTimer(5 * time.Second)
		defer guard.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-guard.C:
			t.Error("shutdown context did not expire within five seconds")
			return context.DeadlineExceeded
		}
	}, func() { events = append(events, "kill") }, func() { events = append(events, "cleanup") })
	if !reflect.DeepEqual(events, []string{"close", "kill", "cleanup"}) {
		t.Fatalf("shutdown order = %v", events)
	}
}
