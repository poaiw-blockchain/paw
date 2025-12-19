//go:build stress
// +build stress

package stress_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/testutil/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	OperationsCompleted atomic.Uint64
	OperationsFailed    atomic.Uint64
	TotalLatencyNs      atomic.Uint64
	MaxLatencyNs        atomic.Uint64
	MinLatencyNs        atomic.Uint64
}

// RecordOperation records an operation result
func (tm *TestMetrics) RecordOperation(latencyNs uint64, success bool) {
	if success {
		tm.OperationsCompleted.Add(1)
	} else {
		tm.OperationsFailed.Add(1)
	}

	tm.TotalLatencyNs.Add(latencyNs)

	// Update max latency
	for {
		current := tm.MaxLatencyNs.Load()
		if latencyNs <= current || tm.MaxLatencyNs.CompareAndSwap(current, latencyNs) {
			break
		}
	}

	// Update min latency (if not zero)
	for {
		current := tm.MinLatencyNs.Load()
		if current == 0 {
			if tm.MinLatencyNs.CompareAndSwap(0, latencyNs) {
				break
			}
			continue
		}
		if latencyNs >= current || tm.MinLatencyNs.CompareAndSwap(current, latencyNs) {
			break
		}
	}
}

// GetStats returns formatted statistics
func (tm *TestMetrics) GetStats() string {
	completed := tm.OperationsCompleted.Load()
	failed := tm.OperationsFailed.Load()
	total := completed + failed

	if total == 0 {
		return "No operations recorded"
	}

	avgLatency := time.Duration(tm.TotalLatencyNs.Load() / completed)
	maxLatency := time.Duration(tm.MaxLatencyNs.Load())
	minLatency := time.Duration(tm.MinLatencyNs.Load())

	successRate := float64(completed) / float64(total) * 100

	return fmt.Sprintf(`Operation Statistics:
  Total: %d (Success: %d, Failed: %d)
  Success Rate: %.2f%%
  Latency: avg=%v, min=%v, max=%v`,
		total, completed, failed,
		successRate,
		avgLatency, minLatency, maxLatency,
	)
}

// StressTestApp wraps the PAW app for stress testing
type StressTestApp struct {
	App     *app.PAWApp
	Ctx     sdk.Context
	Cleanup func()
}

// NewStressTestApp creates a new app instance for stress testing
func NewStressTestApp(t *testing.T) *StressTestApp {
	t.Helper()

	db := dbm.NewMemDB()
	testApp, ctx := keeper.SetupTestApp(t)

	return &StressTestApp{
		App: testApp,
		Ctx: ctx,
		Cleanup: func() {
			if db != nil {
				db.Close()
			}
		},
	}
}

// NewLightweightStressTestApp creates a lightweight app for high-volume tests
func NewLightweightStressTestApp(t *testing.T) *StressTestApp {
	t.Helper()

	testApp, ctx := keeper.SetupTestApp(t)

	return &StressTestApp{
		App:     testApp,
		Ctx:     ctx,
		Cleanup: func() {},
	}
}

// WorkloadConfig defines the parameters for a stress test workload
type WorkloadConfig struct {
	Duration          time.Duration
	OperationsPerSec  int
	Concurrency       int
	RampUpDuration    time.Duration
	ReportingInterval time.Duration
}

// DefaultWorkloadConfig returns default workload settings
func DefaultWorkloadConfig() WorkloadConfig {
	return WorkloadConfig{
		Duration:          1 * time.Hour,
		OperationsPerSec:  100,
		Concurrency:       10,
		RampUpDuration:    1 * time.Minute,
		ReportingInterval: 5 * time.Minute,
	}
}

// EnduranceWorkloadConfig returns settings for 6-hour endurance test
func EnduranceWorkloadConfig() WorkloadConfig {
	return WorkloadConfig{
		Duration:          6 * time.Hour,
		OperationsPerSec:  50,
		Concurrency:       20,
		RampUpDuration:    5 * time.Minute,
		ReportingInterval: 15 * time.Minute,
	}
}

// MarathonWorkloadConfig returns settings for 24-hour marathon test
func MarathonWorkloadConfig() WorkloadConfig {
	return WorkloadConfig{
		Duration:          24 * time.Hour,
		OperationsPerSec:  30,
		Concurrency:       15,
		RampUpDuration:    10 * time.Minute,
		ReportingInterval: 30 * time.Minute,
	}
}

// WorkloadExecutor executes a stress test workload
type WorkloadExecutor struct {
	config  WorkloadConfig
	metrics *TestMetrics
	monitor *ResourceMonitor
	t       *testing.T
}

// NewWorkloadExecutor creates a new workload executor
func NewWorkloadExecutor(t *testing.T, config WorkloadConfig) *WorkloadExecutor {
	return &WorkloadExecutor{
		config:  config,
		metrics: &TestMetrics{},
		monitor: NewResourceMonitor(30*time.Second, 10000),
		t:       t,
	}
}

// Execute runs the workload
func (we *WorkloadExecutor) Execute(ctx context.Context, operation func(context.Context) error) {
	// Start resource monitoring
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	we.monitor.SetWarningCallback(func(msg string) {
		we.t.Logf("RESOURCE WARNING: %s", msg)
	})
	go we.monitor.Start(monitorCtx)

	// Start workload
	ticker := time.NewTicker(time.Second / time.Duration(we.config.OperationsPerSec))
	defer ticker.Stop()

	reportTicker := time.NewTicker(we.config.ReportingInterval)
	defer reportTicker.Stop()

	startTime := time.Now()
	endTime := startTime.Add(we.config.Duration)

	we.t.Logf("Starting workload: %d ops/sec, %d concurrent workers, duration=%v",
		we.config.OperationsPerSec, we.config.Concurrency, we.config.Duration)

	semaphore := make(chan struct{}, we.config.Concurrency)

	for {
		select {
		case <-ctx.Done():
			we.t.Log("Workload cancelled")
			return

		case <-ticker.C:
			if time.Now().After(endTime) {
				we.t.Log("Workload duration completed")
				we.printFinalReport()
				return
			}

			// Execute operation with concurrency limit
			select {
			case semaphore <- struct{}{}:
				go func() {
					defer func() { <-semaphore }()

					opStart := time.Now()
					err := operation(ctx)
					latency := time.Since(opStart)

					we.metrics.RecordOperation(uint64(latency.Nanoseconds()), err == nil)
				}()
			default:
				// Concurrency limit reached, skip this operation
				we.metrics.OperationsFailed.Add(1)
			}

		case <-reportTicker.C:
			we.printIntermediateReport()
		}
	}
}

// printIntermediateReport prints a progress report
func (we *WorkloadExecutor) printIntermediateReport() {
	we.t.Log("=== Intermediate Report ===")
	we.t.Log(we.metrics.GetStats())
	we.t.Log(we.monitor.GetStats().String())
	we.t.Log("===========================")
}

// printFinalReport prints the final report
func (we *WorkloadExecutor) printFinalReport() {
	we.t.Log("=== Final Report ===")
	we.t.Log(we.metrics.GetStats())
	we.t.Log(we.monitor.GetStats().String())
	we.t.Log("====================")
}

// GetMetrics returns the test metrics
func (we *WorkloadExecutor) GetMetrics() *TestMetrics {
	return we.metrics
}

// GetMonitor returns the resource monitor
func (we *WorkloadExecutor) GetMonitor() *ResourceMonitor {
	return we.monitor
}

// Helper functions for generating test data

// RandomPoolID returns a pool ID between 1 and 100
func RandomPoolID() uint64 {
	return uint64(time.Now().UnixNano()%100) + 1
}

// RandomAmount returns a random amount between 1000 and 1000000
func RandomAmount() math.Int {
	return math.NewInt(int64(time.Now().UnixNano()%999000) + 1000)
}

// RandomAddress generates a random SDK address
func RandomAddress() sdk.AccAddress {
	return sdk.AccAddress(fmt.Sprintf("addr%d", time.Now().UnixNano()))
}

// CreateTestPools creates test pools for stress testing
func CreateTestPools(t *testing.T, sta *StressTestApp, count int) []uint64 {
	t.Helper()

	poolIDs := make([]uint64, count)
	dexKeeper := sta.App.DEXKeeper

	for i := 0; i < count; i++ {
		tokenA := fmt.Sprintf("token%d", i*2)
		tokenB := fmt.Sprintf("token%d", i*2+1)

		pool, err := dexKeeper.CreatePool(
			sta.Ctx,
			dextypes.TestAddr(),
			tokenA,
			tokenB,
			math.NewInt(1000000),
			math.NewInt(1000000),
		)
		if err != nil {
			t.Fatalf("Failed to create pool %d: %v", i, err)
		}
		poolIDs[i] = pool.Id
	}

	t.Logf("Created %d test pools", count)
	return poolIDs
}

// CreateTestComputeProviders creates test compute providers
func CreateTestComputeProviders(t *testing.T, sta *StressTestApp, count int) []sdk.AccAddress {
	t.Helper()

	providers := make([]sdk.AccAddress, count)
	computeKeeper := sta.App.ComputeKeeper

	for i := 0; i < count; i++ {
		addr := RandomAddress()
		provider := computetypes.Provider{
			Address:        addr.String(),
			Moniker:        fmt.Sprintf("provider-%d", i),
			Endpoint:       fmt.Sprintf("http://provider%d.test:8080", i),
			AvailableSpecs: computetypes.ComputeSpec{CpuCores: 8, MemoryMb: 16384, StorageGb: 200, TimeoutSeconds: 3600},
			Pricing: computetypes.Pricing{
				CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.0001"),
				MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.00005"),
				GpuPricePerHour:       math.LegacyZeroDec(),
				StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
			},
			Stake:                  math.NewInt(5_000_000),
			Reputation:             75,
			TotalRequestsCompleted: 10,
			TotalRequestsFailed:    1,
			Active:                 true,
			RegisteredAt:           time.Now(),
			LastActiveAt:           time.Now(),
		}

		err := computeKeeper.SetProvider(sta.Ctx, provider)
		if err != nil {
			t.Fatalf("Failed to create provider %d: %v", i, err)
		}
		providers[i] = addr
	}

	t.Logf("Created %d test compute providers", count)
	return providers
}

// CreateTestOracleFeeds creates test oracle feeds
func CreateTestOracleFeeds(t *testing.T, sta *StressTestApp, count int) []string {
	t.Helper()

	feedIDs := make([]string, count)

	for i := 0; i < count; i++ {
		feedID := fmt.Sprintf("feed%d", i)
		feedIDs[i] = feedID
	}

	t.Logf("Created %d test oracle feeds", count)
	return feedIDs
}
