//go:build performance
// +build performance

// PERF-1.2: Gas Baseline Tests
// Document pool creation gas baseline and other operation gas costs
// NOTE: Requires funded test accounts. Run with: go test -tags=performance ./tests/performance/...
package performance

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// GasBaselineTestSuite measures gas consumption for DEX operations
type GasBaselineTestSuite struct {
	suite.Suite
	k   *keeper.Keeper
	ctx sdk.Context
}

func TestGasBaselineTestSuite(t *testing.T) {
	suite.Run(t, new(GasBaselineTestSuite))
}

func (suite *GasBaselineTestSuite) SetupTest() {
	suite.k, suite.ctx = keepertest.DexKeeper(suite.T())
}

// GasResult captures gas consumption metrics
type GasResult struct {
	Operation    string
	MinGas       uint64
	MaxGas       uint64
	MeanGas      uint64
	Samples      int
	BaselineGas  uint64 // Expected baseline
	WithinBounds bool
}

func (r GasResult) String() string {
	return fmt.Sprintf("%s: min=%d max=%d mean=%d baseline=%d within_bounds=%v (n=%d)",
		r.Operation, r.MinGas, r.MaxGas, r.MeanGas, r.BaselineGas, r.WithinBounds, r.Samples)
}

// measureGas runs an operation and returns gas consumption
func (suite *GasBaselineTestSuite) measureGas(name string, iterations int, baseline uint64, op func(sdk.Context) error) GasResult {
	var totalGas uint64
	var minGas, maxGas uint64 = ^uint64(0), 0
	samples := 0

	for i := 0; i < iterations; i++ {
		// Create a new context with fresh gas meter for each iteration
		gasMeter := storetypes.NewGasMeter(10_000_000)
		ctx := suite.ctx.WithGasMeter(gasMeter)

		err := op(ctx)
		if err != nil {
			continue
		}

		gasUsed := gasMeter.GasConsumed()
		totalGas += gasUsed
		samples++

		if gasUsed < minGas {
			minGas = gasUsed
		}
		if gasUsed > maxGas {
			maxGas = gasUsed
		}
	}

	if samples == 0 {
		return GasResult{Operation: name, Samples: 0, BaselineGas: baseline}
	}

	meanGas := totalGas / uint64(samples)

	// Within bounds if mean is within 20% of baseline
	tolerance := baseline / 5
	withinBounds := meanGas >= baseline-tolerance && meanGas <= baseline+tolerance

	return GasResult{
		Operation:    name,
		MinGas:       minGas,
		MaxGas:       maxGas,
		MeanGas:      meanGas,
		Samples:      samples,
		BaselineGas:  baseline,
		WithinBounds: withinBounds,
	}
}

// TestPoolCreationGasBaseline documents pool creation gas (target: ~50k base)
func (suite *GasBaselineTestSuite) TestPoolCreationGasBaseline() {
	creator := types.TestAddr()
	poolCounter := 0

	result := suite.measureGas("PoolCreation", 50, 50000, func(ctx sdk.Context) error {
		tokenA := fmt.Sprintf("tokenA%d", poolCounter)
		tokenB := fmt.Sprintf("tokenB%d", poolCounter)
		poolCounter++
		_, err := suite.k.CreatePool(ctx, creator, tokenA, tokenB,
			math.NewInt(1_000_000), math.NewInt(1_000_000))
		return err
	})

	suite.T().Logf("PERF-1.2 Pool Creation Gas: %s", result)

	// Document the baseline
	suite.T().Logf("  → Pool creation baseline: %d gas", result.MeanGas)
	suite.T().Logf("  → Expected baseline: ~50,000 gas")

	// Verify gas is reasonable (not excessive)
	suite.Less(result.MeanGas, uint64(200000), "Pool creation should use <200k gas")
}

// TestSwapGasBaseline documents swap gas consumption
func (suite *GasBaselineTestSuite) TestSwapGasBaseline() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	trader := sdk.AccAddress([]byte("gas_test_trader_001"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
		sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(100_000_000_000))))

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	result := suite.measureGas("SingleSwap", 100, 80000, func(ctx sdk.Context) error {
		_, err := suite.k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt",
			math.NewInt(10000), math.ZeroInt())
		return err
	})

	suite.T().Logf("PERF-1.2 Swap Gas: %s", result)
	suite.T().Logf("  → Swap baseline: %d gas", result.MeanGas)
	suite.Less(result.MeanGas, uint64(200000), "Swap should use <200k gas")
}

// TestAddLiquidityGasBaseline documents add liquidity gas
func (suite *GasBaselineTestSuite) TestAddLiquidityGasBaseline() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	provider := sdk.AccAddress([]byte("gas_test_provider01"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, provider,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(100_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(100_000_000_000)),
		))

	result := suite.measureGas("AddLiquidity", 100, 70000, func(ctx sdk.Context) error {
		_, err := suite.k.AddLiquidity(ctx, provider, pool.Id,
			math.NewInt(10000), math.NewInt(10000))
		return err
	})

	suite.T().Logf("PERF-1.2 Add Liquidity Gas: %s", result)
	suite.T().Logf("  → Add liquidity baseline: %d gas", result.MeanGas)
	suite.Less(result.MeanGas, uint64(200000), "AddLiquidity should use <200k gas")
}

// TestRemoveLiquidityGasBaseline documents remove liquidity gas
func (suite *GasBaselineTestSuite) TestRemoveLiquidityGasBaseline() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	provider := sdk.AccAddress([]byte("gas_rm_provider_01"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, provider,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(100_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(100_000_000_000)),
		))

	shares, err := suite.k.AddLiquidity(suite.ctx, provider, pool.Id,
		math.NewInt(50_000_000_000), math.NewInt(50_000_000_000))
	suite.Require().NoError(err)

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)
	sharePerOp := shares.Quo(math.NewInt(200))

	result := suite.measureGas("RemoveLiquidity", 100, 75000, func(ctx sdk.Context) error {
		_, _, err := suite.k.RemoveLiquidity(ctx, provider, pool.Id, sharePerOp)
		return err
	})

	suite.T().Logf("PERF-1.2 Remove Liquidity Gas: %s", result)
	suite.T().Logf("  → Remove liquidity baseline: %d gas", result.MeanGas)
	suite.Less(result.MeanGas, uint64(200000), "RemoveLiquidity should use <200k gas")
}

// TestLimitOrderGasBaseline documents limit order gas
func (suite *GasBaselineTestSuite) TestLimitOrderGasBaseline() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	owner := sdk.AccAddress([]byte("gas_limit_order_01"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, owner,
		sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(100_000_000_000))))

	orderCounter := 0
	result := suite.measureGas("PlaceLimitOrder", 50, 90000, func(ctx sdk.Context) error {
		orderCounter++
		_, err := suite.k.PlaceLimitOrder(ctx, owner, pool.Id,
			keeper.OrderTypeBuy, // Order type
			"upaw", "uusdt",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(11, 1), // 1.1 price
			24*time.Hour,                     // Expiry duration
		)
		return err
	})

	suite.T().Logf("PERF-1.2 Limit Order Gas: %s", result)
	suite.T().Logf("  → Limit order baseline: %d gas", result.MeanGas)
	suite.Less(result.MeanGas, uint64(300000), "PlaceLimitOrder should use <300k gas")
}

// TestGasSummaryReport generates comprehensive gas report
func (suite *GasBaselineTestSuite) TestGasSummaryReport() {
	creator := types.TestAddr()
	pool, err := suite.k.CreatePool(suite.ctx, creator, "upaw", "uusdt",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	suite.Require().NoError(err)

	user := sdk.AccAddress([]byte("gas_summary_user_01"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, user,
		sdk.NewCoins(
			sdk.NewCoin("upaw", math.NewInt(1_000_000_000_000)),
			sdk.NewCoin("uusdt", math.NewInt(1_000_000_000_000)),
		))

	shares, _ := suite.k.AddLiquidity(suite.ctx, user, pool.Id,
		math.NewInt(100_000_000_000), math.NewInt(100_000_000_000))
	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 101)

	poolCounter := 0
	sharePerOp := shares.Quo(math.NewInt(500))

	results := []GasResult{
		suite.measureGas("CreatePool", 20, 50000, func(ctx sdk.Context) error {
			poolCounter++
			_, err := suite.k.CreatePool(ctx, creator,
				fmt.Sprintf("gasA%d", poolCounter), fmt.Sprintf("gasB%d", poolCounter),
				math.NewInt(1_000_000), math.NewInt(1_000_000))
			return err
		}),
		suite.measureGas("Swap(10K)", 50, 80000, func(ctx sdk.Context) error {
			_, err := suite.k.ExecuteSwap(ctx, user, pool.Id, "upaw", "uusdt",
				math.NewInt(10000), math.ZeroInt())
			return err
		}),
		suite.measureGas("Swap(1M)", 50, 80000, func(ctx sdk.Context) error {
			_, err := suite.k.ExecuteSwap(ctx, user, pool.Id, "upaw", "uusdt",
				math.NewInt(1000000), math.ZeroInt())
			return err
		}),
		suite.measureGas("AddLiquidity", 50, 70000, func(ctx sdk.Context) error {
			_, err := suite.k.AddLiquidity(ctx, user, pool.Id,
				math.NewInt(100000), math.NewInt(100000))
			return err
		}),
		suite.measureGas("RemoveLiquidity", 50, 75000, func(ctx sdk.Context) error {
			_, _, err := suite.k.RemoveLiquidity(ctx, user, pool.Id, sharePerOp)
			return err
		}),
	}

	suite.T().Log("\n=== PERF-1.2 GAS BASELINE SUMMARY ===")
	suite.T().Log("| Operation        | Min Gas | Max Gas | Mean Gas | Baseline |")
	suite.T().Log("|------------------|---------|---------|----------|----------|")
	for _, r := range results {
		suite.T().Logf("| %-16s | %7d | %7d | %8d | %8d |",
			r.Operation, r.MinGas, r.MaxGas, r.MeanGas, r.BaselineGas)
	}
	suite.T().Log("=== END GAS REPORT ===\n")
}

// TestGasScalingWithPoolSize tests how gas scales with pool reserves
func (suite *GasBaselineTestSuite) TestGasScalingWithPoolSize() {
	creator := types.TestAddr()
	trader := sdk.AccAddress([]byte("gas_scaling_trader"))
	keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
		sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1_000_000_000_000))))

	poolSizes := []int64{1_000_000, 100_000_000, 1_000_000_000, 10_000_000_000}

	suite.T().Log("\n=== Gas Scaling with Pool Size ===")
	for i, size := range poolSizes {
		tokenA := fmt.Sprintf("scaleA%d", i)
		tokenB := fmt.Sprintf("scaleB%d", i)
		pool, err := suite.k.CreatePool(suite.ctx, creator, tokenA, tokenB,
			math.NewInt(size), math.NewInt(size))
		suite.Require().NoError(err)

		keepertest.FundAccount(suite.T(), suite.k, suite.ctx, trader,
			sdk.NewCoins(sdk.NewCoin(tokenA, math.NewInt(size))))

		result := suite.measureGas(fmt.Sprintf("Swap@%dM", size/1_000_000), 20, 80000, func(ctx sdk.Context) error {
			swapAmt := math.NewInt(size / 100) // 1% of pool
			_, err := suite.k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, swapAmt, math.ZeroInt())
			return err
		})

		suite.T().Logf("  Pool size %dM: %d gas (mean)", size/1_000_000, result.MeanGas)
	}
	suite.T().Log("=== END SCALING TEST ===\n")
}
