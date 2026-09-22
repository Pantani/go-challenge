package carrierproxy

import "errors"

var (
	// ErrInvalidCredentials is returned when the target site rejects the
	// supplied username/password (or either is blank).
	ErrInvalidCredentials = errors.New("carrierproxy: invalid username or password")

	// ErrNotLoggedIn is returned by PolicyProvider methods that need an
	// authenticated session (Policies, DocumentDownload) when Login has not
	// yet succeeded.
	ErrNotLoggedIn = errors.New("carrierproxy: not logged in")

	// ErrNotConfigured is returned by a PolicyProvider method that needs
	// site-specific configuration (such as a target URL) that was not
	// supplied.
	ErrNotConfigured = errors.New("carrierproxy: not configured")

	// ErrMalformedResponse is returned when the target site's response
	// doesn't match the shape a PolicyProvider method expects (such as a
	// policy row with the wrong number of cells). It is never retried,
	// since the same response would just fail to parse again.
	ErrMalformedResponse = errors.New("carrierproxy: malformed response")
)
