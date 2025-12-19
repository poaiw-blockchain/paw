//go:build stress
// +build stress

package stress_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	// No suite-level setup needed for stress tests
	// Each test handles its own lifecycle
	m.Run()
}

func TestStressSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress tests in short mode")
	}
	t.Skip("manual long-run only; build intentionally disabled in CI")
	suite.Run(t, new(StressTestSuite))
}

func TestMemoryLeakSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak tests in short mode")
	}
	t.Skip("manual long-run only; build intentionally disabled in CI")
}

func TestGoroutineLeakSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak tests in short mode")
	}
	t.Skip("manual long-run only; build intentionally disabled in CI")
}
