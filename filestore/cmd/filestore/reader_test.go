package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

type observedBody struct {
	io.Reader
	closeErr error
	closes   int
}

func (b *observedBody) Close() error {
	b.closes++
	return b.closeErr
}

type bodyStore struct {
	filestore.FileProvider
	body io.ReadCloser
}

func (s bodyStore) Get(context.Context, string) (io.ReadCloser, string, error) {
	return s.body, "text/plain", nil
}

func assertReaderCause(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want cause %v", got, want)
	}
}

func TestRunClosesBody(t *testing.T) {
	readErr := errors.New("read failure")
	closeErr := errors.New("close failure")
	cases := []struct {
		name     string
		reader   io.Reader
		closeErr error
		causes   []error
	}{
		{"success", strings.NewReader("data"), nil, []error{nil}},
		{"read failure", iotest.ErrReader(readErr), nil, []error{readErr}},
		{"close failure", strings.NewReader("data"), closeErr, []error{closeErr}},
		{"combined failure", iotest.ErrReader(readErr), closeErr, []error{readErr, closeErr}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &observedBody{Reader: tc.reader, closeErr: tc.closeErr}
			store := bodyStore{FileProvider: mock.NewClient(mock.Config{}), body: body}
			err := run(context.Background(), store)
			if body.closes != 1 {
				t.Fatalf("Close calls = %d, want 1", body.closes)
			}
			for _, cause := range tc.causes {
				assertReaderCause(t, err, cause)
			}
		})
	}
}
