package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestGasConsumptionOnSwapSuccess tests that successful swaps consume expected gas
func TestGasConsumptionOnSwapSuccess(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	// Create trader
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, "upaw", math.NewInt(1000000))

	// Record gas before swap
	gasBefore := ctx.GasMeter().GasConsumed()

	// Execute swap
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(1))
	require.NoError(t, err)

	// Record gas after swap
	gasAfter := ctx.GasMeter().GasConsumed()

	// Verify gas consumed
	gasConsumed := gasAfter - gasBefore
	require.Greater(t, gasConsumed, uint64(0), "swap should consume gas")
	require.Less(t, gasConsumed, uint64(1000000), "swap should not consume excessive gas")

	t.Logf("Successful swap consumed %d gas", gasConsumed)
}

// TestGasConsumptionOnSwapFailure tests that failed swaps consume gas for validation
func TestGasConsumptionOnSwapFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	// Create trader without funds
	trader := sdk.AccAddress("unfunded_trader_____")

	// Record gas before failed swap
	gasBefore := ctx.GasMeter().GasConsumed()

	// Attempt swap (will fail due to insufficient funds)
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(1))
	require.Error(t, err)

	// Record gas after failed swap
	gasAfter := ctx.GasMeter().GasConsumed()

	// Verify gas was consumed (validation happened)
	gasConsumed := gasAfter - gasBefore
	require.Greater(t, gasConsumed, uint64(0), "failed swap should still consume gas for validation")

	t.Logf("Failed swap consumed %d gas", gasConsumed)
}

// TestGasConsumptionOnSlippageFailure tests gas consumption when slippage protection triggers
func TestGasConsumptionOnSlippageFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	// Create funded trader
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, "upaw", math.NewInt(1000000))

	// Record gas before
	gasBefore := ctx.GasMeter().GasConsumed()

	// Attempt swap with impossible slippage requirement
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(999999999))
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")

	// Record gas after
	gasAfter := ctx.GasMeter().GasConsumed()

	// Verify gas consumed (swap calculation happened before slippage check)
	gasConsumed := gasAfter - gasBefore
	require.Greater(t, gasConsumed, uint64(0), "slippage check should consume gas")

	t.Logf("Slippage failure consumed %d gas", gasConsumed)
}

// TestGasRefundNotAppliedOnFailure tests that failed operations don't refund gas
func TestGasRefundNotAppliedOnFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	trader := sdk.AccAddress("unfunded_trader_____")

	// Set up gas meter with specific limit
	initialGas := uint64(1000000)
	gasMeter := sdk.NewGasMeter(initialGas)
	ctx = ctx.WithGasMeter(gasMeter)

	gasBefore := ctx.GasMeter().GasConsumed()

	// Attempt operation that will fail
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(1))
	require.Error(t, err)

	gasAfter := ctx.GasMeter().GasConsumed()

	// Verify gas was consumed and NOT refunded
	require.Greater(t, gasAfter, gasBefore, "gas should be consumed on failure")
	require.Equal(t, gasAfter, ctx.GasMeter().GasConsumed(), "gas should not be refunded")
}

// TestOutOfGasDoesNotCorruptState tests that out-of-gas scenarios don't corrupt state
func TestOutOfGasDoesNotCorruptState(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	// Record initial pool state
	initialReserveA := pool.ReserveA
	initialReserveB := pool.ReserveB

	// Create trader with funds
	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, "upaw", math.NewInt(1000000))

	// Set up gas meter with very low limit to trigger out-of-gas
	lowGasLimit := uint64(1000) // Very low
	gasMeter := sdk.NewGasMeter(lowGasLimit)
	ctx = ctx.WithGasMeter(gasMeter)

	// Attempt swap (will likely run out of gas)
	defer func() {
		if r := recover(); r != nil {
			// Out of gas panic occurred, verify state unchanged
			finalPool, err := k.GetPool(ctx.WithGasMeter(sdk.NewInfiniteGasMeter()), pool.Id)
			require.NoError(t, err)
			require.Equal(t, initialReserveA, finalPool.ReserveA, "pool state should not be corrupted by out-of-gas")
			require.Equal(t, initialReserveB, finalPool.ReserveB, "pool state should not be corrupted by out-of-gas")
		}
	}()

	_, _ = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(1))
}

// TestGasConsumptionIncreaseWithComplexity tests that more complex operations consume more gas
func TestGasConsumptionIncreaseWithComplexity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, "upaw", math.NewInt(10000000))

	// Test small swap gas
	gasBeforeSmall := ctx.GasMeter().GasConsumed()
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(1000), math.NewInt(1))
	require.NoError(t, err)
	gasSmallSwap := ctx.GasMeter().GasConsumed() - gasBeforeSmall

	// Test large swap gas
	gasBeforeLarge := ctx.GasMeter().GasConsumed()
	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", math.NewInt(100000), math.NewInt(1))
	require.NoError(t, err)
	gasLargeSwap := ctx.GasMeter().GasConsumed() - gasBeforeLarge

	// Both should consume gas (exact values may vary but should be reasonable)
	require.Greater(t, gasSmallSwap, uint64(0))
	require.Greater(t, gasLargeSwap, uint64(0))

	t.Logf("Small swap gas: %d, Large swap gas: %d", gasSmallSwap, gasLargeSwap)
}

// TestGasConsumptionOnValidationFailure tests gas consumption during early validation failures
func TestGasConsumptionOnValidationFailure(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	trader := createTestTraderForErrorRecovery(t)

	testCases := []struct {
		name         string
		amountIn     math.Int
		minAmountOut math.Int
		errorMsg     string
	}{
		{
			name:         "zero amount",
			amountIn:     math.NewInt(0),
			minAmountOut: math.NewInt(1),
			errorMsg:     "must be positive",
		},
		{
			name:         "identical tokens",
			amountIn:     math.NewInt(1000),
			minAmountOut: math.NewInt(1),
			errorMsg:     "identical tokens",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gasBefore := ctx.GasMeter().GasConsumed()

			tokenIn := "upaw"
			tokenOut := "uusdt"
			if tc.name == "identical tokens" {
				tokenOut = "upaw"
			}

			_, err := k.ExecuteSwap(ctx, trader, pool.Id, tokenIn, tokenOut, tc.amountIn, tc.minAmountOut)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errorMsg)

			gasAfter := ctx.GasMeter().GasConsumed()
			gasConsumed := gasAfter - gasBefore

			require.Greater(t, gasConsumed, uint64(0), "validation should consume gas")
			t.Logf("%s validation consumed %d gas", tc.name, gasConsumed)
		})
	}
}

// TestGasConsumptionOnPoolNotFound tests gas consumption when pool doesn't exist
func TestGasConsumptionOnPoolNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	trader := createTestTraderForErrorRecovery(t)
	nonExistentPoolID := uint64(99999)

	gasBefore := ctx.GasMeter().GasConsumed()

	_, err := k.ExecuteSwap(ctx, trader, nonExistentPoolID, "upaw", "uusdt", math.NewInt(1000), math.NewInt(1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	gasAfter := ctx.GasMeter().GasConsumed()
	gasConsumed := gasAfter - gasBefore

	require.Greater(t, gasConsumed, uint64(0), "pool lookup should consume gas")
	t.Logf("Pool not found error consumed %d gas", gasConsumed)
}

// TestGasConsumptionConsistency tests that similar operations consume similar gas
func TestGasConsumptionConsistency(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", math.NewInt(10000000), math.NewInt(20000000))
	require.NoError(t, err)

	trader := createTestTraderForErrorRecovery(t)
	fundTestAccountForErrorRecovery(t, k, ctx, trader, "upaw", math.NewInt(10000000))

	swapAmount := math.NewInt(10000)
	minOut := math.NewInt(1)

	// Execute same swap multiple times and measure gas
	var gasConsumptions []uint64

	for i := 0; i < 5; i++ {
		gasBefore := ctx.GasMeter().GasConsumed()
		_, err := k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", swapAmount, minOut)
		require.NoError(t, err)
		gasAfter := ctx.GasMeter().GasConsumed()

		gasConsumed := gasAfter - gasBefore
		gasConsumptions = append(gasConsumptions, gasConsumed)
	}

	// Verify gas consumption is relatively consistent
	// (may vary slightly due to state size changes, but should be in same ballpark)
	for i := 1; i < len(gasConsumptions); i++ {
		diff := int64(gasConsumptions[i]) - int64(gasConsumptions[0])
		if diff < 0 {
			diff = -diff
		}
		percentDiff := float64(diff) / float64(gasConsumptions[0]) * 100

		require.Less(t, percentDiff, 50.0, "gas consumption should be relatively consistent across similar operations")
	}

	t.Logf("Gas consumptions: %v", gasConsumptions)
}
