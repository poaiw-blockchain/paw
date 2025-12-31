// Package performance contains PERF-1 performance baseline tests
// - PERF-1.1: Swap latency baseline (<100ms)
// - PERF-1.2: Gas baseline for pool operations
// - PERF-1.3: Concurrent swap stress tests (100+ TPS)
// - PERF-1.4: Memory profiling for 1000+ pools
package performance

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
