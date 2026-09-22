package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/carrierproxy"
)

// fakeProvider records the URL and credentials run used and returns loginErr.
type fakeProvider struct {
	loginErr           error
	loginURL           string
	username, password string
}

func (f *fakeProvider) Login(username, password string) error {
	f.username, f.password = username, password
	return f.loginErr
}

func (f *fakeProvider) Policies() ([]carrierproxy.Policy, error) { return nil, nil }

func (f *fakeProvider) DocumentDownload(string) (io.ReadCloser, error) { return nil, nil }

func TestRun(t *testing.T) {
	loginErr := errors.New("site unreachable")
	creds := map[string]string{envUsername: "alice", envPassword: "pw"}

	cases := map[string]struct {
		env      map[string]string
		loginErr error
		wantErr  error
		wantURL  string
		wantOut  string
	}{
		"missing credentials": {env: map[string]string{envUsername: "alice"}, wantErr: carrierproxy.ErrInvalidCredentials},
		"login failure":       {env: creds, loginErr: loginErr, wantErr: loginErr, wantURL: defaultLoginURL},
		"default url":         {env: creds, wantURL: defaultLoginURL, wantOut: "login to " + defaultLoginURL + " succeeded as alice\n"},
		"custom url": {
			env:     map[string]string{envLoginURL: "https://carrier.example/login", envUsername: "alice", envPassword: "pw"},
			wantURL: "https://carrier.example/login",
			wantOut: "login to https://carrier.example/login succeeded as alice\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			provider := &fakeProvider{loginErr: tc.loginErr}
			var out bytes.Buffer

			err := run(func(k string) string { return tc.env[k] }, func(url string) carrierproxy.PolicyProvider {
				provider.loginURL = url
				return provider
			}, &out)

			assertRun(t, err, tc.wantErr, provider.loginURL, tc.wantURL, out.String(), tc.wantOut)
			if tc.wantURL != "" && (provider.username != "alice" || provider.password != "pw") {
				t.Fatalf("Login got %q/%q, want the environment's credentials", provider.username, provider.password)
			}
		})
	}
}

func assertRun(t *testing.T, err, wantErr error, gotURL, wantURL, gotOut, wantOut string) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
	if gotURL != wantURL {
		t.Fatalf("run() used %q, want %q", gotURL, wantURL)
	}
	if gotOut != wantOut {
		t.Fatalf("run() output = %q, want %q", gotOut, wantOut)
	}
}
