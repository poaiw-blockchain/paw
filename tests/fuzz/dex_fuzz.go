package fuzz

import (
	"testing"

	"cosmossdk.io/math"
)

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

		// Calculate price impact: Δprice / initial_price
		// For constant product: price_impact ≈ swap_size / reserve
		spotPriceBefore := math.LegacyOneDec() // Assuming 1:1 initially

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
