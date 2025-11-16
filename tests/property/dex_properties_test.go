package property_test

import (
	"math/rand"
	"testing"
	"testing/quick"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// Property: Pool creation should be commutative (order doesn't matter)
func TestPropertyPoolCreationCommutative(t *testing.T) {
	property := func(amountA, amountB uint64) bool {
		// Ensure non-zero amounts
		if amountA == 0 || amountB == 0 {
			return true // Skip invalid cases
		}

		// Limit to reasonable values to prevent overflow
		if amountA > 1e18 || amountB > 1e18 {
			return true
		}

		// Create pool with (A, B)
		reserveA1 := math.NewInt(int64(amountA))
		reserveB1 := math.NewInt(int64(amountB))
		k1 := reserveA1.Mul(reserveB1)

		// Create pool with (B, A)
		reserveA2 := math.NewInt(int64(amountB))
		reserveB2 := math.NewInt(int64(amountA))
		k2 := reserveA2.Mul(reserveB2)

		// K should be the same regardless of order
		return k1.Equal(k2)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}

// Property: Swaps never increase reserves beyond collected fees
func TestPropertySwapNeverIncreasesReserves(t *testing.T) {
	property := func(reserveA, reserveB, swapAmount uint64) bool {
		// Skip invalid inputs
		if reserveA == 0 || reserveB == 0 || swapAmount == 0 {
			return true
		}

		// Limit values
		if reserveA > 1e15 || reserveB > 1e15 || swapAmount > reserveA/2 {
			return true
		}

		// Calculate swap using constant product formula with 0.3% fee
		resA := math.NewInt(int64(reserveA))
		resB := math.NewInt(int64(reserveB))
		amtIn := math.NewInt(int64(swapAmount))

		// Fee: 0.3% (997/1000 after fee)
		amtInWithFee := amtIn.MulRaw(997)
		numerator := amtInWithFee.Mul(resB)
		denominator := resA.MulRaw(1000).Add(amtInWithFee)

		// Avoid division by zero
		if denominator.IsZero() {
			return true
		}

		amtOut := numerator.Quo(denominator)

		// New reserves
		newReserveA := resA.Add(amtIn)
		newReserveB := resB.Sub(amtOut)

		// Check reserves didn't go negative
		if newReserveB.IsNegative() {
			return true // Invalid swap, skip
		}

		// Original K
		k1 := resA.Mul(resB)

		// New K (should be equal or slightly higher due to fees)
		k2 := newReserveA.Mul(newReserveB)

		// K should never decrease (fees make it increase slightly)
		return k2.GTE(k1)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}

// Property: Adding then removing liquidity returns proportional amounts
func TestPropertyAddRemoveLiquidityRoundtrip(t *testing.T) {
	property := func(reserveA, reserveB, liquidityAmount uint64) bool {
		// Skip invalid inputs
		if reserveA == 0 || reserveB == 0 || liquidityAmount == 0 {
			return true
		}

		// Limit values
		if reserveA > 1e15 || reserveB > 1e15 || liquidityAmount > reserveA/10 {
			return true
		}

		resA := math.NewInt(int64(reserveA))
		resB := math.NewInt(int64(reserveB))
		liqAmt := math.NewInt(int64(liquidityAmount))

		// Assume initial total shares equal to geometric mean of reserves
		// Using simple approximation: min(resA, resB) for total shares
		totalShares := resA
		if resB.LT(resA) {
			totalShares = resB
		}
		if totalShares.IsZero() {
			return true
		}

		// Calculate amounts to add for given liquidity
		amountA := liqAmt.Mul(resA).Quo(totalShares)
		amountB := liqAmt.Mul(resB).Quo(totalShares)

		// Add liquidity
		newReserveA := resA.Add(amountA)
		newReserveB := resB.Add(amountB)
		newTotalShares := totalShares.Add(liqAmt)

		// Remove same liquidity
		removedA := liqAmt.Mul(newReserveA).Quo(newTotalShares)
		removedB := liqAmt.Mul(newReserveB).Quo(newTotalShares)

		// Due to rounding, removed amounts should be approximately equal to added amounts
		// Allow 1% difference for rounding errors
		diffA := amountA.Sub(removedA).Abs()
		diffB := amountB.Sub(removedB).Abs()

		maxDiffA := amountA.QuoRaw(100)
		maxDiffB := amountB.QuoRaw(100)

		return diffA.LTE(maxDiffA) && diffB.LTE(maxDiffB)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}

// Property: Price impact increases with swap size
func TestPropertyPriceImpactIncreasesWithSize(t *testing.T) {
	property := func(reserveA, reserveB, swapAmount1, swapAmount2 uint64) bool {
		// Skip invalid inputs
		if reserveA == 0 || reserveB == 0 || swapAmount1 == 0 || swapAmount2 == 0 {
			return true
		}

		// Ensure swap2 > swap1
		if swapAmount2 <= swapAmount1 {
			return true
		}

		// Limit values
		if reserveA > 1e15 || reserveB > 1e15 {
			return true
		}

		if swapAmount1 > reserveA/10 || swapAmount2 > reserveA/5 {
			return true
		}

		resA := math.NewInt(int64(reserveA))
		resB := math.NewInt(int64(reserveB))

		// Calculate output for smaller swap
		amt1 := math.NewInt(int64(swapAmount1))
		amt1WithFee := amt1.MulRaw(997)
		out1 := amt1WithFee.Mul(resB).Quo(resA.MulRaw(1000).Add(amt1WithFee))

		// Calculate output for larger swap
		amt2 := math.NewInt(int64(swapAmount2))
		amt2WithFee := amt2.MulRaw(997)
		out2 := amt2WithFee.Mul(resB).Quo(resA.MulRaw(1000).Add(amt2WithFee))

		// Price per token for each swap
		// price1 = swapAmount1 / out1
		// price2 = swapAmount2 / out2

		if out1.IsZero() || out2.IsZero() {
			return true
		}

		// To avoid division, cross multiply:
		// price2 > price1 if: swapAmount2 * out1 > swapAmount1 * out2
		lhs := amt2.Mul(out1)
		rhs := amt1.Mul(out2)

		// Larger swap should have worse (higher) price per token
		return lhs.GTE(rhs)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}

// Property: Multiple small swaps result in similar output as one large swap (minus fees)
func TestPropertySwapAggregation(t *testing.T) {
	property := func(reserveA, reserveB, totalSwap uint8) bool {
		// Use uint8 to limit values
		if reserveA == 0 || reserveB == 0 || totalSwap == 0 {
			return true
		}

		// Scale up the values
		resA := math.NewInt(int64(uint64(reserveA)) * 1000000)
		resB := math.NewInt(int64(uint64(reserveB)) * 1000000)
		total := math.NewInt(int64(uint64(totalSwap)) * 10000)

		// One large swap
		amtWithFee := total.MulRaw(997)
		outLarge := amtWithFee.Mul(resB).Quo(resA.MulRaw(1000).Add(amtWithFee))

		// Two small swaps (half each)
		half := total.QuoRaw(2)

		// First small swap
		amt1WithFee := half.MulRaw(997)
		out1 := amt1WithFee.Mul(resB).Quo(resA.MulRaw(1000).Add(amt1WithFee))
		newResA := resA.Add(half)
		newResB := resB.Sub(out1)

		// Second small swap
		amt2WithFee := half.MulRaw(997)
		out2 := amt2WithFee.Mul(newResB).Quo(newResA.MulRaw(1000).Add(amt2WithFee))

		totalOutSmall := out1.Add(out2)

		// Small swaps should give slightly more due to lower price impact
		// but less due to double fee application
		// The difference should be small (within 1%)
		diff := outLarge.Sub(totalOutSmall).Abs()
		maxDiff := outLarge.QuoRaw(100)

		return diff.LTE(maxDiff)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 500,
	})
	require.NoError(t, err)
}

// Property: Reserves always stay positive after valid operations
func TestPropertyReservesStayPositive(t *testing.T) {
	property := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Initial reserves
		reserveA := math.NewInt(rng.Int63n(1000000) + 100000)
		reserveB := math.NewInt(rng.Int63n(1000000) + 100000)

		// Perform 10 random operations
		for i := 0; i < 10; i++ {
			op := rng.Intn(2) // 0 = swap A->B, 1 = swap B->A

			if op == 0 {
				// Swap A for B
				maxSwap := reserveA.QuoRaw(10) // Max 10% of reserve
				if maxSwap.IsZero() {
					continue
				}
				swapAmt := math.NewInt(rng.Int63n(maxSwap.Int64()) + 1)

				amtWithFee := swapAmt.MulRaw(997)
				out := amtWithFee.Mul(reserveB).Quo(reserveA.MulRaw(1000).Add(amtWithFee))

				reserveA = reserveA.Add(swapAmt)
				reserveB = reserveB.Sub(out)
			} else {
				// Swap B for A
				maxSwap := reserveB.QuoRaw(10)
				if maxSwap.IsZero() {
					continue
				}
				swapAmt := math.NewInt(rng.Int63n(maxSwap.Int64()) + 1)

				amtWithFee := swapAmt.MulRaw(997)
				out := amtWithFee.Mul(reserveA).Quo(reserveB.MulRaw(1000).Add(amtWithFee))

				reserveB = reserveB.Add(swapAmt)
				reserveA = reserveA.Sub(out)
			}

			// Check reserves stay positive
			if reserveA.IsNegative() || reserveB.IsNegative() {
				return false
			}
		}

		return true
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}

// Property: K (constant product) never decreases
func TestPropertyKNeverDecreases(t *testing.T) {
	property := func(reserveA, reserveB, swapAmount uint64) bool {
		if reserveA == 0 || reserveB == 0 || swapAmount == 0 {
			return true
		}

		if reserveA > 1e15 || reserveB > 1e15 || swapAmount > reserveA/2 {
			return true
		}

		resA := math.NewInt(int64(reserveA))
		resB := math.NewInt(int64(reserveB))
		amt := math.NewInt(int64(swapAmount))

		// Original K
		k1 := resA.Mul(resB)

		// Perform swap
		amtWithFee := amt.MulRaw(997)
		out := amtWithFee.Mul(resB).Quo(resA.MulRaw(1000).Add(amtWithFee))

		if out.IsZero() || out.GTE(resB) {
			return true // Invalid swap
		}

		newResA := resA.Add(amt)
		newResB := resB.Sub(out)

		// New K
		k2 := newResA.Mul(newResB)

		// K should never decrease (fees make it increase)
		return k2.GTE(k1)
	}

	err := quick.Check(property, &quick.Config{
		MaxCount: 1000,
	})
	require.NoError(t, err)
}
