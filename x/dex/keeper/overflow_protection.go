package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/paw-chain/paw/x/dex/types"
)

// SafeCalculateSwapOutput performs overflow-protected swap output calculation
// This wraps the constant product AMM formula with explicit overflow checks.
//
// Formula: amountOut = (amountIn * (1 - fee) * reserveOut) / (reserveIn + amountIn * (1 - fee))
//
// SECURITY: All intermediate calculations use SafeMul, SafeAdd, SafeQuo to detect overflow.
// This prevents integer overflow attacks that could drain pool liquidity.
func (k Keeper) SafeCalculateSwapOutput(ctx context.Context, amountIn, reserveIn, reserveOut math.Int, swapFee math.LegacyDec) (math.Int, error) {
	// Validate inputs
	if amountIn.IsZero() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("input amount must be positive")
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("pool reserves must be positive")
	}

	// Calculate amount after fee: amountIn * (1 - fee)
	oneMinusFee := math.LegacyOneDec().Sub(swapFee)
	amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(oneMinusFee)

	// Convert back to Int for safe arithmetic
	amountInAfterFeeInt := amountInAfterFee.TruncateInt()

	// OVERFLOW CHECK 1: numerator = amountInAfterFee * reserveOut
	// This is the most likely overflow point with large reserves
	numerator, err := amountInAfterFeeInt.SafeMul(reserveOut)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in numerator calculation: amountInAfterFee=%s * reserveOut=%s: %v",
			amountInAfterFeeInt.String(), reserveOut.String(), err)
	}

	// OVERFLOW CHECK 2: denominator = reserveIn + amountInAfterFee
	denominator, err := reserveIn.SafeAdd(amountInAfterFeeInt)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in denominator calculation: reserveIn=%s + amountInAfterFee=%s: %v",
			reserveIn.String(), amountInAfterFeeInt.String(), err)
	}

	// OVERFLOW CHECK 3: division (less likely to overflow, but we use SafeQuo for consistency)
	amountOut, err := numerator.SafeQuo(denominator)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in division: numerator=%s / denominator=%s: %v",
			numerator.String(), denominator.String(), err)
	}

	if amountOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("output amount too small")
	}

	if amountOut.GTE(reserveOut) {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("output %s >= reserve %s", amountOut, reserveOut)
	}

	return amountOut, nil
}

// SafeCalculatePoolShares calculates initial pool shares with overflow protection
// Uses geometric mean: sqrt(amountA * amountB)
func (k Keeper) SafeCalculatePoolShares(amountA, amountB math.Int) (math.Int, error) {
	// OVERFLOW CHECK: amountA * amountB
	product, err := amountA.SafeMul(amountB)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow calculating pool shares: amountA=%s * amountB=%s: %v",
			amountA.String(), amountB.String(), err)
	}

	// Square root calculation (uses LegacyDec which has high precision)
	sqrtShares, _ := math.LegacyNewDecFromInt(product).ApproxSqrt()
	shares := sqrtShares.TruncateInt()

	return shares, nil
}

// SafeCalculateAddLiquidityShares calculates shares to mint when adding liquidity
// Formula: min((amountA * totalShares / reserveA), (amountB * totalShares / reserveB))
func (k Keeper) SafeCalculateAddLiquidityShares(amountA, amountB, reserveA, reserveB, totalShares math.Int) (math.Int, error) {
	// Validate inputs
	if reserveA.IsZero() || reserveB.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("pool reserves must be positive")
	}

	if totalShares.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("total shares must be positive")
	}

	// OVERFLOW CHECK 1: amountA * totalShares
	numeratorA, err := amountA.SafeMul(totalShares)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in sharesA calculation: amountA=%s * totalShares=%s: %v",
			amountA.String(), totalShares.String(), err)
	}

	// OVERFLOW CHECK 2: sharesA = numeratorA / reserveA
	sharesA, err := numeratorA.SafeQuo(reserveA)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in sharesA division: numeratorA=%s / reserveA=%s: %v",
			numeratorA.String(), reserveA.String(), err)
	}

	// OVERFLOW CHECK 3: amountB * totalShares
	numeratorB, err := amountB.SafeMul(totalShares)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in sharesB calculation: amountB=%s * totalShares=%s: %v",
			amountB.String(), totalShares.String(), err)
	}

	// OVERFLOW CHECK 4: sharesB = numeratorB / reserveB
	sharesB, err := numeratorB.SafeQuo(reserveB)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in sharesB division: numeratorB=%s / reserveB=%s: %v",
			numeratorB.String(), reserveB.String(), err)
	}

	// Return minimum of the two shares calculations
	shares := math.MinInt(sharesA, sharesB)

	return shares, nil
}

// SafeCalculateRemoveLiquidityAmounts calculates token amounts when removing liquidity
// Formula: amountA = (shares * reserveA / totalShares), amountB = (shares * reserveB / totalShares)
func (k Keeper) SafeCalculateRemoveLiquidityAmounts(shares, reserveA, reserveB, totalShares math.Int) (math.Int, math.Int, error) {
	// Validate inputs
	if totalShares.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("total shares must be positive")
	}

	if shares.GT(totalShares) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("shares exceed total shares")
	}

	// OVERFLOW CHECK 1: shares * reserveA
	numeratorA, err := shares.SafeMul(reserveA)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in amountA calculation: shares=%s * reserveA=%s: %v",
			shares.String(), reserveA.String(), err)
	}

	// OVERFLOW CHECK 2: amountA = numeratorA / totalShares
	amountA, err := numeratorA.SafeQuo(totalShares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in amountA division: numeratorA=%s / totalShares=%s: %v",
			numeratorA.String(), totalShares.String(), err)
	}

	// OVERFLOW CHECK 3: shares * reserveB
	numeratorB, err := shares.SafeMul(reserveB)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in amountB calculation: shares=%s * reserveB=%s: %v",
			shares.String(), reserveB.String(), err)
	}

	// OVERFLOW CHECK 4: amountB = numeratorB / totalShares
	amountB, err := numeratorB.SafeQuo(totalShares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in amountB division: numeratorB=%s / totalShares=%s: %v",
			numeratorB.String(), totalShares.String(), err)
	}

	return amountA, amountB, nil
}

// SafeUpdateReserves updates pool reserves with overflow protection
// Returns new reserves after applying deltas
func (k Keeper) SafeUpdateReserves(reserveA, reserveB, deltaA, deltaB math.Int) (math.Int, math.Int, error) {
	var newReserveA, newReserveB math.Int
	var err error

	// Update reserve A
	if deltaA.IsPositive() {
		newReserveA, err = reserveA.SafeAdd(deltaA)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding to reserveA: %s + %s: %v",
				reserveA.String(), deltaA.String(), err)
		}
	} else if deltaA.IsNegative() {
		newReserveA, err = reserveA.SafeSub(deltaA.Neg())
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting from reserveA: %s - %s: %v",
				reserveA.String(), deltaA.Neg().String(), err)
		}
	} else {
		newReserveA = reserveA
	}

	// Update reserve B
	if deltaB.IsPositive() {
		newReserveB, err = reserveB.SafeAdd(deltaB)
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding to reserveB: %s + %s: %v",
				reserveB.String(), deltaB.String(), err)
		}
	} else if deltaB.IsNegative() {
		newReserveB, err = reserveB.SafeSub(deltaB.Neg())
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting from reserveB: %s - %s: %v",
				reserveB.String(), deltaB.Neg().String(), err)
		}
	} else {
		newReserveB = reserveB
	}

	// Ensure reserves remain positive
	if newReserveA.IsNegative() || newReserveB.IsNegative() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("negative reserves after update: A=%s, B=%s",
			newReserveA.String(), newReserveB.String())
	}

	if newReserveA.IsZero() || newReserveB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("zero reserves after update")
	}

	return newReserveA, newReserveB, nil
}

// SafeValidateConstantProduct validates the constant product invariant with overflow protection
// Returns error if new k-value is less than old k-value (precision loss or calculation error)
func (k Keeper) SafeValidateConstantProduct(oldReserveA, oldReserveB, newReserveA, newReserveB math.Int) error {
	// OVERFLOW CHECK 1: old k-value
	oldK, err := oldReserveA.SafeMul(oldReserveB)
	if err != nil {
		return types.ErrOverflow.Wrapf("overflow calculating old k-value: %s * %s: %v",
			oldReserveA.String(), oldReserveB.String(), err)
	}

	// OVERFLOW CHECK 2: new k-value
	newK, err := newReserveA.SafeMul(newReserveB)
	if err != nil {
		return types.ErrOverflow.Wrapf("overflow calculating new k-value: %s * %s: %v",
			newReserveA.String(), newReserveB.String(), err)
	}

	// Validate invariant: new k >= old k (with rounding, should never decrease)
	if newK.LT(oldK) {
		return types.ErrInvariantViolation.Wrapf(
			"constant product invariant violated: old_k=%s, new_k=%s",
			oldK.String(), newK.String(),
		)
	}

	return nil
}
