package gas

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// Gas limits for DEX operations
const (
	GasCreatePoolMin       = 80000
	GasCreatePoolMax       = 200000
	GasSwapMin             = 60000
	GasSwapMax             = 150000
	GasAddLiquidityMin     = 50000
	GasAddLiquidityMax     = 120000
	GasRemoveLiquidityMin  = 50000
	GasRemoveLiquidityMax  = 120000
	GasConstantProductMin  = 5000
	GasConstantProductMax  = 10000
	GasCircuitBreakerMin   = 10000
	GasCircuitBreakerMax   = 30000
	GasUpdatePoolParamsMin = 20000
	GasUpdatePoolParamsMax = 50000
)

func TestDEXGas_CreatePool(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

	creator := sdk.AccAddress("creator1___________")
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(2000000)

	poolID, err := k.CreatePool(ctx, creator.String(), tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	require.Greater(t, poolID, uint64(0))

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasCreatePoolMax),
		"CreatePool should use <%d gas, used %d", GasCreatePoolMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasCreatePoolMin),
		"CreatePool should use >%d gas, used %d", GasCreatePoolMin, gasUsed)

	t.Logf("CreatePool gas usage: %d", gasUsed)
}

func TestDEXGas_Swap(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Setup: Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	trader := sdk.AccAddress("trader1____________")

	tests := []struct {
		name     string
		amountIn int64
		maxGas   uint64
	}{
		{"small swap", 1000, 100000},
		{"medium swap", 1000000, 120000},
		{"large swap", 100000000, 150000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300000))

			amountOut, err := k.Swap(ctx, trader.String(), poolID, "upaw",
				math.NewInt(tt.amountIn), math.NewInt(1))
			require.NoError(t, err)
			require.True(t, amountOut.GT(math.ZeroInt()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"Swap gas usage exceeds limit: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasSwapMin),
				"Swap gas usage too low: used %d, min %d", gasUsed, GasSwapMin)

			t.Logf("Swap (%s): %d gas, amountOut: %s", tt.name, gasUsed, amountOut.String())
		})
	}
}

func TestDEXGas_ConstantProductCalculation(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Constant product calculation should be cheap (pure math)
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(50000))

	reserveA := math.NewInt(10000000)
	reserveB := math.NewInt(20000000)

	// Calculate constant product k = x * y
	k_value := k.CalculateConstantProduct(ctx, reserveA, reserveB)
	require.True(t, k_value.GT(math.ZeroInt()))

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasConstantProductMax),
		"Constant product calculation should use <%d gas, used %d", GasConstantProductMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasConstantProductMin),
		"Constant product calculation should use >%d gas, used %d", GasConstantProductMin, gasUsed)

	t.Logf("Constant product calculation gas: %d, k = %s", gasUsed, k_value.String())
}

func TestDEXGas_CircuitBreakerCheck(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Setup pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(100000))

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)

	// Calculate price impact
	amountIn := math.NewInt(10000000)
	priceImpact := k.CalculatePriceImpact(ctx, *pool, amountIn, pool.TokenA)

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasCircuitBreakerMax),
		"Circuit breaker check should use <%d gas, used %d", GasCircuitBreakerMax, gasUsed)

	t.Logf("Circuit breaker check gas: %d, price impact: %s", gasUsed, priceImpact.String())
}

func TestDEXGas_AddLiquidity(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Setup pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	tests := []struct {
		name    string
		amountA int64
		amountB int64
		maxGas  uint64
	}{
		{"small liquidity", 10000, 20000, 80000},
		{"medium liquidity", 1000000, 2000000, 100000},
		{"large liquidity", 100000000, 200000000, 120000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

			provider := sdk.AccAddress("provider1__________")

			lpTokens, err := k.AddLiquidity(ctx, provider.String(), poolID,
				math.NewInt(tt.amountA), math.NewInt(tt.amountB), math.NewInt(1))
			require.NoError(t, err)
			require.True(t, lpTokens.GT(math.ZeroInt()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"AddLiquidity gas exceeds limit: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasAddLiquidityMin),
				"AddLiquidity gas too low: used %d, min %d", gasUsed, GasAddLiquidityMin)

			t.Logf("AddLiquidity (%s): %d gas, LP tokens: %s", tt.name, gasUsed, lpTokens.String())
		})
	}
}

func TestDEXGas_RemoveLiquidity(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Setup pool and add liquidity
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	provider := sdk.AccAddress("provider1__________")
	lpTokens, err := k.AddLiquidity(ctx, provider.String(), poolID,
		math.NewInt(10000000), math.NewInt(20000000), math.NewInt(1))
	require.NoError(t, err)

	tests := []struct {
		name     string
		lpAmount int64
		maxGas   uint64
	}{
		{"small removal", 1000, 80000},
		{"medium removal", 100000, 100000},
		{"large removal", 1000000, 120000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

			// Remove a portion of liquidity
			removeAmount := lpTokens.Quo(math.NewInt(10)) // Remove 10%
			if removeAmount.GT(math.NewInt(tt.lpAmount)) {
				removeAmount = math.NewInt(tt.lpAmount)
			}

			amountA, amountB, err := k.RemoveLiquidity(ctx, provider.String(), poolID,
				removeAmount, math.NewInt(1), math.NewInt(1))
			require.NoError(t, err)
			require.True(t, amountA.GT(math.ZeroInt()))
			require.True(t, amountB.GT(math.ZeroInt()))

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"RemoveLiquidity gas exceeds limit: used %d, max %d", gasUsed, tt.maxGas)
			require.Greater(t, gasUsed, uint64(GasRemoveLiquidityMin),
				"RemoveLiquidity gas too low: used %d, min %d", gasUsed, GasRemoveLiquidityMin)

			t.Logf("RemoveLiquidity (%s): %d gas, got %s tokenA, %s tokenB",
				tt.name, gasUsed, amountA.String(), amountB.String())
		})
	}
}

func TestDEXGas_MultipleSwaps(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	// Test that multiple swaps have consistent gas usage
	trader := sdk.AccAddress("trader1____________")
	swapAmount := math.NewInt(1000000)

	var gasUsages []uint64

	for i := 0; i < 5; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300000))

		_, err := k.Swap(ctx, trader.String(), poolID, "upaw", swapAmount, math.NewInt(1))
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		gasUsages = append(gasUsages, gasUsed)

		t.Logf("Swap %d: %d gas", i+1, gasUsed)
	}

	// Gas usage should be relatively consistent
	// Calculate variance
	var sum uint64
	for _, gas := range gasUsages {
		sum += gas
	}
	avg := sum / uint64(len(gasUsages))

	for i, gas := range gasUsages {
		deviation := float64(gas) - float64(avg)
		if deviation < 0 {
			deviation = -deviation
		}
		percentDeviation := (deviation / float64(avg)) * 100

		// Gas should not deviate more than 20% from average
		require.Less(t, percentDeviation, 20.0,
			"Swap %d gas usage deviates %.2f%% from average", i+1, percentDeviation)
	}

	t.Logf("Average gas usage: %d, consistency check passed", avg)
}

func TestDEXGas_PriceImpactCalculation(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)

	// Test price impact calculation for different swap sizes
	tests := []struct {
		name        string
		swapSize    int64
		expectedGas uint64
	}{
		{"tiny swap", 1000, 15000},
		{"small swap", 100000, 20000},
		{"medium swap", 10000000, 25000},
		{"large swap", 100000000, 30000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(50000))

			priceImpact := k.CalculatePriceImpact(ctx, *pool, math.NewInt(tt.swapSize), pool.TokenA)
			require.NotNil(t, priceImpact)

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.expectedGas,
				"Price impact calculation too expensive: used %d, expected <%d", gasUsed, tt.expectedGas)

			t.Logf("Price impact (%s): %d gas, impact: %s%%", tt.name, gasUsed, priceImpact.Mul(math.LegacyNewDec(100)).String())
		})
	}
}

func TestDEXGas_PoolStateRead(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	// Test gas for reading pool state
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(20000))

	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	require.NotNil(t, pool)

	gasUsed := ctx.GasMeter().GasConsumed()

	// Reading state should be cheap
	require.Less(t, gasUsed, uint64(10000),
		"GetPool should be cheap: used %d gas", gasUsed)

	t.Logf("GetPool gas usage: %d", gasUsed)
}

func TestDEXGas_SlippageCalculation(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	require.NoError(t, err)

	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)

	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(50000))

	// Calculate expected output
	amountIn := math.NewInt(1000000)
	expectedOut := k.CalculateSwapOutput(ctx, *pool, amountIn, pool.TokenA)
	require.True(t, expectedOut.GT(math.ZeroInt()))

	gasUsed := ctx.GasMeter().GasConsumed()

	// Slippage calculation should be cheap
	require.Less(t, gasUsed, uint64(15000),
		"Slippage calculation too expensive: used %d gas", gasUsed)

	t.Logf("Slippage calculation gas: %d, output: %s", gasUsed, expectedOut.String())
}

func TestDEXGas_GasRegression(t *testing.T) {
	rawKeeper, ctx := keepertest.DexKeeper(t)
	k := NewDexGasKeeper(rawKeeper)

	// Baseline gas values
	baselines := map[string]uint64{
		"CreatePool":   120000,
		"Swap":         80000,
		"AddLiquidity": 70000,
	}

	tolerance := uint64(100000)

	t.Run("CreatePool baseline", func(t *testing.T) {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

		creator := sdk.AccAddress("creator1___________")
		_, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
			math.NewInt(1000000), math.NewInt(2000000))
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["CreatePool"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"CreatePool gas changed from baseline %d to %d", baseline, gasUsed)
	})

	t.Run("Swap baseline", func(t *testing.T) {
		// Create pool first
		creator := sdk.AccAddress("creator1___________")
		poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdc",
			math.NewInt(1000000000), math.NewInt(2000000000))
		require.NoError(t, err)

		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300000))

		trader := sdk.AccAddress("trader1____________")
		_, err = k.Swap(ctx, trader.String(), poolID, "upaw", math.NewInt(1000000), math.NewInt(1))
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["Swap"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"Swap gas changed from baseline %d to %d", baseline, gasUsed)
	})
}
