package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email/mockemail"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

// erroringMailProvider is an email.MailProvider whose sends always fail,
// used to exercise AddPolicyCoverage's email-service error path (which the
// always-succeeding mockemail.Client cannot trigger).
type erroringMailProvider struct {
	err error
}

func (p erroringMailProvider) Send([]string, json.RawMessage, email.TplID) error {
	return p.err
}

func (p erroringMailProvider) SendWithCC([]string, []string, json.RawMessage, email.TplID) error {
	return p.err
}

func TestAddPolicyVehicle(t *testing.T) {

	t.Parallel()

	type testCase struct {
		method       string
		payload      handlers.AddPolicyVehicleReq
		expectTplID  email.TplID
		expectStatus int
	}

	testCases := map[string]testCase{
		"pass": {
			method: http.MethodPost,
			payload: handlers.AddPolicyVehicleReq{
				EmailTo: "foo@bar.com",
				Message: json.RawMessage(`{"foo":"bar"}`),
			},
			expectTplID:  email.TplAddPolicyVehicle,
			expectStatus: http.StatusOK,
		},
		"fail invalid method": {
			method:       http.MethodGet,
			payload:      handlers.AddPolicyVehicleReq{},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	testFactory := func(tc testCase) func(*testing.T) {
		return func(t *testing.T) {

			payload, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("could nor marshal payload to json: %v", err)
			}

			testEmail := mockemail.NewClient()
			defer testEmail.FlushSendLogs()

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/comms/add-policy-vehicle", bytes.NewReader(payload))

			handlers.AddPolicyVehicle(testEmail)(w, req)

			resp := w.Result()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}

			if resp.StatusCode == http.StatusOK {

				if testEmail.SendLogs().IsEmpty() {
					t.Fatalf("expected email log but got empty")
				}

				lastEmail := testEmail.SendLogs().Last()

				if lastEmail.ExtractTo() != tc.payload.EmailTo {
					t.Fatalf("expected to %v but got %v", tc.payload.EmailTo, lastEmail.ExtractTo())
				}

				if string(lastEmail.ExtractMessage()) != string(tc.payload.Message) {
					t.Fatalf("expected message %v but got %v", tc.payload.Message, lastEmail.ExtractMessage())
				}

				if lastEmail.ExtractTplID() != tc.expectTplID {
					t.Fatalf("expected tpl %v but got %v", tc.expectTplID, lastEmail.ExtractTplID())
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, testFactory(tc))
	}
}

func TestAddPolicyDriver(t *testing.T) {

	t.Parallel()

	type testCase struct {
		method       string
		payload      handlers.AddPolicyDriverReq
		expectTplID  email.TplID
		expectStatus int
	}

	testCases := map[string]testCase{
		"pass": {
			method: http.MethodPost,
			payload: handlers.AddPolicyDriverReq{
				EmailTo: "foo@bar.com",
				Message: json.RawMessage(`{"foo":"bar"}`),
			},
			expectTplID:  email.TplAddPolicyDriver,
			expectStatus: http.StatusOK,
		},
		"fail invalid method": {
			method:       http.MethodGet,
			payload:      handlers.AddPolicyDriverReq{},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	testFactory := func(tc testCase) func(*testing.T) {
		return func(t *testing.T) {

			payload, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("could nor marshal payload to json: %v", err)
			}

			testEmail := mockemail.NewClient()
			defer testEmail.FlushSendLogs()

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/comms/add-policy-driver", bytes.NewReader(payload))

			handlers.AddPolicyDriver(testEmail)(w, req)

			resp := w.Result()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}

			if resp.StatusCode == http.StatusOK {

				if testEmail.SendLogs().IsEmpty() {
					t.Fatalf("expected email log but got empty")
				}

				lastEmail := testEmail.SendLogs().Last()

				if lastEmail.ExtractTo() != tc.payload.EmailTo {
					t.Fatalf("expected to %v but got %v", tc.payload.EmailTo, lastEmail.ExtractTo())
				}

				if string(lastEmail.ExtractMessage()) != string(tc.payload.Message) {
					t.Fatalf("expected message %v but got %v", tc.payload.Message, lastEmail.ExtractMessage())
				}

				if lastEmail.ExtractTplID() != tc.expectTplID {
					t.Fatalf("expected tpl %v but got %v", tc.expectTplID, lastEmail.ExtractTplID())
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, testFactory(tc))
	}
}

func TestAddPolicyAddress(t *testing.T) {

	t.Parallel()

	type testCase struct {
		method       string
		payload      handlers.AddPolicyAddressReq
		expectTplID  email.TplID
		expectStatus int
	}

	testCases := map[string]testCase{
		"pass": {
			method: http.MethodPost,
			payload: handlers.AddPolicyAddressReq{
				EmailTo: "foo@bar.com",
				Message: json.RawMessage(`{"foo":"bar"}`),
			},
			expectTplID:  email.TplAddPolicyAddress,
			expectStatus: http.StatusOK,
		},
		"fail invalid method": {
			method:       http.MethodGet,
			payload:      handlers.AddPolicyAddressReq{},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	testFactory := func(tc testCase) func(*testing.T) {
		return func(t *testing.T) {

			payload, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("could nor marshal payload to json: %v", err)
			}

			testEmail := mockemail.NewClient()
			defer testEmail.FlushSendLogs()

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/comms/add-policy-address", bytes.NewReader(payload))

			handlers.AddPolicyAddress(testEmail)(w, req)

			resp := w.Result()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}

			if resp.StatusCode == http.StatusOK {

				if testEmail.SendLogs().IsEmpty() {
					t.Fatalf("expected email log but got empty")
				}

				lastEmail := testEmail.SendLogs().Last()

				if lastEmail.ExtractTo() != tc.payload.EmailTo {
					t.Fatalf("expected to %v but got %v", tc.payload.EmailTo, lastEmail.ExtractTo())
				}

				if string(lastEmail.ExtractMessage()) != string(tc.payload.Message) {
					t.Fatalf("expected message %v but got %v", tc.payload.Message, lastEmail.ExtractMessage())
				}

				if lastEmail.ExtractTplID() != tc.expectTplID {
					t.Fatalf("expected tpl %v but got %v", tc.expectTplID, lastEmail.ExtractTplID())
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, testFactory(tc))
	}
}

func TestAddPolicyCoverage(t *testing.T) {

	t.Parallel()

	type testCase struct {
		method       string
		rawBody      []byte
		payload      handlers.AddPolicyCoverageReq
		sendErr      error
		expectStatus int
	}

	testCases := map[string]testCase{
		"pass": {
			method: http.MethodPost,
			payload: handlers.AddPolicyCoverageReq{
				EmailTo: "foo@bar.com",
				EmailCC: []string{"cc1@bar.com", "cc2@bar.com"},
				Message: json.RawMessage(`{"foo":"bar"}`),
			},
			expectStatus: http.StatusOK,
		},
		"fail invalid method": {
			method:       http.MethodGet,
			payload:      handlers.AddPolicyCoverageReq{},
			expectStatus: http.StatusMethodNotAllowed,
		},
		"fail invalid payload": {
			method:       http.MethodPost,
			rawBody:      []byte(`{invalid`),
			expectStatus: http.StatusBadRequest,
		},
		"fail send error": {
			method: http.MethodPost,
			payload: handlers.AddPolicyCoverageReq{
				EmailTo: "foo@bar.com",
				Message: json.RawMessage(`{"foo":"bar"}`),
			},
			sendErr:      errors.New("service unavailable"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	testFactory := func(tc testCase) func(*testing.T) {
		return func(t *testing.T) {

			body := tc.rawBody
			if body == nil {
				marshaled, err := json.Marshal(tc.payload)
				if err != nil {
					t.Fatalf("could nor marshal payload to json: %v", err)
				}
				body = marshaled
			}

			testEmail := mockemail.NewClient()
			defer testEmail.FlushSendLogs()

			if !testEmail.SendLogs().IsEmpty() {
				t.Fatalf("expected a fresh mock client to have no recorded sends")
			}

			var provider email.MailProvider = testEmail
			if tc.sendErr != nil {
				provider = erroringMailProvider{err: tc.sendErr}
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/comms/add-policy-coverage", bytes.NewReader(body))

			handlers.AddPolicyCoverage(provider)(w, req)

			resp := w.Result()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}

			if resp.StatusCode == http.StatusOK {

				if testEmail.SendLogs().IsEmpty() {
					t.Fatalf("expected email log but got empty")
				}

				lastEmail := testEmail.SendLogs().Last()

				if lastEmail.ExtractTo() != tc.payload.EmailTo {
					t.Fatalf("expected to %v but got %v", tc.payload.EmailTo, lastEmail.ExtractTo())
				}

				if !reflect.DeepEqual(lastEmail.ExtractCC(), tc.payload.EmailCC) {
					t.Fatalf("expected cc %v but got %v", tc.payload.EmailCC, lastEmail.ExtractCC())
				}

				if string(lastEmail.ExtractMessage()) != string(tc.payload.Message) {
					t.Fatalf("expected message %v but got %v", tc.payload.Message, lastEmail.ExtractMessage())
				}

				if lastEmail.ExtractTplID() != email.TplAddPolicyCoverage {
					t.Fatalf("expected tpl %v but got %v", email.TplAddPolicyCoverage, lastEmail.ExtractTplID())
				}
			}
		}
	}

	for name, tc := range testCases {
		t.Run(name, testFactory(tc))
	}
}
