package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// AddLiquiditySecure adds liquidity with comprehensive security checks
func (k Keeper) AddLiquiditySecure(ctx context.Context, provider sdk.AccAddress, poolID uint64, amountA, amountB math.Int) (math.Int, error) {
	// Execute with reentrancy protection
	var shares math.Int
	err := k.WithReentrancyGuard(ctx, poolID, "add_liquidity", func() error {
		var execErr error
		shares, execErr = k.addLiquidityInternal(ctx, provider, poolID, amountA, amountB)
		return execErr
	})

	return shares, err
}

// addLiquidityInternal is the internal implementation with all security checks
func (k Keeper) addLiquidityInternal(ctx context.Context, provider sdk.AccAddress, poolID uint64, amountA, amountB math.Int) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Gas metering: Base liquidity operation cost
	sdkCtx.GasMeter().ConsumeGas(40000, "dex_add_liquidity_base")

	// 1. Input validation
	if amountA.IsZero() || amountA.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("amount A must be positive")
	}
	sdkCtx.GasMeter().ConsumeGas(1000, "dex_add_liquidity_validation")

	if amountB.IsZero() || amountB.IsNegative() {
		return math.ZeroInt(), types.ErrInvalidInput.Wrap("amount B must be positive")
	}

	// 2. Get pool and validate state
	sdkCtx.GasMeter().ConsumeGas(5000, "dex_add_liquidity_pool_lookup")
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// 3. Check circuit breaker
	if err := k.CheckCircuitBreaker(ctx, pool, "add_liquidity"); err != nil {
		return math.ZeroInt(), err
	}

	// 4. Calculate optimal amounts and shares (proportional liquidity)
	var finalAmountA, finalAmountB, newShares math.Int

	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("cannot add liquidity to empty pool")
	}

	// Calculate optimal amounts to maintain price ratio
	optimalAmountB, err := SafeMul(amountA, pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), err
	}
	optimalAmountB, err = SafeQuo(optimalAmountB, pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), err
	}

	optimalAmountA, err := SafeMul(amountB, pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), err
	}
	optimalAmountA, err = SafeQuo(optimalAmountA, pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Use the optimal amounts that fit within provided limits
	if optimalAmountB.LTE(amountB) {
		finalAmountA = amountA
		finalAmountB = optimalAmountB
	} else {
		finalAmountA = optimalAmountA
		finalAmountB = amountB
	}

	// 5. Calculate shares to mint (proportional to contribution)
	sdkCtx.GasMeter().ConsumeGas(8000, "dex_add_liquidity_calculation")
	sharesA, err := SafeMul(finalAmountA, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), err
	}
	sharesA, err = SafeQuo(sharesA, pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), err
	}

	sharesB, err := SafeMul(finalAmountB, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), err
	}
	sharesB, err = SafeQuo(sharesB, pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Use minimum to maintain proportionality
	newShares = sharesA
	if sharesB.LT(sharesA) {
		newShares = sharesB
	}

	if newShares.IsZero() {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("liquidity contribution too small")
	}

	// 6. Store old k for invariant check
	oldK := pool.ReserveA.Mul(pool.ReserveB)

	// 7. Transfer tokens FIRST (checks-effects-interactions)
	sdkCtx.GasMeter().ConsumeGas(15000, "dex_add_liquidity_token_transfer")
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(pool.TokenA, finalAmountA)
	coinB := sdk.NewCoin(pool.TokenB, finalAmountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, provider, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// 8. Update pool state AFTER receiving tokens
	pool.ReserveA, err = SafeAdd(pool.ReserveA, finalAmountA)
	if err != nil {
		return math.ZeroInt(), err
	}

	pool.ReserveB, err = SafeAdd(pool.ReserveB, finalAmountB)
	if err != nil {
		return math.ZeroInt(), err
	}

	pool.TotalShares, err = SafeAdd(pool.TotalShares, newShares)
	if err != nil {
		return math.ZeroInt(), err
	}

	// 9. Validate invariant
	if err := k.ValidatePoolInvariant(ctx, pool, oldK); err != nil {
		return math.ZeroInt(), err
	}

	// 10. Validate final pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), err
	}

	// 11. Save pool
	sdkCtx.GasMeter().ConsumeGas(10000, "dex_add_liquidity_state_update")
	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), err
	}

	// 12. Update user's liquidity position
	currentShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), err
	}

	newTotalShares, err := SafeAdd(currentShares, newShares)
	if err != nil {
		return math.ZeroInt(), err
	}

	if err := k.SetLiquidity(ctx, poolID, provider, newTotalShares); err != nil {
		return math.ZeroInt(), err
	}

	// 13. Record liquidity action for flash loan protection
	if err := k.SetLastLiquidityActionBlock(ctx, poolID, provider); err != nil {
		return math.ZeroInt(), err
	}

	// 14. Emit event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"liquidity_added",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount_a", finalAmountA.String()),
			sdk.NewAttribute("amount_b", finalAmountB.String()),
			sdk.NewAttribute("shares", newShares.String()),
			sdk.NewAttribute("total_shares", pool.TotalShares.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, provider.String()),
		),
	})

	return newShares, nil
}

// RemoveLiquiditySecure removes liquidity with comprehensive security checks
func (k Keeper) RemoveLiquiditySecure(ctx context.Context, provider sdk.AccAddress, poolID uint64, shares math.Int) (math.Int, math.Int, error) {
	// Execute with reentrancy protection
	var amountA, amountB math.Int
	err := k.WithReentrancyGuard(ctx, poolID, "remove_liquidity", func() error {
		var execErr error
		amountA, amountB, execErr = k.removeLiquidityInternal(ctx, provider, poolID, shares)
		return execErr
	})

	return amountA, amountB, err
}

// removeLiquidityInternal is the internal implementation with all security checks
func (k Keeper) removeLiquidityInternal(ctx context.Context, provider sdk.AccAddress, poolID uint64, shares math.Int) (math.Int, math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Gas metering: Base remove liquidity operation cost
	sdkCtx.GasMeter().ConsumeGas(40000, "dex_remove_liquidity_base")

	// 1. Input validation
	if shares.IsZero() || shares.IsNegative() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidInput.Wrap("shares must be positive")
	}
	sdkCtx.GasMeter().ConsumeGas(1000, "dex_remove_liquidity_validation")

	// 2. Flash loan protection - check minimum lock period
	if err := k.CheckFlashLoanProtection(ctx, poolID, provider); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 3. Get pool and validate state
	sdkCtx.GasMeter().ConsumeGas(5000, "dex_remove_liquidity_pool_lookup")
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 4. Check circuit breaker
	if err := k.CheckCircuitBreaker(ctx, pool, "remove_liquidity"); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 5. Check user's liquidity position
	userShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if shares.GT(userShares) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientShares.Wrapf(
			"insufficient shares: have %s, need %s",
			userShares, shares,
		)
	}

	// 6. Calculate amounts to return (proportional to shares)
	sdkCtx.GasMeter().ConsumeGas(8000, "dex_remove_liquidity_calculation")
	amountA, err := SafeMul(shares, pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	amountA, err = SafeQuo(amountA, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	amountB, err := SafeMul(shares, pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}
	amountB, err = SafeQuo(amountB, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("withdrawal amounts too small")
	}

	// 7. Update pool state BEFORE transfers (checks-effects-interactions)
	pool.ReserveA, err = SafeSub(pool.ReserveA, amountA)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	pool.ReserveB, err = SafeSub(pool.ReserveB, amountB)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	pool.TotalShares, err = SafeSub(pool.TotalShares, shares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 9. Validate pool state (note: k will decrease, which is expected)
	if err := k.ValidatePoolState(pool); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 10. Save pool
	sdkCtx.GasMeter().ConsumeGas(10000, "dex_remove_liquidity_state_update")
	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 11. Update user's liquidity position
	newUserShares, err := SafeSub(userShares, shares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if err := k.SetLiquidity(ctx, poolID, provider, newUserShares); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 12. Record liquidity action for flash loan protection
	if err := k.SetLastLiquidityActionBlock(ctx, poolID, provider); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// 13. Transfer tokens to provider LAST
	sdkCtx.GasMeter().ConsumeGas(15000, "dex_remove_liquidity_token_transfer")
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(pool.TokenA, amountA)
	coinB := sdk.NewCoin(pool.TokenB, amountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, provider, sdk.NewCoins(coinA, coinB)); err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// 14. Emit event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"liquidity_removed",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount_a", amountA.String()),
			sdk.NewAttribute("amount_b", amountB.String()),
			sdk.NewAttribute("shares", shares.String()),
			sdk.NewAttribute("remaining_shares", newUserShares.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, provider.String()),
		),
	})

	return amountA, amountB, nil
}
