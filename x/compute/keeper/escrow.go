package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// EscrowState is an alias to the protobuf type for escrow state management
type EscrowState = types.EscrowState

// LockEscrow creates a secure escrow lock with timeout and challenge period
// This prevents funds from being locked indefinitely and implements atomic state transitions
func (k Keeper) LockEscrow(ctx context.Context, requester, provider sdk.AccAddress, amount math.Int, requestID uint64, timeoutSeconds uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate inputs
	if amount.IsZero() || amount.IsNegative() {
		return fmt.Errorf("escrow amount must be positive")
	}

	// Check for minimum payment threshold to prevent spam
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	minPayment := params.MinProviderStake.QuoRaw(100) // 1% of min stake as minimum payment
	if amount.LT(minPayment) {
		return fmt.Errorf("payment %s below minimum threshold %s", amount.String(), minPayment.String())
	}

	// Generate unique nonce for this escrow BEFORE any state changes
	// This is done early to ensure we have a nonce even if the atomic set fails
	nonce, err := k.getNextEscrowNonce(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate escrow nonce: %w", err)
	}

	now := sdkCtx.BlockTime()
	expiresAt := now.Add(types.SecondsToDuration(timeoutSeconds))

	// Create escrow state record to prepare for atomic commit
	escrowState := &EscrowState{
		RequestId:       requestID,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          amount,
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       expiresAt,
		ReleasedAt:      nil,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 0,
		Nonce:           nonce,
	}

	// TWO-PHASE COMMIT: Use CacheContext to ensure ALL operations succeed or ALL fail
	// This prevents catastrophic failures where bank transfer succeeds but state update fails
	// The CacheContext acts as a transaction boundary - changes are only committed if writeFn is called
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Phase 1: Bank transfer - this must happen in the cached context
	coins := sdk.NewCoins(sdk.NewCoin("upaw", amount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(cacheCtx, requester, types.ModuleName, coins); err != nil {
		// Transfer failed - cache is automatically discarded, no cleanup needed
		return fmt.Errorf("failed to lock escrow funds: %w", err)
	}

	// Phase 2: Store escrow state (atomically - only if it doesn't exist)
	// This prevents the double-lock race condition where two concurrent calls
	// could both pass an earlier existence check and overwrite each other
	if err := k.SetEscrowStateIfNotExists(cacheCtx, *escrowState); err != nil {
		// State storage failed - cache is automatically discarded
		// Bank transfer is rolled back automatically - no catastrophic failure possible
		return fmt.Errorf("failed to store escrow state atomically: %w", err)
	}

	// Phase 3: Create timeout index for automatic cleanup
	// This is CRITICAL - without timeout index, funds can be locked permanently
	if err := k.setEscrowTimeoutIndex(cacheCtx, requestID, expiresAt); err != nil {
		// Timeout index creation failed - cache is automatically discarded
		// Both bank transfer and state update are rolled back - no catastrophic failure possible
		return fmt.Errorf("failed to create timeout index atomically: %w", err)
	}

	// COMMIT: All operations succeeded - write the cached context to the parent context
	// This makes all changes permanent atomically
	writeFn()

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"escrow_locked",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("requester", requester.String()),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("expires_at", expiresAt.Format(time.RFC3339)),
			sdk.NewAttribute("nonce", fmt.Sprintf("%d", nonce)),
		),
	)

	// Record escrow locked metrics
	if k.metrics != nil {
		k.metrics.EscrowLocked.With(map[string]string{
			"denom": "upaw",
		}).Add(float64(amount.Int64()))

		k.metrics.EscrowBalance.With(map[string]string{
			"denom": "upaw",
		}).Add(float64(amount.Int64()))
	}

	return nil
}

// ReleaseEscrow releases escrowed funds to provider with challenge period
// Implements atomic check-and-release to prevent double-spending
func (k Keeper) ReleaseEscrow(ctx context.Context, requestID uint64, releaseImmediate bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get escrow state with lock
	escrowState, err := k.GetEscrowState(ctx, requestID)
	if err != nil {
		return fmt.Errorf("escrow not found for request %d: %w", requestID, err)
	}

	// Skip if already processed
	if escrowState.Status == types.ESCROW_STATUS_RELEASED || escrowState.Status == types.ESCROW_STATUS_REFUNDED {
		return nil
	}

	// CRITICAL: Check current status atomically
	if escrowState.Status != types.ESCROW_STATUS_LOCKED &&
		escrowState.Status != types.ESCROW_STATUS_CHALLENGED {
		return fmt.Errorf("escrow cannot be released in status %s", escrowState.Status.String())
	}

	now := sdkCtx.BlockTime()

	// Check if escrow has expired
	if now.After(escrowState.ExpiresAt) {
		return fmt.Errorf("escrow expired, must be refunded")
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("ReleaseEscrow: get params: %w", err)
	}

	// Implement challenge period unless immediate release requested (governance override)
	if !releaseImmediate {
		if escrowState.ChallengeEndsAt == nil {
			// First release attempt - start challenge period
			challengeEnds := now.Add(types.SecondsToDuration(params.EscrowReleaseDelaySeconds))
			escrowState.ChallengeEndsAt = &challengeEnds
			escrowState.Status = types.ESCROW_STATUS_CHALLENGED
			escrowState.ReleaseAttempts++

			if err := k.SetEscrowState(ctx, *escrowState); err != nil {
				return fmt.Errorf("failed to update escrow to challenged status: %w", err)
			}

			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"escrow_challenged",
					sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
					sdk.NewAttribute("challenge_ends_at", challengeEnds.Format(time.RFC3339)),
				),
			)

			return nil // Wait for challenge period
		}

		// Challenge period exists - check if it has passed
		if now.Before(*escrowState.ChallengeEndsAt) {
			return fmt.Errorf("challenge period active until %s", escrowState.ChallengeEndsAt.Format(time.RFC3339))
		}
	}

	// Validate preconditions
	if escrowState.Amount.IsZero() {
		return fmt.Errorf("escrow amount is zero")
	}

	provider, err := sdk.AccAddressFromBech32(escrowState.Provider)
	if err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	// TWO-PHASE COMMIT: Use CacheContext to ensure bank transfer and state update are atomic
	// This prevents catastrophic failure where payment is sent but state update fails
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Phase 1: Transfer funds to provider
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, provider, coins); err != nil {
		// Transfer failed - cache is automatically discarded, state remains LOCKED/CHALLENGED
		return fmt.Errorf("failed to release payment: %w", err)
	}

	// Phase 2: Update escrow state to RELEASED
	escrowState.Status = types.ESCROW_STATUS_RELEASED
	escrowState.ReleasedAt = &now
	escrowState.ReleaseAttempts++

	if err := k.SetEscrowState(cacheCtx, *escrowState); err != nil {
		// State update failed - cache is automatically discarded
		// Bank transfer is rolled back - no catastrophic failure possible
		return fmt.Errorf("failed to update escrow state to released: %w", err)
	}

	// Phase 3: Remove from timeout index
	k.removeEscrowTimeoutIndex(cacheCtx, requestID)

	// COMMIT: All operations succeeded - write the cached context atomically
	writeFn()

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"escrow_released",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("amount", escrowState.Amount.String()),
			sdk.NewAttribute("nonce", fmt.Sprintf("%d", escrowState.Nonce)),
		),
	)

	// Record escrow released metrics
	if k.metrics != nil {
		k.metrics.EscrowReleased.With(map[string]string{
			"denom": "upaw",
		}).Add(float64(escrowState.Amount.Int64()))

		k.metrics.EscrowBalance.With(map[string]string{
			"denom": "upaw",
		}).Sub(float64(escrowState.Amount.Int64()))
	}

	return nil
}

// RefundEscrow refunds escrowed funds to requester (timeout or cancellation)
// Implements atomic check-and-refund to prevent double-refund
func (k Keeper) RefundEscrow(ctx context.Context, requestID uint64, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get escrow state
	escrowState, err := k.GetEscrowState(ctx, requestID)
	if err != nil {
		return fmt.Errorf("escrow not found for request %d: %w", requestID, err)
	}

	// Skip if already processed or not locked
	if escrowState.Status != types.ESCROW_STATUS_LOCKED {
		return nil
	}

	// Already refunded check (shouldn't happen due to status guard)
	if escrowState.RefundedAt != nil {
		return nil
	}

	now := sdkCtx.BlockTime()
	requester, err := sdk.AccAddressFromBech32(escrowState.Requester)
	if err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}

	// TWO-PHASE COMMIT: Use CacheContext to ensure bank transfer and state update are atomic
	// This prevents catastrophic failure where refund is sent but state update fails
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Phase 1: Transfer funds back to requester
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, requester, coins); err != nil {
		// Transfer failed - cache is automatically discarded, state remains LOCKED
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	// Phase 2: Update escrow state to REFUNDED
	escrowState.Status = types.ESCROW_STATUS_REFUNDED
	escrowState.RefundedAt = &now

	if err := k.SetEscrowState(cacheCtx, *escrowState); err != nil {
		// State update failed - cache is automatically discarded
		// Bank transfer is rolled back - no catastrophic failure possible
		return fmt.Errorf("failed to update escrow state to refunded: %w", err)
	}

	// Phase 3: Remove from timeout index
	k.removeEscrowTimeoutIndex(cacheCtx, requestID)

	// COMMIT: All operations succeeded - write the cached context atomically
	writeFn()

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"escrow_refunded",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("requester", requester.String()),
			sdk.NewAttribute("amount", escrowState.Amount.String()),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("nonce", fmt.Sprintf("%d", escrowState.Nonce)),
		),
	)

	// Record escrow refunded metrics
	if k.metrics != nil {
		k.metrics.EscrowRefunded.With(map[string]string{
			"denom": "upaw",
		}).Add(float64(escrowState.Amount.Int64()))

		k.metrics.EscrowBalance.With(map[string]string{
			"denom": "upaw",
		}).Sub(float64(escrowState.Amount.Int64()))
	}

	return nil
}

// ProcessExpiredEscrows processes all expired escrows and refunds them
// Should be called in EndBlocker
func (k Keeper) ProcessExpiredEscrows(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	expiredCount := 0

	err := k.IterateEscrowTimeouts(ctx, now, func(requestID uint64, expiresAt time.Time) (stop bool, err error) {
		// Attempt to refund
		if err := k.RefundEscrow(ctx, requestID, "timeout"); err != nil {
			// Log error but continue processing
			k.recordEscrowWarning(ctx, requestID, fmt.Sprintf("failed to refund expired escrow: %v", err))
			return false, nil
		}

		expiredCount++
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("failed to process expired escrows: %w", err)
	}

	if expiredCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"escrows_expired_processed",
				sdk.NewAttribute("count", fmt.Sprintf("%d", expiredCount)),
				sdk.NewAttribute("timestamp", now.Format(time.RFC3339)),
			),
		)
	}

	return nil
}

// GetEscrowState retrieves the escrow state for a request
func (k Keeper) GetEscrowState(ctx context.Context, requestID uint64) (*EscrowState, error) {
	store := k.getStore(ctx)
	bz := store.Get(EscrowStateKey(requestID))

	if bz == nil {
		return nil, fmt.Errorf("escrow state not found")
	}

	var state EscrowState
	if err := k.cdc.Unmarshal(bz, &state); err != nil {
		return nil, fmt.Errorf("GetEscrowState: unmarshal: %w", err)
	}

	return &state, nil
}

// SetEscrowState stores the escrow state
func (k Keeper) SetEscrowState(ctx context.Context, state EscrowState) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&state)
	if err != nil {
		return fmt.Errorf("SetEscrowState: marshal: %w", err)
	}

	store.Set(EscrowStateKey(state.RequestId), bz)
	return nil
}

// SetEscrowStateIfNotExists atomically creates a new escrow state only if one doesn't already exist.
// This prevents the double-lock race condition where two concurrent LockEscrow calls could both
// pass the existence check and overwrite each other's escrow state.
// Returns an error if the escrow already exists.
func (k Keeper) SetEscrowStateIfNotExists(ctx context.Context, state EscrowState) error {
	store := k.getStore(ctx)
	key := EscrowStateKey(state.RequestId)

	// Atomic check-and-set: if key exists, return error
	// This is atomic because store operations are serialized within a single block execution
	if store.Has(key) {
		return fmt.Errorf("escrow already exists for request %d", state.RequestId)
	}

	bz, err := k.cdc.Marshal(&state)
	if err != nil {
		return fmt.Errorf("SetEscrowStateIfNotExists: marshal: %w", err)
	}

	store.Set(key, bz)
	return nil
}

// getNextEscrowNonce generates a unique nonce for escrow operations
func (k Keeper) getNextEscrowNonce(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextEscrowNonceKey)

	var nextNonce uint64 = 1
	if bz != nil {
		nextNonce = binary.BigEndian.Uint64(bz)
	}

	// Increment and store
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, nextNonce+1)
	store.Set(NextEscrowNonceKey, nextBz)

	return nextNonce, nil
}

// setEscrowTimeoutIndex creates an index entry for timeout processing
func (k Keeper) setEscrowTimeoutIndex(ctx context.Context, requestID uint64, expiresAt time.Time) error {
	store := k.getStore(ctx)

	// Create composite key: timeout_timestamp + request_id
	key := EscrowTimeoutKey(expiresAt, requestID)
	store.Set(key, []byte{})

	// Also set reverse index: requestID -> timestamp (for O(1) deletion)
	reverseKey := EscrowTimeoutReverseKey(requestID)
	timestampBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBz, types.SaturateInt64ToUint64(expiresAt.Unix()))
	store.Set(reverseKey, timestampBz)

	return nil
}

// removeEscrowTimeoutIndex removes the timeout index entry using the reverse index
// for O(1) lookup instead of iterating over all timeout entries.
func (k Keeper) removeEscrowTimeoutIndex(ctx context.Context, requestID uint64) {
	store := k.getStore(ctx)

	// Use reverse index to find timestamp in O(1)
	reverseKey := EscrowTimeoutReverseKey(requestID)
	timestampBz := store.Get(reverseKey)

	if timestampBz == nil {
		// Reverse index not found, fall back to iteration for backward compatibility
		k.removeEscrowTimeoutIndexSlow(ctx, requestID)
		return
	}

	// Reconstruct the timeout key and delete
	timestamp := binary.BigEndian.Uint64(timestampBz)
	expiresAt := time.Unix(types.SaturateUint64ToInt64(timestamp), 0)
	timeoutKey := EscrowTimeoutKey(expiresAt, requestID)

	// Delete both the timeout index and the reverse index
	store.Delete(timeoutKey)
	store.Delete(reverseKey)
}

// removeEscrowTimeoutIndexSlow is the fallback O(n) method for escrows created
// before the reverse index was added. It iterates to find the timeout entry.
func (k Keeper) removeEscrowTimeoutIndexSlow(ctx context.Context, requestID uint64) {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, EscrowTimeoutPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Extract request ID from key (last 8 bytes)
		key := iterator.Key()
		if len(key) >= 8 {
			rid := binary.BigEndian.Uint64(key[len(key)-8:])
			if rid == requestID {
				store.Delete(key)
				return
			}
		}
	}
}

// IterateEscrowTimeouts iterates over expired escrows
func (k Keeper) IterateEscrowTimeouts(ctx context.Context, beforeTime time.Time, cb func(requestID uint64, expiresAt time.Time) (stop bool, err error)) error {
	store := k.getStore(ctx)

	// Create end key for range query
	endKeyTime := beforeTime.Add(1 * time.Second) // Slightly after to be inclusive
	endKey := EscrowTimeoutKey(endKeyTime, ^uint64(0))

	iterator := store.Iterator(EscrowTimeoutPrefix, endKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		// Extract timestamp and request ID from key
		if len(key) < len(EscrowTimeoutPrefix)+8+8 {
			continue
		}

		offset := len(EscrowTimeoutPrefix)
		timestampUnix := types.SaturateUint64ToInt64(binary.BigEndian.Uint64(key[offset : offset+8]))
		requestID := binary.BigEndian.Uint64(key[offset+8:])

		expiresAt := time.Unix(timestampUnix, 0)

		stop, err := cb(requestID, expiresAt)
		if err != nil {
			return fmt.Errorf("IterateTimeoutEscrows: callback: %w", err)
		}
		if stop {
			break
		}
	}

	return nil
}

// recordCatastrophicFailure records a catastrophic escrow failure for manual resolution.
// This function both emits an event and persists the failure to state for permanent record.
func (k Keeper) recordCatastrophicFailure(ctx context.Context, requestID uint64, account sdk.AccAddress, amount math.Int, reason string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Emit event for real-time monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"escrow_catastrophic_failure",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("timestamp", sdkCtx.BlockTime().Format(time.RFC3339)),
			sdk.NewAttribute("severity", "CRITICAL"),
		),
	)

	// CRITICAL: Also persist to state so failures are not lost if events are missed
	if err := k.StoreCatastrophicFailure(ctx, requestID, account, amount, reason); err != nil {
		// If we can't even store the catastrophic failure, log to chain via extra event
		// This is the absolute last resort - a catastrophic failure of the catastrophic failure system
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"escrow_catastrophic_failure_storage_failed",
				sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
				sdk.NewAttribute("account", account.String()),
				sdk.NewAttribute("amount", amount.String()),
				sdk.NewAttribute("reason", reason),
				sdk.NewAttribute("storage_error", err.Error()),
				sdk.NewAttribute("severity", "CRITICAL"),
			),
		)
	}
}

// recordEscrowWarning records a non-critical escrow warning
func (k Keeper) recordEscrowWarning(ctx context.Context, requestID uint64, message string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"escrow_warning",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("message", message),
			sdk.NewAttribute("timestamp", sdkCtx.BlockTime().Format(time.RFC3339)),
		),
	)
}

// Additional key definitions
var (
	EscrowStateKeyPrefix = []byte{0x20}
	EscrowTimeoutPrefix  = []byte{0x21}
	NextEscrowNonceKey   = []byte{0x22}
)

func EscrowStateKey(requestID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, requestID)
	return append(EscrowStateKeyPrefix, bz...)
}

func EscrowTimeoutKey(expiresAt time.Time, requestID uint64) []byte {
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, types.SaturateInt64ToUint64(expiresAt.Unix()))

	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)

	return append(append(EscrowTimeoutPrefix, timeBz...), idBz...)
}

// StoreCatastrophicFailure persists a catastrophic failure record to state.
// This ensures that critical failures are permanently recorded and can be queried later.
func (k Keeper) StoreCatastrophicFailure(ctx context.Context, requestID uint64, account sdk.AccAddress, amount math.Int, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Get next failure ID
	failureID, err := k.getNextCatastrophicFailureID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next catastrophic failure ID: %w", err)
	}

	// Create failure record
	failure := &types.CatastrophicFailure{
		Id:          failureID,
		RequestId:   requestID,
		Account:     account.String(),
		Amount:      amount,
		Reason:      reason,
		OccurredAt:  sdkCtx.BlockTime(),
		BlockHeight: sdkCtx.BlockHeight(),
		Resolved:    false,
		ResolvedAt:  nil,
	}

	// Marshal and store
	bz, err := k.cdc.Marshal(failure)
	if err != nil {
		return fmt.Errorf("failed to marshal catastrophic failure: %w", err)
	}

	store.Set(CatastrophicFailureKey(failureID), bz)
	return nil
}

// getNextCatastrophicFailureID generates a unique ID for catastrophic failure records
func (k Keeper) getNextCatastrophicFailureID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextCatastrophicFailureIDKey)

	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}

	// Increment and store
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, nextID+1)
	store.Set(NextCatastrophicFailureIDKey, nextBz)

	return nextID, nil
}

// GetCatastrophicFailure retrieves a catastrophic failure record by ID
func (k Keeper) GetCatastrophicFailure(ctx context.Context, failureID uint64) (*types.CatastrophicFailure, error) {
	store := k.getStore(ctx)
	bz := store.Get(CatastrophicFailureKey(failureID))

	if bz == nil {
		return nil, fmt.Errorf("catastrophic failure %d not found", failureID)
	}

	var failure types.CatastrophicFailure
	if err := k.cdc.Unmarshal(bz, &failure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal catastrophic failure: %w", err)
	}

	return &failure, nil
}

// GetAllCatastrophicFailures retrieves all catastrophic failure records
func (k Keeper) GetAllCatastrophicFailures(ctx context.Context) ([]*types.CatastrophicFailure, error) {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, CatastrophicFailureKeyPrefix)
	defer iterator.Close()

	var failures []*types.CatastrophicFailure
	for ; iterator.Valid(); iterator.Next() {
		var failure types.CatastrophicFailure
		if err := k.cdc.Unmarshal(iterator.Value(), &failure); err != nil {
			return nil, fmt.Errorf("failed to unmarshal catastrophic failure: %w", err)
		}
		failures = append(failures, &failure)
	}

	return failures, nil
}

// GetUnresolvedCatastrophicFailures retrieves all unresolved catastrophic failure records
func (k Keeper) GetUnresolvedCatastrophicFailures(ctx context.Context) ([]*types.CatastrophicFailure, error) {
	allFailures, err := k.GetAllCatastrophicFailures(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetUnresolvedCatastrophicFailures: get all: %w", err)
	}

	var unresolved []*types.CatastrophicFailure
	for _, failure := range allFailures {
		if !failure.Resolved {
			unresolved = append(unresolved, failure)
		}
	}

	return unresolved, nil
}
