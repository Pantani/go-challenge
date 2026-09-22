package carrierproxy

import "errors"

var (
	// ErrInvalidCredentials is returned when the target site rejects the
	// supplied username/password (or either is blank).
	ErrInvalidCredentials = errors.New("carrierproxy: invalid username or password")

	// ErrNotImplemented is returned by the methods outside this challenge's
	// partial PolicyProvider implementation.
	ErrNotImplemented = errors.New("carrierproxy: not implemented")

	// ErrNotConfigured is returned by a PolicyProvider method that needs
	// site-specific configuration (such as a target URL) that was not
	// supplied.
	ErrNotConfigured = errors.New("carrierproxy: not configured")
)
