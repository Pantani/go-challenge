package main

import (
	"context"
	"log"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func main() {
	client := mock.NewClient(mock.Config{Bucket: mock.Bucket{Name: "local"}})
	if err := client.Purge(context.Background(), "example.txt"); err != nil {
		log.Fatal(err)
	}
	log.Print("filestore challenge ready")
}
