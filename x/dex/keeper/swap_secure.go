package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// ExecuteSwapSecure performs a token swap with comprehensive security checks
// This is the production-grade version with all security features enabled
func (k Keeper) ExecuteSwapSecure(ctx context.Context, trader sdk.AccAddress, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) (math.Int, error) {
	// Execute with reentrancy protection
	var amountOut math.Int
	err := k.WithReentrancyGuard(ctx, poolID, "swap", func() error {
		var execErr error
		amountOut, execErr = k.executeSwapInternal(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
		return execErr
	})

	return amountOut, err
}

// executeSwapInternal is the internal swap implementation with all security checks
func (k Keeper) executeSwapInternal(ctx context.Context, trader sdk.AccAddress, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Gas metering: Base swap operation cost
	sdkCtx.GasMeter().ConsumeGas(50000, "dex_swap_base")

	// 1. Input Validation
	if amountIn.IsZero() || amountIn.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("swap amount must be positive")
	}
	sdkCtx.GasMeter().ConsumeGas(1000, "dex_swap_validation")

	if tokenIn == tokenOut {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrap("cannot swap identical tokens")
	}

	if minAmountOut.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("min amount out cannot be negative")
	}

	// 2. Get pool and validate state
	sdkCtx.GasMeter().ConsumeGas(5000, "dex_swap_pool_lookup")
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	// Validate pool state before operation
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// 3. Check circuit breaker
	if err := k.CheckCircuitBreaker(ctx, pool, "swap"); err != nil {
		return math.ZeroInt(), err
	}

	// 4. Determine which token is being swapped
	var reserveIn, reserveOut math.Int
	var isTokenAIn bool

	if tokenIn == pool.TokenA && tokenOut == pool.TokenB {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
		isTokenAIn = true
	} else if tokenIn == pool.TokenB && tokenOut == pool.TokenA {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
		isTokenAIn = false
	} else {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrapf(
			"invalid token pair for pool %d: expected %s/%s, got %s/%s",
			poolID, pool.TokenA, pool.TokenB, tokenIn, tokenOut,
		)
	}

	// 5. Validate swap size (MEV protection)
	if err := k.ValidateSwapSize(amountIn, reserveIn); err != nil {
		return math.ZeroInt(), err
	}

	// 6. Get swap parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	// 7. Calculate swap output using constant product formula with fees
	sdkCtx.GasMeter().ConsumeGas(10000, "dex_swap_calculation")
	amountOut, err := k.CalculateSwapOutputSecure(ctx, amountIn, reserveIn, reserveOut, params.SwapFee, params.MaxPoolDrainPercent)
	if err != nil {
		return math.ZeroInt(), err
	}

	// 8. Validate price impact
	if err := k.ValidatePriceImpact(amountIn, reserveIn, reserveOut, amountOut); err != nil {
		return math.ZeroInt(), err
	}

	// 9. Check slippage protection
	if amountOut.LT(minAmountOut) {
		return math.ZeroInt(), types.ErrSlippageTooHigh.Wrapf(
			"slippage too high: expected at least %s, got %s",
			minAmountOut, amountOut,
		)
	}

	// 10. Calculate fees
	feeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()
	amountInAfterFee := amountIn.Sub(feeAmount)
	if amountInAfterFee.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("fee amount exceeds swap amount")
	}
	if amountInAfterFee.IsZero() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("swap amount too small after fees")
	}

	// 11. Store old k for invariant check
	oldK := pool.ReserveA.Mul(pool.ReserveB)

	// 12. Transfer input tokens FIRST (checks-effects-interactions pattern)
	sdkCtx.GasMeter().ConsumeGas(15000, "dex_swap_token_transfer")
	moduleAddr := k.GetModuleAddress()

	coinIn := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer input tokens: %v", err)
	}

	// 12.5 Collect and transfer fees now that module holds tokens
	lpFee, protocolFee, err := k.CollectSwapFees(ctx, poolID, tokenIn, amountIn)
	if err != nil {
		return math.ZeroInt(), err
	}
	feeAmount, err = SafeAdd(lpFee, protocolFee)
	if err != nil {
		return math.ZeroInt(), err
	}
	// 13. Update pool reserves AFTER receiving tokens
	if isTokenAIn {
		pool.ReserveA, err = SafeAdd(pool.ReserveA, amountInAfterFee)
		if err != nil {
			return math.ZeroInt(), err
		}
		pool.ReserveB, err = SafeSub(pool.ReserveB, amountOut)
		if err != nil {
			return math.ZeroInt(), err
		}
	} else {
		pool.ReserveB, err = SafeAdd(pool.ReserveB, amountInAfterFee)
		if err != nil {
			return math.ZeroInt(), err
		}
		pool.ReserveA, err = SafeSub(pool.ReserveA, amountOut)
		if err != nil {
			return math.ZeroInt(), err
		}
	}

	// 14. Validate invariant (k should increase or stay same due to fees)
	if err := k.ValidatePoolInvariant(ctx, pool, oldK); err != nil {
		return math.ZeroInt(), err
	}

	// 15. Validate final pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// 16. Save updated pool
	sdkCtx.GasMeter().ConsumeGas(8000, "dex_swap_state_update")
	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), err
	}

	// 17. Transfer output tokens to trader
	coinOut := sdk.NewCoin(tokenOut, amountOut)
	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer output tokens: %v", err)
	}

	// 18. Emit comprehensive event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDexSwap,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute(types.AttributeKeyTokenIn, tokenIn),
			sdk.NewAttribute(types.AttributeKeyTokenOut, tokenOut),
			sdk.NewAttribute(types.AttributeKeyAmountIn, amountIn.String()),
			sdk.NewAttribute(types.AttributeKeyAmountOut, amountOut.String()),
			sdk.NewAttribute(types.AttributeKeyFee, feeAmount.String()),
			sdk.NewAttribute(types.AttributeKeyReserveA, pool.ReserveA.String()),
			sdk.NewAttribute(types.AttributeKeyReserveB, pool.ReserveB.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, trader.String()),
		),
	})

	return amountOut, nil
}

// CalculateSwapOutputSecure calculates swap output with overflow protection
func (k Keeper) CalculateSwapOutputSecure(
	ctx context.Context,
	amountIn, reserveIn, reserveOut math.Int,
	swapFee, maxPoolDrainPercent math.LegacyDec,
) (math.Int, error) {
	// Input validation
	if amountIn.IsZero() || amountIn.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("input amount must be positive")
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("pool reserves must be positive")
	}

	// Validate fee is in valid range [0, 1)
	if swapFee.IsNegative() || swapFee.GTE(math.LegacyOneDec()) {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("swap fee must be in range [0, 1)")
	}

	if maxPoolDrainPercent.IsNegative() || maxPoolDrainPercent.GT(math.LegacyOneDec()) {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("max pool drain percent must be within [0, 1]")
	}

	// Calculate amount after fee: amountIn * (1 - fee)
	oneMinusFee := math.LegacyOneDec().Sub(swapFee)
	if oneMinusFee.IsNegative() || oneMinusFee.GT(math.LegacyOneDec()) {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("invalid fee calculation")
	}

	amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(oneMinusFee)

	// Calculate output using constant product formula
	// amountOut = (amountInAfterFee * reserveOut) / (reserveIn + amountInAfterFee)
	numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveOut))
	denominator := math.LegacyNewDecFromInt(reserveIn).Add(amountInAfterFee)

	if denominator.IsZero() || denominator.IsNegative() {
		return math.ZeroInt(), types.ErrDivisionByZero.Wrap("invalid denominator in swap calculation")
	}

	amountOut := numerator.Quo(denominator).TruncateInt()

	// Validate output
	if amountOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("output amount too small")
	}

	if amountOut.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("negative output amount")
	}

	// Ensure we don't drain the pool
	if amountOut.GTE(reserveOut) {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf(
			"insufficient liquidity: output %s >= reserve %s",
			amountOut, reserveOut,
		)
	}

	// Additional safety check: verify output is reasonable (less than configured drain percent)
	maxOutput := math.LegacyNewDecFromInt(reserveOut).Mul(maxPoolDrainPercent).TruncateInt()
	if amountOut.GT(maxOutput) {
		percent := maxPoolDrainPercent.MulInt64(100)
		return math.ZeroInt(), types.ErrSwapTooLarge.Wrapf(
			"swap would drain too much liquidity: output %s > limit %s (%s%% of reserves)",
			amountOut,
			maxOutput,
			percent.String(),
		)
	}

	return amountOut, nil
}

// SimulateSwapSecure simulates a swap with all validations but no state changes
func (k Keeper) SimulateSwapSecure(ctx context.Context, poolID uint64, tokenIn, tokenOut string, amountIn math.Int) (math.Int, error) {
	// Input validation
	if amountIn.IsZero() || amountIn.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrap("cannot swap identical tokens")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	// Validate pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// Check circuit breaker
	if err := k.CheckCircuitBreaker(ctx, pool, "simulate_swap"); err != nil {
		return math.ZeroInt(), err
	}

	// Determine reserves
	var reserveIn, reserveOut math.Int

	if tokenIn == pool.TokenA && tokenOut == pool.TokenB {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if tokenIn == pool.TokenB && tokenOut == pool.TokenA {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrapf(
			"invalid token pair for pool %d: expected %s/%s, got %s/%s",
			poolID, pool.TokenA, pool.TokenB, tokenIn, tokenOut,
		)
	}

	// Validate swap size
	if err := k.ValidateSwapSize(amountIn, reserveIn); err != nil {
		return math.ZeroInt(), err
	}

	// Get parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Calculate output
	amountOut, err := k.CalculateSwapOutputSecure(ctx, amountIn, reserveIn, reserveOut, params.SwapFee, params.MaxPoolDrainPercent)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Validate price impact
	if err := k.ValidatePriceImpact(amountIn, reserveIn, reserveOut, amountOut); err != nil {
		return math.ZeroInt(), err
	}

	return amountOut, nil
}

// GetSpotPriceSecure returns the spot price with validation
func (k Keeper) GetSpotPriceSecure(ctx context.Context, poolID uint64, tokenIn, tokenOut string) (math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	// Validate pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return math.LegacyZeroDec(), err
	}

	// Determine reserves
	var reserveIn, reserveOut math.Int

	if tokenIn == pool.TokenA && tokenOut == pool.TokenB {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if tokenIn == pool.TokenB && tokenOut == pool.TokenA {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return math.LegacyZeroDec(), types.ErrInvalidTokenPair.Wrapf("invalid token pair for pool %d", poolID)
	}

	if reserveIn.IsZero() {
		return math.LegacyZeroDec(), types.ErrInsufficientLiquidity.Wrap("reserve in is zero")
	}

	if reserveOut.IsZero() {
		return math.LegacyZeroDec(), types.ErrInsufficientLiquidity.Wrap("reserve out is zero")
	}

	// Spot price = reserveOut / reserveIn
	spotPrice := math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))

	return spotPrice, nil
}
