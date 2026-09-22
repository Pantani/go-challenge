package main

import "testing"

// main has no flags or CLI args and its only failure path is a deterministic
// success against the in-memory mock, so calling it directly is safe here.
func TestMainSmoke(t *testing.T) {
	main()
}
