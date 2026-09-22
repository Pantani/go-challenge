package browser

import (
	"context"
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
	return c.PoliciesContext(context.Background())
}

// PoliciesContext is Policies with caller cancellation.
func (c *Client) PoliciesContext(ctx context.Context) ([]carrierproxy.Policy, error) {
	if c.opts.policiesURL == "" {
		return nil, fmt.Errorf("%w: Policies needs WithPoliciesURL", carrierproxy.ErrNotConfigured)
	}

	creds, err := c.storedCredentials()
	if err != nil {
		return nil, err
	}

	return attemptWithRetries(ctx, c, func(ctx context.Context) ([]carrierproxy.Policy, error) {
		return c.scrapePolicies(ctx, creds)
	})
}

// scrapePolicies is a single Policies attempt: re-authenticate, navigate
// to the policies page, and parse its rows. It is the unit of work
// withRetries repeats.
func (c *Client) scrapePolicies(ctx context.Context, creds credentials) ([]carrierproxy.Policy, error) {
	ctx, cancel := context.WithTimeout(ctx, c.opts.timeout)
	defer cancel()
	pg, closePage, err := c.authenticatedPage(ctx, creds.username, creds.password)
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

// parsePolicyRows reads each row into a Policy via parsePolicyRow,
// keeping only the ones that aren't a non-data row.
func parsePolicyRows(rows []element, cellSelector string) ([]carrierproxy.Policy, error) {
	policies := make([]carrierproxy.Policy, 0, len(rows))
	for _, row := range rows {
		policy, ok, err := parsePolicyRow(row, cellSelector)
		if err != nil {
			return nil, err
		}
		if ok {
			policies = append(policies, policy)
		}
	}
	return policies, nil
}

// parsePolicyRow reads one row's first two cells into a Policy. ok is
// false for a non-data row (zero matching cells, such as a header row
// using <th> instead of <td>), which the caller should skip rather than
// keep. The row selector is documented to match one element per policy,
// so a row with exactly one cell is malformed data, not a header, and is
// reported as carrierproxy.ErrMalformedResponse rather than skipped.
func parsePolicyRow(row element, cellSelector string) (policy carrierproxy.Policy, ok bool, err error) {
	cells, err := row.Elements(cellSelector)
	if err != nil {
		return carrierproxy.Policy{}, false, fmt.Errorf("carrierproxy: read policy row: %w", err)
	}
	if len(cells) == 0 {
		return carrierproxy.Policy{}, false, nil
	}
	if len(cells) < 2 {
		return carrierproxy.Policy{}, false, fmt.Errorf("%w: policy row has %d cells, expected at least 2", carrierproxy.ErrMalformedResponse, len(cells))
	}

	policy, err = newPolicy(cells)
	if err != nil {
		return carrierproxy.Policy{}, false, err
	}
	return policy, true, nil
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
