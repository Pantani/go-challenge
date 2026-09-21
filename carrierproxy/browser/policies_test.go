package browser

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

const testPoliciesURL = "https://example.com/policies"

func TestClientPolicies(t *testing.T) {
	t.Parallel()

	t.Run("requires WithPoliciesURL", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL)
		if _, err := c.Policies(); !errors.Is(err, carrierproxy.ErrNotConfigured) {
			t.Fatalf("expected ErrNotConfigured, got %v", err)
		}
	})

	t.Run("requires a prior successful Login", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		if _, err := c.Policies(); !errors.Is(err, carrierproxy.ErrNotLoggedIn) {
			t.Fatalf("expected ErrNotLoggedIn, got %v", err)
		}
	})

	t.Run("lists policies after a successful login", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(0))
		c.rememberCredentials("tomsmith", "SuperSecretPassword!")

		fp := successPage()
		fp.elementLists["tr"] = []*fakeElement{
			fakeRow("td", "CARRIER-1", "POL-100"),
			fakeRow("td", "CARRIER-2", "POL-200"),
		}
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }

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
	})

	t.Run("retries a transient failure", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL), WithRetries(1))
		c.sleep = func(time.Duration) {}
		c.rememberCredentials("tomsmith", "SuperSecretPassword!")

		fp := successPage()
		fp.elementLists["tr"] = []*fakeElement{fakeRow("td", "CARRIER-1", "POL-100")}
		calls := 0
		c.newPage = func(time.Duration) (page, func(), error) {
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
	})
}

func TestScrapePolicies(t *testing.T) {
	t.Parallel()

	creds := credentials{username: "tomsmith", password: "SuperSecretPassword!"}

	t.Run("re-authentication fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		wantErr := errors.New("boom")
		c.newPage = func(time.Duration) (page, func(), error) { return nil, nil, wantErr }
		if _, err := c.scrapePolicies(creds); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("navigating to the policies page fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		fp := successPage()
		wantErr := errors.New("boom")
		fp.navigateErrByURL[testPoliciesURL] = wantErr
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if _, err := c.scrapePolicies(creds); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("waiting for the policies page to load fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		fp := successPage()
		wantErr := errors.New("boom")
		// First WaitLoad call belongs to the login itself and must
		// succeed; the second is scrapePolicies' own and is the one
		// under test here.
		fp.waitLoadErrs = []error{nil, wantErr}
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if _, err := c.scrapePolicies(creds); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("listing rows fails", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		fp := successPage()
		wantErr := errors.New("boom")
		fp.elementListsErr["tr"] = wantErr
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() {}, nil }
		if _, err := c.scrapePolicies(creds); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("releases the page once done", func(t *testing.T) {
		t.Parallel()
		c := NewClient(testLoginURL, WithPoliciesURL(testPoliciesURL))
		fp := successPage()
		closed := false
		c.newPage = func(time.Duration) (page, func(), error) { return fp, func() { closed = true }, nil }
		if _, err := c.scrapePolicies(creds); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !closed {
			t.Fatal("expected the page to be released")
		}
	})
}

func TestParsePolicyRows(t *testing.T) {
	t.Parallel()

	t.Run("reads the first two cells of each row", func(t *testing.T) {
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
	})

	t.Run("skips rows with fewer than two cells, such as a header row", func(t *testing.T) {
		t.Parallel()
		rows := []element{
			fakeRow("td"),
			fakeRow("td", "only-one"),
			fakeRow("td", "CARRIER-1", "POL-100"),
		}
		got, err := parsePolicyRows(rows, "td")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 policy, got %d: %+v", len(got), got)
		}
	})

	t.Run("propagates a cell-listing error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("boom")
		row := &fakeElement{subElementsErr: map[string]error{"td": wantErr}}
		if _, err := parsePolicyRows([]element{row}, "td"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("propagates a cell-reading error from within a row", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("boom")
		row := &fakeElement{subElements: map[string][]*fakeElement{
			"td": {{textErr: wantErr}, {text: "POL-100"}},
		}}
		if _, err := parsePolicyRows([]element{row}, "td"); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("empty input yields an empty, non-nil slice", func(t *testing.T) {
		t.Parallel()
		got, err := parsePolicyRows(nil, "td")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("expected an empty non-nil slice, got %#v", got)
		}
	})
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
