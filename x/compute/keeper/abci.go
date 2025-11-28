package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// BeginBlocker is called at the beginning of every block
// It handles time-based initialization and scheduled tasks
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Update provider reputation scores based on performance
	if err := k.UpdateProviderReputations(ctx); err != nil {
		sdkCtx.Logger().Error("failed to update provider reputations", "error", err)
		// Don't return error - log and continue
	}

	// Process pending dispute resolutions
	if err := k.ProcessPendingDisputes(ctx); err != nil {
		sdkCtx.Logger().Error("failed to process pending disputes", "error", err)
		// Don't return error - log and continue
	}

	// Emit begin block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_begin_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// EndBlocker is called at the end of every block
// It handles time-based operations like escrow timeouts and cleanup
func (k Keeper) EndBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Process expired escrows and refund them automatically
	if err := k.ProcessExpiredEscrows(ctx); err != nil {
		sdkCtx.Logger().Error("failed to process expired escrows", "error", err)
		// Don't return error - log and continue to prevent block production halt
	}

	// Cleanup expired nonces to prevent unbounded state growth
	if err := k.CleanupExpiredNonces(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup expired nonces", "error", err)
		// Don't return error - log and continue
	}

	// Emit end block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_end_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// CleanupExpiredNonces removes old nonce entries to prevent state bloat
// Nonces older than 1000 blocks are safe to remove
func (k Keeper) CleanupExpiredNonces(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Keep nonces for the last 1000 blocks
	const NonceRetentionBlocks = 1000
	cutoffHeight := currentHeight - NonceRetentionBlocks

	if cutoffHeight <= 0 {
		return nil // Don't cleanup in early blocks
	}

	store := k.getStore(ctx)
	cleanedCount := 0

	// Iterate through heights that need cleanup
	// We clean up one block at a time to avoid unbounded gas consumption
	// The cleanup will catch up over multiple blocks if needed
	for height := cutoffHeight - 10; height < cutoffHeight; height++ {
		if height <= 0 {
			continue
		}

		// Get all nonces at this height
		heightPrefix := NonceByHeightPrefixForHeight(height)
		iterator := storetypes.KVStorePrefixIterator(store, heightPrefix)
		defer iterator.Close()

		noncesToDelete := [][]byte{}
		for ; iterator.Valid(); iterator.Next() {
			noncesToDelete = append(noncesToDelete, iterator.Key())
			cleanedCount++
		}

		// Delete the nonces
		for _, key := range noncesToDelete {
			store.Delete(key)

			// Also delete the actual nonce entry
			// Extract provider and nonce from the index key
			// Format: NonceByHeightPrefix(1) + height(8) + provider(20) + nonce(8)
			if len(key) >= 37 { // 1 + 8 + 20 + 8
				providerStart := 9  // After prefix(1) + height(8)
				providerEnd := 29   // providerStart + 20
				nonceStart := 29

				if len(key) >= nonceStart+8 {
					provider := sdk.AccAddress(key[providerStart:providerEnd])
					nonceBytes := key[nonceStart : nonceStart+8]
					nonce := binary.BigEndian.Uint64(nonceBytes)

					// Delete the actual nonce entry
					nonceKey := NonceKey(provider, nonce)
					store.Delete(nonceKey)
				}
			}
		}
	}

	// Emit event with cleanup statistics
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"nonces_cleaned",
			sdk.NewAttribute("cutoff_height", fmt.Sprintf("%d", cutoffHeight)),
			sdk.NewAttribute("current_height", fmt.Sprintf("%d", currentHeight)),
			sdk.NewAttribute("nonces_cleaned", fmt.Sprintf("%d", cleanedCount)),
		),
	)

	return nil
}

// UpdateProviderReputations updates reputation scores for all providers
func (k Keeper) UpdateProviderReputations(ctx context.Context) error {
	// This is called in BeginBlock to update provider reputation scores
	// based on their performance in the previous blocks
	return k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
		// Provider reputation decay or updates would go here
		// For now, this is a placeholder for the reputation system
		return false, nil
	})
}

// ProcessPendingDisputes processes any disputes that need resolution
func (k Keeper) ProcessPendingDisputes(ctx context.Context) error {
	// sdkCtx := sdk.UnwrapSDKContext(ctx)

	/*
	// Iterate through disputes and check for resolution timeouts
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, DisputeKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var dispute types.Dispute
		if err := k.cdc.Unmarshal(iter.Value(), &dispute); err != nil {
			continue
		}
	*/

		// Check if dispute has timed out (30 days = 2,592,000 seconds)
		/*
		const DisputeTimeoutSeconds = 2592000
		
		if sdkCtx.BlockTime().Unix()-dispute.CreatedAt > DisputeTimeoutSeconds {
			if dispute.Status == types.DisputeStatus_PENDING {
				// Auto-resolve timed out disputes in favor of requester
				dispute.Status = types.DisputeStatus_RESOLVED
				dispute.Resolution = "Auto-resolved due to timeout"

				// Save updated dispute
				bz, err := k.cdc.Marshal(&dispute)
				if err != nil {
					continue
				}
				store.Set(iter.Key(), bz)

				// Emit event
				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"dispute_auto_resolved",
						sdk.NewAttribute("dispute_id", fmt.Sprintf("%d", dispute.Id)),
						sdk.NewAttribute("request_id", fmt.Sprintf("%d", dispute.RequestId)),
					),
				)
			}
		}
		*/
	return nil
}
