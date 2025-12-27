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

// AddLiquidity adds liquidity with comprehensive security checks
// CODE-LOW-4 DOCUMENTATION: Reentrancy Guard Implementation
//
// REENTRANCY ATTACK EXPLANATION:
// A reentrancy attack occurs when an attacker calls back into a contract/module
// during its execution, before the first invocation has completed. This can allow
// an attacker to:
// 1. Drain funds by repeatedly withdrawing before balance updates
// 2. Manipulate state by executing operations in unexpected order
// 3. Bypass security checks that assume single execution path
//
// Classic example: The DAO hack (Ethereum, 2016) - $60M stolen via reentrancy
//
// HOW THE GUARD WORKS:
// 1. Before executing the operation, we acquire a lock specific to this pool+operation
// 2. The lock is stored in the KVStore (persistent) and optionally in-memory (tests)
// 3. If the lock already exists, we reject the operation with ErrReentrancy
// 4. After operation completes (or fails), we release the lock via defer
// 5. This ensures ONLY ONE instance of an operation can execute at a time per pool
//
// ATTACKS PREVENTED:
// - Flash loan attacks: Attacker cannot add liquidity, execute swap, remove liquidity atomically
// - State manipulation: Attacker cannot call AddLiquidity recursively to bypass checks
// - Race conditions: Prevents concurrent modifications to same pool state
// - Callback attacks: External contract calls cannot re-enter DEX functions
//
// IMPLEMENTATION DETAILS:
// - Lock key format: "{poolID}:{operation}" (e.g., "42:add_liquidity")
// - Lock storage: KVStore ensures persistence across context boundaries
// - Lock cleanup: defer ensures release even on panic/error
// - Optional in-memory guard: Used in tests for additional verification
//
// COSMOS SDK SECURITY NOTE:
// Unlike Ethereum smart contracts, Cosmos SDK modules don't have external contract calls
// in the traditional sense. However, reentrancy can still occur via:
// - Module-to-module calls (e.g., DEX calling Bank, which calls hooks)
// - Message server handlers calling keeper methods
// - Ante handlers or post-handlers triggering operations
//
// This guard provides defense-in-depth even though the attack surface is smaller
// than Ethereum. Production DeFi protocols prioritize security over minimal code.
func (k Keeper) AddLiquidity(ctx context.Context, provider sdk.AccAddress, poolID uint64, amountA, amountB math.Int) (math.Int, error) {
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
	if err := k.checkPoolPriceDeviation(ctx, pool, "add_liquidity"); err != nil {
		return math.ZeroInt(), err
	}

	// 4. Calculate optimal amounts and shares (proportional liquidity)
	var finalAmountA, finalAmountB, newShares math.Int

	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("cannot add liquidity to empty pool")
	}

	// Calculate optimal amounts to maintain price ratio
	// math.Int.Mul is safe from overflow (uses big.Int internally)
	optimalAmountB := amountA.Mul(pool.ReserveB)

	// Check for division by zero before Quo
	if pool.ReserveA.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool reserve A is zero")
	}
	optimalAmountB = optimalAmountB.Quo(pool.ReserveA)

	// math.Int.Mul is safe from overflow (uses big.Int internally)
	optimalAmountA := amountB.Mul(pool.ReserveA)

	// Check for division by zero before Quo
	if pool.ReserveB.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool reserve B is zero")
	}
	optimalAmountA = optimalAmountA.Quo(pool.ReserveB)

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
	// math.Int.Mul is safe from overflow (uses big.Int internally)
	sharesA := finalAmountA.Mul(pool.TotalShares)

	// Check for division by zero before Quo (already validated above, but explicit check)
	if pool.ReserveA.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool reserve A is zero")
	}
	sharesA = sharesA.Quo(pool.ReserveA)

	// math.Int.Mul is safe from overflow (uses big.Int internally)
	sharesB := finalAmountB.Mul(pool.TotalShares)

	// Check for division by zero before Quo (already validated above, but explicit check)
	if pool.ReserveB.IsZero() {
		return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool reserve B is zero")
	}
	sharesB = sharesB.Quo(pool.ReserveB)

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
	// math.Int.Add is safe from overflow (uses big.Int internally)
	pool.ReserveA = pool.ReserveA.Add(finalAmountA)
	pool.ReserveB = pool.ReserveB.Add(finalAmountB)
	pool.TotalShares = pool.TotalShares.Add(newShares)

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

	// math.Int.Add is safe from overflow (uses big.Int internally)
	newTotalShares := currentShares.Add(newShares)

	if err := k.SetLiquidity(ctx, poolID, provider, newTotalShares); err != nil {
		return math.ZeroInt(), err
	}

	// 13. Record liquidity action for flash loan protection
	if err := k.SetLastLiquidityActionBlock(ctx, poolID, provider); err != nil {
		return math.ZeroInt(), err
	}

	// 14. Mark pool as active for activity-based tracking
	if err := k.MarkPoolActive(ctx, poolID); err != nil {
		// Log error but don't fail the operation - activity tracking is non-critical
		sdkCtx.Logger().Error("failed to mark pool active", "pool_id", poolID, "error", err)
	}

	// 15. Emit event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDexAddLiquidity,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
			sdk.NewAttribute(types.AttributeKeyAmountA, finalAmountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, finalAmountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, newShares.String()),
			sdk.NewAttribute(types.AttributeKeyTotalShares, pool.TotalShares.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, provider.String()),
		),
	})

	return newShares, nil
}

// RemoveLiquidity removes liquidity with comprehensive security checks
func (k Keeper) RemoveLiquidity(ctx context.Context, provider sdk.AccAddress, poolID uint64, shares math.Int) (math.Int, math.Int, error) {
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
	if err := k.checkPoolPriceDeviation(ctx, pool, "remove_liquidity"); err != nil {
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
	// math.Int.Mul is safe from overflow (uses big.Int internally)
	amountA := shares.Mul(pool.ReserveA)

	// Check for division by zero before Quo
	if pool.TotalShares.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool total shares is zero")
	}
	amountA = amountA.Quo(pool.TotalShares)

	// math.Int.Mul is safe from overflow (uses big.Int internally)
	amountB := shares.Mul(pool.ReserveB)

	// Check for division by zero before Quo
	if pool.TotalShares.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool total shares is zero")
	}
	amountB = amountB.Quo(pool.TotalShares)

	if amountA.IsZero() || amountB.IsZero() {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("withdrawal amounts too small")
	}

	// SEC-6: Enforce minimum reserves - pools cannot be fully drained
	// This is critical to prevent:
	// 1. Price manipulation via dust amounts
	// 2. Flash loan attacks that drain pools
	// 3. Griefing attacks that leave pools unusable
	remainingReserveA := pool.ReserveA.Sub(amountA)
	remainingReserveB := pool.ReserveB.Sub(amountB)
	minReserves := math.NewInt(MinimumReserves)

	// Always enforce minimum reserves - full draining is NOT allowed
	// Pools must maintain at least MinimumReserves (1000 base units) in each token
	if remainingReserveA.LT(minReserves) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrMinimumReserves.Wrapf(
			"withdrawal would leave reserve A at %s, minimum required is %s",
			remainingReserveA, minReserves,
		)
	}
	if remainingReserveB.LT(minReserves) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrMinimumReserves.Wrapf(
			"withdrawal would leave reserve B at %s, minimum required is %s",
			remainingReserveB, minReserves,
		)
	}

	// 7. Update pool state BEFORE transfers (checks-effects-interactions)
	// Check for underflow before subtraction
	if pool.ReserveA.LT(amountA) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf(
			"insufficient reserve A: have %s, need %s",
			pool.ReserveA, amountA,
		)
	}
	pool.ReserveA = pool.ReserveA.Sub(amountA)

	// Check for underflow before subtraction
	if pool.ReserveB.LT(amountB) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientLiquidity.Wrapf(
			"insufficient reserve B: have %s, need %s",
			pool.ReserveB, amountB,
		)
	}
	pool.ReserveB = pool.ReserveB.Sub(amountB)

	// Check for underflow before subtraction
	if pool.TotalShares.LT(shares) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientShares.Wrapf(
			"insufficient total shares: have %s, need %s",
			pool.TotalShares, shares,
		)
	}
	pool.TotalShares = pool.TotalShares.Sub(shares)

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
	// Check for underflow before subtraction (already validated above in step 5, but explicit check)
	if userShares.LT(shares) {
		return math.ZeroInt(), math.ZeroInt(), types.ErrInsufficientShares.Wrapf(
			"insufficient user shares: have %s, need %s",
			userShares, shares,
		)
	}
	newUserShares := userShares.Sub(shares)

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

	// 14. Mark pool as active for activity-based tracking
	if err := k.MarkPoolActive(ctx, poolID); err != nil {
		// Log error but don't fail the operation - activity tracking is non-critical
		sdkCtx.Logger().Error("failed to mark pool active", "pool_id", poolID, "error", err)
	}

	// 15. Emit event
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDexRemoveLiquidity,
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyProvider, provider.String()),
			sdk.NewAttribute(types.AttributeKeyAmountA, amountA.String()),
			sdk.NewAttribute(types.AttributeKeyAmountB, amountB.String()),
			sdk.NewAttribute(types.AttributeKeyShares, shares.String()),
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

// DefaultMaxLiquidityIterations is the maximum number of liquidity positions to iterate
// in a single call to prevent unbounded iteration. This can be overridden using
// IterateLiquidityByPoolPaginated for queries requiring pagination.
const DefaultMaxLiquidityIterations = 10000

// IterateLiquidityByPool iterates over all liquidity positions in a pool with a safety limit.
// PERF-3: Added maximum iteration limit to prevent unbounded iteration that could cause timeout.
// For genesis export and other operations requiring all records, the callback can return false
// to continue iteration, but the function will stop after DefaultMaxLiquidityIterations.
// Use IterateLiquidityByPoolPaginated for queries requiring explicit pagination.
func (k Keeper) IterateLiquidityByPool(ctx context.Context, poolID uint64, cb func(provider sdk.AccAddress, shares math.Int) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, LiquidityKeyByPoolPrefix(poolID))
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		// PERF-3: Safety limit to prevent unbounded iteration
		count++
		if count > DefaultMaxLiquidityIterations {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Warn("IterateLiquidityByPool reached maximum iteration limit",
				"pool_id", poolID,
				"limit", DefaultMaxLiquidityIterations,
			)
			break
		}

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

// LiquidityPosition represents a single liquidity provider's position in a pool.
type LiquidityPosition struct {
	Provider sdk.AccAddress
	Shares   math.Int
}

// PaginatedLiquidityResult contains the paginated result of liquidity positions.
type PaginatedLiquidityResult struct {
	Positions  []LiquidityPosition
	NextKey    []byte // nil if no more results
	TotalCount uint64 // total positions returned in this page
}

// IterateLiquidityByPoolPaginated returns a paginated list of liquidity positions for a pool.
// PERF-3: This function provides explicit pagination support with limit and startAfter parameters
// to prevent unbounded iteration and support efficient queries on pools with many providers.
//
// Parameters:
//   - poolID: The pool to query liquidity positions for
//   - startAfter: Provider address to start after (nil for first page)
//   - limit: Maximum number of positions to return (capped at DefaultMaxLiquidityIterations)
//
// Returns:
//   - PaginatedLiquidityResult with positions, next key for pagination, and count
func (k Keeper) IterateLiquidityByPoolPaginated(
	ctx context.Context,
	poolID uint64,
	startAfter sdk.AccAddress,
	limit uint64,
) (*PaginatedLiquidityResult, error) {
	store := k.getStore(ctx)

	// Cap limit to prevent excessive iteration
	if limit == 0 || limit > uint64(DefaultMaxLiquidityIterations) {
		limit = uint64(DefaultMaxLiquidityIterations)
	}

	prefix := LiquidityKeyByPoolPrefix(poolID)
	var iterator storetypes.Iterator

	if startAfter == nil {
		// Start from beginning
		iterator = storetypes.KVStorePrefixIterator(store, prefix)
	} else {
		// Start after the given key
		startKey := LiquidityKey(poolID, startAfter)
		// Add 1 byte to skip the exact match (start AFTER, not AT)
		startKey = append(startKey, 0x00)
		endKey := storetypes.PrefixEndBytes(prefix)
		iterator = store.Iterator(startKey, endKey)
	}
	defer iterator.Close()

	positions := make([]LiquidityPosition, 0, limit)
	var lastProvider sdk.AccAddress

	for count := uint64(0); iterator.Valid() && count < limit; iterator.Next() {
		var shares math.Int
		if err := shares.Unmarshal(iterator.Value()); err != nil {
			return nil, err
		}

		// Extract provider address from key
		key := iterator.Key()
		providerBytes := key[len(prefix):]
		provider := sdk.AccAddress(providerBytes)

		positions = append(positions, LiquidityPosition{
			Provider: provider,
			Shares:   shares,
		})
		lastProvider = provider
		count++
	}

	result := &PaginatedLiquidityResult{
		Positions:  positions,
		TotalCount: uint64(len(positions)),
	}

	// Check if there are more results
	if iterator.Valid() && lastProvider != nil {
		result.NextKey = lastProvider.Bytes()
	}

	return result, nil
}

// GetLiquidityProviderCount returns the total number of liquidity providers for a pool.
// PERF-3: This function iterates through all providers but only counts them,
// avoiding the overhead of unmarshaling values. Useful for estimating pagination needs.
func (k Keeper) GetLiquidityProviderCount(ctx context.Context, poolID uint64) (uint64, error) {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, LiquidityKeyByPoolPrefix(poolID))
	defer iterator.Close()

	count := uint64(0)
	for ; iterator.Valid(); iterator.Next() {
		count++
		// Safety limit: if we exceed this, we know there are "many" providers
		if count > uint64(DefaultMaxLiquidityIterations) {
			return count, nil
		}
	}
	return count, nil
}
