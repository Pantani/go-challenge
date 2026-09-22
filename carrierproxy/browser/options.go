package browser

import "time"

type (
	// options holds the configurable parts of a Client's behavior.
	// newOptions returns it populated with defaults; Option values then
	// adjust individual fields.
	options struct {
		usernameSelector   string
		passwordSelector   string
		submitSelector     string
		resultSelector     string
		successClass       string
		timeout            time.Duration
		retries            int
		retryDelay         time.Duration
		policiesURL        string
		policyRowSelector  string
		policyCellSelector string
		documentURL        func(downloadKey string) string
	}

	// Option configures a Client built by NewClient.
	Option func(*options)
)

// WithUsernameSelector overrides the CSS selector used to locate the
// username field. Defaults to "#username".
func WithUsernameSelector(selector string) Option {
	return func(o *options) { o.usernameSelector = selector }
}

// WithPasswordSelector overrides the CSS selector used to locate the
// password field. Defaults to "#password".
func WithPasswordSelector(selector string) Option {
	return func(o *options) { o.passwordSelector = selector }
}

// WithSubmitSelector overrides the CSS selector used to locate the submit
// control. Defaults to "button[type='submit']".
func WithSubmitSelector(selector string) Option {
	return func(o *options) { o.submitSelector = selector }
}

// WithResultSelector overrides the CSS selector used to locate the
// post-submit result banner. It should match an element that only exists
// once the site has responded to the submission, since it is waited for
// as soon as the form is submitted. Defaults to "#flash".
func WithResultSelector(selector string) Option {
	return func(o *options) { o.resultSelector = selector }
}

// WithSuccessClass overrides the CSS class that marks the result banner as
// a success rather than a failure; it must match a whole class token
// ("success" does not match "unsuccessful") and must not be empty (Login
// returns carrierproxy.ErrNotConfigured). Defaults to "success".
func WithSuccessClass(class string) Option {
	return func(o *options) { o.successClass = class }
}

// WithTimeout overrides the shared time budget for one Login, Policies, or
// DocumentDownload attempt, excluding later retries. The attempt starts its
// budget before browser launch and uses it for browser operations and any HTTP
// download/body reads. A successful document body retains that context until
// EOF, a read error, Close, or the deadline. Defaults to 30 seconds. Rod
// v0.116.2 has context-unaware dependency operations, so this is not an
// absolute wall-clock bound for browser startup or cleanup.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithRetries overrides how many additional attempts are made after a
// transient failure (browser launch, navigation, a not-yet-rendered form,
// ...) in Login, or after re-authenticating for Policies or
// DocumentDownload. carrierproxy.ErrInvalidCredentials,
// carrierproxy.ErrMalformedResponse and carrierproxy.ErrNotConfigured are
// never retried, since the same attempt would just fail again. 0 disables
// retries. Negative values retain one-attempt behavior. The maximum
// representable int returns carrierproxy.ErrNotConfigured because the attempt
// count would overflow. Defaults to 2 (3 attempts total).
func WithRetries(n int) Option {
	return func(o *options) { o.retries = n }
}

// WithRetryDelay overrides how long a retrying call waits before each
// retry. Defaults to 1 second.
func WithRetryDelay(d time.Duration) Option {
	return func(o *options) { o.retryDelay = d }
}

// WithPoliciesURL sets the page Policies scrapes for policy rows. Required
// for Policies; it returns carrierproxy.ErrNotConfigured without it.
func WithPoliciesURL(url string) Option {
	return func(o *options) { o.policiesURL = url }
}

// WithPolicyRowSelector overrides the CSS selector that matches one
// element per policy on the WithPoliciesURL page. Defaults to "tr".
func WithPolicyRowSelector(selector string) Option {
	return func(o *options) { o.policyRowSelector = selector }
}

// WithPolicyCellSelector overrides the CSS selector, relative to each
// policy row, that matches its cells; Policies reads the first as
// CarrierID and the second as PolicyNumber. Defaults to "td".
func WithPolicyCellSelector(selector string) Option {
	return func(o *options) { o.policyCellSelector = selector }
}

// WithDocumentURL sets the function DocumentDownload uses to turn an
// escapedDownloadKey into the URL it fetches. The argument is a
// url.PathEscape-encoded single path segment; append it directly to a trusted
// document path without decoding or encoding it again. The callback chooses
// the origin and can return an unrelated URL, so Client cannot guarantee an
// origin or path boundary. Required for DocumentDownload, which returns
// carrierproxy.ErrNotConfigured without it.
func WithDocumentURL(build func(downloadKey string) string) Option {
	return func(o *options) { o.documentURL = build }
}

// newOptions returns the default options, ready to be adjusted by Option
// values.
func newOptions() options {
	return options{
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
}
