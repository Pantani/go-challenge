package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/handlers"
)

func paddedRequest(size int, trailing bool) string {
	if trailing {
		return validBody + strings.Repeat(" ", size-len(validBody))
	}
	const prefix = `{"email_to":"foo@bar.com","message":"`
	const suffix = `"}`
	return prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix
}

func boundaryCases() map[string]handlerCase {
	cases := make(map[string]handlerCase, 6)
	for _, trailing := range []bool{false, true} {
		for _, size := range []int{(1 << 20) - 1, 1 << 20, (1 << 20) + 1} {
			body := paddedRequest(size, trailing)
			cases[fmt.Sprintf("bytes=%d/trailing=%t", size, trailing)] = boundaryCase(body, size, trailing)
		}
	}
	return cases
}

func boundaryCase(body string, size int, trailing bool) handlerCase {
	tc := handlerCase{method: http.MethodPost, body: body, expectStatus: http.StatusOK, expectTo: "foo@bar.com"}
	if size > 1<<20 {
		tc.expectStatus = http.StatusRequestEntityTooLarge
		tc.expectBody = "request body too large\n"
		return tc
	}
	tc.expectMsg = `{"foo":"bar"}`
	if !trailing {
		const prefix = `{"email_to":"foo@bar.com","message":`
		tc.expectMsg = body[len(prefix) : len(body)-1]
	}
	return tc
}

func TestPayloadByteBoundaries(t *testing.T) {
	tests := []struct {
		name string
		ctor func(email.MailProvider) http.HandlerFunc
		tpl  email.TplID
	}{
		{"vehicle", handlers.AddPolicyVehicle, email.TplAddPolicyVehicle},
		{"driver", handlers.AddPolicyDriver, email.TplAddPolicyDriver},
		{"address", handlers.AddPolicyAddress, email.TplAddPolicyAddress},
		{"coverage", handlers.AddPolicyCoverage, email.TplAddPolicyCoverage},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { runHandlerCases(t, tc.ctor, tc.tpl, boundaryCases()) })
	}
}

func TestBoundaryFixtureLengths(t *testing.T) {
	for _, size := range []int{(1 << 20) - 1, 1 << 20, (1 << 20) + 1} {
		assertEqual(t, "message-padded byte count", len(paddedRequest(size, false)), size)
		assertEqual(t, "whitespace-padded byte count", len(paddedRequest(size, true)), size)
	}
}

func TestAcceptedJSONPolicy(t *testing.T) {
	cases := map[string]string{
		"unknown field":                  `{"email_to":"foo@bar.com","message":{},"extra":true}`,
		"trailing whitespace":            validBody + " \n\t",
		"string message":                 `{"email_to":"foo@bar.com","message":"text"}`,
		"number message":                 `{"email_to":"foo@bar.com","message":7}`,
		"boolean message":                `{"email_to":"foo@bar.com","message":false}`,
		"array message":                  `{"email_to":"foo@bar.com","message":[1]}`,
		"duplicate recipient last value": `{"email_to":"first@bar.com","email_to":"foo@bar.com","message":{}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { assertAcceptedJSON(t, body, "") })
	}
}

func TestContentTypeIsNotRequired(t *testing.T) {
	for _, contentType := range []string{"", "text/plain", "application/json"} {
		t.Run(fmt.Sprintf("type=%q", contentType), func(t *testing.T) {
			assertAcceptedJSON(t, validBody, contentType)
		})
	}
}

func assertAcceptedJSON(t *testing.T, body, contentType string) {
	t.Helper()
	p := &deliverySpy{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	handlers.AddPolicyCoverage(p)(w, req)
	assertEqual(t, "status", w.Code, http.StatusOK)
	assertEqual(t, "empty success body", w.Body.String(), "")
	assertEqual(t, "Send count", len(p.send), 1)
	assertEqual(t, "SendWithCC count", len(p.sendCC), 0)
	assertEqual(t, "recipient", p.send[0].to, []string{"foo@bar.com"})
}
