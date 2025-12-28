package keeper

import (
	"context"
	"fmt"
	"sort"
	"sync"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// maxAggregationWorkers is the maximum number of parallel goroutines for price aggregation.
// PERF-8: Using 4 workers provides good parallelism without overwhelming CPU resources.
const maxAggregationWorkers = 4

// AssetAggregationResult holds the computed result from parallel asset price aggregation.
// PERF-8: This struct captures computation results so writes can be serialized.
type AssetAggregationResult struct {
	Asset          string
	Price          types.Price
	Snapshot       types.PriceSnapshot
	FilteredData   *FilteredPriceData
	AggregatedDec  sdkmath.LegacyDec
	MinHeight      int64
	Err            error
}

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

	// Check geographic diversity periodically (P2-SEC-2 mitigation)
	params, err := k.GetParams(ctx)
	if err == nil && params.DiversityCheckInterval > 0 {
		// Check diversity every N blocks
		if sdkCtx.BlockHeight()%int64(params.DiversityCheckInterval) == 0 {
			if err := k.MonitorGeographicDiversity(ctx); err != nil {
				sdkCtx.Logger().Error("geographic diversity monitoring failed", "error", err)
				// Don't return error - log and continue
			}
		}
	}

	// Prune expired GeoIP cache entries periodically (P2-PERF-1 mitigation)
	// Prune every 100 blocks (~10 minutes) to prevent unbounded cache growth
	if err == nil && sdkCtx.BlockHeight()%100 == 0 {
		if k.geoIPManager != nil {
			pruned := k.geoIPManager.PruneCacheExpired()
			if pruned > 0 {
				sdkCtx.Logger().Debug("pruned expired GeoIP cache entries",
					"pruned", pruned,
					"height", sdkCtx.BlockHeight(),
				)
				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"geoip_cache_pruned",
						sdk.NewAttribute("count", fmt.Sprintf("%d", pruned)),
						sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
					),
				)
			}
		}
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

	// Prune expired IBC nonces to prevent unbounded state growth
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	if err != nil {
		sdkCtx.Logger().Error("failed to prune expired nonces", "error", err)
		// Don't return error - log and continue
	} else if prunedCount > 0 {
		sdkCtx.Logger().Info("pruned expired nonces", "count", prunedCount)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"nonces_pruned",
				sdk.NewAttribute("count", fmt.Sprintf("%d", prunedCount)),
				sdk.NewAttribute("height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			),
		)
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
// SEC-9: Uses MaxOutlierHistoryBlocks constant to control retention period.
//
// Performance optimization: Uses amortized cleanup to avoid O(n×m) iteration every block.
// The cleanup is distributed across blocks using modulo-based scheduling, processing only
// a subset of the total work per block. This prevents block timeouts with large validator sets.
//
// Complexity: O(k) per block where k = maxCleanupPerBlock, amortized to O(n) over cleanupCycle blocks
func (k Keeper) CleanupOldOutlierHistoryGlobal(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// SEC-9: Keep outlier history for the last MaxOutlierHistoryBlocks blocks
	// History older than this is cleaned up to prevent unbounded state growth
	minHeight := currentHeight - MaxOutlierHistoryBlocks

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
		defer iterator.Close()

		// P3-PERF-3: Pre-size with estimated capacity (validators * assets)
		submissionsToDelete := make([][]byte, 0, 100)
		for ; iterator.Valid(); iterator.Next() {
			submissionsToDelete = append(submissionsToDelete, iterator.Key())
			cleanedCount++
		}
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

// AggregatePrices aggregates validator price submissions for all assets using parallel processing.
// PERF-8: Uses a worker pool to compute aggregations in parallel, then serializes writes.
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

	// PERF-8: Convert set to slice for parallel processing
	assets := make([]string, 0, len(assetSet))
	for asset := range assetSet {
		assets = append(assets, asset)
	}

	// PERF-8: Sort assets for deterministic ordering of writes
	// Map iteration order is random in Go, so we must sort for consensus
	sort.Strings(assets)

	// PERF-8: Parallel aggregation using worker pool
	results := k.aggregateAssetsParallel(sdkCtx, assets)

	// PERF-8: Apply results sequentially (writes must be serialized for determinism)
	aggregatedCount := 0
	for _, result := range results {
		if result.Err != nil {
			sdkCtx.Logger().Error("failed to aggregate price",
				"asset", result.Asset,
				"error", result.Err,
			)
			if k.metrics != nil && k.metrics.AggregationCount != nil {
				k.metrics.AggregationCount.With(map[string]string{
					"asset":  result.Asset,
					"status": "error",
				}).Inc()
			}
			continue
		}

		// Apply writes for successful aggregation
		if err := k.applyAggregationResult(ctx, sdkCtx, result); err != nil {
			sdkCtx.Logger().Error("failed to apply aggregation result",
				"asset", result.Asset,
				"error", err,
			)
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

// aggregateAssetsParallel computes aggregations for multiple assets in parallel.
// PERF-8: Uses a bounded worker pool to limit concurrency and prevent CPU overload.
// Reads are done in parallel using CacheContext for isolation; writes are returned for
// sequential application to maintain deterministic state.
func (k Keeper) aggregateAssetsParallel(sdkCtx sdk.Context, assets []string) []AssetAggregationResult {
	numAssets := len(assets)
	if numAssets == 0 {
		return nil
	}

	// Pre-allocate results slice
	results := make([]AssetAggregationResult, numAssets)

	// For small numbers of assets, process sequentially to avoid goroutine overhead
	if numAssets <= 2 {
		for i, asset := range assets {
			results[i] = k.computeAssetAggregation(sdkCtx, asset)
		}
		return results
	}

	// PERF-8: Worker pool pattern with semaphore for bounded concurrency
	sem := make(chan struct{}, maxAggregationWorkers)
	var wg sync.WaitGroup

	for i, asset := range assets {
		wg.Add(1)
		go func(idx int, a string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			// PERF-8: Use CacheContext for read isolation in each goroutine
			// CacheContext creates an isolated cache - reads are safe but writes won't persist
			cacheCtx, _ := sdkCtx.CacheContext()
			results[idx] = k.computeAssetAggregation(cacheCtx, a)
		}(i, asset)
	}

	wg.Wait()
	return results
}

// computeAssetAggregation performs the read-heavy computation for a single asset.
// PERF-8: This function is safe to call from multiple goroutines with CacheContext.
// It returns all computed data needed for writes without performing any writes itself.
func (k Keeper) computeAssetAggregation(sdkCtx sdk.Context, asset string) AssetAggregationResult {
	result := AssetAggregationResult{Asset: asset}

	validatorPrices, err := k.GetValidatorPricesByAsset(sdkCtx, asset)
	if err != nil {
		result.Err = err
		return result
	}

	if len(validatorPrices) == 0 {
		result.Err = fmt.Errorf("no price submissions for asset: %s", asset)
		return result
	}

	params, err := k.GetParams(sdkCtx)
	if err != nil {
		result.Err = err
		return result
	}

	totalVotingPower, validPrices, err := k.calculateVotingPower(sdkCtx, validatorPrices)
	if err != nil {
		result.Err = err
		return result
	}

	if len(validPrices) == 0 {
		result.Err = fmt.Errorf("no valid price submissions for asset: %s", asset)
		return result
	}

	submittedVotingPower := int64(0)
	for _, vp := range validPrices {
		submittedVotingPower += vp.VotingPower
	}

	votePercentage := sdkmath.LegacyNewDec(submittedVotingPower).Quo(sdkmath.LegacyNewDec(totalVotingPower))
	if votePercentage.LT(params.VoteThreshold) {
		result.Err = fmt.Errorf("insufficient voting power: %s < %s", votePercentage.String(), params.VoteThreshold.String())
		return result
	}

	// Multi-stage statistical outlier detection
	filteredData, err := k.detectAndFilterOutliers(sdkCtx, asset, validPrices)
	if err != nil {
		result.Err = err
		return result
	}

	if len(filteredData.ValidPrices) == 0 {
		result.Err = fmt.Errorf("all prices filtered as outliers for asset: %s", asset)
		return result
	}

	// Calculate weighted median from filtered prices
	aggregatedPrice, err := k.calculateWeightedMedian(filteredData.ValidPrices)
	if err != nil {
		result.Err = err
		return result
	}

	// Prepare result struct for later write application
	result.Price = types.Price{
		Asset:         asset,
		Price:         aggregatedPrice,
		BlockHeight:   sdkCtx.BlockHeight(),
		BlockTime:     sdkCtx.BlockTime().Unix(),
		NumValidators: uint32(len(filteredData.ValidPrices)),
	}

	result.Snapshot = types.PriceSnapshot{
		Asset:       asset,
		Price:       aggregatedPrice,
		BlockHeight: sdkCtx.BlockHeight(),
		BlockTime:   sdkCtx.BlockTime().Unix(),
	}

	result.FilteredData = filteredData
	result.AggregatedDec = aggregatedPrice
	result.MinHeight = sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)

	return result
}

// applyAggregationResult writes the computed aggregation result to state.
// PERF-8: This function must be called sequentially to maintain deterministic state ordering.
func (k Keeper) applyAggregationResult(ctx context.Context, sdkCtx sdk.Context, result AssetAggregationResult) error {
	// Handle outlier slashing and emit events
	for _, outlier := range result.FilteredData.FilteredOutliers {
		if err := k.handleOutlierSlashing(ctx, result.Asset, outlier); err != nil {
			sdkCtx.Logger().Error("failed to slash outlier validator",
				"validator", outlier.ValidatorAddr,
				"asset", result.Asset,
				"severity", outlier.Severity,
				"error", err.Error(),
			)
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeOracleOutlier,
				sdk.NewAttribute(types.AttributeKeyValidator, outlier.ValidatorAddr),
				sdk.NewAttribute(types.AttributeKeyAsset, result.Asset),
				sdk.NewAttribute(types.AttributeKeyPrice, outlier.Price.String()),
				sdk.NewAttribute(types.AttributeKeySeverity, fmt.Sprintf("%d", outlier.Severity)),
				sdk.NewAttribute(types.AttributeKeyDeviation, outlier.Deviation.String()),
				sdk.NewAttribute(types.AttributeKeyReason, outlier.Reason),
				sdk.NewAttribute(types.AttributeKeyMedian, result.FilteredData.Median.String()),
				sdk.NewAttribute(types.AttributeKeyMAD, result.FilteredData.MAD.String()),
			),
		)
	}

	// Write aggregated price
	if err := k.SetPrice(ctx, result.Price); err != nil {
		return err
	}

	// Write price snapshot
	if err := k.SetPriceSnapshot(ctx, result.Snapshot); err != nil {
		return err
	}

	// Delete old snapshots
	if err := k.DeleteOldSnapshots(ctx, result.Asset, result.MinHeight); err != nil {
		return err
	}

	// Emit aggregation event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOraclePriceAggregated,
			sdk.NewAttribute(types.AttributeKeyAsset, result.Asset),
			sdk.NewAttribute(types.AttributeKeyPrice, result.AggregatedDec.String()),
			sdk.NewAttribute(types.AttributeKeyNumValidators, fmt.Sprintf("%d", len(result.FilteredData.ValidPrices))),
			sdk.NewAttribute(types.AttributeKeyNumOutliers, fmt.Sprintf("%d", len(result.FilteredData.FilteredOutliers))),
			sdk.NewAttribute(types.AttributeKeyMedian, result.FilteredData.Median.String()),
			sdk.NewAttribute(types.AttributeKeyMAD, result.FilteredData.MAD.String()),
		),
	)

	// Record metrics
	if k.metrics != nil {
		if k.metrics.AggregationCount != nil {
			k.metrics.AggregationCount.With(map[string]string{
				"asset":  result.Asset,
				"status": "success",
			}).Inc()
		}

		if k.metrics.ValidatorParticipation != nil {
			k.metrics.ValidatorParticipation.With(map[string]string{
				"asset": result.Asset,
			}).Set(float64(len(result.FilteredData.ValidPrices)))
		}

		if k.metrics.OutliersDetected != nil {
			severityCounts := make(map[string]float64)
			for _, outlier := range result.FilteredData.FilteredOutliers {
				severityKey := fmt.Sprintf("%d", outlier.Severity)
				severityCounts[severityKey]++
			}
			for severity, count := range severityCounts {
				k.metrics.OutliersDetected.With(map[string]string{
					"asset":    result.Asset,
					"severity": severity,
				}).Add(count)
			}
		}
	}

	return nil
}

// UpdateValidatorPowers updates validator oracle info with current staking power
// and caches the total voting power for efficient price aggregation.
// PERF-2: Total voting power is cached here once per block instead of being
// recalculated O(n*m) times (n validators * m assets) during price aggregation.
func (k Keeper) UpdateValidatorPowers(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get all validators from staking module
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}

	updatedCount := 0
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	totalVotingPower := int64(0)

	// Update or create validator oracle info and compute total voting power
	for _, validator := range validators {
		valAddr := validator.GetOperator()

		// PERF-2: Accumulate total voting power for bonded validators only
		if validator.IsBonded() {
			totalVotingPower += validator.GetConsensusPower(powerReduction)
		}

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

	// PERF-2: Cache total voting power for use in price aggregation
	// This eliminates O(n) iteration per asset during calculateVotingPower
	k.SetCachedTotalVotingPower(ctx, totalVotingPower)

	if updatedCount > 0 {
		sdkCtx.Logger().Debug("updated validator powers",
			"count", updatedCount,
			"total_voting_power", totalVotingPower,
		)
	}

	return nil
}
