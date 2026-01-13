//go:build performance
// +build performance

// PERF-1.3: Stress Test Concurrent Swaps
// Target: 100+ TPS without triggering circuit breaker
// NOTE: Requires funded test accounts. Run with: go test -tags=performance ./tests/performance/...
package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// ConcurrentSwapTestSuite tests high-throughput swap operations
type ConcurrentSwapTestSuite struct {
	suite.Suite
	k   *keeper.Keeper
	ctx sdk.Context
}

func TestConcurrentSwapTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentSwapTestSuite))
}

func (suite *ConcurrentSwapTestSuite) SetupTest() {
	suite.k, suite.ctx = keepertest.DexKeeper(suite.T())
}

// TPSResult captures throughput test results
type TPSResult struct {
	TargetTPS       int
	AchievedTPS     float64
	TotalOperations uint64
	SuccessCount    uint64
	FailureCount    uint64
	Duration        time.Duration
	CircuitTripped  bool
	PassedTarget    bool
}

func (r TPSResult) String() string {
	return fmt.Sprintf("Target: %d TPS, Achieved: %.2f TPS, Success: %d/%d, Duration: %v, Circuit: %v, Passed: %v",
		r.TargetTPS, r.AchievedTPS, r.SuccessCount, r.TotalOperations, r.Duration, r.CircuitTripped, r.PassedTarget)
}

// TestConcurrentSwaps100TPS tests 100 TPS target
func (suite *ConcurrentSwapTestSuite) TestConcurrentSwaps100TPS() {
	result := suite.runTPSTest(100, 10*time.Second)
	suite.T().Logf("PERF-1.3 Result: %s", result)
	suite.True(result.PassedTarget, "Should achieve 100+ TPS, got %.2f", result.AchievedTPS)
}

// TestConcurrentSwaps200TPS tests 200 TPS stress level
func (suite *ConcurrentSwapTestSuite) TestConcurrentSwaps200TPS() {
	result := suite.runTPSTest(200, 10*time.Second)
	suite.T().Logf("PERF-1.3 Stress Result (200 TPS): %s", result)
	// This is a stress test - we document the result but don't fail
	suite.T().Logf("  → Achieved %.1f%% of target", (result.AchievedTPS/200)*100)
}

// TestConcurrentSwapsScaling tests TPS scaling from 50 to 500
func (suite *ConcurrentSwapTestSuite) TestConcurrentSwapsScaling() {
	targets := []int{50, 100, 150, 200, 300, 500}

	suite.T().Log("\n=== PERF-1.3 TPS SCALING TEST ===")
	suite.T().Log("| Target TPS | Achieved TPS | Success Rate | Circuit Tripped |")
	suite.T().Log("|------------|--------------|--------------|-----------------|")

	for _, target := range targets {
		result := suite.runTPSTest(target, 5*time.Second)
		successRate := float64(result.SuccessCount) / float64(result.TotalOperations) * 100
		suite.T().Logf("| %10d | %12.2f | %11.1f%% | %15v |",
			target, result.AchievedTPS, successRate, result.CircuitTripped)

		// Add small delay between tests
		time.Sleep(100 * time.Millisecond)
	}
	suite.T().Log("=== END SCALING TEST ===\n")
}

// runTPSTest executes a throughput test at the target TPS
func (suite *ConcurrentSwapTestSuite) runTPSTest(targetTPS int, duration time.Duration) TPSResult {
	// Setup: Create multiple pools for distribution
	creator := types.TestAddr()
	numPools := 10
	pools := make([]uint64, numPools)

	for i := 0; i < numPools; i++ {
		tokenA := fmt.Sprintf("tpsA%d", i)
		tokenB := fmt.Sprintf("tpsB%d", i)
		pool, err := suite.k.CreatePool(suite.ctx, creator, tokenA, tokenB,
			math.NewInt(10_000_000_000), math.NewInt(10_000_000_000))
		suite.Require().NoError(err)
		pools[i] = pool.Id
	}

	// Advance past flash loan protection
	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	// Create worker pool
	numWorkers := runtime.NumCPU() * 2
	if numWorkers < 8 {
		numWorkers = 8
	}

	var (
		successCount atomic.Uint64
		failureCount atomic.Uint64
		circuitTrips atomic.Uint64
	)

	// Rate limiter channel
	ticker := time.NewTicker(time.Second / time.Duration(targetTPS))
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	operationChan := make(chan int, targetTPS*2)

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			trader := sdk.AccAddress([]byte(fmt.Sprintf("tps_trader_%03d", workerID)))

			// Fund trader with all tokens
			for i := 0; i < numPools; i++ {
				tokenA := fmt.Sprintf("tpsA%d", i)
				keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
					sdk.NewCoins(sdk.NewCoin(tokenA, math.NewInt(1_000_000_000_000))))
			}

			for opNum := range operationChan {
				poolIdx := opNum % numPools
				tokenA := fmt.Sprintf("tpsA%d", poolIdx)
				tokenB := fmt.Sprintf("tpsB%d", poolIdx)

				_, err := suite.k.ExecuteSwap(suite.ctx, trader, pools[poolIdx],
					tokenA, tokenB, math.NewInt(1000), math.ZeroInt())

				if err != nil {
					failureCount.Add(1)
					// Check for circuit breaker errors
					if isCircuitBreakerError(err) {
						circuitTrips.Add(1)
					}
				} else {
					successCount.Add(1)
				}
			}
		}(w)
	}

	// Generate operations at target rate
	opNum := 0
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			close(operationChan)
			wg.Wait()

			totalOps := successCount.Load() + failureCount.Load()
			elapsed := time.Since(startTime)
			achievedTPS := float64(successCount.Load()) / elapsed.Seconds()

			return TPSResult{
				TargetTPS:       targetTPS,
				AchievedTPS:     achievedTPS,
				TotalOperations: totalOps,
				SuccessCount:    successCount.Load(),
				FailureCount:    failureCount.Load(),
				Duration:        elapsed,
				CircuitTripped:  circuitTrips.Load() > 0,
				PassedTarget:    achievedTPS >= float64(targetTPS),
			}

		case <-ticker.C:
			select {
			case operationChan <- opNum:
				opNum++
			default:
				// Channel full, skip this tick
			}
		}
	}
}

func isCircuitBreakerError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "circuit") || contains(errStr, "rate limit") ||
		contains(errStr, "too many") || contains(errStr, "throttl")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

// TestConcurrentSwapsMultiPool tests concurrent swaps across multiple pools
func (suite *ConcurrentSwapTestSuite) TestConcurrentSwapsMultiPool() {
	creator := types.TestAddr()
	numPools := 20

	pools := make([]struct {
		id     uint64
		tokenA string
		tokenB string
	}, numPools)

	for i := 0; i < numPools; i++ {
		pools[i].tokenA = fmt.Sprintf("mpA%d", i)
		pools[i].tokenB = fmt.Sprintf("mpB%d", i)
		pool, err := suite.k.CreatePool(suite.ctx, creator, pools[i].tokenA, pools[i].tokenB,
			math.NewInt(5_000_000_000), math.NewInt(5_000_000_000))
		suite.Require().NoError(err)
		pools[i].id = pool.Id
	}

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	var wg sync.WaitGroup
	var successCount, failCount atomic.Uint64

	numGoroutines := 50
	opsPerGoroutine := 100

	startTime := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()

			trader := sdk.AccAddress([]byte(fmt.Sprintf("mp_trader_%04d", gid)))

			// Fund with all tokens
			for _, p := range pools {
				keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
					sdk.NewCoins(sdk.NewCoin(p.tokenA, math.NewInt(100_000_000_000))))
			}

			for op := 0; op < opsPerGoroutine; op++ {
				poolIdx := (gid + op) % numPools
				p := pools[poolIdx]

				_, err := suite.k.ExecuteSwap(suite.ctx, trader, p.id,
					p.tokenA, p.tokenB, math.NewInt(1000), math.ZeroInt())

				if err != nil {
					failCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}
		}(g)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	totalOps := successCount.Load() + failCount.Load()
	tps := float64(successCount.Load()) / elapsed.Seconds()
	successRate := float64(successCount.Load()) / float64(totalOps) * 100

	suite.T().Logf("PERF-1.3 Multi-Pool Concurrent Test:")
	suite.T().Logf("  → Total operations: %d", totalOps)
	suite.T().Logf("  → Successful: %d (%.1f%%)", successCount.Load(), successRate)
	suite.T().Logf("  → Failed: %d", failCount.Load())
	suite.T().Logf("  → Duration: %v", elapsed)
	suite.T().Logf("  → Achieved TPS: %.2f", tps)

	suite.GreaterOrEqual(tps, 100.0, "Should achieve at least 100 TPS across multiple pools")
}

// TestSwapBurstLoad tests handling of burst traffic
func (suite *ConcurrentSwapTestSuite) TestSwapBurstLoad() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "burst_a", "burst_b",
		math.NewInt(50_000_000_000), math.NewInt(50_000_000_000))
	suite.Require().NoError(err)

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	// Burst: Send 500 swaps as fast as possible
	burstSize := 500
	var wg sync.WaitGroup
	var successCount atomic.Uint64

	startTime := time.Now()

	for i := 0; i < burstSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			trader := sdk.AccAddress([]byte(fmt.Sprintf("burst_%05d", idx)))
			keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
				sdk.NewCoins(sdk.NewCoin("burst_a", math.NewInt(1_000_000_000))))

			_, err := suite.k.ExecuteSwap(suite.ctx, trader, pool.Id,
				"burst_a", "burst_b", math.NewInt(1000), math.ZeroInt())
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	burstTPS := float64(successCount.Load()) / elapsed.Seconds()
	successRate := float64(successCount.Load()) / float64(burstSize) * 100

	suite.T().Logf("PERF-1.3 Burst Load Test:")
	suite.T().Logf("  → Burst size: %d swaps", burstSize)
	suite.T().Logf("  → Successful: %d (%.1f%%)", successCount.Load(), successRate)
	suite.T().Logf("  → Duration: %v", elapsed)
	suite.T().Logf("  → Burst TPS: %.2f", burstTPS)

	// Should handle burst without complete failure
	suite.Greater(successRate, 50.0, "Should handle at least 50% of burst load")
}

// BenchmarkConcurrentSwaps provides Go benchmark for concurrent swaps
func BenchmarkConcurrentSwaps(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()
	pool, _ := k.CreatePool(ctx, creator, "bench_a", "bench_b",
		math.NewInt(100_000_000_000), math.NewInt(100_000_000_000))

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 101)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		trader := sdk.AccAddress([]byte(fmt.Sprintf("bench_%d", time.Now().UnixNano())))
		keepertest.FundAccount(b, k, ctx, trader,
			sdk.NewCoins(sdk.NewCoin("bench_a", math.NewInt(1_000_000_000_000))))

		for pb.Next() {
			_, _ = k.ExecuteSwap(ctx, trader, pool.Id, "bench_a", "bench_b",
				math.NewInt(1000), math.ZeroInt())
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/sec")
}
