package browser

import (
	"reflect"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	options := newOptions()
	WithUsernameSelector("#user")(&options)
	WithPasswordSelector("#password")(&options)
	WithSubmitSelector("#submit")(&options)
	WithResultSelector("#result")(&options)
	WithSuccessClass("ok")(&options)
	WithTimeout(5 * time.Second)(&options)

	want := browserOptions("#user", "#password", "#submit", "#result", "ok", 5*time.Second)
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("options = %+v, want %+v", options, want)
	}
}

func browserOptions(username, password, submit, result, success string, timeout time.Duration) options {
	return options{
		usernameSelector: username,
		passwordSelector: password,
		submitSelector:   submit,
		resultSelector:   result,
		successClass:     success,
		timeout:          timeout,
	}
}
