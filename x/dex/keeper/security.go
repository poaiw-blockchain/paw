package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// Security constants
const (
	// Maximum price deviation allowed before circuit breaker triggers (20%)
	MaxPriceDeviation = "0.2"

	// Maximum single swap size as percentage of pool reserves (10%)
	MaxSwapSizePercent = "0.1"

	// Minimum lock period for LP tokens to prevent flash loan attacks (1 block)
	MinLPLockBlocks = int64(1)

	// Maximum pools to prevent unbounded iteration
	MaxPools = uint64(1000)

	// Price update tolerance for invariant checks (0.1%)
	PriceUpdateTolerance = "0.001"
)

// CircuitBreakerState represents the circuit breaker status for a pool
type CircuitBreakerState struct {
	Enabled           bool
	PausedUntil       time.Time
	LastPrice         math.LegacyDec
	TriggeredBy       string
	TriggerReason     string
	NotificationsSent int
	LastNotification  time.Time
	PersistenceKey    string
}

// WithReentrancyGuard executes a function with reentrancy protection
// Stores locks in KVStore to ensure they persist across context boundaries
func (k Keeper) WithReentrancyGuard(ctx context.Context, poolID uint64, operation string, fn func() error) error {
	lockKey := fmt.Sprintf("%d:%s", poolID, operation)

	// Acquire lock using KVStore
	if err := k.acquireReentrancyLock(ctx, lockKey); err != nil {
		return err
	}

	// Ensure lock is released even if function panics
	defer k.releaseReentrancyLock(ctx, lockKey)

	return fn()
}

// acquireReentrancyLock attempts to acquire a reentrancy lock from the KVStore
func (k Keeper) acquireReentrancyLock(ctx context.Context, lockKey string) error {
	store := k.getStore(ctx)
	key := ReentrancyLockKey(lockKey)

	// Check if lock already exists
	if store.Has(key) {
		return types.ErrReentrancy.Wrapf("operation %s is already locked", lockKey)
	}

	// Acquire lock by setting a marker in the store
	store.Set(key, []byte{0x01})
	return nil
}

// releaseReentrancyLock releases a reentrancy lock from the KVStore
func (k Keeper) releaseReentrancyLock(ctx context.Context, lockKey string) {
	store := k.getStore(ctx)
	key := ReentrancyLockKey(lockKey)
	store.Delete(key)
}

// ValidatePoolInvariant checks the constant product invariant k = x * y
/*
func (k Keeper) ValidateSwapSize(ctx sdk.Context, pool types.Pool, amountIn math.Int) error {
    // ...
}

func (k Keeper) ValidatePriceImpact(ctx sdk.Context, pool types.Pool, amountIn math.Int, amountOut math.Int) error {
    // ...
}
*/
func (k Keeper) ValidatePoolInvariant(ctx context.Context, pool *types.Pool, oldK math.Int) error {
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return nil // Empty pools don't have invariant
	}

	// Calculate new k
	newK := pool.ReserveA.Mul(pool.ReserveB)

	// k should never decrease (can increase due to fees)
	if newK.LT(oldK) {
		return types.ErrInvariantViolation.Wrapf(
			"constant product invariant violated: old_k=%s, new_k=%s",
			oldK.String(), newK.String(),
		)
	}

	return nil
}

// ValidatePoolState performs comprehensive pool state validation
func (k Keeper) ValidatePoolState(pool *types.Pool) error {
	// Check reserves are non-negative
	if pool.ReserveA.IsNegative() {
		return types.ErrInvalidPoolState.Wrapf("negative reserve A: %s", pool.ReserveA)
	}
	if pool.ReserveB.IsNegative() {
		return types.ErrInvalidPoolState.Wrapf("negative reserve B: %s", pool.ReserveB)
	}

	// Check total shares are non-negative
	if pool.TotalShares.IsNegative() {
		return types.ErrInvalidPoolState.Wrapf("negative total shares: %s", pool.TotalShares)
	}

	// If pool has reserves, it must have shares
	if (!pool.ReserveA.IsZero() || !pool.ReserveB.IsZero()) && pool.TotalShares.IsZero() {
		return types.ErrInvalidPoolState.Wrap("pool has reserves but no shares")
	}

	// If pool has shares, it must have reserves
	if !pool.TotalShares.IsZero() && (pool.ReserveA.IsZero() || pool.ReserveB.IsZero()) {
		return types.ErrInvalidPoolState.Wrap("pool has shares but missing reserves")
	}

	return nil
}

// CheckCircuitBreaker checks if circuit breaker should be triggered
func (k Keeper) CheckCircuitBreaker(ctx context.Context, pool *types.Pool, operation string) error {
	state, err := k.GetCircuitBreakerState(ctx, pool.Id)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check if pool is currently paused
	if state.Enabled && sdkCtx.BlockTime().Before(state.PausedUntil) {
		return types.ErrCircuitBreakerTriggered.Wrapf(
			"pool %d paused until %s, reason: %s",
			pool.Id, state.PausedUntil, state.TriggerReason,
		)
	}

	// Calculate current price
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return nil
	}

	currentPrice := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))

	// Check for significant price deviation
	if !state.LastPrice.IsZero() {
		maxDeviation, err := math.LegacyNewDecFromStr(MaxPriceDeviation)
		if err != nil {
			return err
		}

		var deviation math.LegacyDec
		if currentPrice.GT(state.LastPrice) {
			deviation = currentPrice.Sub(state.LastPrice).Quo(state.LastPrice)
		} else {
			deviation = state.LastPrice.Sub(currentPrice).Quo(state.LastPrice)
		}

		if deviation.GT(maxDeviation) {
			// Trigger circuit breaker
			state.Enabled = true
			state.PausedUntil = sdkCtx.BlockTime().Add(1 * time.Hour)
			state.TriggerReason = fmt.Sprintf("price deviation of %s%% during %s", deviation.Mul(math.LegacyNewDec(100)), operation)

			if err := k.SetCircuitBreakerState(ctx, pool.Id, state); err != nil {
				return err
			}

			// Emit event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"circuit_breaker_triggered",
					sdk.NewAttribute("pool_id", fmt.Sprintf("%d", pool.Id)),
					sdk.NewAttribute("reason", state.TriggerReason),
					sdk.NewAttribute("paused_until", state.PausedUntil.String()),
				),
			)

			// TASK 61: Send circuit breaker notification
			if err := k.SendCircuitBreakerNotification(ctx, pool.Id, "triggered", state.TriggerReason, sdkCtx.BlockTime()); err != nil {
				sdkCtx.Logger().Error("failed to send circuit breaker notification", "error", err)
			}

			return types.ErrCircuitBreakerTriggered.Wrapf("triggered due to: %s", state.TriggerReason)
		}
	}

	// Update last known price
	state.LastPrice = currentPrice
	if err := k.SetCircuitBreakerState(ctx, pool.Id, state); err != nil {
		return err
	}

	return nil
}

// GetCircuitBreakerState retrieves circuit breaker state for a pool
func (k Keeper) GetCircuitBreakerState(ctx context.Context, poolID uint64) (CircuitBreakerState, error) {
	store := k.getStore(ctx)
	bz := store.Get(CircuitBreakerKey(poolID))
	if bz == nil {
		return CircuitBreakerState{
			Enabled:       false,
			LastPrice:     math.LegacyZeroDec(),
			TriggerReason: "",
		}, nil
	}

	var state CircuitBreakerState
	// Use encoding/json for non-protobuf types
	if err := json.Unmarshal(bz, &state); err != nil {
		return CircuitBreakerState{}, err
	}
	return state, nil
}

// SetCircuitBreakerState saves circuit breaker state for a pool
func (k Keeper) SetCircuitBreakerState(ctx context.Context, poolID uint64, state CircuitBreakerState) error {
	store := k.getStore(ctx)
	// Use encoding/json for non-protobuf types
	bz, err := json.Marshal(&state)
	if err != nil {
		return err
	}
	store.Set(CircuitBreakerKey(poolID), bz)
	return nil
}





// CheckFlashLoanProtection prevents same-block liquidity manipulation
/*
func (k Keeper) CheckFlashLoanProtection(ctx context.Context, poolID uint64, provider sdk.AccAddress) error {
	lastActionBlock, err := k.GetLastLiquidityActionBlock(ctx, poolID, provider)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := sdkCtx.BlockHeight()

	// Require minimum blocks between add and remove liquidity
	if currentBlock-lastActionBlock < MinLPLockBlocks {
		return types.ErrFlashLoanDetected.Wrapf(
			"must wait %d blocks between liquidity operations (last: %d, current: %d)",
			MinLPLockBlocks, lastActionBlock, currentBlock,
		)
	}

	return nil
}
*/

// GetLastLiquidityActionBlock retrieves the last block height when user modified liquidity
/*
func (k Keeper) GetLastLiquidityActionBlock(ctx context.Context, poolID uint64, provider sdk.AccAddress) (int64, error) {
    // ...
}

func (k Keeper) SetLastLiquidityActionBlock(ctx context.Context, poolID uint64, provider sdk.AccAddress) error {
    // ...
}

func SafeAdd(a, b math.Int) (math.Int, error) {
    // ...
}

func SafeSub(a, b math.Int) (math.Int, error) {
    // ...
}

func SafeMul(a, b math.Int) (math.Int, error) {
    // ...
}

func SafeQuo(a, b math.Int) (math.Int, error) {
    // ...
}
*/






// EmergencyPausePool pauses all operations on a pool (governance only)
func (k Keeper) EmergencyPausePool(ctx context.Context, poolID uint64, reason string, duration time.Duration) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	state := CircuitBreakerState{
		Enabled:       true,
		PausedUntil:   sdkCtx.BlockTime().Add(duration),
		TriggeredBy:   "governance",
		TriggerReason: reason,
	}

	if err := k.SetCircuitBreakerState(ctx, poolID, state); err != nil {
		return err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"pool_emergency_paused",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("paused_until", state.PausedUntil.String()),
		),
	)

	return nil
}

// UnpausePool removes circuit breaker pause (governance only)
func (k Keeper) UnpausePool(ctx context.Context, poolID uint64) error {
	state, err := k.GetCircuitBreakerState(ctx, poolID)
	if err != nil {
		return err
	}

	state.Enabled = false
	state.PausedUntil = time.Time{}
	state.TriggerReason = ""

	if err := k.SetCircuitBreakerState(ctx, poolID, state); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"pool_unpaused",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
		),
	)

	return nil
}

// TASK 61: SendCircuitBreakerNotification sends notifications when circuit breaker triggers or recovers
func (k Keeper) SendCircuitBreakerNotification(ctx context.Context, poolID uint64, eventType, reason string, timestamp time.Time) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get current state to update notification tracking
	state, err := k.GetCircuitBreakerState(ctx, poolID)
	if err != nil {
		return err
	}

	// Increment notification counter
	state.NotificationsSent++
	state.LastNotification = timestamp

	// Persist updated state
	if err := k.SetCircuitBreakerState(ctx, poolID, state); err != nil {
		return err
	}

	// Emit detailed notification event for external monitoring systems
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"circuit_breaker_notification",
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("event_type", eventType),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("timestamp", timestamp.Format(time.RFC3339)),
			sdk.NewAttribute("notification_count", fmt.Sprintf("%d", state.NotificationsSent)),
			sdk.NewAttribute("severity", k.getNotificationSeverity(eventType)),
		),
	)

	// Log for operators
	sdkCtx.Logger().Info("circuit breaker notification sent",
		"pool_id", poolID,
		"event_type", eventType,
		"reason", reason,
		"notification_count", state.NotificationsSent,
	)

	return nil
}

// getNotificationSeverity determines severity level for monitoring
func (k Keeper) getNotificationSeverity(eventType string) string {
	switch eventType {
	case "triggered":
		return "critical"
	case "recovery":
		return "info"
	default:
		return "warning"
	}
}

// PersistCircuitBreakerState ensures circuit breaker state survives restarts
func (k Keeper) PersistCircuitBreakerState(ctx context.Context, poolID uint64) error {
	state, err := k.GetCircuitBreakerState(ctx, poolID)
	if err != nil {
		return err
	}

	// State is already persisted via SetCircuitBreakerState which uses KVStore
	// This function provides explicit persistence guarantee and can be extended
	// for additional backup mechanisms if needed

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Debug("circuit breaker state persisted",
		"pool_id", poolID,
		"enabled", state.Enabled,
		"paused_until", state.PausedUntil,
	)

	return nil
}
