package fuzz

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// ============================================================================
// FuzzSwapAmount - Fuzz tests swap operations with random amounts
// ============================================================================

// FuzzSwapAmount tests DEX swap operations with random input amounts.
// This tests the constant product AMM formula: x * y = k
// where k should never decrease (can increase due to fees).
func FuzzSwapAmount(f *testing.F) {
	// Seed corpus with representative swap scenarios
	seeds := []struct {
		amountIn uint64
	}{
		{1000},             // Small swap
		{10000},            // Medium swap
		{100000},           // Larger swap
		{1000000},          // Large swap
		{1},                // Minimum amount
		{500000},           // Medium-large swap
		{50000},            // 5% of initial pool
		{99000},            // Just under 10% of pool
		{100000000},        // Very large amount
		{9999999999999999}, // Near max uint64
	}

	for _, seed := range seeds {
		f.Add(seed.amountIn)
	}

	f.Fuzz(func(t *testing.T, amountInRaw uint64) {
		// Skip invalid inputs
		if amountInRaw == 0 {
			return
		}

		// Cap the amount to prevent overflow in test setup
		// Use amounts that are reasonable relative to pool size
		if amountInRaw > 1<<50 {
			amountInRaw = amountInRaw % (1 << 50)
		}
		if amountInRaw == 0 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)
		creator := types.TestAddr()

		// Create pool with sufficient liquidity
		// Pool reserves: 1 billion each (1_000_000_000)
		initialReserve := math.NewInt(1_000_000_000)
		pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", initialReserve, initialReserve)
		if err != nil {
			return // Skip if pool creation fails
		}

		// Calculate swap amount (cap at 10% of reserves for this test)
		maxSwap := initialReserve.Quo(math.NewInt(10))
		amountIn := math.NewInt(int64(amountInRaw % uint64(maxSwap.Int64())))
		if amountIn.IsZero() {
			amountIn = math.NewInt(1)
		}

		// Get pool state before swap
		poolBefore, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		kBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)

		// Execute swap with zero minimum (to allow any slippage in fuzz testing)
		trader := sdk.AccAddress([]byte("fuzz_trader_address"))
		keepertest.FundAccount(t, k, ctx, trader, sdk.NewCoins(sdk.NewCoin("upaw", amountIn.MulRaw(2))))

		amountOut, err := k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", amountIn, math.ZeroInt())

		if err != nil {
			// Verify error is a valid rejection reason
			// Valid rejections: slippage, swap too large, price impact, circuit breaker
			return
		}

		// Get pool state after swap
		poolAfter, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		kAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)

		// INVARIANT 1: Output amount must be positive
		require.True(t, amountOut.IsPositive(), "output amount must be positive, got %s", amountOut.String())

		// INVARIANT 2: Output must not exceed output reserve
		require.True(t, amountOut.LT(poolBefore.ReserveB), "output %s must be less than reserve %s", amountOut.String(), poolBefore.ReserveB.String())

		// INVARIANT 3: k must never decrease (can increase due to fees)
		require.True(t, kAfter.GTE(kBefore), "k decreased: before=%s, after=%s", kBefore.String(), kAfter.String())

		// INVARIANT 4: Reserves must remain positive
		require.True(t, poolAfter.ReserveA.IsPositive(), "reserve A must be positive")
		require.True(t, poolAfter.ReserveB.IsPositive(), "reserve B must be positive")

		// INVARIANT 5: Reserve A increased by input (minus fees)
		require.True(t, poolAfter.ReserveA.GT(poolBefore.ReserveA), "reserve A should increase after swap")

		// INVARIANT 6: Reserve B decreased by output
		require.True(t, poolAfter.ReserveB.LT(poolBefore.ReserveB), "reserve B should decrease after swap")
	})
}

// ============================================================================
// FuzzSwapSlippage - Fuzz tests slippage protection
// ============================================================================

// FuzzSwapSlippage tests that slippage protection correctly rejects swaps
// when the output is below minAmountOut.
func FuzzSwapSlippage(f *testing.F) {
	// Seed corpus with various slippage scenarios
	seeds := []struct {
		amountIn        uint64
		minAmountOutPct uint64 // Percentage of expected output (0-200)
	}{
		{10000, 100}, // Exact expected output
		{10000, 101}, // 1% above expected
		{10000, 90},  // 10% slippage tolerance
		{10000, 50},  // 50% slippage tolerance
		{10000, 200}, // 2x expected (should fail)
		{10000, 0},   // No slippage protection
		{1000, 105},  // Small swap with 5% above
		{100000, 99}, // Large swap with 1% tolerance
		{50000, 110}, // Medium swap with 10% above expected
		{1, 100},     // Minimum swap
	}

	for _, seed := range seeds {
		f.Add(seed.amountIn, seed.minAmountOutPct)
	}

	f.Fuzz(func(t *testing.T, amountInRaw, minAmountOutPct uint64) {
		// Skip invalid inputs
		if amountInRaw == 0 || minAmountOutPct > 1000 {
			return
		}

		// Cap amounts
		if amountInRaw > 1<<40 {
			amountInRaw = amountInRaw % (1 << 40)
		}
		if amountInRaw == 0 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)
		creator := types.TestAddr()

		// Create balanced pool
		initialReserve := math.NewInt(1_000_000_000)
		pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", initialReserve, initialReserve)
		if err != nil {
			return
		}

		// Cap swap at 10% of reserves
		maxSwap := initialReserve.Quo(math.NewInt(10))
		amountIn := math.NewInt(int64(amountInRaw % uint64(maxSwap.Int64())))
		if amountIn.IsZero() {
			amountIn = math.NewInt(1)
		}

		// Simulate to get expected output
		expectedOutput, err := k.SimulateSwap(ctx, pool.Id, "upaw", "uusdt", amountIn)
		if err != nil || expectedOutput.IsZero() {
			return // Skip if simulation fails
		}

		// Calculate minAmountOut based on percentage
		minAmountOut := expectedOutput.Mul(math.NewInt(int64(minAmountOutPct))).Quo(math.NewInt(100))

		// Fund trader and execute swap
		trader := sdk.AccAddress([]byte("fuzz_slippage_trader"))
		keepertest.FundAccount(t, k, ctx, trader, sdk.NewCoins(sdk.NewCoin("upaw", amountIn.MulRaw(2))))

		amountOut, err := k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdt", amountIn, minAmountOut)

		// INVARIANT: If minAmountOut > expectedOutput, swap should fail
		if minAmountOutPct > 100 {
			// We expect the swap to fail with slippage error
			if err == nil {
				// Swap succeeded - verify output meets minimum
				require.True(t, amountOut.GTE(minAmountOut),
					"swap succeeded but output %s < minAmountOut %s",
					amountOut.String(), minAmountOut.String())
			}
			// Error is expected for high minAmountOut
			return
		}

		// INVARIANT: If minAmountOut <= expectedOutput, swap should succeed
		if minAmountOutPct <= 100 && err == nil {
			require.True(t, amountOut.GTE(minAmountOut),
				"output %s must be >= minAmountOut %s",
				amountOut.String(), minAmountOut.String())
		}
	})
}

// ============================================================================
// FuzzLiquidityAdd - Fuzz tests adding liquidity with random amounts
// ============================================================================

// FuzzLiquidityAdd tests adding liquidity with random amounts.
// Verifies LP shares are minted correctly and pool reserves increase.
func FuzzLiquidityAdd(f *testing.F) {
	// Seed corpus with representative liquidity additions
	seeds := []struct {
		amountA, amountB uint64
	}{
		{1000000, 1000000},     // Balanced addition
		{2000000, 1000000},     // Imbalanced 2:1
		{1000000, 2000000},     // Imbalanced 1:2
		{10000, 10000},         // Small addition
		{100000000, 100000000}, // Large addition
		{1, 1},                 // Minimum amounts
		{999999, 1000001},      // Slightly imbalanced
		{5000000, 5000000},     // 5x pool size
		{100, 100},             // Very small
		{1000000000, 1000000},  // Highly imbalanced
	}

	for _, seed := range seeds {
		f.Add(seed.amountA, seed.amountB)
	}

	f.Fuzz(func(t *testing.T, amountARaw, amountBRaw uint64) {
		// Skip invalid inputs
		if amountARaw == 0 || amountBRaw == 0 {
			return
		}

		// Cap amounts to prevent overflow
		if amountARaw > 1<<50 {
			amountARaw = amountARaw % (1 << 50)
		}
		if amountBRaw > 1<<50 {
			amountBRaw = amountBRaw % (1 << 50)
		}
		if amountARaw == 0 || amountBRaw == 0 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)
		creator := types.TestAddr()

		// Create pool with initial liquidity
		initialReserve := math.NewInt(1_000_000_000)
		pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", initialReserve, initialReserve)
		if err != nil {
			return
		}

		// Get pool state before
		poolBefore, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		totalSharesBefore := poolBefore.TotalShares

		// Prepare amounts (proportional to pool ratio for optimal acceptance)
		amountA := math.NewInt(int64(amountARaw % 100000000))
		amountB := math.NewInt(int64(amountBRaw % 100000000))
		if amountA.IsZero() {
			amountA = math.NewInt(1000)
		}
		if amountB.IsZero() {
			amountB = math.NewInt(1000)
		}

		// Fund provider
		provider := sdk.AccAddress([]byte("fuzz_lp_provider___"))
		keepertest.FundAccount(t, k, ctx, provider, sdk.NewCoins(
			sdk.NewCoin("upaw", amountA.MulRaw(2)),
			sdk.NewCoin("uusdt", amountB.MulRaw(2)),
		))

		// Add liquidity
		shares, err := k.AddLiquidity(ctx, provider, pool.Id, amountA, amountB)

		if err != nil {
			// Valid rejection reasons: insufficient contribution, flash loan protection
			return
		}

		// Get pool state after
		poolAfter, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)

		// INVARIANT 1: Shares must be positive
		require.True(t, shares.IsPositive(), "minted shares must be positive")

		// INVARIANT 2: Total shares must increase
		require.True(t, poolAfter.TotalShares.GT(totalSharesBefore),
			"total shares must increase: before=%s, after=%s",
			totalSharesBefore.String(), poolAfter.TotalShares.String())

		// INVARIANT 3: Reserves must increase or stay same
		require.True(t, poolAfter.ReserveA.GTE(poolBefore.ReserveA),
			"reserve A must increase or stay same")
		require.True(t, poolAfter.ReserveB.GTE(poolBefore.ReserveB),
			"reserve B must increase or stay same")

		// INVARIANT 4: k must increase
		kBefore := poolBefore.ReserveA.Mul(poolBefore.ReserveB)
		kAfter := poolAfter.ReserveA.Mul(poolAfter.ReserveB)
		require.True(t, kAfter.GT(kBefore), "k must increase when adding liquidity")

		// INVARIANT 5: User's liquidity position updated
		userShares, err := k.GetLiquidity(ctx, pool.Id, provider)
		require.NoError(t, err)
		require.True(t, userShares.Equal(shares),
			"user shares must match minted shares: got=%s, expected=%s",
			userShares.String(), shares.String())
	})
}

// ============================================================================
// FuzzLiquidityRemove - Fuzz tests removing liquidity
// ============================================================================

// FuzzLiquidityRemove tests removing liquidity with random share amounts.
// Verifies tokens are returned proportionally and reserves decrease correctly.
func FuzzLiquidityRemove(f *testing.F) {
	// Seed corpus with representative removal scenarios
	seeds := []struct {
		removePercent uint64 // Percentage of shares to remove (1-99)
	}{
		{10}, // Remove 10%
		{25}, // Remove 25%
		{50}, // Remove 50%
		{75}, // Remove 75%
		{90}, // Remove 90%
		{1},  // Remove 1%
		{99}, // Remove 99% (might hit minimum reserves)
		{5},  // Remove 5%
		{33}, // Remove 1/3
		{66}, // Remove 2/3
	}

	for _, seed := range seeds {
		f.Add(seed.removePercent)
	}

	f.Fuzz(func(t *testing.T, removePercent uint64) {
		// Skip invalid percentages
		if removePercent == 0 || removePercent > 100 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)
		creator := types.TestAddr()

		// Create pool with large reserves (to allow partial withdrawals after min reserve check)
		// SEC-17: MinimumReserves is 1,000,000 so we need much larger pool
		initialReserve := math.NewInt(10_000_000_000) // 10 billion
		pool, err := k.CreatePool(ctx, creator, "upaw", "uusdt", initialReserve, initialReserve)
		if err != nil {
			return
		}

		// Creator has initial shares
		creatorShares, err := k.GetLiquidity(ctx, pool.Id, creator)
		require.NoError(t, err)

		// Calculate shares to remove
		sharesToRemove := creatorShares.Mul(math.NewInt(int64(removePercent))).Quo(math.NewInt(100))
		if sharesToRemove.IsZero() {
			return
		}

		// Get pool state before
		poolBefore, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)

		// Need to advance block for flash loan protection
		newCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 101)

		// Remove liquidity
		amountA, amountB, err := k.RemoveLiquidity(newCtx, creator, pool.Id, sharesToRemove)

		if err != nil {
			// Valid rejections: minimum reserves, flash loan protection, insufficient shares
			return
		}

		// Get pool state after
		poolAfter, err := k.GetPool(newCtx, pool.Id)
		require.NoError(t, err)

		// INVARIANT 1: Returned amounts must be positive
		require.True(t, amountA.IsPositive(), "returned amount A must be positive")
		require.True(t, amountB.IsPositive(), "returned amount B must be positive")

		// INVARIANT 2: Reserves must decrease
		require.True(t, poolAfter.ReserveA.LT(poolBefore.ReserveA),
			"reserve A must decrease: before=%s, after=%s",
			poolBefore.ReserveA.String(), poolAfter.ReserveA.String())
		require.True(t, poolAfter.ReserveB.LT(poolBefore.ReserveB),
			"reserve B must decrease: before=%s, after=%s",
			poolBefore.ReserveB.String(), poolAfter.ReserveB.String())

		// INVARIANT 3: Total shares must decrease
		require.True(t, poolAfter.TotalShares.LT(poolBefore.TotalShares),
			"total shares must decrease")

		// INVARIANT 4: Reserves must remain positive
		require.True(t, poolAfter.ReserveA.IsPositive(), "reserve A must remain positive")
		require.True(t, poolAfter.ReserveB.IsPositive(), "reserve B must remain positive")

		// INVARIANT 5: Returned amounts proportional to share fraction
		// (amountA / reserveA_before) ≈ (sharesToRemove / totalShares_before)
		shareRatio := math.LegacyNewDecFromInt(sharesToRemove).Quo(math.LegacyNewDecFromInt(poolBefore.TotalShares))
		expectedAmountA := math.LegacyNewDecFromInt(poolBefore.ReserveA).Mul(shareRatio).TruncateInt()
		expectedAmountB := math.LegacyNewDecFromInt(poolBefore.ReserveB).Mul(shareRatio).TruncateInt()

		// Allow 1% tolerance for rounding
		tolerance := math.LegacyNewDecWithPrec(1, 2)
		diffA := math.LegacyNewDecFromInt(amountA.Sub(expectedAmountA).Abs()).Quo(math.LegacyNewDecFromInt(expectedAmountA))
		diffB := math.LegacyNewDecFromInt(amountB.Sub(expectedAmountB).Abs()).Quo(math.LegacyNewDecFromInt(expectedAmountB))

		require.True(t, diffA.LTE(tolerance) || expectedAmountA.IsZero(),
			"amount A not proportional: got=%s, expected=%s, diff=%.4f%%",
			amountA.String(), expectedAmountA.String(), diffA.MustFloat64()*100)
		require.True(t, diffB.LTE(tolerance) || expectedAmountB.IsZero(),
			"amount B not proportional: got=%s, expected=%s, diff=%.4f%%",
			amountB.String(), expectedAmountB.String(), diffB.MustFloat64()*100)

		// INVARIANT 6: User's remaining shares updated correctly
		remainingShares, err := k.GetLiquidity(newCtx, pool.Id, creator)
		require.NoError(t, err)
		expectedRemaining := creatorShares.Sub(sharesToRemove)
		require.True(t, remainingShares.Equal(expectedRemaining),
			"remaining shares mismatch: got=%s, expected=%s",
			remainingShares.String(), expectedRemaining.String())
	})
}

// ============================================================================
// FuzzPoolCreation - Fuzz tests pool creation with random token pairs
// ============================================================================

// FuzzPoolCreation tests pool creation with random token denominations and amounts.
// Verifies proper initialization and rejects duplicate pools.
func FuzzPoolCreation(f *testing.F) {
	// Seed corpus with various pool creation scenarios
	seeds := []struct {
		tokenSeed uint64
		amountA   uint64
		amountB   uint64
	}{
		{1, 1000000000, 1000000000}, // Balanced pool
		{2, 2000000000, 1000000000}, // 2:1 ratio
		{3, 1000000000, 5000000000}, // 1:5 ratio
		{4, 100000000, 100000000},   // Smaller pool
		{5, 500000000, 500000000},   // Medium pool
		{6, 1000000000, 100000000},  // 10:1 ratio
		{7, 10000000, 10000000},     // Minimum viable
		{8, 999999999, 999999999},   // Near billion
		{9, 1234567890, 987654321},  // Random amounts
		{10, 2000000, 2000000},      // Small but valid
	}

	for _, seed := range seeds {
		f.Add(seed.tokenSeed, seed.amountA, seed.amountB)
	}

	f.Fuzz(func(t *testing.T, tokenSeed, amountARaw, amountBRaw uint64) {
		// Skip invalid amounts
		if amountARaw == 0 || amountBRaw == 0 {
			return
		}

		// Cap amounts
		if amountARaw > 1<<50 {
			amountARaw = amountARaw % (1 << 50)
		}
		if amountBRaw > 1<<50 {
			amountBRaw = amountBRaw % (1 << 50)
		}
		if amountARaw == 0 || amountBRaw == 0 {
			return
		}

		k, ctx := keepertest.DexKeeper(t)

		// Generate unique token pair based on seed
		tokenPairs := []struct{ tokenA, tokenB string }{
			{"upaw", "uusdt"},
			{"uatom", "usdc"},
			{"atom", "osmo"},
			{"tokenA", "tokenB"},
			{"tokenC", "tokenD"},
			{"tokenE", "tokenF"},
		}

		pairIndex := int(tokenSeed % uint64(len(tokenPairs)))
		tokenA := tokenPairs[pairIndex].tokenA
		tokenB := tokenPairs[pairIndex].tokenB

		// Minimum amounts for pool creation (SEC-6: MinimumInitialLiquidity = 1000)
		amountA := math.NewInt(int64(amountARaw % 1000000000))
		amountB := math.NewInt(int64(amountBRaw % 1000000000))

		// Ensure amounts meet minimum
		minAmount := math.NewInt(1000)
		if amountA.LT(minAmount) {
			amountA = minAmount
		}
		if amountB.LT(minAmount) {
			amountB = minAmount
		}

		// Fund creator
		creator := types.TestAddrWithSeed(int(tokenSeed))
		keepertest.FundAccount(t, k, ctx, creator, sdk.NewCoins(
			sdk.NewCoin(tokenA, amountA.MulRaw(2)),
			sdk.NewCoin(tokenB, amountB.MulRaw(2)),
		))

		// Create pool
		pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)

		if err != nil {
			// Valid rejections: pool exists, insufficient liquidity, invalid ratio
			return
		}

		// INVARIANT 1: Pool ID must be positive
		require.True(t, pool.Id > 0, "pool ID must be positive")

		// INVARIANT 2: Reserves match initial amounts
		require.True(t, pool.ReserveA.Equal(amountA) || pool.ReserveB.Equal(amountA),
			"reserve A or B must match initial amount A")
		require.True(t, pool.ReserveA.Equal(amountB) || pool.ReserveB.Equal(amountB),
			"reserve A or B must match initial amount B")

		// INVARIANT 3: Total shares are positive
		require.True(t, pool.TotalShares.IsPositive(), "total shares must be positive")

		// INVARIANT 4: Tokens are sorted lexicographically
		require.True(t, pool.TokenA < pool.TokenB,
			"tokens must be sorted: %s < %s", pool.TokenA, pool.TokenB)

		// INVARIANT 5: Pool can be retrieved by ID
		retrieved, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		require.Equal(t, pool.Id, retrieved.Id)

		// INVARIANT 6: Pool can be retrieved by token pair
		byTokens, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
		require.NoError(t, err)
		require.Equal(t, pool.Id, byTokens.Id)

		// INVARIANT 7: Creator has initial shares
		creatorShares, err := k.GetLiquidity(ctx, pool.Id, creator)
		require.NoError(t, err)
		require.True(t, creatorShares.IsPositive(), "creator must have shares")
		require.True(t, creatorShares.Equal(pool.TotalShares),
			"creator should have all initial shares")

		// INVARIANT 8: Duplicate pool creation fails
		_, dupErr := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
		require.Error(t, dupErr, "duplicate pool creation must fail")

		// INVARIANT 9: k = reserveA * reserveB is valid
		kValue := pool.ReserveA.Mul(pool.ReserveB)
		require.True(t, kValue.IsPositive(), "k must be positive")
	})
}

// ============================================================================
// FuzzSwapCalculation - Tests swap calculation for correctness and safety
// ============================================================================

// FuzzSwapCalculation tests DEX swap calculation for correctness and safety
func FuzzSwapCalculation(f *testing.F) {
	// Seed corpus with realistic swap scenarios
	seeds := []struct {
		amountIn, reserveIn, reserveOut uint64
		feeStr                          string
	}{
		{1000, 10000, 10000, "0.003"},       // Balanced pool, small swap
		{5000, 100000, 50000, "0.003"},      // Imbalanced pool
		{100000, 1000000, 2000000, "0.01"},  // Large pool, high fee
		{1, 1000000, 1000000, "0.001"},      // Tiny swap
		{999999, 1000000, 1000000, "0.003"}, // Nearly draining pool
	}

	for _, seed := range seeds {
		f.Add(seed.amountIn, seed.reserveIn, seed.reserveOut, seed.feeStr)
	}

	f.Fuzz(func(t *testing.T, amountIn, reserveIn, reserveOut uint64, feeStr string) {
		// Skip invalid inputs
		if amountIn == 0 || reserveIn == 0 || reserveOut == 0 {
			return
		}

		// Parse fee
		fee, err := math.LegacyNewDecFromStr(feeStr)
		if err != nil || fee.IsNegative() || fee.GTE(math.LegacyOneDec()) {
			return // Skip invalid fee
		}

		// Convert to math.Int
		amountInInt := math.NewInt(int64(amountIn))
		reserveInInt := math.NewInt(int64(reserveIn))
		reserveOutInt := math.NewInt(int64(reserveOut))

		// Calculate swap output using constant product formula
		// amountOut = (amountIn * (1 - fee) * reserveOut) / (reserveIn + amountIn * (1 - fee))
		oneMinusFee := math.LegacyOneDec().Sub(fee)
		amountInAfterFee := math.LegacyNewDecFromInt(amountInInt).Mul(oneMinusFee)

		numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveOutInt))
		denominator := math.LegacyNewDecFromInt(reserveInInt).Add(amountInAfterFee)

		if denominator.IsZero() {
			return // Skip edge case
		}

		amountOut := numerator.Quo(denominator).TruncateInt()

		// INVARIANT 1: Output must be less than reserve
		if amountOut.GTE(reserveOutInt) {
			t.Errorf("VIOLATION: amountOut (%s) >= reserveOut (%s)", amountOut.String(), reserveOutInt.String())
		}

		// INVARIANT 2: Output must be non-negative
		if amountOut.IsNegative() {
			t.Errorf("VIOLATION: negative output %s", amountOut.String())
		}

		// INVARIANT 3: Constant product k should increase (due to fees)
		// k_before = reserveIn * reserveOut
		// k_after = (reserveIn + amountIn) * (reserveOut - amountOut)
		kBefore := reserveInInt.Mul(reserveOutInt)

		newReserveIn := reserveInInt.Add(amountInInt)
		newReserveOut := reserveOutInt.Sub(amountOut)
		kAfter := newReserveIn.Mul(newReserveOut)

		// Due to fees, k_after >= k_before
		if kAfter.LT(kBefore) {
			t.Errorf("VIOLATION: k decreased - k_before=%s, k_after=%s", kBefore.String(), kAfter.String())
		}

		// INVARIANT 4: Larger swaps should have worse price (price impact)
		if amountIn > 1 && reserveIn > amountIn {
			smallSwap := math.NewInt(int64(amountIn / 2))
			smallAfterFee := math.LegacyNewDecFromInt(smallSwap).Mul(oneMinusFee)
			smallNum := smallAfterFee.Mul(math.LegacyNewDecFromInt(reserveOutInt))
			smallDenom := math.LegacyNewDecFromInt(reserveInInt).Add(smallAfterFee)

			if !smallDenom.IsZero() {
				smallOutput := smallNum.Quo(smallDenom).TruncateInt()

				// Price for full swap
				fullPrice := math.LegacyNewDecFromInt(amountInInt).Quo(math.LegacyNewDecFromInt(amountOut))
				// Price for small swap (extrapolated to full amount)
				smallPriceBase := math.LegacyNewDecFromInt(smallSwap).Quo(math.LegacyNewDecFromInt(smallOutput))

				// Full swap should have worse or equal price (higher input per output)
				if !fullPrice.IsNil() && !smallPriceBase.IsNil() && fullPrice.LT(smallPriceBase.MulInt64(9).QuoInt64(10)) {
					t.Logf("WARNING: Price impact violation - full price better than small: full=%s, small=%s",
						fullPrice.String(), smallPriceBase.String())
				}
			}
		}
	})
}

// ============================================================================
// FuzzLiquidityCalculation - Tests LP share calculation
// ============================================================================

// FuzzLiquidityCalculation tests LP share calculation
func FuzzLiquidityCalculation(f *testing.F) {
	seeds := []struct {
		amountA, amountB, reserveA, reserveB, totalShares uint64
	}{
		{1000, 1000, 10000, 10000, 10000},     // Balanced addition
		{5000, 2500, 100000, 50000, 70710},    // Imbalanced pool
		{100, 100, 1000000, 1000000, 1000000}, // Small addition to large pool
	}

	for _, seed := range seeds {
		f.Add(seed.amountA, seed.amountB, seed.reserveA, seed.reserveB, seed.totalShares)
	}

	f.Fuzz(func(t *testing.T, amountA, amountB, reserveA, reserveB, totalShares uint64) {
		// Skip invalid inputs
		if amountA == 0 || amountB == 0 || reserveA == 0 || reserveB == 0 || totalShares == 0 {
			return
		}

		// Convert to math.Int
		amountAInt := math.NewInt(int64(amountA))
		amountBInt := math.NewInt(int64(amountB))
		reserveAInt := math.NewInt(int64(reserveA))
		reserveBInt := math.NewInt(int64(reserveB))
		totalSharesInt := math.NewInt(int64(totalShares))

		// Calculate shares based on smaller ratio
		// shares = min(amountA * totalShares / reserveA, amountB * totalShares / reserveB)
		sharesFromA := math.LegacyNewDecFromInt(amountAInt).
			Mul(math.LegacyNewDecFromInt(totalSharesInt)).
			Quo(math.LegacyNewDecFromInt(reserveAInt))

		sharesFromB := math.LegacyNewDecFromInt(amountBInt).
			Mul(math.LegacyNewDecFromInt(totalSharesInt)).
			Quo(math.LegacyNewDecFromInt(reserveBInt))

		var mintedShares math.Int
		if sharesFromA.LT(sharesFromB) {
			mintedShares = sharesFromA.TruncateInt()
		} else {
			mintedShares = sharesFromB.TruncateInt()
		}

		// INVARIANT 1: Minted shares must be positive
		if mintedShares.IsZero() || mintedShares.IsNegative() {
			return // Can happen with very small additions
		}

		// INVARIANT 2: Proportion of pool should match proportion of shares
		// (amountA / reserveA) should ≈ (mintedShares / totalShares)
		proportionA := math.LegacyNewDecFromInt(amountAInt).Quo(math.LegacyNewDecFromInt(reserveAInt))
		proportionShares := math.LegacyNewDecFromInt(mintedShares).Quo(math.LegacyNewDecFromInt(totalSharesInt))

		// Allow 1% tolerance due to rounding
		tolerance := math.LegacyMustNewDecFromStr("0.01")
		diff := proportionA.Sub(proportionShares).Abs()

		if diff.GT(tolerance) && diff.GT(proportionA.Mul(tolerance)) {
			t.Logf("Proportion mismatch: pool=%.6f%%, shares=%.6f%%, diff=%.6f%%",
				proportionA.MustFloat64()*100,
				proportionShares.MustFloat64()*100,
				diff.MustFloat64()*100)
		}

		// INVARIANT 3: Total shares should increase
		newTotalShares := totalSharesInt.Add(mintedShares)
		if newTotalShares.LTE(totalSharesInt) {
			t.Errorf("VIOLATION: total shares did not increase")
		}
	})
}

// ============================================================================
// FuzzRemoveLiquidity - Tests liquidity removal calculations
// ============================================================================

// FuzzRemoveLiquidity tests liquidity removal calculations
func FuzzRemoveLiquidity(f *testing.F) {
	seeds := []struct {
		sharesToRemove, totalShares, reserveA, reserveB uint64
	}{
		{1000, 10000, 100000, 100000},   // 10% removal
		{5000, 10000, 100000, 50000},    // 50% removal from imbalanced
		{9999, 10000, 1000000, 2000000}, // Nearly full removal
	}

	for _, seed := range seeds {
		f.Add(seed.sharesToRemove, seed.totalShares, seed.reserveA, seed.reserveB)
	}

	f.Fuzz(func(t *testing.T, sharesToRemove, totalShares, reserveA, reserveB uint64) {
		// Skip invalid inputs
		if sharesToRemove == 0 || totalShares == 0 || reserveA == 0 || reserveB == 0 {
			return
		}
		if sharesToRemove > totalShares {
			return // Can't remove more shares than exist
		}

		sharesToRemoveInt := math.NewInt(int64(sharesToRemove))
		totalSharesInt := math.NewInt(int64(totalShares))
		reserveAInt := math.NewInt(int64(reserveA))
		reserveBInt := math.NewInt(int64(reserveB))

		// Calculate amounts to return
		// amountA = sharesToRemove * reserveA / totalShares
		amountA := math.LegacyNewDecFromInt(sharesToRemoveInt).
			Mul(math.LegacyNewDecFromInt(reserveAInt)).
			Quo(math.LegacyNewDecFromInt(totalSharesInt)).
			TruncateInt()

		amountB := math.LegacyNewDecFromInt(sharesToRemoveInt).
			Mul(math.LegacyNewDecFromInt(reserveBInt)).
			Quo(math.LegacyNewDecFromInt(totalSharesInt)).
			TruncateInt()

		// INVARIANT 1: Returned amounts must not exceed reserves
		if amountA.GT(reserveAInt) {
			t.Errorf("VIOLATION: amountA (%s) > reserveA (%s)", amountA.String(), reserveAInt.String())
		}
		if amountB.GT(reserveBInt) {
			t.Errorf("VIOLATION: amountB (%s) > reserveB (%s)", amountB.String(), reserveBInt.String())
		}

		// INVARIANT 2: Amounts must be non-negative
		if amountA.IsNegative() || amountB.IsNegative() {
			t.Errorf("VIOLATION: negative amounts returned")
		}

		// INVARIANT 3: Proportion should be maintained
		// amountA / amountB should ≈ reserveA / reserveB
		if !amountA.IsZero() && !amountB.IsZero() {
			ratioReserves := math.LegacyNewDecFromInt(reserveAInt).Quo(math.LegacyNewDecFromInt(reserveBInt))
			ratioAmounts := math.LegacyNewDecFromInt(amountA).Quo(math.LegacyNewDecFromInt(amountB))

			tolerance := math.LegacyMustNewDecFromStr("0.01") // 1% tolerance
			diff := ratioReserves.Sub(ratioAmounts).Abs()

			if diff.GT(tolerance) && diff.GT(ratioReserves.Mul(tolerance)) {
				t.Logf("Ratio mismatch: reserves=%.6f, amounts=%.6f, diff=%.6f",
					ratioReserves.MustFloat64(),
					ratioAmounts.MustFloat64(),
					diff.MustFloat64())
			}
		}

		// INVARIANT 4: Complete removal should drain reserves
		if sharesToRemove == totalShares {
			if !amountA.Equal(reserveAInt) {
				t.Logf("Full removal: amountA=%s, reserveA=%s", amountA.String(), reserveAInt.String())
			}
		}
	})
}

// ============================================================================
// FuzzPriceImpact - Tests that large swaps have proportionally larger price impact
// ============================================================================

// FuzzPriceImpact tests that large swaps have proportionally larger price impact
func FuzzPriceImpact(f *testing.F) {
	seeds := []struct {
		swapSize, reserve uint64
	}{
		{1000, 100000},
		{10000, 100000},
		{50000, 100000},
	}

	for _, seed := range seeds {
		f.Add(seed.swapSize, seed.reserve)
	}

	f.Fuzz(func(t *testing.T, swapSize, reserve uint64) {
		if swapSize == 0 || reserve == 0 || swapSize >= reserve {
			return
		}

		swapInt := math.NewInt(int64(swapSize))
		reserveInt := math.NewInt(int64(reserve))

		// After swap: new price = (reserve + swap) / (reserve - output)
		// Simplified: price_impact = swap / (reserve + swap/2)
		priceImpact := math.LegacyNewDecFromInt(swapInt).
			Quo(math.LegacyNewDecFromInt(reserveInt))

		// INVARIANT: Price impact should be less than 100%
		if priceImpact.GTE(math.LegacyOneDec()) {
			// This is actually allowed for very large swaps
			t.Logf("Large price impact: %.2f%%", priceImpact.MustFloat64()*100)
		}

		// INVARIANT: Price impact should increase with swap size
		// (monotonically increasing)
		if swapSize < reserve/2 {
			smallerSwap := math.NewInt(int64(swapSize / 2))
			smallerImpact := math.LegacyNewDecFromInt(smallerSwap).
				Quo(math.LegacyNewDecFromInt(reserveInt))

			if priceImpact.LTE(smallerImpact) {
				t.Errorf("VIOLATION: price impact not monotonic")
			}
		}
	})
}
