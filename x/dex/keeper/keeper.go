package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Keeper maintains the state of the DEX module
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
}

// NewKeeper creates a new DEX Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// CreatePool creates a new liquidity pool
func (k Keeper) CreatePool(
	ctx sdk.Context,
	creator string,
	tokenA string,
	tokenB string,
	amountA sdk.Int,
	amountB sdk.Int,
) (uint64, error) {
	// Validate inputs
	if tokenA == tokenB {
		return 0, types.ErrSameToken
	}

	if amountA.LTE(sdk.ZeroInt()) || amountB.LTE(sdk.ZeroInt()) {
		return 0, types.ErrInvalidAmount
	}

	// Ensure consistent token ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
		amountA, amountB = amountB, amountA
	}

	// Check if pool already exists
	if k.GetPoolByTokens(ctx, tokenA, tokenB) != nil {
		return 0, types.ErrPoolAlreadyExists
	}

	// Get next pool ID
	poolId := k.GetNextPoolId(ctx)

	// Transfer tokens from creator to module account
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return 0, err
	}

	moduleAddr := k.GetModuleAddress()
	coinsToTransfer := sdk.NewCoins(
		sdk.NewCoin(tokenA, amountA),
		sdk.NewCoin(tokenB, amountB),
	)

	if err := k.bankKeeper.SendCoins(ctx, creatorAddr, moduleAddr, coinsToTransfer); err != nil {
		return 0, err
	}

	// Create pool
	pool := types.NewPool(poolId, tokenA, tokenB, amountA, amountB, creator)

	// Store pool
	k.SetPool(ctx, pool)
	k.SetPoolByTokens(ctx, tokenA, tokenB, poolId)

	// Give initial liquidity shares to creator
	k.SetLiquidity(ctx, poolId, creator, pool.TotalShares)

	// Increment pool count
	k.SetNextPoolId(ctx, poolId+1)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"create_pool",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("creator", creator),
			sdk.NewAttribute("token_a", tokenA),
			sdk.NewAttribute("token_b", tokenB),
			sdk.NewAttribute("amount_a", amountA.String()),
			sdk.NewAttribute("amount_b", amountB.String()),
		),
	)

	return poolId, nil
}

// Swap executes a token swap using the constant product AMM formula
func (k Keeper) Swap(
	ctx sdk.Context,
	trader string,
	poolId uint64,
	tokenIn string,
	tokenOut string,
	amountIn sdk.Int,
	minAmountOut sdk.Int,
) (sdk.Int, error) {
	// Get pool
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return sdk.ZeroInt(), types.ErrPoolNotFound
	}

	// Validate tokens
	if tokenIn != pool.TokenA && tokenIn != pool.TokenB {
		return sdk.ZeroInt(), types.ErrInvalidTokenDenom
	}
	if tokenOut != pool.TokenA && tokenOut != pool.TokenB {
		return sdk.ZeroInt(), types.ErrInvalidTokenDenom
	}
	if tokenIn == tokenOut {
		return sdk.ZeroInt(), types.ErrSameToken
	}

	// Determine reserves
	var reserveIn, reserveOut sdk.Int
	if tokenIn == pool.TokenA {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	}

	// Calculate output amount using AMM formula with 0.3% fee
	amountOut := k.CalculateSwapAmount(reserveIn, reserveOut, amountIn)

	// Check minimum output
	if amountOut.LT(minAmountOut) {
		return sdk.ZeroInt(), types.ErrMinAmountOut
	}

	// Transfer tokens
	traderAddr, err := sdk.AccAddressFromBech32(trader)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	moduleAddr := k.GetModuleAddress()

	// Transfer input tokens from trader to module
	coinIn := sdk.NewCoins(sdk.NewCoin(tokenIn, amountIn))
	if err := k.bankKeeper.SendCoins(ctx, traderAddr, moduleAddr, coinIn); err != nil {
		return sdk.ZeroInt(), err
	}

	// Transfer output tokens from module to trader
	coinOut := sdk.NewCoins(sdk.NewCoin(tokenOut, amountOut))
	if err := k.bankKeeper.SendCoins(ctx, moduleAddr, traderAddr, coinOut); err != nil {
		return sdk.ZeroInt(), err
	}

	// Update pool reserves
	if tokenIn == pool.TokenA {
		pool.ReserveA = pool.ReserveA.Add(amountIn)
		pool.ReserveB = pool.ReserveB.Sub(amountOut)
	} else {
		pool.ReserveB = pool.ReserveB.Add(amountIn)
		pool.ReserveA = pool.ReserveA.Sub(amountOut)
	}

	k.SetPool(ctx, *pool)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("trader", trader),
			sdk.NewAttribute("token_in", tokenIn),
			sdk.NewAttribute("token_out", tokenOut),
			sdk.NewAttribute("amount_in", amountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
		),
	)

	return amountOut, nil
}

// CalculateSwapAmount calculates the output amount using constant product AMM formula
// Formula: amountOut = (amountIn * 997 * reserveOut) / (reserveIn * 1000 + amountIn * 997)
// This implements a 0.3% fee (997/1000 = 0.997)
func (k Keeper) CalculateSwapAmount(reserveIn, reserveOut, amountIn sdk.Int) sdk.Int {
	// amountInWithFee = amountIn * 997
	amountInWithFee := amountIn.Mul(math.NewInt(997))

	// numerator = amountInWithFee * reserveOut
	numerator := amountInWithFee.Mul(reserveOut)

	// denominator = (reserveIn * 1000) + amountInWithFee
	denominator := reserveIn.Mul(math.NewInt(1000)).Add(amountInWithFee)

	// amountOut = numerator / denominator
	amountOut := numerator.Quo(denominator)

	return amountOut
}

// AddLiquidity adds liquidity to an existing pool
func (k Keeper) AddLiquidity(
	ctx sdk.Context,
	provider string,
	poolId uint64,
	amountA sdk.Int,
	amountB sdk.Int,
) (sdk.Int, error) {
	// Get pool
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return sdk.ZeroInt(), types.ErrPoolNotFound
	}

	// Calculate shares to mint based on pool ratio
	// shares = min(amountA / reserveA, amountB / reserveB) * totalShares
	ratioA := amountA.Mul(pool.TotalShares).Quo(pool.ReserveA)
	ratioB := amountB.Mul(pool.TotalShares).Quo(pool.ReserveB)

	var sharesToMint sdk.Int
	if ratioA.LT(ratioB) {
		sharesToMint = ratioA
	} else {
		sharesToMint = ratioB
	}

	if sharesToMint.LTE(sdk.ZeroInt()) {
		return sdk.ZeroInt(), types.ErrInvalidAmount
	}

	// Transfer tokens from provider to module
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	moduleAddr := k.GetModuleAddress()
	coinsToTransfer := sdk.NewCoins(
		sdk.NewCoin(pool.TokenA, amountA),
		sdk.NewCoin(pool.TokenB, amountB),
	)

	if err := k.bankKeeper.SendCoins(ctx, providerAddr, moduleAddr, coinsToTransfer); err != nil {
		return sdk.ZeroInt(), err
	}

	// Update pool reserves and total shares
	pool.ReserveA = pool.ReserveA.Add(amountA)
	pool.ReserveB = pool.ReserveB.Add(amountB)
	pool.TotalShares = pool.TotalShares.Add(sharesToMint)

	k.SetPool(ctx, *pool)

	// Update provider's liquidity shares
	currentShares := k.GetLiquidity(ctx, poolId, provider)
	k.SetLiquidity(ctx, poolId, provider, currentShares.Add(sharesToMint))

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"add_liquidity",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("provider", provider),
			sdk.NewAttribute("amount_a", amountA.String()),
			sdk.NewAttribute("amount_b", amountB.String()),
			sdk.NewAttribute("shares", sharesToMint.String()),
		),
	)

	return sharesToMint, nil
}

// RemoveLiquidity removes liquidity from a pool
func (k Keeper) RemoveLiquidity(
	ctx sdk.Context,
	provider string,
	poolId uint64,
	shares sdk.Int,
) (sdk.Int, sdk.Int, error) {
	// Get pool
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return sdk.ZeroInt(), sdk.ZeroInt(), types.ErrPoolNotFound
	}

	// Check provider has enough shares
	providerShares := k.GetLiquidity(ctx, poolId, provider)
	if providerShares.LT(shares) {
		return sdk.ZeroInt(), sdk.ZeroInt(), types.ErrInsufficientShares
	}

	// Calculate amounts to return
	// amountA = shares * reserveA / totalShares
	// amountB = shares * reserveB / totalShares
	amountA := shares.Mul(pool.ReserveA).Quo(pool.TotalShares)
	amountB := shares.Mul(pool.ReserveB).Quo(pool.TotalShares)

	// Transfer tokens from module to provider
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return sdk.ZeroInt(), sdk.ZeroInt(), err
	}

	moduleAddr := k.GetModuleAddress()
	coinsToTransfer := sdk.NewCoins(
		sdk.NewCoin(pool.TokenA, amountA),
		sdk.NewCoin(pool.TokenB, amountB),
	)

	if err := k.bankKeeper.SendCoins(ctx, moduleAddr, providerAddr, coinsToTransfer); err != nil {
		return sdk.ZeroInt(), sdk.ZeroInt(), err
	}

	// Update pool reserves and total shares
	pool.ReserveA = pool.ReserveA.Sub(amountA)
	pool.ReserveB = pool.ReserveB.Sub(amountB)
	pool.TotalShares = pool.TotalShares.Sub(shares)

	k.SetPool(ctx, *pool)

	// Update provider's liquidity shares
	k.SetLiquidity(ctx, poolId, provider, providerShares.Sub(shares))

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"remove_liquidity",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("provider", provider),
			sdk.NewAttribute("amount_a", amountA.String()),
			sdk.NewAttribute("amount_b", amountB.String()),
			sdk.NewAttribute("shares", shares.String()),
		),
	)

	return amountA, amountB, nil
}

// GetPool retrieves a pool by ID
func (k Keeper) GetPool(ctx sdk.Context, poolId uint64) *types.Pool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PoolKey)
	bz := store.Get(types.GetPoolKey(poolId))
	if bz == nil {
		return nil
	}

	var pool types.Pool
	k.cdc.MustUnmarshal(bz, &pool)
	return &pool
}

// SetPool stores a pool
func (k Keeper) SetPool(ctx sdk.Context, pool types.Pool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PoolKey)
	bz := k.cdc.MustMarshal(&pool)
	store.Set(types.GetPoolKey(pool.Id), bz)
}

// GetPoolByTokens retrieves a pool by token pair
func (k Keeper) GetPoolByTokens(ctx sdk.Context, tokenA, tokenB string) *types.Pool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetPoolByTokensKey(tokenA, tokenB))
	if bz == nil {
		return nil
	}

	poolId := sdk.BigEndianToUint64(bz)
	return k.GetPool(ctx, poolId)
}

// SetPoolByTokens stores the pool ID for a token pair
func (k Keeper) SetPoolByTokens(ctx sdk.Context, tokenA, tokenB string, poolId uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetPoolByTokensKey(tokenA, tokenB), sdk.Uint64ToBigEndian(poolId))
}

// GetNextPoolId gets the next pool ID
func (k Keeper) GetNextPoolId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PoolCountKey)
	if bz == nil {
		return 1
	}
	return sdk.BigEndianToUint64(bz)
}

// SetNextPoolId sets the next pool ID
func (k Keeper) SetNextPoolId(ctx sdk.Context, poolId uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PoolCountKey, sdk.Uint64ToBigEndian(poolId))
}

// GetLiquidity gets a provider's liquidity shares
func (k Keeper) GetLiquidity(ctx sdk.Context, poolId uint64, provider string) sdk.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetLiquidityKey(poolId, provider))
	if bz == nil {
		return sdk.ZeroInt()
	}

	var shares sdk.Int
	k.cdc.MustUnmarshal(bz, &shares)
	return shares
}

// SetLiquidity sets a provider's liquidity shares
func (k Keeper) SetLiquidity(ctx sdk.Context, poolId uint64, provider string, shares sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&shares)
	store.Set(types.GetLiquidityKey(poolId, provider), bz)
}

// GetModuleAddress returns the module account address
func (k Keeper) GetModuleAddress() sdk.AccAddress {
	return sdk.AccAddress([]byte(types.ModuleName))
}
