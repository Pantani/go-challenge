package browser

import (
	"context"
	"time"
)

const shutdownTimeout = 2 * time.Second

// shutdownBrowser gives graceful CDP close its own budget even after the
// caller cancels. The callback must honor ctx; kill and cleanup have no
// context and therefore no absolute wall-clock bound.
func shutdownBrowser(closeBrowser func(context.Context) error, kill, cleanup func()) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := closeBrowser(ctx); err != nil {
		kill()
	}
	cleanup()
}
