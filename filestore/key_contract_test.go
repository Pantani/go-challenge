package filestore_test

import (
	"context"
	"testing"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
)

func checkOpaqueObjects(t *testing.T, store filestore.PresignedFileProvider) {
	t.Helper()
	keys := []string{"", ".", "..", "/a", "a", "a//b", "a/b", "x/../y", "y", "../outside"}
	for index, key := range keys {
		mustSet(t, store, key, string(rune('A'+index)), "text/plain")
	}
	for index, key := range keys {
		assertFile(t, store, key, string(rune('A'+index)), "text/plain")
	}
}

func TestOpaqueObjectKeys(t *testing.T) {
	forEachProvider(t, checkOpaqueObjects)
}

func TestOpaqueTransferKeys(t *testing.T) {
	forEachProvider(t, func(t *testing.T, store filestore.PresignedFileProvider) {
		mustSet(t, store, "x/../source", "payload", "text/plain")
		if err := store.Copy(context.Background(), "x/../source", "/copy//key"); err != nil {
			t.Fatal(err)
		}
		if err := store.Move(context.Background(), "/copy//key", "./moved"); err != nil {
			t.Fatal(err)
		}
		assertFile(t, store, "./moved", "payload", "text/plain")
		assertMissing(t, store, "moved")
		assertMissing(t, store, "/copy//key")
	})
}
