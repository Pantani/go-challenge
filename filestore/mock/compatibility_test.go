package mock

import "testing"

func TestLegacyPositionalConfig(t *testing.T) {
	client := NewClient(Config{Bucket{"legacy", nil}, "base"})
	if client.basePath != "base" {
		t.Fatalf("base path = %q", client.basePath)
	}
}
