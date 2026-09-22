package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// fakeProvider records the credentials Login was called with and returns
// loginErr; Policies and DocumentDownload are never reached by run.
type fakeProvider struct {
	loginErr           error
	calls              int
	username, password string
}

func (f *fakeProvider) Login(username, password string) error {
	f.calls++
	f.username, f.password = username, password
	return f.loginErr
}

func (f *fakeProvider) Policies() ([]carrierproxy.Policy, error) { return nil, nil }

func (f *fakeProvider) DocumentDownload(string) (io.ReadCloser, error) { return nil, nil }

// failingWriter fails every Write with err.
type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRun(t *testing.T) {
	t.Parallel()

	loginErr := errors.New("site unreachable")
	writeErr := errors.New("stdout closed")

	cases := map[string]struct {
		env       map[string]string
		loginErr  error
		out       io.Writer
		wantErr   error
		wantCalls int
		wantOut   string
	}{
		"both credentials missing": {
			env:     map[string]string{},
			wantErr: carrierproxy.ErrInvalidCredentials,
		},
		"username missing": {
			env:     map[string]string{envPassword: "pw"},
			wantErr: carrierproxy.ErrInvalidCredentials,
		},
		"password missing": {
			env:     map[string]string{envUsername: "alice"},
			wantErr: carrierproxy.ErrInvalidCredentials,
		},
		"login failure is wrapped": {
			env:       map[string]string{envUsername: "alice", envPassword: "pw"},
			loginErr:  loginErr,
			wantErr:   loginErr,
			wantCalls: 1,
		},
		"success reports the user and site": {
			env:       map[string]string{envUsername: "alice", envPassword: "pw"},
			wantCalls: 1,
			wantOut:   "login to " + demoLoginURL + " succeeded as alice\n",
		},
		"a failing writer is reported": {
			env:       map[string]string{envUsername: "alice", envPassword: "pw"},
			out:       failingWriter{writeErr},
			wantErr:   writeErr,
			wantCalls: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			provider := &fakeProvider{loginErr: tc.loginErr}
			var buf bytes.Buffer
			var out io.Writer = &buf
			if tc.out != nil {
				out = tc.out
			}

			err := run(func(key string) string { return tc.env[key] }, provider, out)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if provider.calls != tc.wantCalls {
				t.Fatalf("got %d Login calls, want %d", provider.calls, tc.wantCalls)
			}
			if tc.wantCalls > 0 && (provider.username != tc.env[envUsername] || provider.password != tc.env[envPassword]) {
				t.Fatalf("Login got %q/%q, want the environment's %q/%q", provider.username, provider.password, tc.env[envUsername], tc.env[envPassword])
			}
			if got := buf.String(); got != tc.wantOut {
				t.Fatalf("got output %q, want %q", got, tc.wantOut)
			}
		})
	}
}

func TestRunMissingCredentialsNamesTheVariables(t *testing.T) {
	t.Parallel()

	err := run(func(string) string { return "" }, &fakeProvider{}, io.Discard)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, name := range []string{envUsername, envPassword} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error %q should name %s so the user knows what to set", err, name)
		}
	}
}
