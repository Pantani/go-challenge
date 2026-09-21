package browser

import (
	"reflect"
	"testing"
	"time"
)

func TestNewOptions(t *testing.T) {
	t.Parallel()

	got := newOptions()
	want := options{
		usernameSelector:   "#username",
		passwordSelector:   "#password",
		submitSelector:     "button[type='submit']",
		resultSelector:     "#flash",
		successClass:       "success",
		timeout:            30 * time.Second,
		retries:            2,
		retryDelay:         1 * time.Second,
		policyRowSelector:  "tr",
		policyCellSelector: "td",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWithUsernameSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithUsernameSelector("#u")(&o)
	if o.usernameSelector != "#u" {
		t.Fatalf("got %q, want %q", o.usernameSelector, "#u")
	}
}

func TestWithPasswordSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithPasswordSelector("#p")(&o)
	if o.passwordSelector != "#p" {
		t.Fatalf("got %q, want %q", o.passwordSelector, "#p")
	}
}

func TestWithSubmitSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithSubmitSelector("#s")(&o)
	if o.submitSelector != "#s" {
		t.Fatalf("got %q, want %q", o.submitSelector, "#s")
	}
}

func TestWithResultSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithResultSelector("#r")(&o)
	if o.resultSelector != "#r" {
		t.Fatalf("got %q, want %q", o.resultSelector, "#r")
	}
}

func TestWithSuccessClass(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithSuccessClass("ok")(&o)
	if o.successClass != "ok" {
		t.Fatalf("got %q, want %q", o.successClass, "ok")
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithTimeout(5 * time.Second)(&o)
	if o.timeout != 5*time.Second {
		t.Fatalf("got %v, want %v", o.timeout, 5*time.Second)
	}
}

func TestWithRetries(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithRetries(5)(&o)
	if o.retries != 5 {
		t.Fatalf("got %d, want %d", o.retries, 5)
	}
}

func TestWithRetryDelay(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithRetryDelay(50 * time.Millisecond)(&o)
	if o.retryDelay != 50*time.Millisecond {
		t.Fatalf("got %v, want %v", o.retryDelay, 50*time.Millisecond)
	}
}

func TestWithPoliciesURL(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithPoliciesURL("https://example.com/policies")(&o)
	if o.policiesURL != "https://example.com/policies" {
		t.Fatalf("got %q, want %q", o.policiesURL, "https://example.com/policies")
	}
}

func TestWithPolicyRowSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithPolicyRowSelector(".policy-row")(&o)
	if o.policyRowSelector != ".policy-row" {
		t.Fatalf("got %q, want %q", o.policyRowSelector, ".policy-row")
	}
}

func TestWithPolicyCellSelector(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithPolicyCellSelector(".cell")(&o)
	if o.policyCellSelector != ".cell" {
		t.Fatalf("got %q, want %q", o.policyCellSelector, ".cell")
	}
}

func TestWithDocumentURL(t *testing.T) {
	t.Parallel()
	o := newOptions()
	WithDocumentURL(func(key string) string { return "https://example.com/docs/" + key })(&o)
	if got, want := o.documentURL("abc"), "https://example.com/docs/abc"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
