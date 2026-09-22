package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/sendgrid"
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
		expectCC    []string
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
			expectCC:    []string{"cc@bar.com"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

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

			last := testEmail.SendLogs().Last()
			if last == nil {
				t.Fatalf("expected an email to be sent")
			}

			if got := last.ExtractTplID(); got != tc.expectTplID {
				t.Fatalf("expected tpl %v but got %v", tc.expectTplID, got)
			}

			if got := last.ExtractCC(); strings.Join(got, ",") != strings.Join(tc.expectCC, ",") {
				t.Fatalf("expected cc %v but got %v", tc.expectCC, got)
			}
		})
	}
}

// TestRoutesRejections confirms the mux surfaces handler-level rejections
// (wrong method, bad payload) and 404s for anything outside the four comms
// routes.
func TestRoutesRejections(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		method       string
		path         string
		body         string
		expectStatus int
		expectBody   string
	}{
		"unknown route": {
			method:       http.MethodPost,
			path:         "/api/comms/unknown-route",
			body:         `{}`,
			expectStatus: http.StatusNotFound,
		},
		"wrong method": {
			method:       http.MethodGet,
			path:         "/api/comms/add-policy-coverage",
			expectStatus: http.StatusMethodNotAllowed,
			expectBody:   "method not allowed\n",
		},
		"invalid cc entry": {
			method:       http.MethodPost,
			path:         "/api/comms/add-policy-coverage",
			body:         `{"email_to":"foo@bar.com","email_cc":["bad"],"message":{}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "email_cc[0]: is not a valid email address\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testEmail := mockemail.NewClient()
			srv := httptest.NewServer(newMux(testEmail))
			defer srv.Close()

			req, err := http.NewRequest(tc.method, srv.URL+tc.path, strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("building request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			if tc.expectBody != "" && string(body) != tc.expectBody {
				t.Fatalf("expected body %q but got %q", tc.expectBody, body)
			}

			if !testEmail.SendLogs().IsEmpty() {
				t.Fatalf("expected no email to be sent but got %v", testEmail.SendLogs())
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		env    map[string]string
		expect config
	}{
		"defaults when unset": {
			env: map[string]string{},
			expect: config{
				addr:  ":8090",
				email: sendgrid.Config{APIKey: "FOOBAR123XYZ", FromName: "Foo Bar", FromAddress: "foo@bar.com"},
			},
		},
		"defaults when empty": {
			env: map[string]string{
				"COMMS_ADDR":               "",
				"COMMS_SENDGRID_API_KEY":   "",
				"COMMS_EMAIL_FROM_NAME":    "",
				"COMMS_EMAIL_FROM_ADDRESS": "",
			},
			expect: config{
				addr:  ":8090",
				email: sendgrid.Config{APIKey: "FOOBAR123XYZ", FromName: "Foo Bar", FromAddress: "foo@bar.com"},
			},
		},
		"overrides": {
			env: map[string]string{
				"COMMS_ADDR":               "127.0.0.1:9000",
				"COMMS_SENDGRID_API_KEY":   "SG.real",
				"COMMS_EMAIL_FROM_NAME":    "Comms",
				"COMMS_EMAIL_FROM_ADDRESS": "comms@example.com",
			},
			expect: config{
				addr:  "127.0.0.1:9000",
				email: sendgrid.Config{APIKey: "SG.real", FromName: "Comms", FromAddress: "comms@example.com"},
			},
		},
		"partial override keeps other defaults": {
			env: map[string]string{"COMMS_ADDR": ":1234"},
			expect: config{
				addr:  ":1234",
				email: sendgrid.Config{APIKey: "FOOBAR123XYZ", FromName: "Foo Bar", FromAddress: "foo@bar.com"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := loadConfig(func(k string) string { return tc.env[k] })

			if got != tc.expect {
				t.Fatalf("expected %+v but got %+v", tc.expect, got)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	mux := newMux(mockemail.NewClient())
	srv := newServer(":0", mux)

	if srv.Addr != ":0" {
		t.Fatalf("expected addr %q but got %q", ":0", srv.Addr)
	}
	if srv.Handler != mux {
		t.Fatalf("expected the mux to be installed as the handler")
	}

	for name, d := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if d <= 0 {
			t.Fatalf("expected %s to be set", name)
		}
	}
}

// TestRunReturnsListenError covers the wiring in run without a routable
// listener: the port is already held by the test, so ListenAndServe fails
// immediately and run must surface that error rather than swallow it.
func TestRunReturnsListenError(t *testing.T) {
	// Not parallel: silences the process-wide logger for run's startup line.

	prev := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(prev) })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	defer func() { _ = ln.Close() }()

	cfg := loadConfig(func(string) string { return "" })
	cfg.addr = ln.Addr().String()

	err = run(cfg)

	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("expected address-in-use error but got %v", err)
	}
}
