package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// BeginBlocker is called at the beginning of every block
// It handles time-based initialization and scheduled tasks
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Refresh provider reputation cache if needed
	// This is done first to ensure fresh cache for incoming requests
	shouldRefresh, err := k.ShouldRefreshCache(ctx)
	if err != nil {
		sdkCtx.Logger().Error("failed to check cache refresh", "error", err)
	} else if shouldRefresh {
		if err := k.RefreshProviderCache(ctx); err != nil {
			sdkCtx.Logger().Error("failed to refresh provider cache", "error", err)
			// Don't return error - log and continue
		}
	}

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

	// Cleanup old request rate limit data to prevent state bloat
	if err := k.CleanupOldRequestRateLimitData(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup old request rate limit data", "error", err)
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
// It uses the configurable nonce_retention_blocks parameter from module params
func (k Keeper) CleanupExpiredNonces(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get retention period from params
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	retentionBlocks := params.NonceRetentionBlocks
	if retentionBlocks <= 0 {
		retentionBlocks = 17280 // Default to ~24 hours
	}

	cutoffHeight := currentHeight - retentionBlocks

	if cutoffHeight <= 0 {
		return nil // Don't cleanup in early blocks
	}

	store := k.getStore(ctx)
	cleanedCount := 0

	// Clean up in batches to avoid unbounded gas consumption
	// Process up to 100 blocks worth of nonces per cleanup cycle
	const maxBlocksPerCleanup = 100
	startHeight := cutoffHeight - maxBlocksPerCleanup
	if startHeight < 0 {
		startHeight = 0
	}

	// Iterate through heights that need cleanup
	for height := startHeight; height < cutoffHeight; height++ {
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
			// Format: NonceByHeightPrefix(2) + height(8) + provider(20) + nonce(8)
			if len(key) >= 38 { // 2 + 8 + 20 + 8
				providerStart := 10 // After prefix(2) + height(8)
				providerEnd := 30   // providerStart + 20
				nonceStart := 30

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

	// Update metrics
	k.metrics.RecordNonceCleanup(cleanedCount)

	// Emit event with cleanup statistics
	if cleanedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"nonces_cleaned",
				sdk.NewAttribute("cutoff_height", fmt.Sprintf("%d", cutoffHeight)),
				sdk.NewAttribute("current_height", fmt.Sprintf("%d", currentHeight)),
				sdk.NewAttribute("nonces_cleaned", fmt.Sprintf("%d", cleanedCount)),
				sdk.NewAttribute("retention_blocks", fmt.Sprintf("%d", retentionBlocks)),
			),
		)
	}

	return nil
}

// UpdateProviderReputations updates reputation scores for all providers
// This implements a comprehensive reputation system based on:
// - Success rate (completed jobs / total jobs)
// - Response time (time to complete jobs)
// - Dispute history (disputes lost vs total)
// - Uptime (time since last activity vs total time since registration)
func (k Keeper) UpdateProviderReputations(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	now := sdkCtx.BlockTime()

	// Get module parameters for reputation calculation
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Reputation update interval (every 100 blocks for efficiency)
	const reputationUpdateInterval = 100
	if currentHeight%reputationUpdateInterval != 0 {
		return nil
	}

	updatedCount := 0
	totalProviders := 0

	iterErr := k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
		totalProviders++

		// Calculate reputation factors
		successRate := k.calculateSuccessRate(ctx, provider.Address)
		disputeScore := k.calculateDisputeScore(ctx, provider.Address)
		uptimeScore := k.calculateUptimeScore(ctx, provider, now)

		// Weighted reputation calculation
		// Success rate: 40%, Dispute score: 30%, Uptime: 30%
		const (
			successWeight = 40
			disputeWeight = 30
			uptimeWeight  = 30
		)

		newReputation := (successRate*successWeight + disputeScore*disputeWeight + uptimeScore*uptimeWeight) / 100

		// Apply reputation decay for inactive providers (>16 hours of inactivity)
		timeSinceActive := now.Sub(provider.LastActiveAt)
		inactiveThreshold := 16 * 60 * 60 // 16 hours in seconds
		if timeSinceActive.Seconds() > float64(inactiveThreshold) {
			excessInactive := timeSinceActive.Seconds() - float64(inactiveThreshold)
			decayFactor := excessInactive / (50 * 60 * 60) // Gradual decay over 50 hours
			if decayFactor > 0.5 {
				decayFactor = 0.5 // Max 50% decay
			}
			newReputation = uint32(float64(newReputation) * (1 - decayFactor))
		}

		// Ensure reputation is within bounds [0, 100]
		if newReputation > 100 {
			newReputation = 100
		}

		// Check if provider meets minimum reputation threshold
		oldReputation := provider.Reputation
		minReputation := params.MinReputationScore
		if minReputation == 0 {
			minReputation = 10 // Default minimum
		}

		// Update provider reputation
		if newReputation != oldReputation {
			provider.Reputation = newReputation

			// Deactivate provider if reputation falls below threshold
			if newReputation < uint32(minReputation) && provider.Active {
				provider.Active = false
				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"provider_deactivated_low_reputation",
						sdk.NewAttribute("provider", provider.Address),
						sdk.NewAttribute("reputation", fmt.Sprintf("%d", newReputation)),
						sdk.NewAttribute("threshold", fmt.Sprintf("%d", minReputation)),
					),
				)
			}

			// Save updated provider
			providerAddr, addrErr := sdk.AccAddressFromBech32(provider.Address)
			if addrErr != nil {
				return false, nil // Continue to next provider
			}
			if setErr := k.SetProvider(ctx, provider); setErr != nil {
				sdkCtx.Logger().Error("failed to update provider reputation",
					"provider", provider.Address,
					"error", setErr,
				)
				return false, nil // Continue to next provider
			}

			updatedCount++

			// Emit reputation change event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"provider_reputation_updated",
					sdk.NewAttribute("provider", providerAddr.String()),
					sdk.NewAttribute("old_reputation", fmt.Sprintf("%d", oldReputation)),
					sdk.NewAttribute("new_reputation", fmt.Sprintf("%d", newReputation)),
					sdk.NewAttribute("success_rate", fmt.Sprintf("%d", successRate)),
					sdk.NewAttribute("dispute_score", fmt.Sprintf("%d", disputeScore)),
					sdk.NewAttribute("uptime_score", fmt.Sprintf("%d", uptimeScore)),
				),
			)
		}

		return false, nil // Continue iterating
	})

	if iterErr != nil {
		return fmt.Errorf("failed to iterate providers for reputation update: %w", iterErr)
	}

	// Emit summary event
	if updatedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"reputation_update_summary",
				sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
				sdk.NewAttribute("total_providers", fmt.Sprintf("%d", totalProviders)),
				sdk.NewAttribute("updated_count", fmt.Sprintf("%d", updatedCount)),
			),
		)
	}

	return nil
}

// calculateSuccessRate calculates the success rate for a provider (0-100)
func (k Keeper) calculateSuccessRate(ctx context.Context, providerAddr string) uint32 {
	stats, err := k.GetProviderStats(ctx, providerAddr)
	if err != nil || stats == nil {
		return 50 // Default to neutral score if no stats
	}

	totalJobs := stats.TotalJobsCompleted + stats.TotalJobsFailed
	if totalJobs == 0 {
		return 50 // No history, default to neutral
	}

	// Calculate success percentage
	successRate := (stats.TotalJobsCompleted * 100) / totalJobs
	return types.SaturateUint64ToUint32(successRate)
}

// calculateDisputeScore calculates reputation impact from disputes (0-100)
// Lower disputes = higher score
func (k Keeper) calculateDisputeScore(ctx context.Context, providerAddr string) uint32 {
	stats, err := k.GetProviderStats(ctx, providerAddr)
	if err != nil || stats == nil {
		return 100 // No disputes = perfect score
	}

	totalDisputes := stats.TotalDisputes
	disputesLost := stats.DisputesLost

	if totalDisputes == 0 {
		return 100 // No disputes = perfect score
	}

	// Calculate score based on dispute loss rate
	// 0 disputes lost = 100, all disputes lost = 0
	disputeLossRate := (disputesLost * 100) / totalDisputes
	score := int64(100 - types.SaturateUint64ToUint32(disputeLossRate))

	// Additional penalty for high dispute volume
	// More than 10 disputes total starts reducing score
	if totalDisputes > 10 {
		penaltyBase := types.SaturateUint64ToInt64(totalDisputes - 10)
		penalty := penaltyBase * 2
		if penalty > 30 {
			penalty = 30 // Max 30% penalty
		}
		score = score - penalty
		if score < 0 {
			score = 0
		}
	}

	return types.SaturateInt64ToUint32(score)
}

// calculateUptimeScore calculates uptime score (0-100) based on activity recency
func (k Keeper) calculateUptimeScore(_ context.Context, provider types.Provider, now time.Time) uint32 {
	// Calculate time since registration
	timeSinceRegistration := now.Sub(provider.RegisteredAt)
	if timeSinceRegistration.Seconds() <= 0 {
		return 100 // Just registered
	}

	// Check time since last activity
	// A provider is considered active if they've processed jobs recently
	timeSinceActive := now.Sub(provider.LastActiveAt)
	minutesSinceActive := timeSinceActive.Minutes()

	// Score based on activity recency
	// < 10 minutes: Very active (100)
	// < 1 hour: Active (90)
	// < 4 hours: Somewhat active (70)
	// < 8 hours: Marginally active (50)
	// > 8 hours: Inactive (30)
	switch {
	case minutesSinceActive <= 10:
		return 100 // Very active
	case minutesSinceActive <= 60:
		return 90 // Active
	case minutesSinceActive <= 240:
		return 70 // Somewhat active
	case minutesSinceActive <= 480:
		return 50 // Marginally active
	default:
		return 30 // Inactive
	}
}

// ProviderStats tracks provider performance metrics
type ProviderStats struct {
	TotalJobsCompleted uint64
	TotalJobsFailed    uint64
	TotalDisputes      uint64
	DisputesLost       uint64
	TotalEarnings      uint64
	AverageJobTime     uint64 // in seconds
}

// GetProviderStats retrieves stats for a provider
func (k Keeper) GetProviderStats(ctx context.Context, providerAddr string) (*ProviderStats, error) {
	store := k.getStore(ctx)
	key := ProviderStatsKey(providerAddr)

	bz := store.Get(key)
	if bz == nil {
		return nil, nil
	}

	var stats ProviderStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider stats: %w", err)
	}

	return &stats, nil
}

// SetProviderStats stores stats for a provider
func (k Keeper) SetProviderStats(ctx context.Context, providerAddr string, stats *ProviderStats) error {
	store := k.getStore(ctx)
	key := ProviderStatsKey(providerAddr)

	bz, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal provider stats: %w", err)
	}

	store.Set(key, bz)
	return nil
}

// IncrementProviderJobCompleted increments the job completed count
func (k Keeper) IncrementProviderJobCompleted(ctx context.Context, providerAddr string) error {
	stats, err := k.GetProviderStats(ctx, providerAddr)
	if err != nil {
		stats = &ProviderStats{}
	}
	if stats == nil {
		stats = &ProviderStats{}
	}

	stats.TotalJobsCompleted++
	return k.SetProviderStats(ctx, providerAddr, stats)
}

// IncrementProviderJobFailed increments the job failed count
func (k Keeper) IncrementProviderJobFailed(ctx context.Context, providerAddr string) error {
	stats, err := k.GetProviderStats(ctx, providerAddr)
	if err != nil {
		stats = &ProviderStats{}
	}
	if stats == nil {
		stats = &ProviderStats{}
	}

	stats.TotalJobsFailed++
	return k.SetProviderStats(ctx, providerAddr, stats)
}

// IncrementProviderDispute increments the dispute counts
func (k Keeper) IncrementProviderDispute(ctx context.Context, providerAddr string, lost bool) error {
	stats, err := k.GetProviderStats(ctx, providerAddr)
	if err != nil {
		stats = &ProviderStats{}
	}
	if stats == nil {
		stats = &ProviderStats{}
	}

	stats.TotalDisputes++
	if lost {
		stats.DisputesLost++
	}
	return k.SetProviderStats(ctx, providerAddr, stats)
}

// ProcessPendingDisputes processes any disputes that need resolution
func (k Keeper) ProcessPendingDisputes(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, DisputeKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var dispute types.Dispute
		if err := k.cdc.Unmarshal(iter.Value(), &dispute); err != nil {
			continue
		}

		switch dispute.Status {
		case types.DISPUTE_STATUS_EVIDENCE_SUBMISSION:
			if now.After(dispute.EvidenceEndsAt) {
				dispute.Status = types.DISPUTE_STATUS_VOTING
				_ = k.setDispute(ctx, dispute)
			}
		case types.DISPUTE_STATUS_VOTING:
			if now.After(dispute.VotingEndsAt) {
				// auto-resolve if authority configured
				authAddr, err := sdk.AccAddressFromBech32(k.authority)
				if err != nil {
					continue
				}
				if res, err := k.ResolveDispute(ctx, authAddr, dispute.Id); err == nil {
					_ = k.SettleDisputeOutcome(ctx, dispute.Id, res)
				}
			}
		}
	}

	return nil
}
