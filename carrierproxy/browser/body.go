package browser

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// attemptBody owns the attempt deadline after the browser page is released.
type attemptBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *attemptBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.cancel()
	}
	return n, err
}

func (b *attemptBody) Close() error {
	b.cancel()
	return b.ReadCloser.Close()
}

// documentResult releases a successful response if cancellation wins at the
// retry boundary, before the caller can take ownership of its body.
func documentResult(body io.ReadCloser, err error) (io.ReadCloser, error) {
	if err == nil {
		return body, nil
	}
	if body != nil {
		if closeErr := body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("carrierproxy: close canceled document: %w", closeErr))
		}
	}
	return nil, err
}
