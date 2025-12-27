package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// BeginBlocker is called at the beginning of every block
// It handles time-based pool maintenance and fee distribution
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Update pool TWAPs (Time Weighted Average Price)
	if err := k.UpdatePoolTWAPs(ctx); err != nil {
		sdkCtx.Logger().Error("failed to update pool TWAPs", "error", err)
		// Don't return error - log and continue
	}

	// Distribute protocol fees to the community pool or fee collector
	if err := k.DistributeProtocolFees(ctx); err != nil {
		sdkCtx.Logger().Error("failed to distribute protocol fees", "error", err)
		// Don't return error - log and continue
	}

	// Emit begin block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_begin_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// EndBlocker is called at the end of every block
// It handles time-based operations like circuit breaker recovery and cleanup
func (k Keeper) EndBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Process expired limit orders (must happen before matching)
	// This refunds tokens for orders that have passed their expiration time
	if err := k.ProcessExpiredOrders(ctx); err != nil {
		sdkCtx.Logger().Error("failed to process expired orders", "error", err)
		// Don't return error - log and continue to prevent block production halt
	}

	// Match limit orders against current pool prices
	// Uses batching and gas limits to prevent chain halt with large order books
	if err := k.MatchAllOrders(ctx); err != nil {
		sdkCtx.Logger().Error("failed to match limit orders", "error", err)
		// Don't return error - log and continue to prevent block production halt
	}

	// Process circuit breaker auto-recovery for all pools
	if err := k.ProcessCircuitBreakerRecovery(ctx); err != nil {
		sdkCtx.Logger().Error("failed to process circuit breaker recovery", "error", err)
		// Don't return error - log and continue to prevent block production halt
	}

	// Cleanup old rate limit data to prevent unbounded state growth
	if err := k.CleanupOldRateLimitData(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup old rate limit data", "error", err)
		// Don't return error - log and continue
	}

	// Cleanup inactive pools from active set to prevent unbounded growth
	if err := k.CleanupInactivePools(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup inactive pools", "error", err)
		// Don't return error - log and continue
	}

	// Cleanup expired swap commitments (commit-reveal scheme for large swaps)
	if err := k.CleanupExpiredCommitments(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup expired swap commitments", "error", err)
		// Don't return error - log and continue
	}

	// Emit end block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_end_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// ProcessCircuitBreakerRecovery automatically recovers pools from circuit breaker pause
// when the pause duration has expired
func (k Keeper) ProcessCircuitBreakerRecovery(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	recoveredCount := 0

	// Iterate over all pools to check circuit breaker states
	err := k.IteratePools(ctx, func(pool types.Pool) bool {
		cbState, err := k.GetPoolCircuitBreakerState(ctx, pool.Id)
		if err != nil {
			sdkCtx.Logger().Error("failed to get circuit breaker state", "pool_id", pool.Id, "error", err)
			return false // Continue iteration
		}

		// Check if circuit breaker is enabled and pause period has expired
		if cbState.Enabled && !cbState.PausedUntil.IsZero() && now.After(cbState.PausedUntil) {
			// Auto-recover the pool
			cbState.Enabled = false
			oldPausedUntil := cbState.PausedUntil
			cbState.PausedUntil = time.Time{}
			oldReason := cbState.TriggerReason
			cbState.TriggerReason = ""

			if err := k.SetCircuitBreakerState(ctx, pool.Id, cbState); err != nil {
				sdkCtx.Logger().Error("failed to recover pool from circuit breaker",
					"pool_id", pool.Id,
					"error", err,
				)
				return false // Continue iteration
			}

			// Emit recovery event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"circuit_breaker_recovered",
					sdk.NewAttribute("pool_id", fmt.Sprintf("%d", pool.Id)),
					sdk.NewAttribute("paused_until", oldPausedUntil.Format(time.RFC3339)),
					sdk.NewAttribute("trigger_reason", oldReason),
					sdk.NewAttribute("recovered_at", now.Format(time.RFC3339)),
				),
			)

			// TASK 61: Send circuit breaker notification
			if err := k.SendCircuitBreakerNotification(ctx, pool.Id, "recovery", oldReason, now); err != nil {
				sdkCtx.Logger().Error("failed to send circuit breaker recovery notification",
					"pool_id", pool.Id,
					"error", err,
				)
			}

			sdkCtx.Logger().Info("auto-recovered pool from circuit breaker",
				"pool_id", pool.Id,
				"was_paused_until", oldPausedUntil,
			)

			recoveredCount++
		}

		return false // Continue iteration
	})

	if err != nil {
		return fmt.Errorf("failed to iterate pools for circuit breaker recovery: %w", err)
	}

	if recoveredCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"circuit_breakers_recovered",
				sdk.NewAttribute("count", fmt.Sprintf("%d", recoveredCount)),
				sdk.NewAttribute("timestamp", now.Format(time.RFC3339)),
			),
		)
	}

	return nil
}

// CleanupOldRateLimitData removes old rate limit tracking data to prevent state bloat
func (k Keeper) CleanupOldRateLimitData(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Rate limit data older than 24 hours can be safely removed
	// Since rate limits are typically enforced on shorter windows
	const RateLimitRetentionBlocks = 86400 // ~24 hours at 1s blocks

	cutoffHeight := sdkCtx.BlockHeight() - RateLimitRetentionBlocks

	if cutoffHeight <= 0 {
		return nil // Don't cleanup in early blocks
	}

	cleanedCount := 0

	// Iterate through heights that need cleanup
	// We clean up a range of blocks to avoid unbounded gas consumption
	for height := cutoffHeight - 10; height < cutoffHeight; height++ {
		if height <= 0 {
			continue
		}

		// Use IIFE to ensure iterator is closed at end of each iteration
		// This prevents iterator accumulation (defer inside loop would keep all open until function returns)
		func() {
			// Get all rate limits at this height
			heightPrefix := RateLimitByHeightPrefixForHeight(height)
			iterator := store.Iterator(heightPrefix, storetypes.PrefixEndBytes(heightPrefix))
			defer iterator.Close() // Now properly closes at end of this anonymous function

			rateLimitsToDelete := [][]byte{}
			for ; iterator.Valid(); iterator.Next() {
				rateLimitsToDelete = append(rateLimitsToDelete, iterator.Key())
				cleanedCount++
			}
			// Delete the rate limit index entries
			for _, key := range rateLimitsToDelete {
				store.Delete(key)

				// Also delete the actual rate limit entry
				// Extract user and window from the index key
				// Format: RateLimitByHeightPrefix(2) + height(8) + user(20) + window(8) = 38 bytes
				// Keys are namespaced with 2-byte prefix
				if len(key) >= 38 { // 2 + 8 + 20 + 8
					userStart := 10 // After prefix(2) + height(8)
					userEnd := 30   // userStart + 20
					windowStart := 30

					if len(key) >= windowStart+8 {
						user := sdk.AccAddress(key[userStart:userEnd])
						windowBytes := key[windowStart : windowStart+8]
						window := int64(binary.BigEndian.Uint64(windowBytes))

						// Delete the actual rate limit entry
						rateLimitKey := RateLimitKey(user, window)
						store.Delete(rateLimitKey)
					}
				}
			}
		}()
	}

	// Emit event with cleanup statistics
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"rate_limit_data_cleaned",
			sdk.NewAttribute("cutoff_height", fmt.Sprintf("%d", cutoffHeight)),
			sdk.NewAttribute("current_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			sdk.NewAttribute("rate_limits_cleaned", fmt.Sprintf("%d", cleanedCount)),
		),
	)

	return nil
}

// ActivePoolTTL is the duration after which an inactive pool is removed from the active set
// Pools with no swaps or liquidity changes for this duration are removed from the active index
// This prevents unbounded growth of the active pools tracking set
const ActivePoolTTL = 24 * time.Hour

// CleanupInactivePools removes pools from the active set if they haven't had activity
// within the ActivePoolTTL window. This prevents unbounded growth of the active pools index.
//
// The cleanup is based on the LastTimestamp field in the PoolTWAP record, which is updated
// on every swap or liquidity change via MarkPoolActive and UpdateCumulativePriceOnSwap.
//
// Note: This only removes the pool from the "active" index - it does NOT delete the pool itself
// or any liquidity/TWAP data. The pool remains fully functional and will be re-added to the
// active set on the next swap or liquidity operation.
func (k Keeper) CleanupInactivePools(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)
	currentTime := sdkCtx.BlockTime()

	// Calculate the cutoff timestamp - pools inactive longer than this get cleaned up
	cutoffTime := currentTime.Add(-ActivePoolTTL)
	cutoffTimestamp := cutoffTime.Unix()

	cleanedCount := 0
	scannedCount := 0

	// Iterate through all pools in the active set
	iter := storetypes.KVStorePrefixIterator(store, ActivePoolsKeyPrefix)
	defer iter.Close()

	// Collect pools to remove (can't delete while iterating)
	poolsToRemove := []uint64{}

	for ; iter.Valid(); iter.Next() {
		scannedCount++

		// Extract pool ID from key: ActivePoolsKeyPrefix(1 byte) + poolID(8 bytes)
		poolID := sdk.BigEndianToUint64(iter.Key()[len(ActivePoolsKeyPrefix):])

		// Get the pool's TWAP record to check last activity timestamp
		twapRecord, found, err := k.GetPoolTWAP(ctx, poolID)
		if err != nil {
			sdkCtx.Logger().Error("failed to get pool TWAP for cleanup check",
				"pool_id", poolID,
				"error", err,
			)
			continue // Skip this pool, continue iteration
		}

		// If no TWAP record exists, the pool has never had a swap
		// In this case, we still want to keep it in the active set since it was
		// explicitly marked active (likely via liquidity addition)
		if !found {
			continue
		}

		// Check if the pool's last activity was before the cutoff
		if twapRecord.LastTimestamp < cutoffTimestamp {
			poolsToRemove = append(poolsToRemove, poolID)
		}
	}

	// Remove inactive pools from the active set
	for _, poolID := range poolsToRemove {
		store.Delete(ActivePoolKey(poolID))
		cleanedCount++

		sdkCtx.Logger().Debug("removed inactive pool from active set",
			"pool_id", poolID,
			"cutoff_time", cutoffTime.Format(time.RFC3339),
		)
	}

	// Emit event with cleanup statistics
	if cleanedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"active_pools_cleaned",
				sdk.NewAttribute("scanned_count", fmt.Sprintf("%d", scannedCount)),
				sdk.NewAttribute("cleaned_count", fmt.Sprintf("%d", cleanedCount)),
				sdk.NewAttribute("cutoff_time", cutoffTime.Format(time.RFC3339)),
				sdk.NewAttribute("ttl_hours", fmt.Sprintf("%d", int(ActivePoolTTL.Hours()))),
			),
		)

		sdkCtx.Logger().Info("cleaned up inactive pools from active set",
			"scanned", scannedCount,
			"cleaned", cleanedCount,
			"ttl", ActivePoolTTL.String(),
		)
	}

	return nil
}

// UpdatePoolTWAPs is deprecated and now a no-op.
// TWAP updates are now lazy and triggered only on swaps via UpdateCumulativePriceOnSwap.
// This fixes the O(n) performance issue where we iterated all pools every block.
func (k Keeper) UpdatePoolTWAPs(ctx context.Context) error {
	// No-op: TWAP calculation is now lazy (only on query or swap)
	// This eliminates O(n) iteration every block
	return nil
}

// DistributeProtocolFees distributes accumulated protocol fees
func (k Keeper) DistributeProtocolFees(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Collect all protocol fees
	iter := storetypes.KVStorePrefixIterator(store, types.ProtocolFeeKeyPrefix)
	defer iter.Close()

	totalFees := sdk.NewCoins()
	distributionCount := 0
	for ; iter.Valid(); iter.Next() {
		var amount math.Int
		if err := amount.Unmarshal(iter.Value()); err != nil {
			continue
		}

		// Extract denom from key
		denom := string(iter.Key()[len(types.ProtocolFeeKeyPrefix):])
		totalFees = totalFees.Add(sdk.NewCoin(denom, amount))

		// Clear the fee accumulator
		store.Delete(iter.Key())

		if !amount.IsZero() {
			distributionCount++
		}
	}

	if !totalFees.IsZero() {
		sdkCtx.Logger().Info("distributing protocol fees", "amount", totalFees)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"dex_protocol_fees_distributed",
				sdk.NewAttribute("amount", totalFees.String()),
				sdk.NewAttribute("distribution_count", fmt.Sprintf("%d", distributionCount)),
			),
		)
	}

	return nil
}
