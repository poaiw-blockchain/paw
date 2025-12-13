package simulation

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Run the simulation tests
	os.Exit(m.Run())
}
