package browser

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

const testPoliciesURL = "https://example.com/policies"

func TestClientPolicies(t *testing.T) {
	t.Parallel()

	t.Run("requires WithPoliciesURL", testClientPoliciesMissingURL)

	t.Run("requires a prior successful Login", testClientPoliciesMissingLogin)

	t.Run("lists policies after a successful login", testClientPoliciesSuccess)

	t.Run("retries a transient failure", testClientPoliciesTransientFailure)

	t.Run("exhausts retries and returns the last error with no policies", testClientPoliciesRetriesExhausted)

	t.Run("never retries a malformed row", testClientPoliciesMalformedRow)
}

func testClientPoliciesMissingURL(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL)
	if _, err := c.Policies(); !errors.Is(err, carrierproxy.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func testClientPoliciesMissingLogin(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	if _, err := c.Policies(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func testClientPoliciesSuccess(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(0))
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")

	fp := successPage()
	fp.elementLists["tr"] = []*fakeElement{
		fakeRow("td", "CARRIER-1", "POL-100"),
		fakeRow("td", "CARRIER-2", "POL-200"),
	}
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }

	got, err := c.Policies()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []carrierproxy.Policy{
		{CarrierID: "CARRIER-1", PolicyNumber: "POL-100"},
		{CarrierID: "CARRIER-2", PolicyNumber: "POL-200"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func testClientPoliciesTransientFailure(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(1))
	c.wait = func(context.Context, time.Duration) error { return nil }
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")

	fp := successPage()
	fp.elementLists["tr"] = []*fakeElement{fakeRow("td", "CARRIER-1", "POL-100")}
	calls := 0
	c.newPage = func(context.Context) (page, func(), error) {
		calls++
		if calls == 1 {
			return nil, nil, errors.New("transient")
		}
		return fp, func() {}, nil
	}

	got, err := c.Policies()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(got))
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func testClientPoliciesRetriesExhausted(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(2))
	c.wait = func(context.Context, time.Duration) error { return nil }
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")
	wantErr := errors.New("persistent")
	calls := 0
	c.newPage = func(context.Context) (page, func(), error) { calls++; return nil, nil, wantErr }

	got, err := c.Policies()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != nil {
		t.Fatalf("expected no policies on failure, got %+v", got)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
}

func testClientPoliciesMalformedRow(t *testing.T) {
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(2))
	c.wait = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep: a malformed row must not be retried")
		return nil
	}
	c.rememberCredentials("tomsmith", "SuperSecretPassword!")

	fp := successPage()
	fp.elementLists["tr"] = []*fakeElement{fakeRow("td", "only-one")}
	calls := 0
	c.newPage = func(context.Context) (page, func(), error) {
		calls++
		return fp, func() {}, nil
	}

	if _, err := c.Policies(); !errors.Is(err, carrierproxy.ErrMalformedResponse) {
		t.Fatalf("expected ErrMalformedResponse, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", calls)
	}
}

func TestScrapePolicies(t *testing.T) {
	t.Parallel()

	t.Run("re-authentication fails", testScrapePoliciesAuthenticationFailure)

	t.Run("navigating to the policies page fails", testScrapePoliciesNavigateFailure)

	t.Run("waiting for the policies page to load fails", testScrapePoliciesWaitLoadFailure)

	t.Run("listing rows fails", testScrapePoliciesRowsFailure)

	t.Run("releases the page once done", testScrapePoliciesReleasesPage)
}

func testScrapePoliciesAuthenticationFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	wantErr := errors.New("boom")
	c.newPage = func(context.Context) (page, func(), error) { return nil, nil, wantErr }
	if _, err := c.scrapePolicies(context.Background(), creds); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testScrapePoliciesNavigateFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	fp := successPage()
	wantErr := errors.New("boom")
	fp.navigateErrByURL[testPoliciesURL] = wantErr
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if _, err := c.scrapePolicies(context.Background(), creds); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testScrapePoliciesWaitLoadFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	fp := successPage()
	wantErr := errors.New("boom")
	// First WaitLoad call belongs to the login itself and must
	// succeed; the second is scrapePolicies' own and is the one
	// under test here.
	fp.waitLoadErrs = []error{nil, wantErr}
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if _, err := c.scrapePolicies(context.Background(), creds); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testScrapePoliciesRowsFailure(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	fp := successPage()
	wantErr := errors.New("boom")
	fp.elementListsErr["tr"] = wantErr
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() {}, nil }
	if _, err := c.scrapePolicies(context.Background(), creds); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testScrapePoliciesReleasesPage(t *testing.T) {
	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}
	t.Parallel()
	c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
	fp := successPage()
	closed := false
	c.newPage = func(context.Context) (page, func(), error) { return fp, func() { closed = true }, nil }
	if _, err := c.scrapePolicies(context.Background(), creds); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closed {
		t.Fatal("expected the page to be released")
	}
}

func TestParsePolicyRows(t *testing.T) {
	t.Parallel()

	t.Run("reads the first two cells of each row", testParsePolicyRowsFirstTwoCells)

	t.Run("skips a header row with zero matching cells", testParsePolicyRowsHeader)

	t.Run("a row with exactly one cell is malformed, not a header, and is an error", testParsePolicyRowsMalformedRow)

	t.Run("propagates a cell-listing error", testParsePolicyRowsCellListFailure)

	t.Run("propagates a cell-reading error from within a row", testParsePolicyRowsCellReadFailure)

	t.Run("empty input yields an empty, non-nil slice", testParsePolicyRowsEmpty)
}

func testParsePolicyRowsFirstTwoCells(t *testing.T) {
	t.Parallel()
	rows := []element{
		fakeRow("td", "CARRIER-1", "POL-100"),
		fakeRow("td", "CARRIER-2", "POL-200", "extra-cell-ignored"),
	}
	got, err := parsePolicyRows(rows, "td")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []carrierproxy.Policy{
		{CarrierID: "CARRIER-1", PolicyNumber: "POL-100"},
		{CarrierID: "CARRIER-2", PolicyNumber: "POL-200"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func testParsePolicyRowsHeader(t *testing.T) {
	t.Parallel()
	rows := []element{
		fakeRow("td"),
		fakeRow("td", "CARRIER-1", "POL-100"),
	}
	got, err := parsePolicyRows(rows, "td")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 policy, got %d: %+v", len(got), got)
	}
}

func testParsePolicyRowsMalformedRow(t *testing.T) {
	t.Parallel()
	rows := []element{fakeRow("td", "only-one")}
	if _, err := parsePolicyRows(rows, "td"); !errors.Is(err, carrierproxy.ErrMalformedResponse) {
		t.Fatalf("expected ErrMalformedResponse, got %v", err)
	}
}

func testParsePolicyRowsCellListFailure(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")
	row := &fakeElement{subElementsErr: map[string]error{"td": wantErr}}
	if _, err := parsePolicyRows([]element{row}, "td"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testParsePolicyRowsCellReadFailure(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")
	row := &fakeElement{subElements: map[string][]*fakeElement{
		"td": {{textErr: wantErr}, {text: "POL-100"}},
	}}
	if _, err := parsePolicyRows([]element{row}, "td"); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func testParsePolicyRowsEmpty(t *testing.T) {
	t.Parallel()
	got, err := parsePolicyRows(nil, "td")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected an empty non-nil slice, got %#v", got)
	}
}

func TestNewPolicy(t *testing.T) {
	t.Parallel()

	t.Run("trims whitespace from both cells", func(t *testing.T) {
		t.Parallel()
		cells := []element{
			&fakeElement{text: "  CARRIER-1  "},
			&fakeElement{text: "  POL-100  "},
		}
		got, err := newPolicy(cells)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := carrierproxy.Policy{CarrierID: "CARRIER-1", PolicyNumber: "POL-100"}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("propagates a carrier ID read error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("boom")
		cells := []element{&fakeElement{textErr: wantErr}, &fakeElement{text: "POL-100"}}
		if _, err := newPolicy(cells); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("propagates a policy number read error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("boom")
		cells := []element{&fakeElement{text: "CARRIER-1"}, &fakeElement{textErr: wantErr}}
		if _, err := newPolicy(cells); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})
}
