package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		return math.ZeroInt(), fmt.Errorf("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), fmt.Errorf("cannot swap identical tokens")
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
		return math.ZeroInt(), fmt.Errorf("invalid token pair for pool %d: expected %s/%s, got %s/%s",
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
		return math.ZeroInt(), fmt.Errorf("fee amount exceeds swap amount: fee %s, amount %s", feeAmount.String(), amountIn.String())
	}
	if amountInAfterFee.IsZero() {
		return math.ZeroInt(), fmt.Errorf("swap amount too small after fees")
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
		return math.ZeroInt(), fmt.Errorf("slippage too high: expected at least %s, got %s", minAmountOut, amountOut)
	}

	// Update pool reserves
	if isTokenAIn {
		pool.ReserveA = pool.ReserveA.Add(amountInAfterFee)
		pool.ReserveB = pool.ReserveB.Sub(amountOut)
	} else {
		pool.ReserveB = pool.ReserveB.Add(amountInAfterFee)
		pool.ReserveA = pool.ReserveA.Sub(amountOut)
	}

	// Save updated pool
	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), err
	}

	// Execute token transfers
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	// Transfer input tokens from trader to module
	coinIn := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to transfer input tokens: %w", err)
	}

	// Transfer output tokens from module to trader
	lpFee, protocolFee, err := k.CollectSwapFees(ctx, poolID, tokenIn, amountIn)
	if err != nil {
		return math.ZeroInt(), err
	}
	feeAmount, err = SafeAdd(lpFee, protocolFee)
	if err != nil {
		return math.ZeroInt(), err
	}
	coinOut := sdk.NewCoin(tokenOut, amountOut)
	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to transfer output tokens: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_executed",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute("token_in", tokenIn),
			sdk.NewAttribute("token_out", tokenOut),
			sdk.NewAttribute("amount_in", amountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
			sdk.NewAttribute("fee", feeAmount.String()),
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
		return math.ZeroInt(), fmt.Errorf("input amount must be positive")
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return math.ZeroInt(), fmt.Errorf("pool reserves must be positive")
	}

	// Calculate amount after fee: amountIn * (1 - fee)
	oneMinusFee := math.LegacyOneDec().Sub(swapFee)
	amountInAfterFee := math.LegacyNewDecFromInt(amountIn).Mul(oneMinusFee)

	// Calculate output: (amountInAfterFee * reserveOut) / (reserveIn + amountInAfterFee)
	numerator := amountInAfterFee.Mul(math.LegacyNewDecFromInt(reserveOut))
	denominator := math.LegacyNewDecFromInt(reserveIn).Add(amountInAfterFee)

	amountOut := numerator.Quo(denominator).TruncateInt()

	if amountOut.IsZero() {
		return math.ZeroInt(), fmt.Errorf("output amount too small")
	}

	if amountOut.GTE(reserveOut) {
		return math.ZeroInt(), fmt.Errorf("insufficient liquidity: output %s >= reserve %s", amountOut, reserveOut)
	}

	return amountOut, nil
}

// SimulateSwap simulates a swap without executing it
func (k Keeper) SimulateSwap(ctx context.Context, poolID uint64, tokenIn, tokenOut string, amountIn math.Int) (math.Int, error) {
	// Validate inputs
	if amountIn.IsZero() {
		return math.ZeroInt(), fmt.Errorf("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), fmt.Errorf("cannot swap identical tokens")
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
		return math.ZeroInt(), fmt.Errorf("invalid token pair for pool %d: expected %s/%s, got %s/%s",
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
		return math.ZeroInt(), fmt.Errorf("fee amount exceeds swap amount: fee %s, amount %s", feeAmount.String(), amountIn.String())
	}
	if amountInAfterFee.IsZero() {
		return math.ZeroInt(), fmt.Errorf("swap amount too small after fees")
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
		return math.LegacyZeroDec(), fmt.Errorf("invalid token pair for pool %d", poolID)
	}

	if reserveIn.IsZero() {
		return math.LegacyZeroDec(), fmt.Errorf("reserve is zero")
	}

	// Spot price = reserveOut / reserveIn
	spotPrice := math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))
	return spotPrice, nil
}
