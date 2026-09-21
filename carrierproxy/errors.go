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
)
