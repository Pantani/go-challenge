// Command filestore is a small demo of the file store abstraction: it stores
// a file in the in-memory provider, copies it to a new key and reads the
// copy back through the provider-neutral interface.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func main() {
	store := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "local"}})
	if err := run(context.Background(), store); err != nil {
		log.Fatal(err)
	}
	log.Print("filestore challenge ready")
}

func run(ctx context.Context, store filestore.FileProvider) error {
	const src, dst = "example.txt", "example-copy.txt"

	if err := store.Set(ctx, src, []byte("hello, filestore"), "text/plain"); err != nil {
		return err
	}
	if err := store.Copy(ctx, src, dst); err != nil {
		return err
	}
	body, contentType, err := store.Get(ctx, dst)
	if err != nil {
		return err
	}
	data, err := readAndClose(body)
	if err != nil {
		return err
	}
	log.Printf("copied %s to %s: %d bytes (%s)", src, dst, len(data), contentType)
	return nil
}

func readAndClose(body io.ReadCloser) ([]byte, error) {
	data, readErr := io.ReadAll(body)
	closeErr := body.Close()
	return data, errors.Join(readerError("read", readErr), readerError("close", closeErr))
}

func readerError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("filestore demo: %s body: %w", op, err)
}
