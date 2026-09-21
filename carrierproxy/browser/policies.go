package browser

import (
	"fmt"
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// Policies lists the policies found in the rows of the WithPoliciesURL
// page, reading each row's first two cells (WithPolicyRowSelector,
// WithPolicyCellSelector) as CarrierID and PolicyNumber. It requires
// WithPoliciesURL and returns carrierproxy.ErrNotConfigured without it,
// and carrierproxy.ErrNotLoggedIn if Login has not yet succeeded. Like
// Login, transient failures are retried (see WithRetries).
func (c *Client) Policies() ([]carrierproxy.Policy, error) {
	if c.opts.policiesURL == "" {
		return nil, fmt.Errorf("%w: Policies needs WithPoliciesURL", carrierproxy.ErrNotConfigured)
	}

	creds, err := c.storedCredentials()
	if err != nil {
		return nil, err
	}

	var policies []carrierproxy.Policy
	err = c.withRetries(func() error {
		var attemptErr error
		policies, attemptErr = c.scrapePolicies(creds)
		return attemptErr
	})
	return policies, err
}

// scrapePolicies is a single Policies attempt: re-authenticate, navigate
// to the policies page, and parse its rows. It is the unit of work
// withRetries repeats.
func (c *Client) scrapePolicies(creds credentials) ([]carrierproxy.Policy, error) {
	pg, closePage, err := c.authenticatedPage(creds.username, creds.password)
	if err != nil {
		return nil, err
	}
	defer closePage()

	if err := pg.Navigate(c.opts.policiesURL); err != nil {
		return nil, fmt.Errorf("carrierproxy: open policies page: %w", err)
	}
	if err := pg.WaitLoad(); err != nil {
		return nil, fmt.Errorf("carrierproxy: open policies page: %w", err)
	}

	rows, err := pg.Elements(c.opts.policyRowSelector)
	if err != nil {
		return nil, fmt.Errorf("carrierproxy: list policy rows: %w", err)
	}

	return parsePolicyRows(rows, c.opts.policyCellSelector)
}

// parsePolicyRows reads the first two cells of each row as CarrierID and
// PolicyNumber, skipping rows with fewer than two matching cells (such as
// a header row).
func parsePolicyRows(rows []element, cellSelector string) ([]carrierproxy.Policy, error) {
	policies := make([]carrierproxy.Policy, 0, len(rows))
	for _, row := range rows {
		cells, err := row.Elements(cellSelector)
		if err != nil {
			return nil, fmt.Errorf("carrierproxy: read policy row: %w", err)
		}
		if len(cells) < 2 {
			continue
		}

		policy, err := newPolicy(cells)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

// newPolicy reads a Policy from a row's first two cells.
func newPolicy(cells []element) (carrierproxy.Policy, error) {
	carrierID, err := cells[0].Text()
	if err != nil {
		return carrierproxy.Policy{}, fmt.Errorf("carrierproxy: read carrier ID: %w", err)
	}
	policyNumber, err := cells[1].Text()
	if err != nil {
		return carrierproxy.Policy{}, fmt.Errorf("carrierproxy: read policy number: %w", err)
	}
	return carrierproxy.Policy{
		CarrierID:    strings.TrimSpace(carrierID),
		PolicyNumber: strings.TrimSpace(policyNumber),
	}, nil
}
