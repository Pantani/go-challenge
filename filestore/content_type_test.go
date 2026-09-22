package filestore_test

import (
	"context"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

// checkContentType catches a provider that loses the default or an explicit
// content type while storing or transferring an object.
func checkContentType(t *testing.T, store filestore.PresignedFileProvider, input, want string) {
	t.Helper()
	mustSet(t, store, "source", "data", input)
	assertFile(t, store, "source", "data", want)
	if err := store.Copy(context.Background(), "source", "copy"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, store, "copy", "data", want)
	if err := store.Move(context.Background(), "copy", "moved"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, store, "moved", "data", want)
}

func TestContentTypeContract(t *testing.T) {
	for _, input := range []string{"", "text/plain", "application/custom"} {
		t.Run(input, func(t *testing.T) {
			want := input
			if want == "" {
				want = "application/octet-stream"
			}
			forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
				checkContentType(t, store, input, want)
			})
		})
	}
}
