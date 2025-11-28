package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		return math.ZeroInt(), fmt.Errorf("liquidity amounts must be positive")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), err
	}

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
		return math.ZeroInt(), fmt.Errorf("liquidity contribution too small")
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
		return math.ZeroInt(), fmt.Errorf("failed to transfer tokens: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"liquidity_added",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount_a", finalAmountA.String()),
			sdk.NewAttribute("amount_b", finalAmountB.String()),
			sdk.NewAttribute("shares", newShares.String()),
		),
	)

	return newShares, nil
}

// RemoveLiquidity removes liquidity from a pool
func (k Keeper) RemoveLiquidity(ctx context.Context, provider sdk.AccAddress, poolID uint64, shares math.Int) (math.Int, math.Int, error) {
	// Validate inputs
	if shares.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("shares must be positive")
	}

	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	// Check user's liquidity position
	userShares, err := k.GetLiquidity(ctx, poolID, provider)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if shares.GT(userShares) {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("insufficient shares: have %s, need %s", userShares, shares)
	}

	// Calculate amounts to return (proportional to shares)
	amountA := shares.Mul(pool.ReserveA).Quo(pool.TotalShares)
	amountB := shares.Mul(pool.ReserveB).Quo(pool.TotalShares)

	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("withdrawal amounts too small")
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
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("failed to transfer tokens: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"liquidity_removed",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount_a", amountA.String()),
			sdk.NewAttribute("amount_b", amountB.String()),
			sdk.NewAttribute("shares", shares.String()),
		),
	)

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
