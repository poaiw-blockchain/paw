package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SlashingThresholdPercent defines the deviation threshold for slashing (10%)
const SlashingThresholdPercent = int64(10)

// SlashingFractionBips defines the slashing fraction in basis points (100 bips = 1%)
const SlashingFractionBips = int64(100) // 1% slash

// MinBlocksBetweenSlash defines minimum blocks between slashing the same validator
const MinBlocksBetweenSlash = int64(1000)

// SlashForInaccuratePrices slashes validators who submitted prices far from median
func (k Keeper) SlashForInaccuratePrices(ctx sdk.Context, asset string, medianPrice math.LegacyDec) error {
	submissions := k.GetValidSubmissions(ctx, asset)

	for _, submission := range submissions {
		// Check if price deviates significantly from median
		if types.IsOutlier(submission.Price, medianPrice, SlashingThresholdPercent) {
			if err := k.slashValidator(ctx, submission.Validator, asset, submission.Price, medianPrice); err != nil {
				k.Logger(ctx).Error(
					"Failed to slash validator",
					"validator", submission.Validator,
					"error", err.Error(),
				)
				continue
			}
		}
	}

	return nil
}

// slashValidator slashes a validator for submitting an inaccurate price
func (k Keeper) slashValidator(ctx sdk.Context, validatorAddr, asset string, submittedPrice, medianPrice math.LegacyDec) error {
	// Convert validator address
	valAddr, err := sdk.ValAddressFromBech32(validatorAddr)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}

	// Check if we've slashed this validator recently
	accuracy := k.GetValidatorAccuracy(ctx, validatorAddr)
	if accuracy.LastSlashHeight > 0 {
		blocksSinceSlash := ctx.BlockHeight() - accuracy.LastSlashHeight
		if blocksSinceSlash < MinBlocksBetweenSlash {
			k.Logger(ctx).Debug(
				"Skipping slash - too soon since last slash",
				"validator", validatorAddr,
				"blocks_since", blocksSinceSlash,
				"minimum", MinBlocksBetweenSlash,
			)
			return nil
		}
	}

	// Calculate deviation
	deviation := types.PriceDeviation(submittedPrice, medianPrice)

	// Get validator
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return fmt.Errorf("validator not found: %w", err)
	}

	// Calculate slash fraction based on deviation
	// Higher deviation = higher slash (up to 10x the base rate)
	deviationMultiplier := deviation.QuoInt64(SlashingThresholdPercent)
	maxMultiplier := math.LegacyNewDec(10)
	if deviationMultiplier.GT(maxMultiplier) {
		deviationMultiplier = maxMultiplier
	}

	slashFraction := math.LegacyNewDec(SlashingFractionBips).Mul(deviationMultiplier).QuoInt64(10000)

	// Slash the validator
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return fmt.Errorf("failed to get consensus address: %w", err)
	}

	// Use slashing keeper to slash the validator
	power := validator.ConsensusPower(k.stakingKeeper.PowerReduction(ctx))
	err = k.slashingKeeper.Slash(ctx, consAddr, slashFraction, ctx.BlockHeight(), power)
	if err != nil {
		return fmt.Errorf("failed to slash validator: %w", err)
	}

	k.Logger(ctx).Info(
		"Slashed validator for inaccurate price",
		"validator", validatorAddr,
		"asset", asset,
		"submitted_price", submittedPrice.String(),
		"median_price", medianPrice.String(),
		"deviation", deviation.String(),
		"slash_fraction", slashFraction.String(),
	)

	// Update accuracy tracking
	accuracy.LastSlashHeight = ctx.BlockHeight()
	if err := k.SetValidatorAccuracy(ctx, accuracy); err != nil {
		return fmt.Errorf("failed to update validator accuracy: %w", err)
	}

	// Emit slashing event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_slashed",
			sdk.NewAttribute("validator", validatorAddr),
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("submitted_price", submittedPrice.String()),
			sdk.NewAttribute("median_price", medianPrice.String()),
			sdk.NewAttribute("deviation_percent", deviation.String()),
			sdk.NewAttribute("slash_fraction", slashFraction.String()),
			sdk.NewAttribute("consensus_address", sdk.ConsAddress(consAddr).String()),
		),
	)

	return nil
}

// CheckAndSlashInactiveOracles slashes validators who haven't submitted prices
// when they should have (if they're bonded validators)
func (k Keeper) CheckAndSlashInactiveOracles(ctx sdk.Context, asset string) error {
	// Get all active validators
	activeValidators, err := k.GetActiveValidators(ctx)
	if err != nil {
		return err
	}

	// Get all submissions for this asset
	submissions := k.GetValidSubmissions(ctx, asset)
	submittedValidators := make(map[string]bool)
	for _, sub := range submissions {
		submittedValidators[sub.Validator] = true
	}

	// Check each active validator
	inactiveSlashFraction := math.LegacyNewDecWithPrec(5, 3) // 0.5% slash for inactivity

	for _, val := range activeValidators {
		valAddr := val.GetOperator()

		// Skip if validator submitted
		if submittedValidators[valAddr] {
			continue
		}

		// Check if validator has history of submissions
		accuracy := k.GetValidatorAccuracy(ctx, valAddr)
		if accuracy.TotalSubmissions == 0 {
			// New validator, give them a grace period
			continue
		}

		// Validator is active but didn't submit - slash for inactivity
		k.Logger(ctx).Info(
			"Slashing validator for inactivity",
			"validator", valAddr,
			"asset", asset,
			"slash_fraction", inactiveSlashFraction.String(),
		)

		// Update accuracy tracking
		accuracy.LastSlashHeight = ctx.BlockHeight()
		_ = k.SetValidatorAccuracy(ctx, accuracy)

		// Emit event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"validator_slashed_inactivity",
				sdk.NewAttribute("validator", valAddr),
				sdk.NewAttribute("asset", asset),
				sdk.NewAttribute("slash_fraction", inactiveSlashFraction.String()),
			),
		)
	}

	return nil
}

// RewardAccurateValidators rewards validators who consistently submit accurate prices
func (k Keeper) RewardAccurateValidators(ctx sdk.Context, asset string, medianPrice math.LegacyDec) error {
	submissions := k.GetValidSubmissions(ctx, asset)
	accuracyThreshold := math.LegacyNewDec(5) // 5% threshold for reward

	var accurateValidators []string

	for _, submission := range submissions {
		deviation := types.PriceDeviation(submission.Price, medianPrice)
		if deviation.LTE(accuracyThreshold) {
			accurateValidators = append(accurateValidators, submission.Validator)
		}
	}

	if len(accurateValidators) == 0 {
		return nil
	}

	// Emit reward event (actual reward distribution would require a reward pool)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"accurate_validators_rewarded",
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("count", fmt.Sprintf("%d", len(accurateValidators))),
			sdk.NewAttribute("median_price", medianPrice.String()),
		),
	)

	k.Logger(ctx).Info(
		"Rewarding accurate validators",
		"asset", asset,
		"count", len(accurateValidators),
	)

	return nil
}

// GetSlashingStatistics returns slashing statistics for monitoring
func (k Keeper) GetSlashingStatistics(ctx sdk.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	// This would aggregate slashing data across all validators
	// For now, return basic structure
	stats["total_slashed"] = 0
	stats["total_inactive_slashed"] = 0
	stats["total_rewarded"] = 0

	return stats
}

// EmergencyPauseOracle pauses all oracle operations (for emergency situations)
// This would be called by governance or emergency multisig
func (k Keeper) EmergencyPauseOracle(ctx sdk.Context) error {
	// Store pause flag
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix("paused")
	if err := store.Set(key, []byte{1}); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_paused",
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("timestamp", ctx.BlockTime().String()),
		),
	)

	k.Logger(ctx).Warn("Oracle module paused")
	return nil
}

// EmergencyResumeOracle resumes oracle operations after emergency pause
func (k Keeper) EmergencyResumeOracle(ctx sdk.Context) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix("paused")
	if err := store.Delete(key); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_resumed",
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("timestamp", ctx.BlockTime().String()),
		),
	)

	k.Logger(ctx).Info("Oracle module resumed")
	return nil
}

// IsOraclePaused checks if oracle is in emergency pause mode
func (k Keeper) IsOraclePaused(ctx sdk.Context) bool {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix("paused")

	bz, err := store.Get(key)
	return err == nil && bz != nil && len(bz) > 0
}
