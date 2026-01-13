//go:build performance
// +build performance

// PERF-1.4: Memory Profiling for Large Pool Iterations
// Tests memory usage with 1000+ pools
// NOTE: Requires funded test accounts. Run with: go test -tags=performance ./tests/performance/...
package performance

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// MemoryProfileTestSuite tests memory consumption with large pool counts
type MemoryProfileTestSuite struct {
	suite.Suite
	k   *keeper.Keeper
	ctx sdk.Context
}

func TestMemoryProfileTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryProfileTestSuite))
}

func (suite *MemoryProfileTestSuite) SetupTest() {
	suite.k, suite.ctx = keepertest.DexKeeper(suite.T())
}

// MemorySnapshot captures memory state at a point in time
type MemorySnapshot struct {
	HeapAlloc   uint64
	HeapInuse   uint64
	HeapObjects uint64
	StackInuse  uint64
	NumGC       uint32
	Timestamp   time.Time
}

func takeMemorySnapshot() MemorySnapshot {
	runtime.GC() // Force GC for accurate measurement
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemorySnapshot{
		HeapAlloc:   m.HeapAlloc,
		HeapInuse:   m.HeapInuse,
		HeapObjects: m.HeapObjects,
		StackInuse:  m.StackInuse,
		NumGC:       m.NumGC,
		Timestamp:   time.Now(),
	}
}

func (s MemorySnapshot) String() string {
	return fmt.Sprintf("Heap: %.2f MB (inuse: %.2f MB), Objects: %d, Stack: %.2f KB",
		float64(s.HeapAlloc)/1024/1024,
		float64(s.HeapInuse)/1024/1024,
		s.HeapObjects,
		float64(s.StackInuse)/1024)
}

// MemoryGrowthResult captures memory growth during operations
type MemoryGrowthResult struct {
	Operation       string
	PoolCount       int
	StartMemory     MemorySnapshot
	EndMemory       MemorySnapshot
	HeapGrowthMB    float64
	ObjectGrowth    int64
	MemoryPerPoolKB float64
	Duration        time.Duration
}

func (r MemoryGrowthResult) String() string {
	return fmt.Sprintf("%s (%d pools): Growth=%.2f MB, PerPool=%.2f KB, Objects=%+d, Duration=%v",
		r.Operation, r.PoolCount, r.HeapGrowthMB, r.MemoryPerPoolKB, r.ObjectGrowth, r.Duration)
}

// TestMemoryWith1000Pools tests memory usage with 1000 pools
func (suite *MemoryProfileTestSuite) TestMemoryWith1000Pools() {
	result := suite.measurePoolCreationMemory(1000)
	suite.T().Logf("PERF-1.4 1000 Pools: %s", result)

	// Memory should not grow excessively
	suite.Less(result.HeapGrowthMB, 500.0, "Memory growth for 1000 pools should be <500MB")
	suite.Less(result.MemoryPerPoolKB, 500.0, "Memory per pool should be <500KB")
}

// TestMemoryWith2000Pools tests memory scaling to 2000 pools
func (suite *MemoryProfileTestSuite) TestMemoryWith2000Pools() {
	result := suite.measurePoolCreationMemory(2000)
	suite.T().Logf("PERF-1.4 2000 Pools: %s", result)

	suite.Less(result.HeapGrowthMB, 1000.0, "Memory growth for 2000 pools should be <1GB")
}

// TestMemoryScaling tests memory scaling from 100 to 2000 pools
func (suite *MemoryProfileTestSuite) TestMemoryScaling() {
	poolCounts := []int{100, 250, 500, 1000, 1500, 2000}

	suite.T().Log("\n=== PERF-1.4 MEMORY SCALING TEST ===")
	suite.T().Log("| Pools | Heap Growth (MB) | Per Pool (KB) | Objects Growth | Duration |")
	suite.T().Log("|-------|------------------|---------------|----------------|----------|")

	var previousResult *MemoryGrowthResult

	for _, count := range poolCounts {
		// Reset keeper for each test
		suite.k, suite.ctx = keepertest.DexKeeper(suite.T())

		result := suite.measurePoolCreationMemory(count)

		suite.T().Logf("| %5d | %16.2f | %13.2f | %14d | %8v |",
			count, result.HeapGrowthMB, result.MemoryPerPoolKB, result.ObjectGrowth, result.Duration)

		// Check for linear scaling (memory per pool should be roughly constant)
		if previousResult != nil {
			ratio := result.MemoryPerPoolKB / previousResult.MemoryPerPoolKB
			if ratio > 2.0 {
				suite.T().Logf("  WARNING: Non-linear scaling detected (ratio=%.2f)", ratio)
			}
		}
		previousResult = &result

		// Give GC time to settle
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}

	suite.T().Log("=== END MEMORY SCALING TEST ===\n")
}

// measurePoolCreationMemory creates pools and measures memory impact
func (suite *MemoryProfileTestSuite) measurePoolCreationMemory(numPools int) MemoryGrowthResult {
	creator := types.TestAddr()

	// Take initial snapshot
	startSnapshot := takeMemorySnapshot()
	startTime := time.Now()

	// Create pools
	for i := 0; i < numPools; i++ {
		tokenA := fmt.Sprintf("memA%d", i)
		tokenB := fmt.Sprintf("memB%d", i)
		_, err := suite.k.CreatePool(suite.ctx, creator, tokenA, tokenB,
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		if err != nil {
			suite.T().Logf("Pool creation failed at %d: %v", i, err)
			break
		}

		// Periodic GC to simulate realistic conditions
		if i > 0 && i%100 == 0 {
			runtime.GC()
		}
	}

	duration := time.Since(startTime)

	// Take final snapshot
	endSnapshot := takeMemorySnapshot()

	heapGrowth := float64(endSnapshot.HeapAlloc-startSnapshot.HeapAlloc) / 1024 / 1024
	objectGrowth := int64(endSnapshot.HeapObjects) - int64(startSnapshot.HeapObjects)
	perPool := heapGrowth * 1024 / float64(numPools) // KB per pool

	return MemoryGrowthResult{
		Operation:       "CreatePools",
		PoolCount:       numPools,
		StartMemory:     startSnapshot,
		EndMemory:       endSnapshot,
		HeapGrowthMB:    heapGrowth,
		ObjectGrowth:    objectGrowth,
		MemoryPerPoolKB: perPool,
		Duration:        duration,
	}
}

// TestPoolIterationMemory tests memory usage when iterating over pools
func (suite *MemoryProfileTestSuite) TestPoolIterationMemory() {
	creator := types.TestAddr()
	numPools := 1000

	// Create pools first
	for i := 0; i < numPools; i++ {
		_, _ = suite.k.CreatePool(suite.ctx, creator,
			fmt.Sprintf("iterA%d", i), fmt.Sprintf("iterB%d", i),
			math.NewInt(1_000_000), math.NewInt(1_000_000))
	}

	runtime.GC()
	startSnapshot := takeMemorySnapshot()

	// Iterate over all pools multiple times
	iterations := 10
	for iter := 0; iter < iterations; iter++ {
		for poolID := uint64(1); poolID <= uint64(numPools); poolID++ {
			pool, err := suite.k.GetPool(suite.ctx, poolID)
			if err != nil {
				continue
			}
			// Simulate reading pool data
			_ = pool.ReserveA.Add(pool.ReserveB)
		}
	}

	endSnapshot := takeMemorySnapshot()
	heapGrowth := float64(endSnapshot.HeapAlloc-startSnapshot.HeapAlloc) / 1024 / 1024

	suite.T().Logf("PERF-1.4 Pool Iteration Memory:")
	suite.T().Logf("  → Pools: %d", numPools)
	suite.T().Logf("  → Iterations: %d", iterations)
	suite.T().Logf("  → Heap growth during iteration: %.2f MB", heapGrowth)
	suite.T().Logf("  → Start: %s", startSnapshot)
	suite.T().Logf("  → End: %s", endSnapshot)

	// Iteration should not cause significant memory growth (no leaks)
	suite.Less(heapGrowth, 50.0, "Pool iteration should not grow heap by >50MB")
}

// TestSwapMemoryWithManyPools tests swap memory usage with many pools
func (suite *MemoryProfileTestSuite) TestSwapMemoryWithManyPools() {
	creator := types.TestAddr()
	numPools := 500
	pools := make([]uint64, numPools)

	// Create pools
	for i := 0; i < numPools; i++ {
		pool, err := suite.k.CreatePool(suite.ctx, creator,
			fmt.Sprintf("swapMemA%d", i), fmt.Sprintf("swapMemB%d", i),
			math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
		if err != nil {
			break
		}
		pools[i] = pool.Id
	}

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	trader := sdk.AccAddress([]byte("swap_mem_trader_01"))
	for i := 0; i < numPools; i++ {
		keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
			sdk.NewCoins(sdk.NewCoin(fmt.Sprintf("swapMemA%d", i), math.NewInt(10_000_000_000))))
	}

	runtime.GC()
	startSnapshot := takeMemorySnapshot()

	// Execute swaps across all pools
	swapsPerPool := 10
	for round := 0; round < swapsPerPool; round++ {
		for i, poolID := range pools {
			if poolID == 0 {
				continue
			}
			tokenA := fmt.Sprintf("swapMemA%d", i)
			tokenB := fmt.Sprintf("swapMemB%d", i)
			_, _ = suite.k.ExecuteSwap(suite.ctx, trader, poolID, tokenA, tokenB,
				math.NewInt(10000), math.ZeroInt())
		}
	}

	endSnapshot := takeMemorySnapshot()
	totalSwaps := numPools * swapsPerPool
	heapGrowth := float64(endSnapshot.HeapAlloc-startSnapshot.HeapAlloc) / 1024 / 1024
	perSwapKB := heapGrowth * 1024 / float64(totalSwaps)

	suite.T().Logf("PERF-1.4 Swap Memory with %d Pools:", numPools)
	suite.T().Logf("  → Total swaps: %d", totalSwaps)
	suite.T().Logf("  → Heap growth: %.2f MB", heapGrowth)
	suite.T().Logf("  → Memory per swap: %.4f KB", perSwapKB)

	// Memory per swap should be minimal (no leak)
	suite.Less(perSwapKB, 10.0, "Memory per swap should be <10KB (no leak)")
}

// TestLiquidityProviderMemory tests memory with many liquidity providers
func (suite *MemoryProfileTestSuite) TestLiquidityProviderMemory() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "lpMemA", "lpMemB",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	numProviders := 500

	runtime.GC()
	startSnapshot := takeMemorySnapshot()

	// Add many liquidity providers
	for i := 0; i < numProviders; i++ {
		provider := sdk.AccAddress([]byte(fmt.Sprintf("lp_provider_%04d", i)))
		keepertest.FundAccount(suite.T(), suite.k, suite.ctx, provider,
			sdk.NewCoins(
				sdk.NewCoin("lpMemA", math.NewInt(1_000_000_000)),
				sdk.NewCoin("lpMemB", math.NewInt(1_000_000_000)),
			))

		_, err := suite.k.AddLiquidity(suite.ctx, provider, pool.Id,
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		if err != nil {
			suite.T().Logf("Add liquidity failed at provider %d: %v", i, err)
			break
		}
	}

	endSnapshot := takeMemorySnapshot()
	heapGrowth := float64(endSnapshot.HeapAlloc-startSnapshot.HeapAlloc) / 1024 / 1024
	perProviderKB := heapGrowth * 1024 / float64(numProviders)

	suite.T().Logf("PERF-1.4 Liquidity Provider Memory:")
	suite.T().Logf("  → Providers: %d", numProviders)
	suite.T().Logf("  → Heap growth: %.2f MB", heapGrowth)
	suite.T().Logf("  → Memory per provider: %.2f KB", perProviderKB)

	suite.Less(perProviderKB, 100.0, "Memory per LP should be <100KB")
}

// TestMemorySummaryReport generates a comprehensive memory report
func (suite *MemoryProfileTestSuite) TestMemorySummaryReport() {
	suite.T().Log("\n=== PERF-1.4 MEMORY PROFILE SUMMARY ===")

	// Test various pool counts
	poolTests := []int{100, 500, 1000}

	for _, count := range poolTests {
		suite.k, suite.ctx = keepertest.DexKeeper(suite.T())
		result := suite.measurePoolCreationMemory(count)
		suite.T().Logf("  %d pools: %.2f MB total, %.2f KB/pool",
			count, result.HeapGrowthMB, result.MemoryPerPoolKB)
	}

	suite.T().Log("")
	suite.T().Log("Memory efficiency targets:")
	suite.T().Log("  → Pool creation: <500 KB/pool")
	suite.T().Log("  → Pool iteration: No significant growth")
	suite.T().Log("  → Swap operations: <10 KB/swap")
	suite.T().Log("  → Liquidity providers: <100 KB/provider")
	suite.T().Log("=== END MEMORY REPORT ===\n")
}

// BenchmarkPoolCreationMemory provides allocation benchmarks
func BenchmarkPoolCreationMemory(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = k.CreatePool(ctx, creator,
			fmt.Sprintf("benchA%d", i), fmt.Sprintf("benchB%d", i),
			math.NewInt(1_000_000), math.NewInt(1_000_000))
	}
}

// BenchmarkPoolIteration provides iteration benchmarks
func BenchmarkPoolIteration(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	// Create 100 pools
	for i := 0; i < 100; i++ {
		_, _ = k.CreatePool(ctx, creator,
			fmt.Sprintf("iterBenchA%d", i), fmt.Sprintf("iterBenchB%d", i),
			math.NewInt(1_000_000), math.NewInt(1_000_000))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for poolID := uint64(1); poolID <= 100; poolID++ {
			_, _ = k.GetPool(ctx, poolID)
		}
	}

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "lookups/sec")
}
