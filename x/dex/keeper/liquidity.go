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

		// First liquidity provider - calculate initial shares
		// Use geometric mean: sqrt(amountA * amountB) to prevent initial share manipulation
		// This follows Uniswap V2 pattern for initial liquidity
		product := amountA.Mul(amountB)
		sqrtShares, err := math.LegacyNewDecFromInt(product).ApproxSqrt()
		if err != nil {
			return math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrapf("failed to calculate initial shares: %v", err)
		}
		newShares := sqrtShares.TruncateInt()

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
	// Calculate optimal amounts and shares
	// For proportional liquidity: amountA/reserveA = amountB/reserveB = shares/totalShares
	optimalAmountB := amountA.Mul(pool.ReserveB).Quo(pool.ReserveA)
	optimalAmountA := amountB.Mul(pool.ReserveA).Quo(pool.ReserveB)

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

	// Calculate shares to mint (proportional to contribution)
	sharesA := finalAmountA.Mul(pool.TotalShares).Quo(pool.ReserveA)
	sharesB := finalAmountB.Mul(pool.TotalShares).Quo(pool.ReserveB)

	// Use the minimum to maintain proportionality
	newShares := sharesA
	if sharesB.LT(sharesA) {
		newShares = sharesB
	}

	if newShares.IsZero() {
		return math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("liquidity contribution too small")
	}

	// Update pool reserves and total shares
	pool.ReserveA = pool.ReserveA.Add(finalAmountA)
	pool.ReserveB = pool.ReserveB.Add(finalAmountB)
	pool.TotalShares = pool.TotalShares.Add(newShares)

	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), err
	}

	// Update user's liquidity position
	currentShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), err
	}
	newTotalShares := currentShares.Add(newShares)
	if err := k.SetLiquidity(ctx, poolID, provider, newTotalShares); err != nil {
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

	// Now safe to calculate amounts to return (proportional to shares)
	// All denominators are guaranteed non-zero by the checks above
	amountA := shares.Mul(pool.ReserveA).Quo(pool.TotalShares)
	amountB := shares.Mul(pool.ReserveB).Quo(pool.TotalShares)

	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidLiquidityAmount.Wrap("withdrawal amounts too small")
	}

	// Update pool reserves and total shares
	pool.ReserveA = pool.ReserveA.Sub(amountA)
	pool.ReserveB = pool.ReserveB.Sub(amountB)
	pool.TotalShares = pool.TotalShares.Sub(shares)

	if err := k.SetPool(ctx, pool); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// Update user's liquidity position
	newUserShares := userShares.Sub(shares)
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
