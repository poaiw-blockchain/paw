package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetLiquidity retrieves a user's liquidity position in a pool
func (k Keeper) GetLiquidity(ctx context.Context, poolID uint64, provider sdk.AccAddress) (math.Int, error) {
	store := k.getStore(ctx)
	bz := store.Get(LiquidityKey(poolID, provider))
	if bz == nil {
		return math.ZeroInt(), nil
	}

	var shares math.Int
	if err := shares.Unmarshal(bz); err != nil {
		return math.ZeroInt(), err
	}
	return shares, nil
}

// SetLiquidity sets a user's liquidity position in a pool
func (k Keeper) SetLiquidity(ctx context.Context, poolID uint64, provider sdk.AccAddress, shares math.Int) error {
	store := k.getStore(ctx)
	if shares.IsZero() {
		// Remove the liquidity position if shares are zero
		store.Delete(LiquidityKey(poolID, provider))
		return nil
	}

	bz, err := shares.Marshal()
	if err != nil {
		return err
	}
	store.Set(LiquidityKey(poolID, provider), bz)
	return nil
}

// AddLiquidity adds liquidity to an existing pool
func (k Keeper) AddLiquidity(ctx context.Context, provider sdk.AccAddress, poolID uint64, amountA, amountB math.Int) (math.Int, error) {
	// Validate inputs
	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("liquidity amounts must be positive")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), err
	}

	// DIVISION BY ZERO PROTECTION: Explicit zero checks before Quo() operations
	// Check for invalid pool states that would cause division by zero
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		// Pool has no reserves - this is the first liquidity provision
		if !pool.TotalShares.IsZero() {
			// CRITICAL: Pool has shares but no reserves - this is a corrupted state
			return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool has shares but zero reserves")
		}

		// First liquidity provider - calculate initial shares with overflow protection
		// Use geometric mean: sqrt(amountA * amountB) to prevent initial share manipulation
		// This follows Uniswap V2 pattern for initial liquidity
		newShares, err := k.SafeCalculatePoolShares(amountA, amountB)
		if err != nil {
			return math.ZeroInt(), err
		}

		if newShares.IsZero() {
			return math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("initial liquidity amounts too small")
		}

		// Update pool reserves and total shares for first deposit
		pool.ReserveA = amountA
		pool.ReserveB = amountB
		pool.TotalShares = newShares

		if err := k.SetPool(ctx, pool); err != nil {
			return math.ZeroInt(), err
		}

		// Set user's liquidity position
		if err := k.SetLiquidity(ctx, poolID, provider, newShares); err != nil {
			return math.ZeroInt(), err
		}

		// Transfer tokens from provider to module
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		moduleAddr := k.GetModuleAddress()

		coinA := sdk.NewCoin(pool.TokenA, amountA)
		coinB := sdk.NewCoin(pool.TokenB, amountB)

		if err := k.bankKeeper.SendCoins(sdkCtx, provider, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
			return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
		}

		// Emit event
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDexAddLiquidity,
				sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
				sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
				sdk.NewAttribute(types.AttributeKeyAmountA, amountA.String()),
				sdk.NewAttribute(types.AttributeKeyAmountB, amountB.String()),
				sdk.NewAttribute(types.AttributeKeyShares, newShares.String()),
			),
		)

		// Record metrics
		if k.metrics != nil {
			poolIDStr := fmt.Sprintf("%d", poolID)
			k.metrics.LiquidityAdded.WithLabelValues(poolIDStr, pool.TokenA).Add(float64(amountA.Int64()))
			k.metrics.LiquidityAdded.WithLabelValues(poolIDStr, pool.TokenB).Add(float64(amountB.Int64()))
		}

		return newShares, nil
	}

	// Additional safety check: pool must have shares if it has reserves
	if pool.TotalShares.IsZero() {
		// CRITICAL: Pool has reserves but no shares - this is a corrupted state
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool has reserves but zero shares")
	}

	// Now safe to perform division operations - all denominators are guaranteed non-zero
	// Calculate optimal amounts and shares with overflow protection
	// For proportional liquidity: amountA/reserveA = amountB/reserveB = shares/totalShares

	// OVERFLOW CHECK: amountA * pool.ReserveB
	numeratorB, err := amountA.SafeMul(pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow calculating optimal amountB: %s * %s: %v",
			amountA.String(), pool.ReserveB.String(), err)
	}
	optimalAmountB, err := numeratorB.SafeQuo(pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in optimal amountB division: %s / %s: %v",
			numeratorB.String(), pool.ReserveA.String(), err)
	}

	// OVERFLOW CHECK: amountB * pool.ReserveA
	numeratorA, err := amountB.SafeMul(pool.ReserveA)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow calculating optimal amountA: %s * %s: %v",
			amountB.String(), pool.ReserveA.String(), err)
	}
	optimalAmountA, err := numeratorA.SafeQuo(pool.ReserveB)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow in optimal amountA division: %s / %s: %v",
			numeratorA.String(), pool.ReserveB.String(), err)
	}

	var finalAmountA, finalAmountB math.Int

	if optimalAmountB.LTE(amountB) {
		// Use amountA and calculated amountB
		finalAmountA = amountA
		finalAmountB = optimalAmountB
	} else {
		// Use amountB and calculated amountA
		finalAmountA = optimalAmountA
		finalAmountB = amountB
	}

	// Calculate shares to mint with overflow protection
	newShares, err := k.SafeCalculateAddLiquidityShares(finalAmountA, finalAmountB, pool.ReserveA, pool.ReserveB, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), err
	}

	if newShares.IsZero() {
		return math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("liquidity contribution too small")
	}

	// Update pool reserves and total shares with overflow protection
	newReserveA, err := pool.ReserveA.SafeAdd(finalAmountA)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding to reserveA: %s + %s: %v",
			pool.ReserveA.String(), finalAmountA.String(), err)
	}
	newReserveB, err := pool.ReserveB.SafeAdd(finalAmountB)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding to reserveB: %s + %s: %v",
			pool.ReserveB.String(), finalAmountB.String(), err)
	}
	newTotalShares, err := pool.TotalShares.SafeAdd(newShares)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding to total shares: %s + %s: %v",
			pool.TotalShares.String(), newShares.String(), err)
	}

	pool.ReserveA = newReserveA
	pool.ReserveB = newReserveB
	pool.TotalShares = newTotalShares

	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), err
	}

	// Update user's liquidity position with overflow protection
	currentShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), err
	}
	userTotalShares, err := currentShares.SafeAdd(newShares)
	if err != nil {
		return math.ZeroInt(), types.ErrOverflow.Wrapf("overflow adding user shares: %s + %s: %v",
			currentShares.String(), newShares.String(), err)
	}
	if err := k.SetLiquidity(ctx, poolID, provider, userTotalShares); err != nil {
		return math.ZeroInt(), err
	}

	// Transfer tokens from provider to module
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(pool.TokenA, finalAmountA)
	coinB := sdk.NewCoin(pool.TokenB, finalAmountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, provider, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
		return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// Mark pool as active for activity-based tracking
	if err := k.MarkPoolActive(ctx, poolID); err != nil {
		// Log error but don't fail the operation - activity tracking is non-critical
		sdkCtx.Logger().Error("failed to mark pool active", "pool_id", poolID, "error", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexAddLiquidity,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
			sdk.NewAttribute(types.AttributeKeyAmountA, finalAmountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, finalAmountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, newShares.String()),
		),
	)

	// Record liquidity added metrics
	if k.metrics != nil {
		poolIDStr := fmt.Sprintf("%d", poolID)
		k.metrics.LiquidityAdded.WithLabelValues(poolIDStr, pool.TokenA).Add(float64(finalAmountA.Int64()))
		k.metrics.LiquidityAdded.WithLabelValues(poolIDStr, pool.TokenB).Add(float64(finalAmountB.Int64()))
	}

	return newShares, nil
}

// RemoveLiquidity removes liquidity from a pool
func (k Keeper) RemoveLiquidity(ctx context.Context, provider sdk.AccAddress, poolID uint64, shares math.Int) (math.Int, math.Int, error) {
	// Validate inputs
	if shares.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientShares.Wrap("shares must be positive")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// DIVISION BY ZERO PROTECTION: Check pool state before Quo() operations
	if pool.TotalShares.IsZero() {
		// CRITICAL: Pool has no shares - cannot calculate withdrawal amounts
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool has zero total shares")
	}

	// Additional safety: verify reserves are non-zero if shares exist
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		// CRITICAL: Pool has shares but zero reserves - corrupted state
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool has shares but zero reserves")
	}

	// Check user's liquidity position
	userShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if shares.GT(userShares) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientShares.Wrapf("have %s, need %s", userShares, shares)
	}

	// Calculate amounts to return with overflow protection
	amountA, amountB, err := k.SafeCalculateRemoveLiquidityAmounts(shares, pool.ReserveA, pool.ReserveB, pool.TotalShares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("withdrawal amounts too small")
	}

	// Update pool reserves and total shares with overflow protection
	newReserveA, err := pool.ReserveA.SafeSub(amountA)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting from reserveA: %s - %s: %v",
			pool.ReserveA.String(), amountA.String(), err)
	}
	newReserveB, err := pool.ReserveB.SafeSub(amountB)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting from reserveB: %s - %s: %v",
			pool.ReserveB.String(), amountB.String(), err)
	}
	newTotalShares, err := pool.TotalShares.SafeSub(shares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting from total shares: %s - %s: %v",
			pool.TotalShares.String(), shares.String(), err)
	}

	pool.ReserveA = newReserveA
	pool.ReserveB = newReserveB
	pool.TotalShares = newTotalShares

	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// Update user's liquidity position with overflow protection
	newUserShares, err := userShares.SafeSub(shares)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrOverflow.Wrapf("overflow subtracting user shares: %s - %s: %v",
			userShares.String(), shares.String(), err)
	}
	if err := k.SetLiquidity(ctx, poolID, provider, newUserShares); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// Transfer tokens from module to provider
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(pool.TokenA, amountA)
	coinB := sdk.NewCoin(pool.TokenB, amountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, provider, sdk.NewCoins(coinA, coinB)); err != nil {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// Mark pool as active for activity-based tracking
	if err := k.MarkPoolActive(ctx, poolID); err != nil {
		// Log error but don't fail the operation - activity tracking is non-critical
		sdkCtx.Logger().Error("failed to mark pool active", "pool_id", poolID, "error", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexRemoveLiquidity,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
			sdk.NewAttribute(types.AttributeKeyAmountA, amountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, amountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, shares.String()),
		),
	)

	// Record liquidity removed metrics
	if k.metrics != nil {
		poolIDStr := fmt.Sprintf("%d", poolID)
		k.metrics.LiquidityRemoved.WithLabelValues(poolIDStr, pool.TokenA).Add(float64(amountA.Int64()))
		k.metrics.LiquidityRemoved.WithLabelValues(poolIDStr, pool.TokenB).Add(float64(amountB.Int64()))
	}

	return amountA, amountB, nil
}

// IterateLiquidityByPool iterates over all liquidity positions in a pool
func (k Keeper) IterateLiquidityByPool(ctx context.Context, poolID uint64, cb func(provider sdk.AccAddress, shares math.Int) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, LiquidityKeyByPoolPrefix(poolID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var shares math.Int
		if err := shares.Unmarshal(iterator.Value()); err != nil {
			return err
		}

		// Extract provider address from key
		key := iterator.Key()
		providerBytes := key[len(LiquidityKeyByPoolPrefix(poolID)):]
		provider := sdk.AccAddress(providerBytes)

		if cb(provider, shares) {
			break
		}
	}
	return nil
}
