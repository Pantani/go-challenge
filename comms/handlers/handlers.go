// Package handlers exposes the HTTP handlers of the comms API. Every handler
// accepts a JSON POST payload, validates it, and forwards a templated email
// through the injected email.MailProvider.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gloveboxhq/glovebox-go-code-challenge/comms/email"
)

// maxBodyBytes caps the size of an accepted request body.
const maxBodyBytes = 1 << 20 // 1 MiB

// AddPolicyVehicle handles the add-policy-vehicle operation.
func AddPolicyVehicle(emailsvc email.MailProvider) http.HandlerFunc {
	return sendHandler(emailsvc, email.TplAddPolicyVehicle, func(p *AddPolicyVehicleReq) envelope {
		return envelope{to: p.EmailTo, message: p.Message}
	})
}

// AddPolicyDriver handles the add-policy-driver operation.
func AddPolicyDriver(emailsvc email.MailProvider) http.HandlerFunc {
	return sendHandler(emailsvc, email.TplAddPolicyDriver, func(p *AddPolicyDriverReq) envelope {
		return envelope{to: p.EmailTo, message: p.Message}
	})
}

// AddPolicyAddress handles the add-policy-address operation.
func AddPolicyAddress(emailsvc email.MailProvider) http.HandlerFunc {
	return sendHandler(emailsvc, email.TplAddPolicyAddress, func(p *AddPolicyAddressReq) envelope {
		return envelope{to: p.EmailTo, message: p.Message}
	})
}

// AddPolicyCoverage handles the add-policy-coverage operation. It follows
// the same To/Message contract as the other handlers, additionally copying
// EmailCC recipients on the notification email.
func AddPolicyCoverage(emailsvc email.MailProvider) http.HandlerFunc {
	return sendHandler(emailsvc, email.TplAddPolicyCoverage, func(p *AddPolicyCoverageReq) envelope {
		return envelope{to: p.EmailTo, cc: p.EmailCC, message: p.Message}
	})
}

// httpError is a client-facing request failure paired with the HTTP status
// it should be reported with.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string {
	return e.msg
}

// sendHandler is the pipeline every comms operation shares: method check,
// bounded JSON decode, validation, normalisation and delivery. extract maps
// the operation's decoded payload type onto the provider-agnostic envelope.
func sendHandler[T any](emailsvc email.MailProvider, tpl email.TplID, extract func(*T) envelope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		env, herr := parseRequest(w, r, extract)
		if herr != nil {
			http.Error(w, herr.Error(), herr.status)
			return
		}

		if err := env.deliver(emailsvc, tpl); err != nil {
			// Provider failures are an operator concern: log the detail,
			// never echo it back to the caller.
			log.Printf("handlers: %s: error sending email: %v", tpl, err)
			http.Error(w, "error sending email", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// parseRequest decodes and validates the request body, returning the
// normalised envelope ready for delivery.
func parseRequest[T any](w http.ResponseWriter, r *http.Request, extract func(*T) envelope) (envelope, *httpError) {

	var payload T
	if herr := decodeJSON(w, r, &payload); herr != nil {
		return envelope{}, herr
	}

	env := extract(&payload)
	if err := env.validate(); err != nil {
		return envelope{}, &httpError{status: http.StatusBadRequest, msg: err.Error()}
	}

	return env.normalize(), nil
}

// decodeJSON decodes exactly one JSON value from the request body into v,
// rejecting oversized bodies and trailing data.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) *httpError {

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return decodeError(err)
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return trailingDataError(err)
	}

	return nil
}

// trailingDataError classifies a failure to reach EOF after the first JSON
// value: an oversized body is still 413, anything else is trailing data.
func trailingDataError(err error) *httpError {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return decodeError(err)
	}
	return &httpError{status: http.StatusBadRequest, msg: "invalid payload: unexpected data after JSON value"}
}

// decodeError translates a JSON decode failure into the client-facing error.
func decodeError(err error) *httpError {

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return &httpError{status: http.StatusRequestEntityTooLarge, msg: "request body too large"}
	}

	return &httpError{status: http.StatusBadRequest, msg: "invalid payload"}
}

// envelope is the provider-agnostic view of a request payload.
type envelope struct {
	to      string
	cc      []string
	message json.RawMessage
}

// validate checks the envelope is deliverable, naming the offending field
// in the returned error.
func (e envelope) validate() error {

	if err := validateAddress(e.to); err != nil {
		return fmt.Errorf("email_to: %w", err)
	}

	for i, addr := range e.cc {
		if err := validateAddress(addr); err != nil {
			return fmt.Errorf("email_cc[%d]: %w", i, err)
		}
	}

	if isNullJSON(e.message) {
		return errors.New("message: is required")
	}

	return nil
}

// normalize trims the To address and reduces the CC list to unique
// addresses (case-insensitively) that differ from To, preserving order.
func (e envelope) normalize() envelope {
	e.to = strings.TrimSpace(e.to)
	e.cc = normalizeCC(e.cc, e.to)
	return e
}

// deliver sends the envelope through svc, using SendWithCC only when there
// are CC recipients left after normalization.
func (e envelope) deliver(svc email.MailProvider, tpl email.TplID) error {
	if len(e.cc) == 0 {
		return svc.Send([]string{e.to}, e.message, tpl)
	}
	return svc.SendWithCC([]string{e.to}, e.cc, e.message, tpl)
}

// normalizeCC returns cc trimmed, deduplicated case-insensitively and with
// any entry equal to to removed. It returns nil when nothing remains.
func normalizeCC(cc []string, to string) []string {

	if len(cc) == 0 {
		return nil
	}

	seen := map[string]struct{}{strings.ToLower(to): {}}
	out := make([]string, 0, len(cc))

	for _, addr := range cc {
		addr = strings.TrimSpace(addr)
		key := strings.ToLower(addr)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, addr)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// validateAddress requires addr to be a single bare email address (no
// display name), ignoring surrounding whitespace.
func validateAddress(addr string) error {

	addr = strings.TrimSpace(addr)
	if addr == "" {
		return errors.New("is required")
	}

	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return errors.New("is not a valid email address")
	}

	if parsed.Address != addr {
		return errors.New("must be a bare email address")
	}

	return nil
}

// isNullJSON reports whether raw carries no JSON value: absent, blank, or
// the literal null.
func isNullJSON(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
