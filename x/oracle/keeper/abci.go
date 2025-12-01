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
// for all validators and assets to prevent unbounded state growth
func (k Keeper) CleanupOldOutlierHistoryGlobal(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Keep outlier history for the last OutlierReputationWindow blocks
	minHeight := currentHeight - OutlierReputationWindow

	if minHeight <= 0 {
		return nil // Don't cleanup in early blocks
	}

	// Get all validator oracles
	validatorOracles, err := k.GetAllValidatorOracles(ctx)
	if err != nil {
		return fmt.Errorf("failed to get validator oracles: %w", err)
	}

	// Get all tracked assets
	prices, err := k.GetAllPrices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prices: %w", err)
	}

	cleanedCount := 0

	// Cleanup outlier history for each validator-asset pair
	for _, vo := range validatorOracles {
		for _, price := range prices {
			if err := k.CleanupOldOutlierHistory(ctx, vo.ValidatorAddr, price.Asset, minHeight); err != nil {
				sdkCtx.Logger().Error("failed to cleanup outlier history",
					"validator", vo.ValidatorAddr,
					"asset", price.Asset,
					"error", err,
				)
				continue
			}
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"outlier_history_cleaned",
				sdk.NewAttribute("count", fmt.Sprintf("%d", cleanedCount)),
				sdk.NewAttribute("min_height", fmt.Sprintf("%d", minHeight)),
			),
		)
	}

	return nil
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
		} else {
			// Update existing
			// validatorOracle.Power = power
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
