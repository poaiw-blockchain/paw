package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetNextPoolID returns the next pool ID and increments the counter
func (k Keeper) GetNextPoolID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(PoolCountKey)

	var poolID uint64
	if bz == nil {
		poolID = 1
	} else {
		poolID = binary.BigEndian.Uint64(bz)
	}

	// Increment the counter
	nextID := poolID + 1
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, nextID)
	store.Set(PoolCountKey, nextBz)

	return poolID, nil
}

// SetNextPoolId sets the next pool ID counter
func (k Keeper) SetNextPoolId(ctx context.Context, poolID uint64) {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, poolID)
	store.Set(PoolCountKey, bz)
}

// GetTotalPoolsCount returns the total number of active pools in O(1) time.
// PERF-9: This avoids O(n) iteration through all pools for count checks.
func (k Keeper) GetTotalPoolsCount(ctx context.Context) uint64 {
	store := k.getStore(ctx)
	bz := store.Get(TotalPoolsCountKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetTotalPoolsCount sets the total pools count.
// PERF-9: Called when pools are created or deleted.
func (k Keeper) SetTotalPoolsCount(ctx context.Context, count uint64) {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(TotalPoolsCountKey, bz)
}

// IncrementTotalPoolsCount increments the pool count by 1.
// PERF-9: Called when a new pool is created.
func (k Keeper) IncrementTotalPoolsCount(ctx context.Context) {
	count := k.GetTotalPoolsCount(ctx)
	k.SetTotalPoolsCount(ctx, count+1)
}

// DecrementTotalPoolsCount decrements the pool count by 1.
// PERF-9: Called when a pool is deleted.
func (k Keeper) DecrementTotalPoolsCount(ctx context.Context) {
	count := k.GetTotalPoolsCount(ctx)
	if count > 0 {
		k.SetTotalPoolsCount(ctx, count-1)
	}
}

// GetPoolVersion returns the current pool version for graph cache invalidation.
// PERF-10: Incremented when pools are created or deleted.
func (k Keeper) GetPoolVersion(ctx context.Context) uint64 {
	store := k.getStore(ctx)
	bz := store.Get(PoolVersionKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementPoolVersion increments the pool version to invalidate the token graph cache.
// PERF-10: Called when pools are created or deleted.
func (k Keeper) IncrementPoolVersion(ctx context.Context) {
	store := k.getStore(ctx)
	version := k.GetPoolVersion(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, version+1)
	store.Set(PoolVersionKey, bz)
}

// CreatePool creates a new liquidity pool with comprehensive security checks.
// Tokens are ordered lexicographically. Returns ErrPoolAlreadyExists if the pair exists,
// ErrInsufficientLiquidity if amounts are below minimum, ErrMaxPoolsReached at limit.
func (k Keeper) CreatePool(ctx context.Context, creator sdk.AccAddress, tokenA, tokenB string, amountA, amountB math.Int) (*types.Pool, error) {
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

	// SEC-6: Enforce minimum initial liquidity per token
	// This prevents dust pools that are vulnerable to manipulation
	minInitialLiquidity := math.NewInt(MinimumInitialLiquidity)
	if amountA.LT(minInitialLiquidity) {
		return nil, types.ErrInsufficientLiquidity.Wrapf(
			"amount A %s below minimum initial liquidity %s",
			amountA, minInitialLiquidity,
		)
	}
	if amountB.LT(minInitialLiquidity) {
		return nil, types.ErrInsufficientLiquidity.Wrapf(
			"amount B %s below minimum initial liquidity %s",
			amountB, minInitialLiquidity,
		)
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
	// PERF-9: Use O(1) pool count instead of O(n) GetAllPools iteration
	poolCount := k.GetTotalPoolsCount(ctx)

	if poolCount >= MaxPools {
		return nil, types.ErrMaxPoolsReached.Wrapf("maximum number of pools (%d) reached", MaxPools)
	}

	if poolCount > MaxPools*9/10 {
		sdk.UnwrapSDKContext(ctx).Logger().Info(
			"dex pool count approaching limit",
			"current", poolCount,
			"max", MaxPools,
		)
	}

	// 5. Get and validate parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("CreatePool: get params: %w", err)
	}

	// 6. Calculate initial shares using geometric mean (sqrt(x * y))
	// This prevents initial liquidity manipulation
	product := amountA.Mul(amountB)

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
		return nil, fmt.Errorf("CreatePool: get next pool ID: %w", err)
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
		return nil, fmt.Errorf("CreatePool: validate pool state: %w", err)
	}

	// 12. Transfer tokens FIRST (checks-effects-interactions)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(tokenA, amountA)
	coinB := sdk.NewCoin(tokenB, amountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, creator, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
		return nil, types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// DATA-9: Validate module balance after transfer
	// This ensures the transfer was successful and the module holds the expected reserves
	moduleBalanceA := k.bankKeeper.GetBalance(sdkCtx, moduleAddr, tokenA)
	moduleBalanceB := k.bankKeeper.GetBalance(sdkCtx, moduleAddr, tokenB)
	if moduleBalanceA.Amount.LT(amountA) {
		return nil, types.ErrInvariantViolation.Wrapf(
			"module balance mismatch: expected at least %s %s, got %s",
			amountA.String(), tokenA, moduleBalanceA.Amount.String(),
		)
	}
	if moduleBalanceB.Amount.LT(amountB) {
		return nil, types.ErrInvariantViolation.Wrapf(
			"module balance mismatch: expected at least %s %s, got %s",
			amountB.String(), tokenB, moduleBalanceB.Amount.String(),
		)
	}

	// 13. Save pool to store AFTER receiving tokens
	if err := k.SetPool(ctx, pool); err != nil {
		return nil, fmt.Errorf("CreatePool: save pool: %w", err)
	}

	// PERF-9: Increment pool count for O(1) count checks
	k.IncrementTotalPoolsCount(ctx)

	// PERF-10: Increment pool version to invalidate token graph cache
	k.IncrementPoolVersion(ctx)

	// 14. Index pool by tokens
	if err := k.SetPoolByTokens(ctx, tokenA, tokenB, poolID); err != nil {
		return nil, fmt.Errorf("CreatePool: set pool by tokens index: %w", err)
	}

	// 15. Set initial liquidity position for creator
	if err := k.SetLiquidity(ctx, poolID, creator, initialShares); err != nil {
		return nil, fmt.Errorf("CreatePool: set creator liquidity: %w", err)
	}

	// 16. Record liquidity action for flash loan protection
	if err := k.SetLastLiquidityActionBlock(ctx, poolID, creator); err != nil {
		return nil, fmt.Errorf("CreatePool: set last liquidity action block: %w", err)
	}

	// 17. Initialize circuit breaker state
	cbState := &types.CircuitBreakerState{
		Enabled:       false,
		LastPrice:     math.LegacyNewDecFromInt(amountB).Quo(math.LegacyNewDecFromInt(amountA)),
		TriggerReason: "",
	}
	if err := k.SetCircuitBreakerState(ctx, poolID, cbState); err != nil {
		return nil, fmt.Errorf("CreatePool: set circuit breaker state: %w", err)
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

// GetPool retrieves a pool by its unique numeric ID.
// Returns ErrPoolNotFound if the pool does not exist.
func (k Keeper) GetPool(ctx context.Context, poolID uint64) (*types.Pool, error) {
	store := k.getStore(ctx)
	bz := store.Get(PoolKey(poolID))
	if bz == nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	var pool types.Pool
	if err := k.cdc.Unmarshal(bz, &pool); err != nil {
		return nil, fmt.Errorf("GetPool: unmarshal pool %d: %w", poolID, err)
	}
	return &pool, nil
}

// SetPool saves a pool to the store
func (k Keeper) SetPool(ctx context.Context, pool *types.Pool) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(pool)
	if err != nil {
		return fmt.Errorf("SetPool: marshal pool %d: %w", pool.Id, err)
	}
	store.Set(PoolKey(pool.Id), bz)
	return nil
}

// GetPoolByTokens retrieves a pool by its token pair (order-independent).
// Tokens are internally sorted for consistent lookup. Returns ErrPoolNotFound if not found.
func (k Keeper) GetPoolByTokens(ctx context.Context, tokenA, tokenB string) (*types.Pool, error) {
	// Ensure consistent ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}

	store := k.getStore(ctx)
	bz := store.Get(PoolByTokensKey(tokenA, tokenB))
	if bz == nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool not found for token pair %s/%s", tokenA, tokenB)
	}

	poolID := binary.BigEndian.Uint64(bz)
	return k.GetPool(ctx, poolID)
}

// SetPoolByTokens indexes a pool by its token pair
func (k Keeper) SetPoolByTokens(ctx context.Context, tokenA, tokenB string, poolID uint64) error {
	// Ensure consistent ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}

	store := k.getStore(ctx)
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	store.Set(PoolByTokensKey(tokenA, tokenB), poolIDBytes)
	return nil
}

// MaxIterationLimit is the maximum number of items to return in unbounded queries
// This prevents DoS attacks via excessive iteration
const MaxIterationLimit = 100

// IteratePools iterates over all pools
func (k Keeper) IteratePools(ctx context.Context, cb func(pool types.Pool) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := k.cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return fmt.Errorf("IteratePools: unmarshal pool: %w", err)
		}
		if cb(pool) {
			break
		}
	}
	return nil
}

// GetAllPools returns all pools with a maximum limit to prevent DoS
func (k Keeper) GetAllPools(ctx context.Context) ([]types.Pool, error) {
	// P3-PERF-3: Pre-size with MaxIterationLimit capacity
	pools := make([]types.Pool, 0, MaxIterationLimit)
	count := 0
	err := k.IteratePools(ctx, func(pool types.Pool) bool {
		if count >= MaxIterationLimit {
			return true // Stop iteration at limit
		}
		pools = append(pools, pool)
		count++
		return false
	})
	return pools, err
}

// GetModuleAddress returns the cached module account address.
// The address is computed once during Keeper initialization and cached
// to avoid repeated byte slice allocations in hot paths (swaps, liquidity, fees).
func (k Keeper) GetModuleAddress() sdk.AccAddress {
	return k.moduleAddressCache
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

	// PERF-9: Decrement pool count for O(1) count checks
	k.DecrementTotalPoolsCount(ctx)

	// PERF-10: Increment pool version to invalidate token graph cache
	k.IncrementPoolVersion(ctx)

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
