package security_test

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Println("Skipping security suite in short mode")
		os.Exit(0)
	}

	os.Exit(m.Run())
}
