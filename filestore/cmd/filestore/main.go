// Command filestore is a small demo of the file store abstraction: it stores
// a file in the in-memory provider, copies it to a new key and reads the
// copy back through the provider-neutral interface.
package main

import (
	"context"
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
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	log.Printf("copied %s to %s: %d bytes (%s)", src, dst, len(data), contentType)
	return body.Close()
}
