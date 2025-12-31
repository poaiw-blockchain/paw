// PERF-1.1: Swap Latency Baseline Tests
// Target: <100ms for single swap operation
package performance

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// LatencyTestSuite measures operation latencies
type LatencyTestSuite struct {
	suite.Suite
	k   *keeper.Keeper
	ctx sdk.Context
}

func TestLatencyTestSuite(t *testing.T) {
	suite.Run(t, new(LatencyTestSuite))
}

func (suite *LatencyTestSuite) SetupTest() {
	suite.k, suite.ctx = keepertest.DexKeeper(suite.T())
}

// LatencyResult captures timing metrics
type LatencyResult struct {
	Operation   string
	Min         time.Duration
	Max         time.Duration
	Mean        time.Duration
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	Samples     int
	PassedCheck bool
}

func (r LatencyResult) String() string {
	return fmt.Sprintf("%s: min=%v max=%v mean=%v p50=%v p95=%v p99=%v (n=%d, passed=%v)",
		r.Operation, r.Min, r.Max, r.Mean, r.P50, r.P95, r.P99, r.Samples, r.PassedCheck)
}

// measureLatency runs an operation multiple times and returns timing stats
func measureLatency(name string, iterations int, op func() error) LatencyResult {
	latencies := make([]time.Duration, 0, iterations)

	// Warm-up runs
	for i := 0; i < 10; i++ {
		_ = op()
	}

	// Measured runs
	for i := 0; i < iterations; i++ {
		start := time.Now()
		err := op()
		elapsed := time.Since(start)
		if err == nil {
			latencies = append(latencies, elapsed)
		}
	}

	if len(latencies) == 0 {
		return LatencyResult{Operation: name, Samples: 0}
	}

	// Calculate statistics
	var total time.Duration
	min, max := latencies[0], latencies[0]
	for _, l := range latencies {
		total += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}

	// Sort for percentiles
	sortDurations(latencies)

	return LatencyResult{
		Operation:   name,
		Min:         min,
		Max:         max,
		Mean:        total / time.Duration(len(latencies)),
		P50:         percentile(latencies, 50),
		P95:         percentile(latencies, 95),
		P99:         percentile(latencies, 99),
		Samples:     len(latencies),
		PassedCheck: percentile(latencies, 95) < 100*time.Millisecond,
	}
}

func sortDurations(d []time.Duration) {
	for i := range d {
		for j := i + 1; j < len(d); j++ {
			if d[j] < d[i] {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted) * p) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// TestSwapLatencyBaseline tests that single swap completes in <100ms
func (suite *LatencyTestSuite) TestSwapLatencyBaseline() {
	// Create a large pool
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	trader := sdk.AccAddress([]byte("latency_test_trader1"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
		sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1_000_000_000))))

	// Advance blocks past flash loan protection
	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	result := measureLatency("SingleSwap", 1000, func() error {
		_, err := suite.k.ExecuteSwap(suite.ctx, trader, pool.Id, "upaw", "uusdt",
			math.NewInt(10000), math.ZeroInt())
		return err
	})

	suite.T().Logf("PERF-1.1 Result: %s", result)
	suite.True(result.PassedCheck, "Swap P95 latency must be <100ms, got %v", result.P95)
}

// TestSwapLatencyUnderLoad tests swap latency with concurrent operations
func (suite *LatencyTestSuite) TestSwapLatencyUnderLoad() {
	// Create multiple pools
	creator := types.TestAddr()
	pools := make([]uint64, 5)
	for i := 0; i < 5; i++ {
		tokenA := fmt.Sprintf("token%d", i*2)
		tokenB := fmt.Sprintf("token%d", i*2+1)
		pool, err := suite.k.CreatePool(suite.ctx, creator, tokenA, tokenB,
			math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
		suite.Require().NoError(err)
		pools[i] = pool.Id
	}

	trader := sdk.AccAddress([]byte("latency_test_trader2"))
	for i := 0; i < 10; i++ {
		keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
			sdk.NewCoins(sdk.NewCoin(fmt.Sprintf("token%d", i), math.NewInt(1_000_000_000))))
	}

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	// Measure latency while simulating concurrent background load
	var wg sync.WaitGroup
	stopBg := make(chan struct{})

	// Background load: continuous pool queries
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(poolID uint64) {
			defer wg.Done()
			for {
				select {
				case <-stopBg:
					return
				default:
					_, _ = suite.k.GetPool(suite.ctx, poolID)
					time.Sleep(time.Microsecond)
				}
			}
		}(pools[i%len(pools)])
	}

	// Measure swap latency under load
	result := measureLatency("SwapUnderLoad", 500, func() error {
		poolIdx := time.Now().UnixNano() % 5
		tokenA := fmt.Sprintf("token%d", poolIdx*2)
		tokenB := fmt.Sprintf("token%d", poolIdx*2+1)
		_, err := suite.k.ExecuteSwap(suite.ctx, trader, pools[poolIdx], tokenA, tokenB,
			math.NewInt(1000), math.ZeroInt())
		return err
	})

	close(stopBg)
	wg.Wait()

	suite.T().Logf("PERF-1.1 Under Load Result: %s", result)
	suite.True(result.PassedCheck, "Swap under load P95 must be <100ms, got %v", result.P95)
}

// TestAddLiquidityLatency tests add liquidity latency
func (suite *LatencyTestSuite) TestAddLiquidityLatency() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	provider := sdk.AccAddress([]byte("latency_test_provider"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, provider,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(10_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(10_000_000_000)),
		))

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	result := measureLatency("AddLiquidity", 500, func() error {
		_, err := suite.k.AddLiquidity(suite.ctx, provider, pool.Id,
			math.NewInt(10000), math.NewInt(10000))
		return err
	})

	suite.T().Logf("PERF-1.1 AddLiquidity Result: %s", result)
	suite.True(result.P95 < 100*time.Millisecond, "AddLiquidity P95 must be <100ms")
}

// TestRemoveLiquidityLatency tests remove liquidity latency
func (suite *LatencyTestSuite) TestRemoveLiquidityLatency() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	provider := sdk.AccAddress([]byte("latency_rm_provider1"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, provider,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(100_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(100_000_000_000)),
		))

	// Add lots of liquidity first
	shares, err := suite.k.AddLiquidity(suite.ctx, provider, pool.Id,
		math.NewInt(50_000_000_000), math.NewInt(50_000_000_000))
	suite.Require().NoError(err)

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	sharePerOp := shares.Quo(math.NewInt(1000))

	result := measureLatency("RemoveLiquidity", 500, func() error {
		_, _, err := suite.k.RemoveLiquidity(suite.ctx, provider, pool.Id, sharePerOp)
		return err
	})

	suite.T().Logf("PERF-1.1 RemoveLiquidity Result: %s", result)
	suite.True(result.P95 < 100*time.Millisecond, "RemoveLiquidity P95 must be <100ms")
}

// TestLatencySummaryReport generates a comprehensive latency report
func (suite *LatencyTestSuite) TestLatencySummaryReport() {
	creator := types.TestAddr()
	pool, _ := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))

	trader := sdk.AccAddress([]byte("latency_summary_test"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(100_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(100_000_000_000)),
		))

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	results := []LatencyResult{
		measureLatency("SmallSwap(1K)", 200, func() error {
			_, err := suite.k.ExecuteSwap(suite.ctx, trader, pool.Id, "upaw", "uusdt",
				math.NewInt(1000), math.ZeroInt())
			return err
		}),
		measureLatency("MediumSwap(100K)", 200, func() error {
			_, err := suite.k.ExecuteSwap(suite.ctx, trader, pool.Id, "upaw", "uusdt",
				math.NewInt(100000), math.ZeroInt())
			return err
		}),
		measureLatency("LargeSwap(10M)", 200, func() error {
			_, err := suite.k.ExecuteSwap(suite.ctx, trader, pool.Id, "upaw", "uusdt",
				math.NewInt(10000000), math.ZeroInt())
			return err
		}),
	}

	suite.T().Log("\n=== PERF-1.1 LATENCY SUMMARY REPORT ===")
	allPassed := true
	for _, r := range results {
		suite.T().Logf("  %s", r)
		if !r.PassedCheck {
			allPassed = false
		}
	}
	suite.T().Logf("=== All operations under 100ms target: %v ===\n", allPassed)
	suite.True(allPassed, "All swap operations must complete in <100ms P95")
}

// BenchmarkSwapLatency provides Go benchmark for swap latency
func BenchmarkSwapLatency(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	require.NoError(b, err)

	trader := sdk.AccAddress([]byte("bench_swap_latency1"))
	keepertest.FundAccount(b, k, ctx, trader,
		sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1_000_000_000_000))))

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 101)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt",
			math.NewInt(10000), math.ZeroInt())
	}
}

// BenchmarkSwapLatencyParallel measures parallel swap throughput
func BenchmarkSwapLatencyParallel(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt",
		math.NewInt(10_000_000_000), math.NewInt(10_000_000_000))
	require.NoError(b, err)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 101)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		trader := sdk.AccAddress([]byte(fmt.Sprintf("parallel_trader_%d", time.Now().UnixNano())))
		keepertest.FundAccount(b, k, ctx, trader,
			sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1_000_000_000_000))))

		for pb.Next() {
			_, _ = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt",
				math.NewInt(1000), math.ZeroInt())
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "swaps/sec")
}

// TestMemoryDuringLatencyTest monitors memory during latency tests
func (suite *LatencyTestSuite) TestMemoryDuringLatencyTest() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.TotalAlloc

	// Run latency test
	suite.TestSwapLatencyBaseline()

	runtime.ReadMemStats(&m)
	endAlloc := m.TotalAlloc

	allocatedMB := float64(endAlloc-startAlloc) / 1024 / 1024
	suite.T().Logf("Memory allocated during latency test: %.2f MB", allocatedMB)
	suite.Less(allocatedMB, 100.0, "Latency test should not allocate >100MB")
}
