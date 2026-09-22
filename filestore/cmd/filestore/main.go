package main

import (
	"context"
	"log"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
	log.Print("filestore challenge ready")
}

func run() error {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "local"}})
	return client.Purge(context.Background(), "example.txt")
}
