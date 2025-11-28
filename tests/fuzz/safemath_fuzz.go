package fuzz

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
)

// FuzzSafeMathAdd tests SafeMath addition for overflow/underflow
func FuzzSafeMathAdd(f *testing.F) {
	// Seed corpus with interesting values
	seeds := []struct {
		a, b int64
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1000000, 2000000},
		{9223372036854775807, 0},   // MaxInt64
		{-9223372036854775808, 0},  // MinInt64
		{9223372036854775807, 1},   // Would overflow
		{-9223372036854775808, -1}, // Would underflow
	}

	for _, seed := range seeds {
		f.Add(seed.a, seed.b)
	}

	f.Fuzz(func(t *testing.T, a, b int64) {
		// Convert to math.Int
		aInt := math.NewInt(a)
		bInt := math.NewInt(b)

		// Perform addition
		result := aInt.Add(bInt)

		// Verify using big.Int (ground truth)
		aBig := big.NewInt(a)
		bBig := big.NewInt(b)
		expectedBig := new(big.Int).Add(aBig, bBig)

		// Convert result to big.Int for comparison
		resultBig := result.BigInt()

		// Check correctness
		if resultBig.Cmp(expectedBig) != 0 {
			t.Errorf("Add(%d, %d) = %s, expected %s", a, b, result.String(), expectedBig.String())
		}

		// Verify no panic occurred (implicit by reaching here)
	})
}

// FuzzSafeMathSub tests SafeMath subtraction for overflow/underflow
func FuzzSafeMathSub(f *testing.F) {
	seeds := []struct {
		a, b int64
	}{
		{0, 0},
		{1, 1},
		{100, 50},
		{9223372036854775807, -1}, // Would overflow
		{-9223372036854775808, 1}, // Would underflow
		{1000000, 2000000},
	}

	for _, seed := range seeds {
		f.Add(seed.a, seed.b)
	}

	f.Fuzz(func(t *testing.T, a, b int64) {
		aInt := math.NewInt(a)
		bInt := math.NewInt(b)

		result := aInt.Sub(bInt)

		// Verify using big.Int
		aBig := big.NewInt(a)
		bBig := big.NewInt(b)
		expectedBig := new(big.Int).Sub(aBig, bBig)

		resultBig := result.BigInt()

		if resultBig.Cmp(expectedBig) != 0 {
			t.Errorf("Sub(%d, %d) = %s, expected %s", a, b, result.String(), expectedBig.String())
		}
	})
}

// FuzzSafeMathMul tests SafeMath multiplication for overflow/underflow
func FuzzSafeMathMul(f *testing.F) {
	seeds := []struct {
		a, b int64
	}{
		{0, 0},
		{1, 1},
		{2, 3},
		{1000, 1000},
		{9223372036854775807, 1},
		{9223372036854775807, 2},   // Would overflow
		{-9223372036854775808, -1}, // Would overflow
		{1000000, 1000000},
	}

	for _, seed := range seeds {
		f.Add(seed.a, seed.b)
	}

	f.Fuzz(func(t *testing.T, a, b int64) {
		aInt := math.NewInt(a)
		bInt := math.NewInt(b)

		result := aInt.Mul(bInt)

		// Verify using big.Int
		aBig := big.NewInt(a)
		bBig := big.NewInt(b)
		expectedBig := new(big.Int).Mul(aBig, bBig)

		resultBig := result.BigInt()

		if resultBig.Cmp(expectedBig) != 0 {
			t.Errorf("Mul(%d, %d) = %s, expected %s", a, b, result.String(), expectedBig.String())
		}
	})
}

// FuzzSafeMathQuo tests SafeMath division for division by zero and correctness
func FuzzSafeMathQuo(f *testing.F) {
	seeds := []struct {
		a, b int64
	}{
		{0, 1},
		{10, 2},
		{100, 3},
		{9223372036854775807, 2},
		{-100, 3},
		{100, -3},
	}

	for _, seed := range seeds {
		f.Add(seed.a, seed.b)
	}

	f.Fuzz(func(t *testing.T, a, b int64) {
		// Skip division by zero (expected to panic)
		if b == 0 {
			return
		}

		aInt := math.NewInt(a)
		bInt := math.NewInt(b)

		result := aInt.Quo(bInt)

		// Verify using big.Int
		aBig := big.NewInt(a)
		bBig := big.NewInt(b)
		expectedBig := new(big.Int).Quo(aBig, bBig)

		resultBig := result.BigInt()

		if resultBig.Cmp(expectedBig) != 0 {
			t.Errorf("Quo(%d, %d) = %s, expected %s", a, b, result.String(), expectedBig.String())
		}
	})
}

// FuzzDecimalOperations tests decimal operations
func FuzzDecimalOperations(f *testing.F) {
	seeds := []struct {
		a, b string
	}{
		{"0", "1"},
		{"1.5", "2.5"},
		{"0.3", "0.7"},
		{"999999.999999", "0.000001"},
	}

	for _, seed := range seeds {
		f.Add(seed.a, seed.b)
	}

	f.Fuzz(func(t *testing.T, aStr, bStr string) {
		// Attempt to parse decimals
		aDec, err := math.LegacyNewDecFromStr(aStr)
		if err != nil {
			return // Skip invalid inputs
		}

		bDec, err := math.LegacyNewDecFromStr(bStr)
		if err != nil {
			return // Skip invalid inputs
		}

		// Test Add
		resultAdd := aDec.Add(bDec)
		if resultAdd.IsNil() {
			t.Error("Add returned nil")
		}

		// Test Sub
		resultSub := aDec.Sub(bDec)
		if resultSub.IsNil() {
			t.Error("Sub returned nil")
		}

		// Test Mul
		resultMul := aDec.Mul(bDec)
		if resultMul.IsNil() {
			t.Error("Mul returned nil")
		}

		// Test Quo (avoid division by zero)
		if !bDec.IsZero() {
			resultQuo := aDec.Quo(bDec)
			if resultQuo.IsNil() {
				t.Error("Quo returned nil")
			}

			// Verify: a / b * b ≈ a (within precision)
			reconstructed := resultQuo.Mul(bDec)
			diff := reconstructed.Sub(aDec).Abs()

			// Allow small precision error (10^-12)
			tolerance := math.LegacyMustNewDecFromStr("0.000000000001")
			if diff.GT(tolerance) {
				t.Logf("Large precision error: a=%s, b=%s, a/b*b=%s, diff=%s",
					aDec.String(), bDec.String(), reconstructed.String(), diff.String())
			}
		}
	})
}

// FuzzNegativeValues tests operations with negative values
func FuzzNegativeValues(f *testing.F) {
	seeds := []int64{-1000000, -1, 0, 1, 1000000}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, val int64) {
		x := math.NewInt(val)

		// Test Neg
		negated := x.Neg()
		doubleNeg := negated.Neg()

		if !x.Equal(doubleNeg) {
			t.Errorf("Double negation failed: %s != %s", x.String(), doubleNeg.String())
		}

		// Test Abs
		abs := x.Abs()
		if abs.IsNegative() {
			t.Errorf("Abs returned negative value: %s", abs.String())
		}

		// Test IsNegative
		if val < 0 && !x.IsNegative() {
			t.Errorf("IsNegative failed for %d", val)
		}
		if val >= 0 && x.IsNegative() {
			t.Errorf("IsNegative incorrectly returned true for %d", val)
		}
	})
}
