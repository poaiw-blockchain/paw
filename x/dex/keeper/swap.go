package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ExecuteSwap performs a token swap using the constant product AMM formula
func (k Keeper) ExecuteSwap(ctx context.Context, trader sdk.AccAddress, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) (math.Int, error) {
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

	// Calculate swap output using constant product formula with fees
	amountOut, err := k.CalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, params.SwapFee)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Check slippage protection
	if amountOut.LT(minAmountOut) {
		return math.ZeroInt(), fmt.Errorf("slippage too high: expected at least %s, got %s", minAmountOut, amountOut)
	}

	// Calculate fees
	feeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()

	// Update pool reserves
	if isTokenAIn {
		pool.ReserveA = pool.ReserveA.Add(amountIn)
		pool.ReserveB = pool.ReserveB.Sub(amountOut)
	} else {
		pool.ReserveB = pool.ReserveB.Add(amountIn)
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
	return k.CalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, params.SwapFee)
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
