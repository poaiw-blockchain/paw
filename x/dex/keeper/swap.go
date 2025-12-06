package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// ExecuteSwap performs a token swap using the constant product AMM formula
func (k Keeper) ExecuteSwap(ctx context.Context, trader sdk.AccAddress, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) (math.Int, error) {
	// Track swap latency
	start := time.Now()
	defer func() {
		k.metrics.SwapLatency.Observe(time.Since(start).Seconds())
	}()

	// Validate inputs
	if amountIn.IsZero() {
		k.metrics.SwapsTotal.WithLabelValues(fmt.Sprintf("%d", poolID), tokenIn, tokenOut, "failed").Inc()
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrap("cannot swap identical tokens")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Determine which token is being swapped
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
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrapf("invalid token pair for pool %d: expected %s/%s, got %s/%s",
			poolID, pool.TokenA, pool.TokenB, tokenIn, tokenOut)
	}

	// Get swap parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Enforce MEV protection limits on swap size
	if err := k.ValidateSwapSize(amountIn, reserveIn); err != nil {
		return math.ZeroInt(), err
	}

	// Calculate swap output using constant product formula with fees
	feeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()
	amountInAfterFee := amountIn.Sub(feeAmount)
	if amountInAfterFee.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrapf("fee amount exceeds swap amount: fee %s, amount %s", feeAmount.String(), amountIn.String())
	}
	if amountInAfterFee.IsZero() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("swap amount too small after fees")
	}
	amountOut, err := k.CalculateSwapOutput(ctx, amountInAfterFee, reserveIn, reserveOut, math.LegacyZeroDec())
	if err != nil {
		return math.ZeroInt(), err
	}

	// Check slippage protection
	if amountOut.LT(minAmountOut) {
		k.metrics.SwapsTotal.WithLabelValues(fmt.Sprintf("%d", poolID), tokenIn, tokenOut, "failed").Inc()
		// Calculate and record slippage
		expectedOut := math.LegacyNewDecFromInt(minAmountOut)
		actualOut := math.LegacyNewDecFromInt(amountOut)
		if !expectedOut.IsZero() {
			slippagePercent := expectedOut.Sub(actualOut).Quo(expectedOut).Mul(math.LegacyNewDec(100))
			k.metrics.SwapSlippage.Observe(slippagePercent.MustFloat64())
		}
		return math.ZeroInt(), types.ErrSlippageTooHigh.Wrapf("expected at least %s, got %s", minAmountOut, amountOut)
	}

	// ATOMICITY FIX: Execute token transfers FIRST (before updating pool state)
	// This ensures that if transfers fail, pool state remains unchanged
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	// Step 1: Transfer input tokens from trader to module
	coinIn := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer input tokens: %w", err)
	}

	// Step 2: Collect fees (must happen before output transfer)
	lpFee, protocolFee, err := k.CollectSwapFees(ctx, poolID, tokenIn, amountIn)
	if err != nil {
		// Revert the input transfer on fee collection failure
		_ = k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn))
		return math.ZeroInt(), types.WrapWithRecovery(err, "failed to collect swap fees")
	}
	feeAmount, err = SafeAdd(lpFee, protocolFee)
	if err != nil {
		// Revert the input transfer on fee calculation failure
		_ = k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn))
		return math.ZeroInt(), types.WrapWithRecovery(types.ErrOverflow, "failed to calculate total fees: %w", err)
	}

	// Step 3: Transfer output tokens from module to trader
	coinOut := sdk.NewCoin(tokenOut, amountOut)
	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
		// Revert the input transfer on output transfer failure
		_ = k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn))
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer output tokens: %w", err)
	}

	// Step 4: ONLY NOW update pool state (after all transfers succeeded)
	// This ensures pool state is never inconsistent with actual token balances
	if isTokenAIn {
		pool.ReserveA = pool.ReserveA.Add(amountInAfterFee)
		pool.ReserveB = pool.ReserveB.Sub(amountOut)
	} else {
		pool.ReserveB = pool.ReserveB.Add(amountInAfterFee)
		pool.ReserveA = pool.ReserveA.Sub(amountOut)
	}

	// Step 5: Save updated pool
	if err := k.SetPool(ctx, pool); err != nil {
		// Critical failure: transfers succeeded but state update failed
		// This should never happen in normal operation
		return math.ZeroInt(), types.ErrStateCorruption.Wrapf("transfers succeeded but pool state update failed: %w", err)
	}

	// Step 6: Update TWAP cumulative price (lazy update - only on swaps)
	// Calculate current spot price for TWAP oracle
	price0 := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
	price1 := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))
	if err := k.UpdateCumulativePriceOnSwap(ctx, poolID, price0, price1); err != nil {
		// Log error but don't fail the swap - TWAP update is non-critical
		sdkCtx.Logger().Error("failed to update TWAP on swap", "pool_id", poolID, "error", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexSwap,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute(types.AttributeKeyTokenIn, tokenIn),
			sdk.NewAttribute(types.AttributeKeyTokenOut, tokenOut),
			sdk.NewAttribute(types.AttributeKeyAmountIn, amountIn.String()),
			sdk.NewAttribute(types.AttributeKeyAmountOut, amountOut.String()),
			sdk.NewAttribute(types.AttributeKeyFee, feeAmount.String()),
		),
	)

	// Record successful swap metrics
	poolIDStr := fmt.Sprintf("%d", poolID)
	k.metrics.SwapsTotal.WithLabelValues(poolIDStr, tokenIn, tokenOut, "success").Inc()
	k.metrics.SwapVolume.WithLabelValues(poolIDStr, tokenIn).Add(float64(amountIn.Int64()))
	k.metrics.SwapFeesCollected.WithLabelValues(poolIDStr, tokenIn).Add(float64(feeAmount.Int64()))

	// Calculate actual slippage
	if !minAmountOut.IsZero() {
		expectedOut := math.LegacyNewDecFromInt(minAmountOut)
		actualOut := math.LegacyNewDecFromInt(amountOut)
		slippagePercent := expectedOut.Sub(actualOut).Quo(expectedOut).Mul(math.LegacyNewDec(100)).Abs()
		k.metrics.SwapSlippage.Observe(slippagePercent.MustFloat64())
	}

	return amountOut, nil
}

// CalculateSwapOutput calculates the output amount for a swap using the constant product formula
// Formula: amountOut = (amountIn * (1 - fee) * reserveOut) / (reserveIn + amountIn * (1 - fee))
func (k Keeper) CalculateSwapOutput(ctx context.Context, amountIn, reserveIn, reserveOut math.Int, swapFee math.LegacyDec) (math.Int, error) {
	if amountIn.IsZero() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("input amount must be positive")
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("pool reserves must be positive")
	}

	// Calculate amount after fee: amountIn * (1 - fee)
	oneMinusFee := math.LegacyOneDec().Sub(swapFee)
	amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(oneMinusFee)

	// Calculate output: (amountInAfterFee * reserveOut) / (reserveIn + amountInAfterFee)
	numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveOut))
	denominator := math.LegacyNewDecFromInt(reserveIn).Add(amountInAfterFee)

	amountOut := numerator.Quo(denominator).TruncateInt()

	if amountOut.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("output amount too small")
	}

	if amountOut.GTE(reserveOut) {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("output %s >= reserve %s", amountOut, reserveOut)
	}

	return amountOut, nil
}

// SimulateSwap simulates a swap without executing it
func (k Keeper) SimulateSwap(ctx context.Context, poolID uint64, tokenIn, tokenOut string, amountIn math.Int) (math.Int, error) {
	// Validate inputs
	if amountIn.IsZero() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrap("cannot swap identical tokens")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Determine which token is being swapped
	var reserveIn, reserveOut math.Int

	if tokenIn == pool.TokenA && tokenOut == pool.TokenB {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if tokenIn == pool.TokenB && tokenOut == pool.TokenA {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrapf("invalid token pair for pool %d: expected %s/%s, got %s/%s",
			poolID, pool.TokenA, pool.TokenB, tokenIn, tokenOut)
	}

	// Get swap parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Calculate swap output
	feeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()
	amountInAfterFee := amountIn.Sub(feeAmount)
	if amountInAfterFee.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrapf("fee amount exceeds swap amount: fee %s, amount %s", feeAmount.String(), amountIn.String())
	}
	if amountInAfterFee.IsZero() {
		return math.ZeroInt(), types.ErrInvalidSwapAmount.Wrap("swap amount too small after fees")
	}
	return k.CalculateSwapOutput(ctx, amountInAfterFee, reserveIn, reserveOut, math.LegacyZeroDec())
}

// Swap wraps the secure swap execution path for scenarios that expect a simple swap entrypoint.
func (k Keeper) Swap(ctx context.Context, trader sdk.AccAddress, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) (math.Int, error) {
	return k.ExecuteSwapSecure(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
}

// GetSpotPrice returns the spot price of tokenOut in terms of tokenIn
func (k Keeper) GetSpotPrice(ctx context.Context, poolID uint64, tokenIn, tokenOut string) (math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
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
		return math.LegacyZeroDec(), types.ErrInsufficientLiquidity.Wrap("reserve is zero")
	}

	// Spot price = reserveOut / reserveIn
	spotPrice := math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))
	return spotPrice, nil
}
