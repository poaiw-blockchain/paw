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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// StressTestSuite runs sustained load tests
type StressTestSuite struct {
	suite.Suite
}

// TestOneHourSustainedLoad runs a 1-hour sustained load test across all modules
func (suite *StressTestSuite) TestOneHourSustainedLoad() {
	if testing.Short() {
		suite.T().Skip("skipping 1-hour sustained load test in short mode")
	}

	suite.T().Skip("disabled in CI by design; run manually for 1h soak")

	suite.T().Log("Starting 1-hour sustained load test")

	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Minute)
	defer cancel()

	// Create test app
	sta := NewStressTestApp(suite.T())
	defer sta.Cleanup()

	// Setup test data
	poolIDs := CreateTestPools(suite.T(), sta, 50)
	providers := CreateTestComputeProviders(suite.T(), sta, 20)
	feedIDs := CreateTestOracleFeeds(suite.T(), sta, 30)

	suite.T().Logf("Test setup complete: %d pools, %d providers, %d feeds",
		len(poolIDs), len(providers), len(feedIDs))

	// Configure workload
	config := DefaultWorkloadConfig()
	executor := NewWorkloadExecutor(suite.T(), config)

	// Track module-specific operations
	var dexOps, computeOps, oracleOps atomic.Uint64

	// Define mixed workload operation
	operation := func(ctx context.Context) error {
		// Rotate through different module operations
		opType := time.Now().UnixNano() % 3

		switch opType {
		case 0: // DEX operation
			dexOps.Add(1)
			return suite.executeDEXOperation(sta, poolIDs)
		case 1: // Compute operation
			computeOps.Add(1)
			return suite.executeComputeOperation(sta, providers)
		case 2: // Oracle operation
			oracleOps.Add(1)
			return suite.executeOracleOperation(sta, feedIDs)
		default:
			return nil
		}
	}

	// Execute workload
	executor.Execute(ctx, operation)

	// Verify results
	suite.T().Logf("Module operation counts: DEX=%d, Compute=%d, Oracle=%d",
		dexOps.Load(), computeOps.Load(), oracleOps.Load())

	// Check for resource leaks
	stats := executor.GetMonitor().GetStats()
	suite.assertNoGoroutineLeak(stats)
	suite.assertNoMemoryLeak(stats)
}

// TestOneHourDEXFocused runs a 1-hour DEX-focused load test
func (suite *StressTestSuite) TestOneHourDEXFocused() {
	if testing.Short() {
		suite.T().Skip("skipping 1-hour DEX load test in short mode")
	}

	suite.T().Skip("disabled in CI by design; run manually for 1h soak")

	suite.T().Log("Starting 1-hour DEX-focused load test")

	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Minute)
	defer cancel()

	sta := NewStressTestApp(suite.T())
	defer sta.Cleanup()

	poolIDs := CreateTestPools(suite.T(), sta, 100)

	config := DefaultWorkloadConfig()
	config.OperationsPerSec = 200 // Higher rate for single module
	executor := NewWorkloadExecutor(suite.T(), config)

	var swaps, liquidity, queries atomic.Uint64

	operation := func(ctx context.Context) error {
		opType := time.Now().UnixNano() % 10

		switch {
		case opType < 6: // 60% swaps
			swaps.Add(1)
			return suite.executeSwap(sta, poolIDs)
		case opType < 9: // 30% liquidity operations
			liquidity.Add(1)
			return suite.executeLiquidityOp(sta, poolIDs)
		default: // 10% queries
			queries.Add(1)
			return suite.executeQuery(sta, poolIDs)
		}
	}

	executor.Execute(ctx, operation)

	suite.T().Logf("DEX operations: Swaps=%d, Liquidity=%d, Queries=%d",
		swaps.Load(), liquidity.Load(), queries.Load())

	stats := executor.GetMonitor().GetStats()
	suite.assertNoGoroutineLeak(stats)
	suite.assertNoMemoryLeak(stats)
}

// TestOneHourComputeFocused runs a 1-hour compute-focused load test
func (suite *StressTestSuite) TestOneHourComputeFocused() {
	if testing.Short() {
		suite.T().Skip("skipping 1-hour compute load test in short mode")
	}

	suite.T().Skip("disabled in CI by design; run manually for 1h soak")

	suite.T().Log("Starting 1-hour compute-focused load test")

	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Minute)
	defer cancel()

	sta := NewStressTestApp(suite.T())
	defer sta.Cleanup()

	providers := CreateTestComputeProviders(suite.T(), sta, 50)

	config := DefaultWorkloadConfig()
	config.OperationsPerSec = 150
	config.Concurrency = 20
	executor := NewWorkloadExecutor(suite.T(), config)

	var requests, results, disputes atomic.Uint64

	operation := func(ctx context.Context) error {
		opType := time.Now().UnixNano() % 10

		switch {
		case opType < 7: // 70% requests
			requests.Add(1)
			return suite.executeComputeRequest(sta, providers)
		case opType < 9: // 20% results
			results.Add(1)
			return suite.submitComputeResult(sta)
		default: // 10% disputes
			disputes.Add(1)
			return suite.executeComputeDispute(sta)
		}
	}

	executor.Execute(ctx, operation)

	suite.T().Logf("Compute operations: Requests=%d, Results=%d, Disputes=%d",
		requests.Load(), results.Load(), disputes.Load())

	stats := executor.GetMonitor().GetStats()
	suite.assertNoGoroutineLeak(stats)
	suite.assertNoMemoryLeak(stats)
}

// TestOneHourOracleFocused runs a 1-hour oracle-focused load test
func (suite *StressTestSuite) TestOneHourOracleFocused() {
	if testing.Short() {
		suite.T().Skip("skipping 1-hour oracle load test in short mode")
	}

	suite.T().Skip("disabled in CI by design; run manually for 1h soak")

	suite.T().Log("Starting 1-hour oracle-focused load test")

	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Minute)
	defer cancel()

	sta := NewStressTestApp(suite.T())
	defer sta.Cleanup()

	feedIDs := CreateTestOracleFeeds(suite.T(), sta, 100)

	config := DefaultWorkloadConfig()
	config.OperationsPerSec = 250 // Oracle is lightweight
	executor := NewWorkloadExecutor(suite.T(), config)

	var updates, queries, aggregations atomic.Uint64

	operation := func(ctx context.Context) error {
		opType := time.Now().UnixNano() % 10

		switch {
		case opType < 5: // 50% price updates
			updates.Add(1)
			return suite.submitPriceUpdate(sta, feedIDs)
		case opType < 8: // 30% queries
			queries.Add(1)
			return suite.queryOraclePrice(sta, feedIDs)
		default: // 20% aggregations
			aggregations.Add(1)
			return suite.executeOracleAggregation(sta, feedIDs)
		}
	}

	executor.Execute(ctx, operation)

	suite.T().Logf("Oracle operations: Updates=%d, Queries=%d, Aggregations=%d",
		updates.Load(), queries.Load(), aggregations.Load())

	stats := executor.GetMonitor().GetStats()
	suite.assertNoGoroutineLeak(stats)
	suite.assertNoMemoryLeak(stats)
}

// Helper methods for executing operations

func (suite *StressTestSuite) executeDEXOperation(sta *StressTestApp, poolIDs []uint64) error {
	if len(poolIDs) == 0 {
		return fmt.Errorf("no pools available")
	}

	poolID := poolIDs[time.Now().UnixNano()%int64(len(poolIDs))]
	pool, err := sta.App.DEXKeeper.GetPool(sta.Ctx, poolID)
	if err != nil {
		return err
	}
	_, err = sta.App.DEXKeeper.Swap(
		sta.Ctx,
		dextypes.TestAddr(),
		poolID,
		pool.TokenA,
		pool.TokenB,
		math.NewInt(1000),
		math.NewInt(900),
	)
	return err
}

func (suite *StressTestSuite) executeSwap(sta *StressTestApp, poolIDs []uint64) error {
	return suite.executeDEXOperation(sta, poolIDs)
}

func (suite *StressTestSuite) executeLiquidityOp(sta *StressTestApp, poolIDs []uint64) error {
	if len(poolIDs) == 0 {
		return fmt.Errorf("no pools available")
	}

	poolID := poolIDs[time.Now().UnixNano()%int64(len(poolIDs))]

	// 50/50 add vs remove
	if time.Now().UnixNano()%2 == 0 {
		_, err := sta.App.DEXKeeper.AddLiquiditySecure(
			sta.Ctx,
			dextypes.TestAddr(),
			poolID,
			math.NewInt(10000),
			math.NewInt(10000),
		)
		return err
	}

	_, _, err := sta.App.DEXKeeper.RemoveLiquiditySecure(
		sta.Ctx,
		dextypes.TestAddr(),
		poolID,
		math.NewInt(1000),
	)
	return err
}

func (suite *StressTestSuite) executeQuery(sta *StressTestApp, poolIDs []uint64) error {
	if len(poolIDs) == 0 {
		return fmt.Errorf("no pools available")
	}

	poolID := poolIDs[time.Now().UnixNano()%int64(len(poolIDs))]
	_, err := sta.App.DEXKeeper.GetPool(sta.Ctx, poolID)
	return err
}

func (suite *StressTestSuite) executeComputeOperation(sta *StressTestApp, providers []sdk.AccAddress) error {
	// Rotate through different compute operations
	opType := time.Now().UnixNano() % 3
	switch opType {
	case 0:
		return suite.executeComputeRequest(sta, providers)
	case 1:
		return suite.submitComputeResult(sta)
	default:
		return suite.executeComputeDispute(sta)
	}
}

func (suite *StressTestSuite) executeComputeRequest(sta *StressTestApp, providers []sdk.AccAddress) error {
	if len(providers) == 0 {
		return fmt.Errorf("no providers available")
	}

	requester := dextypes.TestAddr()
	provider := providers[time.Now().UnixNano()%int64(len(providers))]
	_, err := sta.App.ComputeKeeper.SubmitRequest(
		sta.Ctx,
		requester,
		computetypes.ComputeSpec{CpuCores: 4, MemoryMb: 4096, StorageGb: 10, TimeoutSeconds: 600},
		"registry/image:latest",
		[]string{"run"},
		nil,
		math.NewInt(100000),
		provider.String(),
	)
	return err
}

func (suite *StressTestSuite) submitComputeResult(sta *StressTestApp) error {
	// Submit a result for a random request
	requestID := uint64(time.Now().UnixNano() % 1000)
	return sta.App.ComputeKeeper.SubmitResult(
		sta.Ctx,
		RandomAddress(),
		requestID,
		"hash",
		"https://example.com/result",
		0,
		"https://example.com/logs",
		[]byte("proof"),
	)
}

func (suite *StressTestSuite) executeComputeDispute(sta *StressTestApp) error {
	requestID := uint64(time.Now().UnixNano() % 1000)
	_, err := sta.App.ComputeKeeper.CreateDispute(
		sta.Ctx,
		RandomAddress(),
		requestID,
		"faulty result",
		[]byte("evidence"),
		math.NewInt(1_000_000),
	)
	return err
}

func (suite *StressTestSuite) executeOracleOperation(sta *StressTestApp, feedIDs []string) error {
	opType := time.Now().UnixNano() % 3
	switch opType {
	case 0:
		return suite.submitPriceUpdate(sta, feedIDs)
	case 1:
		return suite.queryOraclePrice(sta, feedIDs)
	default:
		return suite.executeOracleAggregation(sta, feedIDs)
	}
}

func (suite *StressTestSuite) submitPriceUpdate(sta *StressTestApp, feedIDs []string) error {
	if len(feedIDs) == 0 {
		return fmt.Errorf("no feeds available")
	}

	feedID := feedIDs[time.Now().UnixNano()%int64(len(feedIDs))]
	price := oracletypes.Price{
		Asset:         feedID,
		Price:         math.LegacyNewDec(int64(time.Now().UnixNano()%100000) + 1000),
		BlockHeight:   sta.Ctx.BlockHeight(),
		BlockTime:     sta.Ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	return sta.App.OracleKeeper.SetPrice(sta.Ctx, price)
}

func (suite *StressTestSuite) queryOraclePrice(sta *StressTestApp, feedIDs []string) error {
	if len(feedIDs) == 0 {
		return fmt.Errorf("no feeds available")
	}

	feedID := feedIDs[time.Now().UnixNano()%int64(len(feedIDs))]
	_, err := sta.App.OracleKeeper.GetPrice(sta.Ctx, feedID)
	return err
}

func (suite *StressTestSuite) executeOracleAggregation(sta *StressTestApp, feedIDs []string) error {
	if len(feedIDs) == 0 {
		return fmt.Errorf("no feeds available")
	}

	feedID := feedIDs[time.Now().UnixNano()%int64(len(feedIDs))]

	// Submit multiple prices for aggregation
	for i := 0; i < 5; i++ {
		_ = sta.App.OracleKeeper.SetPrice(sta.Ctx, oracletypes.Price{
			Asset:         feedID,
			Price:         math.LegacyNewDec(int64(time.Now().UnixNano()%10000) + 50000),
			BlockHeight:   sta.Ctx.BlockHeight(),
			BlockTime:     sta.Ctx.BlockTime().Unix(),
			NumValidators: 1,
		})
	}

	// Trigger aggregation
	return sta.App.OracleKeeper.AggregatePrices(sta.Ctx)
}

// Assertion helpers

func (suite *StressTestSuite) assertNoGoroutineLeak(stats ResourceStats) {
	goroutineGrowth := stats.FinalGoroutines - stats.InitialGoroutines

	// Allow up to 50 goroutine growth (some persistent workers are acceptable)
	if goroutineGrowth > 50 {
		suite.T().Errorf("Potential goroutine leak detected: %d goroutines created during test (initial=%d, final=%d)",
			goroutineGrowth, stats.InitialGoroutines, stats.FinalGoroutines)
	} else {
		suite.T().Logf("Goroutine growth acceptable: %+d goroutines", goroutineGrowth)
	}
}

func (suite *StressTestSuite) assertNoMemoryLeak(stats ResourceStats) {
	// Memory can grow due to caching, but excessive growth indicates a leak
	memGrowthPercent := (stats.HeapGrowthMB / stats.InitialHeapMB) * 100

	// Allow up to 200% growth (3x initial size)
	if memGrowthPercent > 200 {
		suite.T().Errorf("Potential memory leak detected: %.2f%% heap growth (%.2f MB -> %.2f MB, growth=%.2f MB)",
			memGrowthPercent, stats.InitialHeapMB, stats.FinalHeapMB, stats.HeapGrowthMB)
	} else {
		suite.T().Logf("Memory growth acceptable: %.2f%% (%.2f MB growth)", memGrowthPercent, stats.HeapGrowthMB)
	}
}
