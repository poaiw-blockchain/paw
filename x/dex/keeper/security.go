package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Security constants - Production-grade parameters calibrated for mainnet security.
//
// These parameters balance attack prevention (MEV, flash loans, price manipulation),
// normal market operations, and system availability. Values are conservative, erring
// on the side of security. Parameters are intentionally NOT governable to prevent
// governance attacks where malicious proposals weaken security.
//
// For detailed governance implementation path and security analysis, see:
// docs/design/SECURITY_PARAMETER_GOVERNANCE.md
const (
	// MaxPriceDeviation = "0.25" (25%)
	// SECURITY RATIONALE:
	// - Triggers circuit breaker if pool price changes >25% in single operation
	// - Prevents flash crash attacks and oracle manipulation
	// - 25% chosen as balance: strict enough to catch attacks, loose enough for volatile crypto markets
	// - Bitcoin has seen 20%+ single-day moves, so 25% allows legitimate volatility
	// - Lower values (e.g., 10%) would trigger false positives during normal market stress
	// - Higher values (e.g., 50%) would allow attackers to drain significant liquidity
	// ATTACK SCENARIO PREVENTED: Attacker cannot manipulate pool price >25% without triggering pause
	MaxPriceDeviation = "0.25"

	// MaxSwapSizePercent = "0.1" (10%)
	// SECURITY RATIONALE:
	// - Limits single swap to max 10% of pool reserves (MEV protection)
	// - Prevents sandwich attacks with excessive slippage
	// - Protects against pool drainage via repeated max-size swaps
	// - 10% allows institutional trades while preventing market manipulation
	// - Example: $1M pool allows max $100K single swap (reasonable for DeFi)
	// - Lower values (e.g., 5%) would fragment large legitimate trades
	// - Higher values (e.g., 20%) would enable significant price impact attacks
	// ATTACK SCENARIO PREVENTED: Attacker cannot drain >10% reserves in single transaction,
	// requiring multiple blocks and exposing manipulation to arbitrageurs
	MaxSwapSizePercent = "0.1"

	// MinLPLockBlocks = 1 (1 block)
	// SECURITY RATIONALE:
	// - Prevents same-block add-liquidity-then-remove (flash loan attack pattern)
	// - Forces minimum 1 block delay between add/remove operations
	// - Exposes flash loan attackers to block time risk and arbitrage
	// - 1 block (~6 seconds) is minimum to break atomic flash loan execution
	// - Higher values (e.g., 10 blocks) would harm legitimate LP experience
	// - 0 blocks would allow atomic flash loan attacks to manipulate pricing
	// ATTACK SCENARIO PREVENTED: Attacker cannot add liquidity, manipulate price via swap,
	// then remove liquidity in same transaction to extract value
	MinLPLockBlocks = int64(1)

	MaxPools = uint64(1000)

	// PriceUpdateTolerance = "0.001" (0.1%)
	// SECURITY RATIONALE:
	// - Invariant check tolerance for k=x*y enforcement
	// - Accounts for rounding errors in decimal math while remaining strict
	// - 0.1% allows ~10 basis points of computational drift
	// - Prevents k-value manipulation attacks within rounding error bounds
	// - Lower values (e.g., 0.01%) would fail legitimate operations due to precision limits
	// - Higher values (e.g., 1%) would allow attackers to slowly drain pools via rounding exploitation
	// ATTACK SCENARIO PREVENTED: Attacker cannot craft sequences of operations that
	// gradually decrease k-value through accumulated rounding errors
	PriceUpdateTolerance = "0.001"
)

// CircuitBreakerState is now defined in types/dex.pb.go (generated from proto)
// Helper functions to convert between time.Time and Unix timestamps for the proto type

// ReentrancyGuard provides lightweight in-memory locks for tests and auxiliary flows.
type ReentrancyGuard struct {
	mu    sync.Mutex
	locks map[string]struct{}
}

// NewReentrancyGuard creates a new guard instance.
func NewReentrancyGuard() *ReentrancyGuard {
	return &ReentrancyGuard{locks: make(map[string]struct{})}
}

// Lock acquires a named lock or returns an error if already held.
func (g *ReentrancyGuard) Lock(key string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.locks[key]; exists {
		return types.ErrReentrancy.Wrapf("reentrancy detected for %s", key)
	}

	g.locks[key] = struct{}{}
	return nil
}

// Unlock releases a named lock.
func (g *ReentrancyGuard) Unlock(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.locks, key)
}

// WithReentrancyGuard executes a function with reentrancy protection
// Stores locks in KVStore to ensure they persist across context boundaries
func (k Keeper) WithReentrancyGuard(ctx context.Context, poolID uint64, operation string, fn func() error) error {
	return k.WithReentrancyGuardAndLock(ctx, poolID, operation, nil, fn)
}

// WithReentrancyGuardAndLock executes a function with reentrancy protection using both
// KVStore-based locks and an optional in-memory guard for test scenarios.
// The guard parameter allows tests to pass an explicit ReentrancyGuard instead of using context values.
func (k Keeper) WithReentrancyGuardAndLock(ctx context.Context, poolID uint64, operation string, guard *ReentrancyGuard, fn func() error) error {
	lockKey := fmt.Sprintf("%d:%s", poolID, operation)

	// Use explicit guard parameter for in-memory locking (primarily for tests)
	if guard != nil {
		if err := guard.Lock(lockKey); err != nil {
			return err
		}
		defer guard.Unlock(lockKey)
	}

	// Acquire lock using KVStore
	if err := k.acquireReentrancyLock(ctx, lockKey); err != nil {
		return err
	}

	// Ensure lock is released even if function panics
	defer k.releaseReentrancyLock(ctx, lockKey)

	return fn()
}

// LockExpirationBlocks is the maximum number of blocks a reentrancy lock can persist.
// SEC-4 FIX: If a lock is older than this, it's considered stale and will be released.
// This prevents permanent lock persistence if a panic occurs before defer runs.
// 2 blocks is sufficient since operations should complete within a single block.
const LockExpirationBlocks = int64(2)

// acquireReentrancyLock attempts to acquire a reentrancy lock from the KVStore.
// SEC-4 FIX: Now includes block height to allow expiration of stale locks.
// SEC-13 FIX: Uses CacheContext for atomic check-and-set to prevent race conditions.
func (k Keeper) acquireReentrancyLock(ctx context.Context, lockKey string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// SEC-13 FIX: Use CacheContext for atomic check-and-set
	// This ensures the lock check and set happen atomically, preventing
	// race conditions where two concurrent transactions both read "no lock"
	// before either writes their lock.
	cacheCtx, writeFn := sdkCtx.CacheContext()
	store := cacheCtx.KVStore(k.storeKey)
	key := ReentrancyLockKey(lockKey)

	// Check if lock already exists
	if existingData := store.Get(key); existingData != nil {
		// SEC-4 FIX: Check if the lock is stale (expired)
		if len(existingData) >= 8 {
			lockHeight := int64(binary.BigEndian.Uint64(existingData[:8]))
			if currentHeight-lockHeight > LockExpirationBlocks {
				// Lock is stale, we can safely override it
				// Log the stale lock cleanup for monitoring
				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"stale_reentrancy_lock_cleared",
						sdk.NewAttribute("lock_key", lockKey),
						sdk.NewAttribute("lock_height", fmt.Sprintf("%d", lockHeight)),
						sdk.NewAttribute("current_height", fmt.Sprintf("%d", currentHeight)),
					),
				)
			} else {
				// Lock is still valid - do NOT call writeFn, cache is discarded
				return types.ErrReentrancy.Wrapf("operation %s is already locked (since block %d)", lockKey, lockHeight)
			}
		} else {
			// Legacy lock without height data - treat as valid for one more block then expire
			return types.ErrReentrancy.Wrapf("operation %s is already locked", lockKey)
		}
	}

	// SEC-4 FIX: Store lock with block height for expiration tracking
	// Format: [8 bytes: block height] + [1 byte: lock marker]
	lockData := make([]byte, 9)
	binary.BigEndian.PutUint64(lockData[:8], uint64(currentHeight))
	lockData[8] = 0x01 // Lock marker
	store.Set(key, lockData)

	// SEC-13 FIX: Commit the atomic check-and-set operation
	writeFn()
	return nil
}

// releaseReentrancyLock releases a reentrancy lock from the KVStore
func (k Keeper) releaseReentrancyLock(ctx context.Context, lockKey string) {
	store := k.getStore(ctx)
	key := ReentrancyLockKey(lockKey)
	store.Delete(key)
}

// CleanupStaleReentrancyLocks cleans up any stale reentrancy locks from previous blocks.
// SEC-4 FIX: This should be called in EndBlocker to ensure stale locks don't persist.
// This is a safety net - normally locks are cleaned up by defer in WithReentrancyGuard.
func (k Keeper) CleanupStaleReentrancyLocks(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	store := k.getStore(ctx)

	// Iterate over all reentrancy locks
	prefix := []byte{0x02, 0x30} // ReentrancyLockPrefix from keys.go
	iterator := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iterator.Close()

	var staleKeys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		lockData := iterator.Value()
		if len(lockData) >= 8 {
			lockHeight := int64(binary.BigEndian.Uint64(lockData[:8]))
			if currentHeight-lockHeight > LockExpirationBlocks {
				staleKeys = append(staleKeys, append([]byte{}, iterator.Key()...))
			}
		}
	}

	// Delete stale locks
	for _, key := range staleKeys {
		store.Delete(key)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"reentrancy_lock_expired",
				sdk.NewAttribute("lock_key", string(key)),
				sdk.NewAttribute("current_height", fmt.Sprintf("%d", currentHeight)),
			),
		)
	}
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

	// If pool has shares, it must have reserves (both must be positive)
	if !pool.TotalShares.IsZero() && (pool.ReserveA.IsZero() || pool.ReserveB.IsZero()) {
		return types.ErrInvalidPoolState.Wrap("pool has shares but missing reserves")
	}

	// SECURITY: Validate reserves are positive before any calculations
	// This prevents division by zero in swap, price, and liquidity calculations
	// For initialized pools (with shares), both reserves MUST be positive
	if !pool.TotalShares.IsZero() {
		if pool.ReserveA.IsZero() {
			return types.ErrInsufficientLiquidity.Wrapf("reserve A is zero for initialized pool %d", pool.Id)
		}
		if pool.ReserveB.IsZero() {
			return types.ErrInsufficientLiquidity.Wrapf("reserve B is zero for initialized pool %d", pool.Id)
		}
		// Additional safety: ensure both reserves are strictly positive
		if !pool.ReserveA.IsPositive() {
			return types.ErrInsufficientLiquidity.Wrapf("reserve A must be positive for pool %d, got %s", pool.Id, pool.ReserveA)
		}
		if !pool.ReserveB.IsPositive() {
			return types.ErrInsufficientLiquidity.Wrapf("reserve B must be positive for pool %d, got %s", pool.Id, pool.ReserveB)
		}
	}

	return nil
}

// checkPoolPriceDeviation checks if circuit breaker should be triggered for a specific pool due to price deviation
func (k Keeper) checkPoolPriceDeviation(ctx context.Context, pool *types.Pool, operation string) error {
	state, err := k.GetPoolCircuitBreakerState(ctx, pool.Id)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pausedUntil := time.Unix(state.PausedUntil, 0)

	// Check if pool is currently paused
	if state.Enabled && sdkCtx.BlockTime().Before(pausedUntil) {
		return types.ErrCircuitBreakerTriggered.Wrapf(
			"pool %d paused until %s, reason: %s",
			pool.Id, pausedUntil, state.TriggerReason,
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
			// Get circuit breaker duration from params (defaults to 1 hour)
			params, paramsErr := k.GetParams(ctx)
			circuitBreakerDuration := time.Hour // fallback default
			if paramsErr == nil && params.CircuitBreakerDurationSeconds > 0 {
				circuitBreakerDuration = time.Duration(params.CircuitBreakerDurationSeconds) * time.Second
			}

			newPausedUntil := sdkCtx.BlockTime().Add(circuitBreakerDuration)
			state.Enabled = true
			state.PausedUntil = newPausedUntil.Unix()
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
					sdk.NewAttribute("paused_until", newPausedUntil.String()),
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

// GetPoolCircuitBreakerState retrieves circuit breaker state for a pool
func (k Keeper) GetPoolCircuitBreakerState(ctx context.Context, poolID uint64) (*types.CircuitBreakerState, error) {
	store := k.getStore(ctx)
	bz := store.Get(CircuitBreakerKey(poolID))
	if bz == nil {
		return &types.CircuitBreakerState{
			Enabled:       false,
			LastPrice:     math.LegacyZeroDec(),
			TriggerReason: "",
		}, nil
	}

	var state types.CircuitBreakerState
	if err := k.cdc.Unmarshal(bz, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SetCircuitBreakerState saves circuit breaker state for a pool
func (k Keeper) SetCircuitBreakerState(ctx context.Context, poolID uint64, state *types.CircuitBreakerState) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(state)
	if err != nil {
		return err
	}
	store.Set(CircuitBreakerKey(poolID), bz)
	return nil
}

// NOTE: CheckFlashLoanProtection is implemented in dex_advanced.go
// It uses params.FlashLoanProtectionBlocks for configurable block delay
// See dex_advanced.go:375-406 for the actual implementation

// EmergencyPausePool pauses all operations on a pool (governance only)
func (k Keeper) EmergencyPausePool(ctx context.Context, poolID uint64, reason string, duration time.Duration) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pausedUntil := sdkCtx.BlockTime().Add(duration)

	state := &types.CircuitBreakerState{
		Enabled:       true,
		PausedUntil:   pausedUntil.Unix(),
		TriggeredBy:   "governance",
		TriggerReason: reason,
		LastPrice:     math.LegacyZeroDec(),
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
			sdk.NewAttribute("paused_until", pausedUntil.String()),
		),
	)

	return nil
}

// UnpausePool removes circuit breaker pause (governance only)
func (k Keeper) UnpausePool(ctx context.Context, poolID uint64) error {
	state, err := k.GetPoolCircuitBreakerState(ctx, poolID)
	if err != nil {
		return err
	}

	state.Enabled = false
	state.PausedUntil = 0
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
	state, err := k.GetPoolCircuitBreakerState(ctx, poolID)
	if err != nil {
		return err
	}

	// Increment notification counter
	state.NotificationsSent++
	state.LastNotification = timestamp.Unix()

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
	state, err := k.GetPoolCircuitBreakerState(ctx, poolID)
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
