package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
)

// SafeMath provides overflow-safe arithmetic operations for DEX module

// SafeAdd adds two math.Int values with overflow checking
func SafeAdd(a, b math.Int) (math.Int, error) {
	// Convert to big.Int for overflow-safe addition
	result := new(big.Int).Add(a.BigInt(), b.BigInt())

	// Check if result is within valid range (< 2^256)
	maxInt := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	if result.Cmp(maxInt) >= 0 {
		return math.Int{}, fmt.Errorf("overflow: addition result exceeds maximum value")
	}

	return math.NewIntFromBigInt(result), nil
}

// SafeSub subtracts two math.Int values with underflow checking
func SafeSub(a, b math.Int) (math.Int, error) {
	// Check for underflow
	if a.LT(b) {
		return math.Int{}, fmt.Errorf("underflow: cannot subtract %s from %s", b.String(), a.String())
	}

	result := new(big.Int).Sub(a.BigInt(), b.BigInt())
	return math.NewIntFromBigInt(result), nil
}

// SafeMul multiplies two math.Int values with overflow checking
func SafeMul(a, b math.Int) (math.Int, error) {
	// Handle zero cases early
	if a.IsZero() || b.IsZero() {
		return math.ZeroInt(), nil
	}

	result := new(big.Int).Mul(a.BigInt(), b.BigInt())

	// Check if result is within valid range
	maxInt := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	if result.Cmp(maxInt) >= 0 {
		return math.Int{}, fmt.Errorf("overflow: multiplication result exceeds maximum value")
	}

	return math.NewIntFromBigInt(result), nil
}

// SafeQuo divides two math.Int values with division by zero checking
func SafeQuo(a, b math.Int) (math.Int, error) {
	if b.IsZero() {
		return math.Int{}, fmt.Errorf("division by zero")
	}

	result := new(big.Int).Quo(a.BigInt(), b.BigInt())
	return math.NewIntFromBigInt(result), nil
}

// SafeDiv is an alias for SafeQuo for consistency
func SafeDiv(a, b math.Int) (math.Int, error) {
	return SafeQuo(a, b)
}

// SafeMulDiv performs (a * b) / c with overflow protection
// This is commonly used in DEX calculations
func SafeMulDiv(a, b, c math.Int) (math.Int, error) {
	if c.IsZero() {
		return math.Int{}, fmt.Errorf("division by zero")
	}

	// Multiply first
	intermediate := new(big.Int).Mul(a.BigInt(), b.BigInt())

	// Check intermediate result
	maxInt := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	if intermediate.Cmp(maxInt) >= 0 {
		return math.Int{}, fmt.Errorf("overflow in multiplication step")
	}

	// Then divide
	result := new(big.Int).Quo(intermediate, c.BigInt())
	return math.NewIntFromBigInt(result), nil
}

// SafeAddUint64 adds two uint64 values with overflow checking
func SafeAddUint64(a, b uint64) (uint64, error) {
	if a > (1<<64 - 1 - b) {
		return 0, fmt.Errorf("overflow: uint64 addition overflow")
	}
	return a + b, nil
}

// SafeMulUint64 multiplies two uint64 values with overflow checking
func SafeMulUint64(a, b uint64) (uint64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}

	result := a * b
	if result/a != b {
		return 0, fmt.Errorf("overflow: uint64 multiplication overflow")
	}
	return result, nil
}

// SafeRatio calculates (numerator * value) / denominator with overflow protection
func SafeRatio(value, numerator, denominator math.Int) (math.Int, error) {
	if denominator.IsZero() {
		return math.Int{}, fmt.Errorf("denominator cannot be zero")
	}

	return SafeMulDiv(value, numerator, denominator)
}
