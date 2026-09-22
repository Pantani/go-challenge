package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
)

// TestRoutes is an integration test: it drives newMux's routes over a real
// httptest server (real listener, real HTTP client) rather than invoking a
// handler func directly, so it proves the route registration in newMux
// dispatches each path to the right handler end to end.
func TestRoutes(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		path        string
		body        string
		expectTplID email.TplID
	}{
		"add-policy-vehicle": {
			path:        "/api/comms/add-policy-vehicle",
			body:        `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`,
			expectTplID: email.TplAddPolicyVehicle,
		},
		"add-policy-driver": {
			path:        "/api/comms/add-policy-driver",
			body:        `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`,
			expectTplID: email.TplAddPolicyDriver,
		},
		"add-policy-address": {
			path:        "/api/comms/add-policy-address",
			body:        `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`,
			expectTplID: email.TplAddPolicyAddress,
		},
		"add-policy-coverage": {
			path:        "/api/comms/add-policy-coverage",
			body:        `{"email_to":"foo@bar.com","email_cc":["cc@bar.com"],"message":{"foo":"bar"}}`,
			expectTplID: email.TplAddPolicyCoverage,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			testEmail := mockemail.NewClient()
			srv := httptest.NewServer(newMux(testEmail))
			defer srv.Close()

			resp, err := http.Post(srv.URL+tc.path, "application/json", strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status %v but got %v", http.StatusOK, resp.StatusCode)
			}

			if testEmail.SendLogs().IsEmpty() {
				t.Fatalf("expected an email to be sent")
			}

			if got := testEmail.SendLogs().Last().ExtractTplID(); got != tc.expectTplID {
				t.Fatalf("expected tpl %v but got %v", tc.expectTplID, got)
			}
		})
	}
}

// TestRoutesNotFound confirms newMux doesn't register anything beyond the
// four comms routes.
func TestRoutesNotFound(t *testing.T) {

	t.Parallel()

	srv := httptest.NewServer(newMux(mockemail.NewClient()))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/comms/unknown-route", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %v but got %v", http.StatusNotFound, resp.StatusCode)
	}
}
