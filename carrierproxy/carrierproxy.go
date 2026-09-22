// Package carrierproxy defines a carrier policy-provider contract.
package carrierproxy

import (
	"io"
)

// PolicyProvider authenticates against a carrier website and can retrieve
// policy data. This challenge implements Login only.
type PolicyProvider interface {
	// Login authenticates against the target site with username and
	// password.
	Login(username, password string) error
	// Policies lists the policies available to the authenticated user.
	Policies() ([]Policy, error)
	// DocumentDownload retrieves the document identified by downloadKey. The
	// caller must always close the returned body.
	DocumentDownload(downloadKey string) (io.ReadCloser, error)
}
