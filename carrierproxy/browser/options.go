package browser

import "time"

type (
	// options holds the configurable parts of a Client's behavior.
	// newOptions returns it populated with defaults; Option values then
	// adjust individual fields.
	options struct {
		usernameSelector string
		passwordSelector string
		submitSelector   string
		resultSelector   string
		successClass     string
		timeout          time.Duration
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

// WithTimeout overrides the time budget for one Login attempt. Defaults to 30
// seconds.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// newOptions returns the default options, ready to be adjusted by Option
// values.
func newOptions() options {
	return options{
		usernameSelector: "#username",
		passwordSelector: "#password",
		submitSelector:   "button[type='submit']",
		resultSelector:   "#flash",
		successClass:     "success",
		timeout:          30 * time.Second,
	}
}
