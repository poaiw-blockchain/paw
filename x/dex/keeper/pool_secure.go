package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// CreatePoolSecure creates a new liquidity pool with comprehensive security checks
func (k Keeper) CreatePoolSecure(ctx context.Context, creator sdk.AccAddress, tokenA, tokenB string, amountA, amountB math.Int) (*types.Pool, error) {
	// 1. Input validation
	if tokenA == tokenB {
		return nil, types.ErrInvalidTokenPair.Wrap("cannot create pool with identical tokens")
	}

	if tokenA == "" || tokenB == "" {
		return nil, types.ErrInvalidInput.Wrap("token denoms cannot be empty")
	}

	if amountA.IsZero() || amountA.IsNegative() {
		return nil, types.ErrInvalidInput.Wrap("amount A must be positive")
	}

	if amountB.IsZero() || amountB.IsNegative() {
		return nil, types.ErrInvalidInput.Wrap("amount B must be positive")
	}

	// 2. Ensure consistent token ordering (lexicographic)
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
		amountA, amountB = amountB, amountA
	}

	// 3. Check if pool already exists
	existingPool, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
	if err == nil && existingPool != nil {
		return nil, types.ErrPoolAlreadyExists.Wrapf("pool already exists for token pair %s/%s", tokenA, tokenB)
	}

	// 4. Check maximum pools limit (DoS prevention)
	pools, err := k.GetAllPools(ctx)
	if err != nil {
		return nil, err
	}

	if uint64(len(pools)) >= MaxPools {
		return nil, types.ErrMaxPoolsReached.Wrapf("maximum number of pools (%d) reached", MaxPools)
	}

	// 5. Get and validate parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// 6. Calculate initial shares using geometric mean (sqrt(x * y))
	// This prevents initial liquidity manipulation
	product, err := SafeMul(amountA, amountB)
	if err != nil {
		return nil, err
	}

	sqrtShares, err := math.LegacyNewDecFromInt(product).ApproxSqrt()
	if err != nil {
		return nil, types.ErrOverflow.Wrapf("failed to calculate square root: %v", err)
	}

	initialShares := sqrtShares.TruncateInt()

	// 7. Check minimum liquidity requirement
	if initialShares.LT(params.MinLiquidity) {
		return nil, types.ErrInsufficientLiquidity.Wrapf(
			"initial liquidity too low: %s < %s",
			initialShares, params.MinLiquidity,
		)
	}

	// 8. Validate amounts aren't too extreme (prevent price manipulation)
	ratio := math.LegacyNewDecFromInt(amountA).Quo(math.LegacyNewDecFromInt(amountB))

	// Check ratio is reasonable (between 1:1000000 and 1000000:1)
	minRatio := math.LegacyNewDecWithPrec(1, 6) // 0.000001
	maxRatio := math.LegacyNewDec(1000000)      // 1000000

	if ratio.LT(minRatio) || ratio.GT(maxRatio) {
		return nil, types.ErrInvalidInput.Wrapf(
			"initial price ratio too extreme: %s (must be between %s and %s)",
			ratio, minRatio, maxRatio,
		)
	}

	// 9. Get next pool ID with reentrancy protection
	poolID, err := k.GetNextPoolID(ctx)
	if err != nil {
		return nil, err
	}

	// 10. Create pool structure
	pool := &types.Pool{
		Id:          poolID,
		TokenA:      tokenA,
		TokenB:      tokenB,
		ReserveA:    amountA,
		ReserveB:    amountB,
		TotalShares: initialShares,
		Creator:     creator.String(),
	}

	// 11. Validate pool state
	if err := k.ValidatePoolState(pool); err != nil {
		return nil, err
	}

	// 12. Transfer tokens FIRST (checks-effects-interactions)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(tokenA, amountA)
	coinB := sdk.NewCoin(tokenB, amountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, creator, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
		return nil, types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// 13. Save pool to store AFTER receiving tokens
	if err := k.SetPool(ctx, pool); err != nil {
		return nil, err
	}

	// 14. Index pool by tokens
	if err := k.SetPoolByTokens(ctx, tokenA, tokenB, poolID); err != nil {
		return nil, err
	}

	// 15. Set initial liquidity position for creator
	if err := k.SetLiquidity(ctx, poolID, creator, initialShares); err != nil {
		return nil, err
	}

	// 16. Record liquidity action for flash loan protection
	if err := k.SetLastLiquidityActionBlock(ctx, poolID, creator); err != nil {
		return nil, err
	}

	// 17. Initialize circuit breaker state
	cbState := CircuitBreakerState{
		Enabled:       false,
		LastPrice:     math.LegacyNewDecFromInt(amountB).Quo(math.LegacyNewDecFromInt(amountA)),
		TriggerReason: "",
	}
	if err := k.SetCircuitBreakerState(ctx, poolID, cbState); err != nil {
		return nil, err
	}

	// 18. Emit comprehensive event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDexPoolCreated,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("creator", creator.String()),
			sdk.NewAttribute(types.AttributeKeyTokenA, tokenA),
			sdk.NewAttribute(types.AttributeKeyTokenB, tokenB),
			sdk.NewAttribute(types.AttributeKeyAmountA, amountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, amountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, initialShares.String()),
			sdk.NewAttribute(types.AttributeKeyPrice, cbState.LastPrice.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, creator.String()),
		),
	})

	return pool, nil
}

// GetPoolSecure retrieves a pool with state validation
func (k Keeper) GetPoolSecure(ctx context.Context, poolID uint64) (*types.Pool, error) {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	if err := k.ValidatePoolState(pool); err != nil {
		return nil, err
	}

	return pool, nil
}

// GetPoolByTokensSecure retrieves a pool by token pair with validation
func (k Keeper) GetPoolByTokensSecure(ctx context.Context, tokenA, tokenB string) (*types.Pool, error) {
	// Validate inputs
	if tokenA == "" || tokenB == "" {
		return nil, types.ErrInvalidInput.Wrap("token denoms cannot be empty")
	}

	if tokenA == tokenB {
		return nil, types.ErrInvalidTokenPair.Wrap("tokens must be different")
	}

	pool, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
	if err != nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool not found for token pair %s/%s", tokenA, tokenB)
	}

	if err := k.ValidatePoolState(pool); err != nil {
		return nil, err
	}

	return pool, nil
}

// GetAllPoolsSecure returns all pools with pagination and validation
func (k Keeper) GetAllPoolsSecure(ctx context.Context, limit, offset uint64) ([]types.Pool, error) {
	var validPools []types.Pool
	count := uint64(0)
	skipped := uint64(0)

	err := k.IteratePools(ctx, func(pool types.Pool) bool {
		// Skip invalid pools
		if err := k.ValidatePoolState(&pool); err != nil {
			return false
		}

		// Apply offset
		if skipped < offset {
			skipped++
			return false
		}

		// Apply limit
		if limit > 0 && count >= limit {
			return true
		}

		validPools = append(validPools, pool)
		count++
		return false
	})

	if err != nil {
		return nil, err
	}

	return validPools, nil
}

// DeletePool removes a pool (governance only - emergency use)
// This function requires governance authority and can only delete empty pools.
func (k Keeper) DeletePool(ctx context.Context, poolID uint64, authority string) error {
	// CRITICAL: Governance-only authorization check
	// Only the governance module can delete pools, even empty ones
	if authority != k.authority {
		return types.ErrUnauthorized.Wrapf(
			"invalid authority; expected %s, got %s", k.authority, authority)
	}

	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	// Verify pool is empty (safety check)
	if !pool.ReserveA.IsZero() || !pool.ReserveB.IsZero() || !pool.TotalShares.IsZero() {
		return types.ErrInvalidPoolState.Wrap("cannot delete pool with active liquidity")
	}

	store := k.getStore(ctx)

	// Delete pool
	store.Delete(PoolKey(poolID))

	// Delete token index
	store.Delete(PoolByTokensKey(pool.TokenA, pool.TokenB))

	// Delete circuit breaker state
	store.Delete(CircuitBreakerKey(poolID))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"pool_deleted",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("authority", authority),
		),
	)

	return nil
}
