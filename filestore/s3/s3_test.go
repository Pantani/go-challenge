package s3_test

import (
	"context"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/s3"
)

func TestClientCopy(t *testing.T) {

	t.Parallel()

	var _ filestore.PresignedFileProvider = (*s3.Client)(nil)

	client := s3.NewClient(s3.Config{Bucket: "test-bucket"})

	if err := client.Copy(context.Background(), "source.txt", "destination.txt"); err != nil {
		t.Fatalf("unexpected error copying file: %v", err)
	}
}
