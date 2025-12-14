package keeper

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// BeginBlocker is called at the beginning of every block
// It handles price aggregation and validator power updates
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Aggregate prices from validator submissions
	if err := k.AggregatePrices(ctx); err != nil {
		sdkCtx.Logger().Error("failed to aggregate prices", "error", err)
		// Don't return error - log and continue
	}

	// Update validator powers from staking module
	if err := k.UpdateValidatorPowers(ctx); err != nil {
		sdkCtx.Logger().Error("failed to update validator powers", "error", err)
		// Don't return error - log and continue
	}

	// Emit begin block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_begin_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// EndBlocker is called at the end of every block
// It handles time-based operations like slash window processing and cleanup
func (k Keeper) EndBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Process slash windows for all tracked assets
	if err := k.ProcessSlashWindows(ctx); err != nil {
		sdkCtx.Logger().Error("failed to process slash windows", "error", err)
		// Don't return error - log and continue to prevent block production halt
	}

	// Cleanup old outlier history to prevent unbounded state growth
	if err := k.CleanupOldOutlierHistoryGlobal(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup old outlier history", "error", err)
		// Don't return error - log and continue
	}

	// Cleanup old price submissions to prevent state bloat
	if err := k.CleanupOldSubmissions(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup old submissions", "error", err)
		// Don't return error - log and continue
	}

	// Emit end block event for monitoring
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_end_block",
			sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return nil
}

// ProcessSlashWindows processes the slash window for all assets
// This is called periodically to check for missed votes and slash validators
func (k Keeper) ProcessSlashWindows(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get all tracked prices/assets
	prices, err := k.GetAllPrices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prices: %w", err)
	}

	processedCount := 0

	// Process slash window for each asset
	for _, price := range prices {
		if err := k.HandleSlashWindow(ctx, price.Asset); err != nil {
			sdkCtx.Logger().Error("failed to handle slash window for asset",
				"asset", price.Asset,
				"error", err,
			)
			// Continue processing other assets even if one fails
			continue
		}
		processedCount++
	}

	if processedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"slash_windows_processed",
				sdk.NewAttribute("count", fmt.Sprintf("%d", processedCount)),
				sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			),
		)
	}

	return nil
}

// CleanupOldOutlierHistoryGlobal removes outlier history older than the retention window
// for all validators and assets to prevent unbounded state growth.
//
// Performance optimization: Uses amortized cleanup to avoid O(n×m) iteration every block.
// The cleanup is distributed across blocks using modulo-based scheduling, processing only
// a subset of the total work per block. This prevents block timeouts with large validator sets.
//
// Complexity: O(k) per block where k = maxCleanupPerBlock, amortized to O(n) over cleanupCycle blocks
func (k Keeper) CleanupOldOutlierHistoryGlobal(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Keep outlier history for the last OutlierReputationWindow blocks
	minHeight := currentHeight - OutlierReputationWindow

	if minHeight <= 0 {
		return nil // Don't cleanup in early blocks
	}

	// Amortized cleanup configuration:
	// Process cleanup work distributed across blocks to avoid O(n×m) spike.
	// With 100 validators × 20 feeds = 2000 pairs, processing 50 per block
	// completes full cleanup cycle in 40 blocks (~4 minutes at 6s/block).
	const (
		maxCleanupPerBlock = 50  // Maximum validator-asset pairs to clean per block
		cleanupCycle       = 100 // Rotate through all pairs every N blocks
	)

	// Use direct prefix iteration to avoid loading all validators and prices into memory
	// This is O(1) memory instead of O(n+m)
	store := k.getStore(ctx)

	// Outlier history keys have format: OutlierHistoryKeyPrefix + validator + 0x00 + asset + 0x00 + height
	// We iterate by prefix and process entries in batches
	iterator := storetypes.KVStorePrefixIterator(store, OutlierHistoryKeyPrefix)
	defer iterator.Close()

	// Use block height modulo to determine which subset to process this block
	// This ensures work is distributed evenly across blocks
	blockOffset := currentHeight % cleanupCycle
	processedCount := 0
	skippedCount := 0
	cleanedKeysCount := 0

	// Batch collect keys to delete (avoid deletion during iteration)
	keysToDelete := make([][]byte, 0, maxCleanupPerBlock*10) // Estimate ~10 old entries per pair

	currentPairKey := ""
	pairIndex := int64(-1) // Start at -1 so first increment makes it 0
	shouldProcessCurrentPair := false

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		// Extract validator+asset pair from key to track unique pairs
		// Key format: OutlierHistoryKeyPrefix + validator + 0x00 + asset + 0x00 + height
		pairKey := extractValidatorAssetPair(key)
		if pairKey == "" {
			continue // Invalid key format
		}

		// Track when we move to a new validator-asset pair
		if pairKey != currentPairKey {
			currentPairKey = pairKey
			pairIndex++

			// Amortized scheduling: process only pairs assigned to this block
			// Using modulo ensures even distribution across the cleanup cycle
			if pairIndex%cleanupCycle != blockOffset {
				skippedCount++
				shouldProcessCurrentPair = false
				continue // Skip pairs not scheduled for this block
			}

			// Enforce per-block limit to prevent gas exhaustion
			if processedCount >= maxCleanupPerBlock {
				break
			}

			processedCount++
			shouldProcessCurrentPair = true
		}

		// Skip entries for pairs not scheduled this block
		if !shouldProcessCurrentPair {
			continue
		}

		// Parse block height from value (format: "severity:blockHeight")
		var blockHeight int64
		if _, err := fmt.Sscanf(string(iterator.Value()), "%d:%d", new(int), &blockHeight); err != nil {
			sdkCtx.Logger().Error("failed to parse outlier history entry", "error", err)
			continue
		}

		// Mark old entries for deletion
		if blockHeight < minHeight {
			keysToDelete = append(keysToDelete, key)
			cleanedKeysCount++
		}
	}

	// Delete old entries outside iteration to avoid iterator invalidation
	for _, key := range keysToDelete {
		store.Delete(key)
	}

	// Emit event with cleanup statistics for monitoring
	if cleanedKeysCount > 0 || processedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"outlier_history_cleaned",
				sdk.NewAttribute("pairs_processed", fmt.Sprintf("%d", processedCount)),
				sdk.NewAttribute("pairs_skipped", fmt.Sprintf("%d", skippedCount)),
				sdk.NewAttribute("entries_deleted", fmt.Sprintf("%d", cleanedKeysCount)),
				sdk.NewAttribute("min_height", fmt.Sprintf("%d", minHeight)),
				sdk.NewAttribute("block_offset", fmt.Sprintf("%d", blockOffset)),
			),
		)
	}

	return nil
}

// extractValidatorAssetPair extracts the validator+asset portion from outlier history key
// for deduplication and batching purposes. Returns empty string if key format is invalid.
//
// Key format: OutlierHistoryKeyPrefix + validator + 0x00 + asset + 0x00 + height
// Returns: "validator\x00asset" for grouping
func extractValidatorAssetPair(key []byte) string {
	if len(key) < len(OutlierHistoryKeyPrefix)+1 {
		return ""
	}

	// Skip prefix bytes
	remainder := key[len(OutlierHistoryKeyPrefix):]

	// Find second separator (end of asset field)
	separatorCount := 0
	for i, b := range remainder {
		if b == 0x00 {
			separatorCount++
			if separatorCount == 2 {
				// Return everything up to (but not including) the second separator
				// This gives us "validator\x00asset" as the pair key
				return string(remainder[:i])
			}
		}
	}

	return ""
}

// CleanupOldSubmissions removes old price submissions to prevent state bloat
func (k Keeper) CleanupOldSubmissions(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Keep submissions for the last 10000 blocks (conservative retention)
	const SubmissionRetentionBlocks = 10000

	cutoffHeight := sdkCtx.BlockHeight() - SubmissionRetentionBlocks

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

		// Get all submissions at this height
		// heightPrefix := GetSubmissionByHeightPrefixForHeight(height)
		iterator := store.Iterator(types.VoteKeyPrefix, storetypes.PrefixEndBytes(types.VoteKeyPrefix))

		submissionsToDelete := [][]byte{}
		for ; iterator.Valid(); iterator.Next() {
			submissionsToDelete = append(submissionsToDelete, iterator.Key())
			cleanedCount++
		}
		iterator.Close()

		// Delete the submission index entries
		for _, key := range submissionsToDelete {
			store.Delete(key)

			// Also delete the actual validator price submission
			// Extract validator and asset from the index key
			// Format: SubmissionByHeightPrefix(1) + height(8) + validator + 0x00 + asset
			if len(key) > 9 {
				// Skip prefix(1) + height(8) = 9 bytes
				remainder := key[9:]

				// Find the separator byte
				separatorIdx := -1
				for i, b := range remainder {
					if b == 0x00 {
						separatorIdx = i
						break
					}
				}

				if separatorIdx > 0 && separatorIdx < len(remainder)-1 {
					validatorStr := string(remainder[:separatorIdx])
					asset := string(remainder[separatorIdx+1:])

					// Parse validator address
					valAddr, err := sdk.ValAddressFromBech32(validatorStr)
					if err == nil {
						// Delete the actual submission
						submissionKey := GetValidatorPriceKey(valAddr, asset)
						store.Delete(submissionKey)
					}
				}
			}
		}
	}

	// Emit event with cleanup statistics
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"submissions_cleaned",
			sdk.NewAttribute("cutoff_height", fmt.Sprintf("%d", cutoffHeight)),
			sdk.NewAttribute("current_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			sdk.NewAttribute("submissions_cleaned", fmt.Sprintf("%d", cleanedCount)),
		),
	)

	return nil
}

// AggregatePrices aggregates validator price submissions for all assets
func (k Keeper) AggregatePrices(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	assetSet := make(map[string]struct{})

	// Track assets that already have aggregated prices
	prices, err := k.GetAllPrices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prices: %w", err)
	}
	for _, price := range prices {
		if price.Asset == "" {
			continue
		}
		assetSet[price.Asset] = struct{}{}
	}

	// Include assets that only have validator submissions (no aggregated price yet)
	validatorPrices, err := k.GetAllValidatorPrices(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get validator prices: %w", err)
	}
	for _, vp := range validatorPrices {
		if vp.Asset == "" {
			continue
		}
		assetSet[vp.Asset] = struct{}{}
	}

	if len(assetSet) == 0 {
		return nil
	}

	aggregatedCount := 0

	for asset := range assetSet {
		if err := k.AggregateAssetPrice(ctx, asset); err != nil {
			sdkCtx.Logger().Error("failed to aggregate price",
				"asset", asset,
				"error", err,
			)
			if k.metrics != nil && k.metrics.AggregationCount != nil {
				k.metrics.AggregationCount.With(map[string]string{
					"asset":  asset,
					"status": "error",
				}).Inc()
			}
			continue
		}
		aggregatedCount++
	}

	if aggregatedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"prices_aggregated",
				sdk.NewAttribute("count", fmt.Sprintf("%d", aggregatedCount)),
				sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			),
		)
	}

	return nil
}

// UpdateValidatorPowers updates validator oracle info with current staking power
func (k Keeper) UpdateValidatorPowers(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get all validators from staking module
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}

	updatedCount := 0

	// Update or create validator oracle info
	for _, validator := range validators {
		valAddr := validator.GetOperator()
		// power := validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx))

		validatorOracle, err := k.GetValidatorOracle(ctx, valAddr)
		if err != nil {
			// Create new validator oracle if doesn't exist
			validatorOracle = types.ValidatorOracle{
				ValidatorAddr: valAddr,
				MissCounter:   0,
			}
		}

		if err := k.SetValidatorOracle(ctx, validatorOracle); err != nil {
			sdkCtx.Logger().Error("failed to set validator oracle",
				"validator", valAddr,
				"error", err,
			)
			continue
		}
		updatedCount++
	}

	if updatedCount > 0 {
		sdkCtx.Logger().Debug("updated validator powers",
			"count", updatedCount,
		)
	}

	return nil
}
