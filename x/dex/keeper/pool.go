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

// CreatePool creates a new liquidity pool
func (k Keeper) CreatePool(ctx context.Context, creator sdk.AccAddress, tokenA, tokenB string, amountA, amountB math.Int) (*types.Pool, error) {
	// Validate inputs
	if tokenA == tokenB {
		return nil, types.ErrInvalidTokenPair.Wrap("cannot create pool with identical tokens")
	}
	if amountA.IsZero() || amountB.IsZero() {
		return nil, types.ErrInvalidLiquidityAmount.Wrap("initial liquidity amounts must be positive")
	}

	// Ensure consistent token ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
		amountA, amountB = amountB, amountA
	}

	// Check if pool already exists
	existingPool, err := k.GetPoolByTokens(ctx, tokenA, tokenB)
	if err == nil && existingPool != nil {
		return nil, types.ErrPoolAlreadyExists.Wrapf("pool already exists for token pair %s/%s", tokenA, tokenB)
	}

	// Check minimum liquidity requirement
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate initial shares (geometric mean)
	totalShares := amountA.Mul(amountB)
	sqrtShares, _ := math.LegacyNewDecFromInt(totalShares).ApproxSqrt()
	initialShares := sqrtShares.TruncateInt()

	if initialShares.LT(params.MinLiquidity) {
		return nil, types.ErrInvalidLiquidityAmount.Wrapf("initial liquidity too low: %s < %s", initialShares, params.MinLiquidity)
	}

	// Get next pool ID
	poolID, err := k.GetNextPoolID(ctx)
	if err != nil {
		return nil, err
	}

	// Create pool
	pool := &types.Pool{
		Id:          poolID,
		TokenA:      tokenA,
		TokenB:      tokenB,
		ReserveA:    amountA,
		ReserveB:    amountB,
		TotalShares: initialShares,
		Creator:     creator.String(),
	}

	// Save pool to store
	if err := k.SetPool(ctx, pool); err != nil {
		return nil, err
	}

	// Index pool by tokens
	if err := k.SetPoolByTokens(ctx, tokenA, tokenB, poolID); err != nil {
		return nil, err
	}

	// Set initial liquidity position for creator
	if err := k.SetLiquidity(ctx, poolID, creator, initialShares); err != nil {
		return nil, err
	}

	// Transfer tokens from creator to module
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress()

	coinA := sdk.NewCoin(tokenA, amountA)
	coinB := sdk.NewCoin(tokenB, amountB)

	if err := k.bankKeeper.SendCoins(sdkCtx, creator, moduleAddr, sdk.NewCoins(coinA, coinB)); err != nil {
		return nil, types.ErrInsufficientLiquidity.Wrapf("failed to transfer tokens: %v", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexPoolCreated,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyTokenA, tokenA),
			sdk.NewAttribute(types.AttributeKeyTokenB, tokenB),
			sdk.NewAttribute(types.AttributeKeyAmountA, amountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, amountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, initialShares.String()),
		),
	)

	return pool, nil
}

// GetPool retrieves a pool by ID
func (k Keeper) GetPool(ctx context.Context, poolID uint64) (*types.Pool, error) {
	store := k.getStore(ctx)
	bz := store.Get(PoolKey(poolID))
	if bz == nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
	}

	var pool types.Pool
	if err := k.cdc.Unmarshal(bz, &pool); err != nil {
		return nil, err
	}
	return &pool, nil
}

// SetPool saves a pool to the store
func (k Keeper) SetPool(ctx context.Context, pool *types.Pool) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(pool)
	if err != nil {
		return err
	}
	store.Set(PoolKey(pool.Id), bz)
	return nil
}

// GetPoolByTokens retrieves a pool by its token pair
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

// IteratePools iterates over all pools
func (k Keeper) IteratePools(ctx context.Context, cb func(pool types.Pool) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PoolKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var pool types.Pool
		if err := k.cdc.Unmarshal(iterator.Value(), &pool); err != nil {
			return err
		}
		if cb(pool) {
			break
		}
	}
	return nil
}

// GetAllPools returns all pools
func (k Keeper) GetAllPools(ctx context.Context) ([]types.Pool, error) {
	var pools []types.Pool
	err := k.IteratePools(ctx, func(pool types.Pool) bool {
		pools = append(pools, pool)
		return false
	})
	return pools, err
}

// GetModuleAddress returns the module account address
func (k Keeper) GetModuleAddress() sdk.AccAddress {
	return sdk.AccAddress([]byte(types.ModuleName))
}
