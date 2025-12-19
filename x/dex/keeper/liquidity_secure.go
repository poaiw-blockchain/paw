package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// AddLiquiditySecure adds liquidity with comprehensive security checks
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
