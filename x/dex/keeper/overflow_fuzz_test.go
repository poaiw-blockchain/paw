package keeper_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// FuzzSwapOverflow tests swap calculations with extreme values
func FuzzSwapOverflow(f *testing.F) {
	// Add seed values
	f.Add(int64(1000000), int64(2000000), int64(100000))         // Normal case
	f.Add(int64(1000000000), int64(2000000000), int64(10000000)) // Large case
	f.Add(int64(1), int64(1), int64(1))                          // Minimum case

	f.Fuzz(func(t *testing.T, amountIn, reserveIn, reserveOut int64) {
		// Skip invalid inputs
		if amountIn <= 0 || reserveIn <= 0 || reserveOut <= 0 {
			return
		}

		// Prevent overflow in initial conversion
		if amountIn > 1<<62 || reserveIn > 1<<62 || reserveOut > 1<<62 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)

		amountInInt := math.NewInt(amountIn)
		reserveInInt := math.NewInt(reserveIn)
		reserveOutInt := math.NewInt(reserveOut)

		// Test SafeCalculateSwapOutput - should never panic
		result, err := k.SafeCalculateSwapOutput(ctx, amountInInt, reserveInInt, reserveOutInt, math.LegacyZeroDec())

		// If error is returned, it should be ErrOverflow or ErrInsufficientLiquidity
		if err != nil {
			require.True(t,
				types.ErrOverflow.Is(err) || types.ErrInsufficientLiquidity.Is(err),
				"unexpected error type: %v", err,
			)
			return
		}

		// If successful, result should be valid
		require.False(t, result.IsNegative(), "result should not be negative")
		require.True(t, result.LT(reserveOutInt), "result should be less than reserve")
	})
}

// FuzzPoolSharesOverflow tests pool share calculation with extreme values
func FuzzPoolSharesOverflow(f *testing.F) {
	// Seed corpus
	f.Add(int64(1000000), int64(2000000))         // Normal
	f.Add(int64(1000000000), int64(2000000000))   // Large
	f.Add(int64(1), int64(1))                     // Minimum
	f.Add(int64(1<<30), int64(1<<30))             // Very large

	f.Fuzz(func(t *testing.T, amountA, amountB int64) {
		// Skip invalid inputs
		if amountA <= 0 || amountB <= 0 {
			return
		}

		// Prevent overflow in initial conversion
		if amountA > 1<<62 || amountB > 1<<62 {
			return
		}

		k, _ := keepertest.DexKeeper(t)

		amountAInt := math.NewInt(amountA)
		amountBInt := math.NewInt(amountB)

		// Test SafeCalculatePoolShares - should never panic
		shares, err := k.SafeCalculatePoolShares(amountAInt, amountBInt)

		// If error is returned, it should be ErrOverflow
		if err != nil {
			require.True(t, types.ErrOverflow.Is(err), "unexpected error type: %v", err)
			return
		}

		// If successful, shares should be positive
		require.True(t, shares.IsPositive(), "shares should be positive")

		// Verify geometric mean property: shares^2 should be approximately amountA * amountB
		// Allow some rounding error
		expectedProduct, err := amountAInt.SafeMul(amountBInt)
		if err == nil {
			shareSquared, err := shares.SafeMul(shares)
			if err == nil {
				// shares^2 should be close to amountA * amountB (within rounding)
				diff := expectedProduct.Sub(shareSquared).Abs()
				maxError := shares.MulRaw(2) // Allow small rounding error
				require.True(t, diff.LTE(maxError), "geometric mean verification failed: diff=%s, maxError=%s", diff, maxError)
			}
		}
	})
}

// FuzzAddLiquiditySharesOverflow tests add liquidity share calculation
func FuzzAddLiquiditySharesOverflow(f *testing.F) {
	// Seed corpus
	f.Add(int64(1000), int64(2000), int64(100000), int64(200000), int64(1000000))

	f.Fuzz(func(t *testing.T, amountA, amountB, reserveA, reserveB, totalShares int64) {
		// Skip invalid inputs
		if amountA <= 0 || amountB <= 0 || reserveA <= 0 || reserveB <= 0 || totalShares <= 0 {
			return
		}

		// Prevent overflow in initial conversion
		if amountA > 1<<50 || amountB > 1<<50 || reserveA > 1<<50 || reserveB > 1<<50 || totalShares > 1<<50 {
			return
		}

		k, _ := keepertest.DexKeeper(t)

		amountAInt := math.NewInt(amountA)
		amountBInt := math.NewInt(amountB)
		reserveAInt := math.NewInt(reserveA)
		reserveBInt := math.NewInt(reserveB)
		totalSharesInt := math.NewInt(totalShares)

		// Test SafeCalculateAddLiquidityShares - should never panic
		shares, err := k.SafeCalculateAddLiquidityShares(amountAInt, amountBInt, reserveAInt, reserveBInt, totalSharesInt)

		// If error is returned, it should be ErrOverflow or ErrInsufficientLiquidity
		if err != nil {
			require.True(t,
				types.ErrOverflow.Is(err) || types.ErrInsufficientLiquidity.Is(err),
				"unexpected error type: %v", err,
			)
			return
		}

		// If successful, shares should be positive
		require.True(t, shares.IsPositive(), "shares should be positive")

		// Verify proportionality: shares should maintain pool ratio
		// sharesA = amountA * totalShares / reserveA
		expectedSharesA, err1 := amountAInt.SafeMul(totalSharesInt)
		if err1 == nil {
			expectedSharesA, err1 = expectedSharesA.SafeQuo(reserveAInt)
		}

		expectedSharesB, err2 := amountBInt.SafeMul(totalSharesInt)
		if err2 == nil {
			expectedSharesB, err2 = expectedSharesB.SafeQuo(reserveBInt)
		}

		if err1 == nil && err2 == nil {
			// Returned shares should be minimum of the two
			minExpected := math.MinInt(expectedSharesA, expectedSharesB)
			require.True(t, shares.Equal(minExpected), "shares should match minimum expected: got %s, expected %s", shares, minExpected)
		}
	})
}

// FuzzRemoveLiquidityAmountsOverflow tests remove liquidity amount calculation
func FuzzRemoveLiquidityAmountsOverflow(f *testing.F) {
	// Seed corpus
	f.Add(int64(10000), int64(100000), int64(200000), int64(1000000))

	f.Fuzz(func(t *testing.T, shares, reserveA, reserveB, totalShares int64) {
		// Skip invalid inputs
		if shares <= 0 || reserveA <= 0 || reserveB <= 0 || totalShares <= 0 {
			return
		}

		// shares must not exceed totalShares
		if shares > totalShares {
			return
		}

		// Prevent overflow in initial conversion
		if shares > 1<<50 || reserveA > 1<<50 || reserveB > 1<<50 || totalShares > 1<<50 {
			return
		}

		k, _ := keepertest.DexKeeper(t)

		sharesInt := math.NewInt(shares)
		reserveAInt := math.NewInt(reserveA)
		reserveBInt := math.NewInt(reserveB)
		totalSharesInt := math.NewInt(totalShares)

		// Test SafeCalculateRemoveLiquidityAmounts - should never panic
		amountA, amountB, err := k.SafeCalculateRemoveLiquidityAmounts(sharesInt, reserveAInt, reserveBInt, totalSharesInt)

		// If error is returned, it should be ErrOverflow or ErrInsufficientLiquidity
		if err != nil {
			require.True(t,
				types.ErrOverflow.Is(err) || types.ErrInsufficientLiquidity.Is(err),
				"unexpected error type: %v", err,
			)
			return
		}

		// If successful, amounts should be positive and less than reserves
		require.True(t, amountA.IsPositive() || amountA.IsZero(), "amountA should be non-negative")
		require.True(t, amountB.IsPositive() || amountB.IsZero(), "amountB should be non-negative")
		require.True(t, amountA.LTE(reserveAInt), "amountA should not exceed reserveA")
		require.True(t, amountB.LTE(reserveBInt), "amountB should not exceed reserveB")

		// Verify proportionality: amountA/reserveA ≈ shares/totalShares
		// amountA = shares * reserveA / totalShares
		expectedAmountA, err1 := sharesInt.SafeMul(reserveAInt)
		if err1 == nil {
			expectedAmountA, err1 = expectedAmountA.SafeQuo(totalSharesInt)
		}

		expectedAmountB, err2 := sharesInt.SafeMul(reserveBInt)
		if err2 == nil {
			expectedAmountB, err2 = expectedAmountB.SafeQuo(totalSharesInt)
		}

		if err1 == nil && err2 == nil {
			require.True(t, amountA.Equal(expectedAmountA), "amountA mismatch: got %s, expected %s", amountA, expectedAmountA)
			require.True(t, amountB.Equal(expectedAmountB), "amountB mismatch: got %s, expected %s", amountB, expectedAmountB)
		}
	})
}

// FuzzConstantProductInvariant tests that k-value never decreases
func FuzzConstantProductInvariant(f *testing.F) {
	// Seed corpus
	f.Add(int64(100000), int64(200000), int64(101000), int64(198000))

	f.Fuzz(func(t *testing.T, oldReserveA, oldReserveB, newReserveA, newReserveB int64) {
		// Skip invalid inputs
		if oldReserveA <= 0 || oldReserveB <= 0 || newReserveA <= 0 || newReserveB <= 0 {
			return
		}

		// Prevent overflow in initial conversion
		if oldReserveA > 1<<50 || oldReserveB > 1<<50 || newReserveA > 1<<50 || newReserveB > 1<<50 {
			return
		}

		k, _ := keepertest.DexKeeper(t)

		oldReserveAInt := math.NewInt(oldReserveA)
		oldReserveBInt := math.NewInt(oldReserveB)
		newReserveAInt := math.NewInt(newReserveA)
		newReserveBInt := math.NewInt(newReserveB)

		// Test SafeValidateConstantProduct - should never panic
		err := k.SafeValidateConstantProduct(oldReserveAInt, oldReserveBInt, newReserveAInt, newReserveBInt)

		// Calculate k-values manually to verify
		oldK, err1 := oldReserveAInt.SafeMul(oldReserveBInt)
		newK, err2 := newReserveAInt.SafeMul(newReserveBInt)

		if err1 != nil || err2 != nil {
			// If multiplication overflows, the function should return overflow error
			if err != nil {
				require.True(t, types.ErrOverflow.Is(err), "expected overflow error, got: %v", err)
			}
			return
		}

		// Check result
		if newK.LT(oldK) {
			// k decreased - should return error
			require.Error(t, err, "should return error when k decreases")
			require.True(t, types.ErrInvariantViolation.Is(err), "should return invariant violation error")
		} else {
			// k increased or stayed same - should succeed
			require.NoError(t, err, "should not return error when k increases or stays same")
		}
	})
}

// TestOverflowProtection_ExtremeValues tests overflow protection with extreme values
func TestOverflowProtection_ExtremeValues(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Test with values near the maximum for math.Int (2^255 - 1)
	maxSafe := new(big.Int).Lsh(big.NewInt(1), 200) // 2^200 (safe for multiplication)
	veryLarge := math.NewIntFromBigInt(maxSafe)

	t.Run("SafeCalculateSwapOutput with very large reserves", func(t *testing.T) {
		amountIn := math.NewInt(1000000)
		reserveIn := veryLarge
		reserveOut := veryLarge

		_, err := k.SafeCalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, math.LegacyZeroDec())
		// Should not panic, may return overflow error or valid result
		if err != nil {
			require.True(t, types.ErrOverflow.Is(err) || types.ErrInsufficientLiquidity.Is(err))
		}
	})

	t.Run("SafeCalculatePoolShares with maximum values", func(t *testing.T) {
		// This should overflow
		_, err := k.SafeCalculatePoolShares(veryLarge, veryLarge)
		require.Error(t, err)
		require.True(t, types.ErrOverflow.Is(err))
	})

	t.Run("SafeCalculateAddLiquidityShares with large values", func(t *testing.T) {
		amountA := math.NewInt(1000000)
		amountB := math.NewInt(2000000)
		reserveA := veryLarge
		reserveB := veryLarge
		totalShares := math.NewInt(1000000000)

		_, err := k.SafeCalculateAddLiquidityShares(amountA, amountB, reserveA, reserveB, totalShares)
		// Should not panic
		if err != nil {
			require.True(t, types.ErrOverflow.Is(err) || types.ErrInsufficientLiquidity.Is(err))
		}
	})
}

// TestOverflowProtection_Integration tests overflow protection in integrated swap scenario
func TestOverflowProtection_Integration(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create a pool with large reserves
	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"

	// Use large but safe amounts
	amountA := math.NewInt(1_000_000_000_000_000) // 1 quadrillion
	amountB := math.NewInt(2_000_000_000_000_000) // 2 quadrillion

	// Fund creator with sufficient tokens
	keepertest.FundAccount(t, k, ctx, creator, sdk.NewCoins(
		sdk.NewCoin(tokenA, amountA.Add(math.NewInt(100_000_000_000_000))),
		sdk.NewCoin(tokenB, amountB.Add(math.NewInt(100_000_000_000_000))),
	))

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)
	require.NotNil(t, pool)

	// Try to swap a very large amount
	trader := createTestTrader(t)
	swapAmount := math.NewInt(100_000_000_000_000) // 100 trillion
	minOut := math.NewInt(1)

	// This should either succeed or fail gracefully with overflow error
	amountOut, err := k.ExecuteSwap(ctx, trader, pool.Id, tokenA, tokenB, swapAmount, minOut)
	if err != nil {
		// Should be a recognized error type, not a panic
		require.True(t,
			types.ErrOverflow.Is(err) ||
				types.ErrInsufficientLiquidity.Is(err) ||
				types.ErrSlippageTooHigh.Is(err) ||
				types.ErrInvalidSwapAmount.Is(err),
			"unexpected error: %v", err,
		)
	} else {
		// If successful, verify result is sane
		require.True(t, amountOut.IsPositive())
		require.True(t, amountOut.LT(amountB)) // Can't get more than reserve
	}
}

// TestOverflowProtection_SequentialSwaps tests that overflow protection works over multiple swaps
func TestOverflowProtection_SequentialSwaps(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	creator := types.TestAddr()
	tokenA := "upaw"
	tokenB := "uusdt"
	amountA := math.NewInt(10_000_000_000) // 10 billion
	amountB := math.NewInt(20_000_000_000) // 20 billion

	// Fund creator with sufficient tokens
	keepertest.FundAccount(t, k, ctx, creator, sdk.NewCoins(
		sdk.NewCoin(tokenA, amountA.MulRaw(2)),
		sdk.NewCoin(tokenB, amountB.MulRaw(2)),
	))

	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	trader := createTestTrader(t)

	// Perform multiple swaps, each time checking for overflow
	numSwaps := 100
	swapAmount := math.NewInt(1_000_000) // 1 million per swap

	for i := 0; i < numSwaps; i++ {
		// Alternate swap direction
		var tokenIn, tokenOut string
		if i%2 == 0 {
			tokenIn = tokenA
			tokenOut = tokenB
		} else {
			tokenIn = tokenB
			tokenOut = tokenA
		}

		_, err := k.ExecuteSwap(ctx, trader, pool.Id, tokenIn, tokenOut, swapAmount, math.NewInt(1))

		// Should not panic, even if it errors
		if err != nil {
			// Verify it's a known error type
			require.True(t,
				types.ErrOverflow.Is(err) ||
					types.ErrInsufficientLiquidity.Is(err) ||
					types.ErrSlippageTooHigh.Is(err) ||
					types.ErrInvalidSwapAmount.Is(err),
				"swap %d failed with unexpected error: %v", i, err,
			)
			break
		}

		// Verify pool state is still valid
		poolState, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		require.True(t, poolState.ReserveA.IsPositive())
		require.True(t, poolState.ReserveB.IsPositive())
		require.True(t, poolState.TotalShares.IsPositive())

		// Verify k-value hasn't decreased
		oldK, _ := pool.ReserveA.SafeMul(pool.ReserveB)
		newK, _ := poolState.ReserveA.SafeMul(poolState.ReserveB)
		require.True(t, newK.GTE(oldK), "k-value decreased after swap %d", i)

		pool = poolState
	}
}
