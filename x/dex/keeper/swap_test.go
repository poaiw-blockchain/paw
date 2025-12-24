package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Helper function to create pool for swap tests
func setupPoolForSwaps(t *testing.T, k *keeper.Keeper, ctx sdk.Context) uint64 {
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10000000) // 10M upaw
	amountB := math.NewInt(20000000) // 20M uusdt

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	return pool.Id
}

// TestSwap_Valid tests successful token swap
func TestSwap_Valid(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000) // 1M upaw
	minAmountOut := math.NewInt(1)   // Accept any amount

	amountOut, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.NoError(t, err)
	require.True(t, amountOut.IsPositive())

	// Verify pool reserves updated
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(10997000), pool.ReserveA)   // 10M + 1M minus swap fees
	require.True(t, pool.ReserveB.LT(math.NewInt(20000000))) // Less than 20M due to output
}

// TestSwap_ZeroAmount tests rejection of zero swap amount
func TestSwap_ZeroAmount(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(0)
	minAmountOut := math.NewInt(0)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be positive")
}

// TestSwap_IdenticalTokens tests rejection of swapping same token
func TestSwap_IdenticalTokens(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "upaw"
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(1)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "identical tokens")
}

// TestSwap_PoolNotFound tests swap with non-existent pool
func TestSwap_PoolNotFound(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	trader := createTestTrader(t)

	poolID := uint64(99999)
	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(1)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestSwap_InvalidTokenPair tests swap with wrong token pair
func TestSwap_InvalidTokenPair(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uatom" // Pool doesn't have uatom
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(1)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid token pair")
}

// TestSwap_SlippageProtection tests slippage limit enforcement
func TestSwap_SlippageProtection(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000)
	// Set minAmountOut to unrealistically high value
	minAmountOut := math.NewInt(100000000)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")
}

// TestSwap_ConstantProductInvariant tests that K doesn't decrease
func TestSwap_ConstantProductInvariant(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	// Get initial pool state
	poolBefore, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	kBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)

	// Execute swap
	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000)
	minAmountOut := math.NewInt(1)

	_, err = k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.NoError(t, err)

	// Get pool state after swap
	poolAfter, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	kAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)

	// K should not decrease (may increase slightly due to fees)
	require.True(t, kAfter.GTE(kBefore), "constant product invariant violated")
}

// TestSwap_FeeDeduction ensures output is lower than a no-fee constant product calculation.
func TestSwap_FeeDeduction(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	poolBefore, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1_000_000)

	// No-fee output for constant-product AMM: out = Δx * R_y / (R_x + Δx)
	noFeeOut := amountIn.Mul(poolBefore.ReserveB).Quo(poolBefore.ReserveA.Add(amountIn))

	actualOut, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, math.NewInt(1))
	require.NoError(t, err)

	require.True(t, actualOut.LT(noFeeOut), "fee not applied: got >= no-fee output")
	require.True(t, actualOut.IsPositive())
}

// TestSwap_FeesCalculated tests that fees are applied correctly
func TestSwap_FeesCalculated(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000)

	// Simulate swap to get expected output
	expectedOut, err := k.SimulateSwap(ctx, poolID, tokenIn, tokenOut, amountIn)
	require.NoError(t, err)

	// Execute actual swap
	actualOut, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, math.NewInt(1))
	require.NoError(t, err)

	// Actual output should match simulation
	require.Equal(t, expectedOut, actualOut)

	// Output should be less than a no-fee calculation would give
	// (since fees are deducted)
	require.True(t, actualOut.LT(amountIn.MulRaw(2))) // Less than 2:1 ratio
}

func TestSwap_FeeCollectorReceivesFees(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1000000)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	expectedFee := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()

	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	balanceBefore := k.BankKeeper().GetBalance(ctx, feeCollectorAddr, tokenIn)

	_, err = k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, math.NewInt(1))
	require.NoError(t, err)

	balanceAfter := k.BankKeeper().GetBalance(ctx, feeCollectorAddr, tokenIn)
	require.True(t, balanceAfter.Amount.Equal(balanceBefore.Amount.Add(expectedFee)))
}

// TestPoolDrainLimit ensures swaps cannot drain more than the configured percentage of reserves.
func TestPoolDrainLimit(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	// Simulate attempting to drain ~50% of reserveOut when limit is 30%.
	amountIn := math.NewInt(5_000_000)
	reserveIn := math.NewInt(10_000_000)
	reserveOut := math.NewInt(20_000_000)

	_, err = k.CalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, params.SwapFee, params.MaxPoolDrainPercent)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrSwapTooLarge)
	require.Contains(t, err.Error(), "drain too much liquidity")
}

// TestSwap_BidirectionalSwap tests swapping in both directions
func TestSwap_BidirectionalSwap(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	// Swap upaw -> uusdt
	amountOut1, err := k.ExecuteSwap(ctx, trader, poolID, "upaw", "uusdt", math.NewInt(1000000), math.NewInt(1))
	require.NoError(t, err)
	require.True(t, amountOut1.IsPositive())

	// Swap uusdt -> upaw
	amountOut2, err := k.ExecuteSwap(ctx, trader, poolID, "uusdt", "upaw", math.NewInt(1000000), math.NewInt(1))
	require.NoError(t, err)
	require.True(t, amountOut2.IsPositive())

	// Both swaps should succeed
	require.True(t, amountOut1.IsPositive())
	require.True(t, amountOut2.IsPositive())
}

// TestSwap_MultipleSwaps tests sequential swaps
func TestSwap_MultipleSwaps(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(100000)

	// Execute 10 small swaps
	for i := 0; i < 10; i++ {
		amountOut, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, math.NewInt(1))
		require.NoError(t, err)
		require.True(t, amountOut.IsPositive())
	}

	// Verify pool reserves changed
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(10997000), pool.ReserveA) // 10M + 1M - fees
}

// TestSwap_LargeAmount ensures large swaps above MEV threshold are rejected
func TestSwap_LargeAmount(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	// Try to swap half the pool
	amountIn := math.NewInt(5000000)
	minAmountOut := math.NewInt(1)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "swap size")
}

// TestSwap_InsufficientLiquidity tests swap exceeding allowed size relative to liquidity
func TestSwap_InsufficientLiquidity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	tokenIn := "upaw"
	tokenOut := "uusdt"
	// Try to swap amount that would drain the pool
	amountIn := math.NewInt(1000000000000) // 1 trillion
	minAmountOut := math.NewInt(1)

	_, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	require.Error(t, err)
	require.Contains(t, err.Error(), "swap size")
}

// TestCalculateSwapOutput tests swap output calculation

// TestSimulateSwap tests swap simulation without execution

// TestGetSpotPrice tests spot price calculation

// TestSwap_PriceImpact tests that large swaps have high price impact
func TestSwap_PriceImpact(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	// Get initial spot price
	spotPriceBefore, err := k.GetSpotPrice(ctx, poolID, "upaw", "uusdt")
	require.NoError(t, err)

	// Execute large swap (10% of pool)
	amountIn := math.NewInt(1000000)
	_, err = k.ExecuteSwap(ctx, trader, poolID, "upaw", "uusdt", amountIn, math.NewInt(1))
	require.NoError(t, err)

	// Get spot price after swap
	spotPriceAfter, err := k.GetSpotPrice(ctx, poolID, "upaw", "uusdt")
	require.NoError(t, err)

	// Price should have changed (price impact)
	require.NotEqual(t, spotPriceBefore, spotPriceAfter)
	// Price should be worse (lower) for the trader
	require.True(t, spotPriceAfter.LT(spotPriceBefore))
}

// TestSwap_RoundingErrors tests handling of dust amounts
func TestSwap_RoundingErrors(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := setupPoolForSwaps(t, k, ctx)
	trader := createTestTrader(t)

	// Very small swap
	tokenIn := "upaw"
	tokenOut := "uusdt"
	amountIn := math.NewInt(1) // 1 unit
	minAmountOut := math.NewInt(0)

	amountOut, err := k.ExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)

	// May fail if output rounds to zero
	if err != nil {
		require.Contains(t, err.Error(), "too small")
	} else {
		require.True(t, amountOut.IsPositive())
	}
}
