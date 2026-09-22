package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewEmailSvc confirms the production email service is built from the
// app config loaded in init.
func TestNewEmailSvc(t *testing.T) {

	t.Parallel()

	if newEmailSvc() == nil {
		t.Fatal("expected an email service")
	}
}

// TestServeEndToEnd boots the comms API exactly as main does (real
// listener, real sendgrid-backed provider from the app config) and drives
// every route over HTTP, including the validation paths, then confirms
// serve returns once its listener is closed.
func TestServeEndToEnd(t *testing.T) {

	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- serve(ln, newEmailSvc()) }()

	base := "http://" + ln.Addr().String()
	client := &http.Client{Timeout: 5 * time.Second}

	testCases := map[string]struct {
		method       string
		path         string
		body         string
		expectStatus int
	}{
		"vehicle":             {http.MethodPost, "/api/comms/add-policy-vehicle", `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`, http.StatusOK},
		"driver":              {http.MethodPost, "/api/comms/add-policy-driver", `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`, http.StatusOK},
		"address":             {http.MethodPost, "/api/comms/add-policy-address", `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`, http.StatusOK},
		"coverage with cc":    {http.MethodPost, "/api/comms/add-policy-coverage", `{"email_to":"foo@bar.com","email_cc":["cc@bar.com"],"message":{"foo":"bar"}}`, http.StatusOK},
		"coverage without cc": {http.MethodPost, "/api/comms/add-policy-coverage", `{"email_to":"foo@bar.com","message":{"foo":"bar"}}`, http.StatusOK},
		"wrong method":        {http.MethodGet, "/api/comms/add-policy-vehicle", ``, http.StatusMethodNotAllowed},
		"malformed body":      {http.MethodPost, "/api/comms/add-policy-driver", `{`, http.StatusBadRequest},
		"unknown route":       {http.MethodPost, "/api/comms/unknown", `{}`, http.StatusNotFound},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			req, err := http.NewRequest(tc.method, base+tc.path, strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.expectStatus {
				t.Fatalf("expected status %v but got %v", tc.expectStatus, resp.StatusCode)
			}
		})
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("expected serve to stop with net.ErrClosed, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after its listener was closed")
	}
}
