package chaos_test

import (
	"os"
	"testing"
)

// TestMain enables all chaos tests - network harness has been updated.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
