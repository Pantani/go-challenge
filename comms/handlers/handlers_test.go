package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

// TestMain silences the handlers' send-error log line so the failure-path
// cases do not clutter test output; TestSendErrorIsLoggedNotEchoed swaps
// in its own writer to assert on that line.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// erroringMailProvider is an email.MailProvider whose sends always fail,
// used to exercise each handler's email-service error path (which the
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

// handlerCase is one request/response expectation shared by every handler.
type handlerCase struct {
	method       string
	body         string
	sendErr      error
	expectStatus int
	expectBody   string // exact response body, including an empty success body
	expectTo     string // recorded To recipient on a 200
	expectCC     []string
	expectMsg    string
}

const validBody = `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`

// commonCases are the behaviours every comms handler must share: they are
// run verbatim against all four handlers.
func commonCases() map[string]handlerCase {
	return map[string]handlerCase{
		"pass": {
			method:       http.MethodPost,
			body:         validBody,
			expectStatus: http.StatusOK,
			expectTo:     "foo@bar.com",
			expectMsg:    `{"foo":"bar"}`,
		},
		"pass trims email_to": {
			method:       http.MethodPost,
			body:         `{"email_to":"  foo@bar.com \n","message":{"foo":"bar"}}`,
			expectStatus: http.StatusOK,
			expectTo:     "foo@bar.com",
			expectMsg:    `{"foo":"bar"}`,
		},
		"fail invalid method": {
			method:       http.MethodGet,
			body:         validBody,
			expectStatus: http.StatusMethodNotAllowed,
			expectBody:   "method not allowed\n",
		},
		"fail invalid payload": {
			method:       http.MethodPost,
			body:         `{invalid`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "invalid payload\n",
		},
		"fail wrong type for email_to": {
			method:       http.MethodPost,
			body:         `{"email_to":["foo@bar.com"],"message":{}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "invalid payload\n",
		},
		"fail trailing json value": {
			method:       http.MethodPost,
			body:         validBody + `{"email_to":"other@bar.com"}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "invalid payload: unexpected data after JSON value\n",
		},
		"fail trailing garbage": {
			method:       http.MethodPost,
			body:         validBody + ` garbage`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "invalid payload: unexpected data after JSON value\n",
		},
		"fail body too large": {
			method:       http.MethodPost,
			body:         `{"email_to":"foo@bar.com","message":{"pad":"` + strings.Repeat("x", 1<<20) + `"}}`,
			expectStatus: http.StatusRequestEntityTooLarge,
			expectBody:   "request body too large\n",
		},
		"fail oversized trailing data": {
			method:       http.MethodPost,
			body:         validBody + strings.Repeat(" ", 1<<20),
			expectStatus: http.StatusRequestEntityTooLarge,
			expectBody:   "request body too large\n",
		},
		"fail missing email_to": {
			method:       http.MethodPost,
			body:         `{"message":{"foo":"bar"}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "email_to: is required\n",
		},
		"fail blank email_to": {
			method:       http.MethodPost,
			body:         `{"email_to":"   ","message":{"foo":"bar"}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "email_to: is required\n",
		},
		"fail malformed email_to": {
			method:       http.MethodPost,
			body:         `{"email_to":"not-an-email","message":{"foo":"bar"}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "email_to: is not a valid email address\n",
		},
		"fail display-name email_to": {
			method:       http.MethodPost,
			body:         `{"email_to":"Foo <foo@bar.com>","message":{"foo":"bar"}}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "email_to: must be a bare email address\n",
		},
		"fail missing message": {
			method:       http.MethodPost,
			body:         `{"email_to":"foo@bar.com"}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "message: is required\n",
		},
		"fail null message": {
			method:       http.MethodPost,
			body:         `{"email_to":"foo@bar.com","message":null}`,
			expectStatus: http.StatusBadRequest,
			expectBody:   "message: is required\n",
		},
		"fail send error": {
			method:       http.MethodPost,
			body:         validBody,
			sendErr:      errors.New("secret provider detail"),
			expectStatus: http.StatusInternalServerError,
			expectBody:   "error sending email\n",
		},
	}
}

type deliveryCall struct {
	to      []string
	cc      []string
	message json.RawMessage
	tpl     email.TplID
}

type deliverySpy struct {
	send   []deliveryCall
	sendCC []deliveryCall
	err    error
}

func capturedCall(to, cc []string, message json.RawMessage, tpl email.TplID) deliveryCall {
	return deliveryCall{slices.Clone(to), slices.Clone(cc), slices.Clone(message), tpl}
}

func (p *deliverySpy) Send(to []string, message json.RawMessage, tpl email.TplID) error {
	p.send = append(p.send, capturedCall(to, nil, message, tpl))
	return p.err
}

func (p *deliverySpy) SendWithCC(to, cc []string, message json.RawMessage, tpl email.TplID) error {
	p.sendCC = append(p.sendCC, capturedCall(to, cc, message, tpl))
	return p.err
}

func assertEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", label, got, want)
	}
}

func assertHandlerResponse(t *testing.T, w *httptest.ResponseRecorder, tc handlerCase) {
	t.Helper()
	assertEqual(t, "status", w.Code, tc.expectStatus)
	assertEqual(t, "body", w.Body.String(), tc.expectBody)
	if tc.expectStatus == http.StatusMethodNotAllowed {
		assertEqual(t, "Allow", w.Header().Get("Allow"), http.MethodPost)
	}
}

func assertHandlerDelivery(t *testing.T, p *deliverySpy, tpl email.TplID, tc handlerCase) {
	t.Helper()
	if tc.sendErr != nil {
		assertEqual(t, "attempts", len(p.send)+len(p.sendCC), 1)
		return
	}
	if tc.expectStatus != http.StatusOK {
		assertEqual(t, "attempts", len(p.send)+len(p.sendCC), 0)
		return
	}
	want := []deliveryCall{capturedCall([]string{tc.expectTo}, tc.expectCC, json.RawMessage(tc.expectMsg), tpl)}
	if len(tc.expectCC) != 0 {
		assertEqual(t, "Send", p.send, []deliveryCall(nil))
		assertEqual(t, "SendWithCC", p.sendCC, want)
		return
	}
	assertEqual(t, "Send", p.send, want)
	assertEqual(t, "SendWithCC", p.sendCC, []deliveryCall(nil))
}

func runHandlerCase(t *testing.T, ctor func(email.MailProvider) http.HandlerFunc, tpl email.TplID, tc handlerCase) {
	t.Helper()
	p := &deliverySpy{err: tc.sendErr}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(tc.method, "/api/comms/"+string(tpl), strings.NewReader(tc.body))
	ctor(p)(w, req)
	assertHandlerResponse(t, w, tc)
	assertHandlerDelivery(t, p, tpl, tc)
}

func runHandlerCases(t *testing.T, ctor func(email.MailProvider) http.HandlerFunc, tpl email.TplID, cases map[string]handlerCase) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runHandlerCase(t, ctor, tpl, tc)
		})
	}
}

func TestAddPolicyVehicle(t *testing.T) {
	t.Parallel()
	runHandlerCases(t, handlers.AddPolicyVehicle, email.TplAddPolicyVehicle, commonCases())
}

func TestAddPolicyDriver(t *testing.T) {
	t.Parallel()
	runHandlerCases(t, handlers.AddPolicyDriver, email.TplAddPolicyDriver, commonCases())
}

func TestAddPolicyAddress(t *testing.T) {
	t.Parallel()
	runHandlerCases(t, handlers.AddPolicyAddress, email.TplAddPolicyAddress, commonCases())
}

// TestAddPolicyCoverage proves the new handler meets the shared contract and
// additionally the CC business rules: CC recipients are copied on the send,
// each CC entry is validated with its index, and the list is normalised
// (trimmed, de-duplicated case-insensitively, never repeating the To
// address) before delivery.
func TestAddPolicyCoverage(t *testing.T) {
	t.Parallel()

	cases := commonCases()

	cases["pass with cc"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["cc1@bar.com","cc2@bar.com"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectCC:     []string{"cc1@bar.com", "cc2@bar.com"},
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["pass with empty cc list"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":[],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["pass with null cc"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":null,"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["pass cc trimmed and deduplicated case-insensitively"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":[" cc@bar.com ","CC@bar.com","other@bar.com","cc@BAR.com"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectCC:     []string{"cc@bar.com", "other@bar.com"},
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["pass cc equal to email_to is dropped"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["Foo@Bar.com","cc@bar.com"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectCC:     []string{"cc@bar.com"},
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["pass cc collapsing entirely into email_to sends without cc"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["foo@bar.com","FOO@BAR.COM"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusOK,
		expectTo:     "foo@bar.com",
		expectMsg:    `{"foo":"bar"}`,
	}
	cases["fail blank cc entry"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["cc1@bar.com",""],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusBadRequest,
		expectBody:   "email_cc[1]: is required\n",
	}
	cases["fail malformed cc entry"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["nope"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusBadRequest,
		expectBody:   "email_cc[0]: is not a valid email address\n",
	}
	cases["fail display-name cc entry"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["cc1@bar.com","cc2@bar.com","CC <cc3@bar.com>"],"message":{"foo":"bar"}}`,
		expectStatus: http.StatusBadRequest,
		expectBody:   "email_cc[2]: must be a bare email address\n",
	}
	cases["fail wrong type for email_cc"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":"cc@bar.com","message":{"foo":"bar"}}`,
		expectStatus: http.StatusBadRequest,
		expectBody:   "invalid payload\n",
	}
	cases["fail send error with cc"] = handlerCase{
		method:       http.MethodPost,
		body:         `{"email_to":"foo@bar.com","email_cc":["cc@bar.com"],"message":{"foo":"bar"}}`,
		sendErr:      errors.New("secret provider detail"),
		expectStatus: http.StatusInternalServerError,
		expectBody:   "error sending email\n",
	}

	runHandlerCases(t, handlers.AddPolicyCoverage, email.TplAddPolicyCoverage, cases)
}

// TestSendErrorIsLoggedNotEchoed pins the 500 contract: the provider's error
// reaches the server log for operators but never the HTTP response body.
func TestSendErrorIsLoggedNotEchoed(t *testing.T) {
	// Not parallel: swaps the process-wide log output.

	var logs bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comms/add-policy-coverage", strings.NewReader(validBody))

	handlers.AddPolicyCoverage(erroringMailProvider{err: errors.New("secret provider detail")})(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %v but got %v", http.StatusInternalServerError, w.Code)
	}

	if strings.Contains(w.Body.String(), "secret provider detail") {
		t.Fatalf("provider error leaked into response body: %q", w.Body.String())
	}

	want := "handlers: add-policy-coverage: error sending email: secret provider detail\n"
	if logs.String() != want {
		t.Fatalf("expected log %q but got %q", want, logs.String())
	}
}
