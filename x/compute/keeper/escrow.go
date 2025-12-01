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

	// Check if escrow already exists (prevent double-lock)
	existingEscrow, err := k.GetEscrowState(ctx, requestID)
	if err == nil && existingEscrow != nil {
		return fmt.Errorf("escrow already exists for request %d", requestID)
	}

	// Generate unique nonce for this escrow
	nonce, err := k.getNextEscrowNonce(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate escrow nonce: %w", err)
	}

	now := sdkCtx.BlockTime()
	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)

	// Transfer funds atomically - CRITICAL: This must succeed or rollback entire operation
	coins := sdk.NewCoins(sdk.NewCoin("upaw", amount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, requester, types.ModuleName, coins); err != nil {
		return fmt.Errorf("failed to lock escrow funds: %w", err)
	}

	// Create escrow state record - AFTER successful transfer
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

	// Store escrow state
	if err := k.SetEscrowState(ctx, *escrowState); err != nil {
		// CRITICAL: If we can't store state, we must refund
		refundCoins := sdk.NewCoins(sdk.NewCoin("upaw", amount))
		if refundErr := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, refundCoins); refundErr != nil {
			// This is catastrophic - funds locked but state not saved
			k.recordCatastrophicFailure(ctx, requestID, requester, amount, "failed to store escrow state and refund")
		}
		return fmt.Errorf("failed to store escrow state: %w", err)
	}

	// Create timeout index for automatic cleanup
	if err := k.setEscrowTimeoutIndex(ctx, requestID, expiresAt); err != nil {
		// Non-critical: escrow is locked, just won't auto-expire
		k.recordEscrowWarning(ctx, requestID, "failed to create timeout index")
	}

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

	// CRITICAL: Check current status atomically
	if escrowState.Status != types.ESCROW_STATUS_LOCKED &&
		escrowState.Status != types.ESCROW_STATUS_CHALLENGED {
		return fmt.Errorf("escrow cannot be released in status %s", escrowState.Status.String())
	}

	// CRITICAL: Prevent double-spending by checking release attempts
	if escrowState.ReleaseAttempts > 0 && escrowState.Status == types.ESCROW_STATUS_RELEASED {
		return fmt.Errorf("escrow already released (double-spend attempt detected)")
	}

	now := sdkCtx.BlockTime()

	// Check if escrow has expired
	if now.After(escrowState.ExpiresAt) {
		return fmt.Errorf("escrow expired, must be refunded")
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Implement challenge period unless immediate release requested (governance override)
	if !releaseImmediate {
		if escrowState.ChallengeEndsAt == nil {
			// First release attempt - start challenge period
			challengeEnds := now.Add(time.Duration(params.EscrowReleaseDelaySeconds) * time.Second)
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

	// ATOMIC RELEASE: Check-Effects-Interactions pattern
	// 1. CHECK: Verify all conditions
	if escrowState.Amount.IsZero() {
		return fmt.Errorf("escrow amount is zero")
	}

	provider, err := sdk.AccAddressFromBech32(escrowState.Provider)
	if err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	// 2. EFFECTS: Update state BEFORE external call
	escrowState.Status = types.ESCROW_STATUS_RELEASED
	escrowState.ReleasedAt = &now
	escrowState.ReleaseAttempts++

	if err := k.SetEscrowState(ctx, *escrowState); err != nil {
		return fmt.Errorf("failed to update escrow state: %w", err)
	}

	// 3. INTERACTIONS: External call AFTER state update (prevents reentrancy)
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, provider, coins); err != nil {
		// CRITICAL: Payment failed but state updated - mark for manual resolution
		k.recordCatastrophicFailure(ctx, requestID, provider, escrowState.Amount, "state updated but payment failed")
		return fmt.Errorf("failed to release payment: %w", err)
	}

	// Remove from timeout index
	k.removeEscrowTimeoutIndex(ctx, requestID)

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

	// CRITICAL: Check current status atomically
	if escrowState.Status != types.ESCROW_STATUS_LOCKED &&
		escrowState.Status != types.ESCROW_STATUS_CHALLENGED {
		return fmt.Errorf("escrow cannot be refunded in status %s", escrowState.Status.String())
	}

	// Already refunded check
	if escrowState.RefundedAt != nil {
		return fmt.Errorf("escrow already refunded (double-refund attempt detected)")
	}

	now := sdkCtx.BlockTime()
	requester, err := sdk.AccAddressFromBech32(escrowState.Requester)
	if err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}

	// ATOMIC REFUND: Check-Effects-Interactions pattern
	// 1. EFFECTS: Update state BEFORE external call
	escrowState.Status = types.ESCROW_STATUS_REFUNDED
	escrowState.RefundedAt = &now

	if err := k.SetEscrowState(ctx, *escrowState); err != nil {
		return fmt.Errorf("failed to update escrow state: %w", err)
	}

	// 2. INTERACTIONS: External call AFTER state update
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, coins); err != nil {
		// CRITICAL: Refund failed but state updated
		k.recordCatastrophicFailure(ctx, requestID, requester, escrowState.Amount, "state updated but refund failed")
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	// Remove from timeout index
	k.removeEscrowTimeoutIndex(ctx, requestID)

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
		return nil, err
	}

	return &state, nil
}

// SetEscrowState stores the escrow state
func (k Keeper) SetEscrowState(ctx context.Context, state EscrowState) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&state)
	if err != nil {
		return err
	}

	store.Set(EscrowStateKey(state.RequestId), bz)
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
	binary.BigEndian.PutUint64(timestampBz, uint64(expiresAt.Unix()))
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
	expiresAt := time.Unix(int64(timestamp), 0)
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
		timestampUnix := int64(binary.BigEndian.Uint64(key[offset : offset+8]))
		requestID := binary.BigEndian.Uint64(key[offset+8:])

		expiresAt := time.Unix(timestampUnix, 0)

		stop, err := cb(requestID, expiresAt)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// recordCatastrophicFailure records a catastrophic escrow failure for manual resolution
func (k Keeper) recordCatastrophicFailure(ctx context.Context, requestID uint64, account sdk.AccAddress, amount math.Int, reason string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

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
	binary.BigEndian.PutUint64(timeBz, uint64(expiresAt.Unix()))

	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)

	return append(append(EscrowTimeoutPrefix, timeBz...), idBz...)
}
