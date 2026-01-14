//go:build chaos

package chaos_test

import (
	"os"
	"testing"
)

// TestMain enables all chaos tests - network harness has been updated.
// These tests require special infrastructure and are excluded from regular CI.
// Run with: go test -tags=chaos ./tests/chaos/...
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
