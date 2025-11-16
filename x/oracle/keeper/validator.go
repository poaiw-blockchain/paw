package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SubmitPrice allows a validator to submit a price for an asset
func (k Keeper) SubmitPrice(ctx sdk.Context, validatorAddr sdk.ValAddress, asset string, price math.LegacyDec) error {
	// Validate inputs
	if asset == "" {
		return fmt.Errorf("asset symbol cannot be empty")
	}
	if price.IsNil() || price.IsNegative() || price.IsZero() {
		return fmt.Errorf("price must be positive")
	}

	// Check if validator is active
	if !k.IsActiveValidator(ctx, validatorAddr) {
		return fmt.Errorf("only active validators can submit prices")
	}

	// Check rate limiting
	params := k.GetParams(ctx)
	if err := k.checkRateLimit(ctx, validatorAddr, asset, params.UpdateInterval); err != nil {
		return err
	}

	// Create submission
	submission := types.NewValidatorPriceSubmission(
		validatorAddr.String(),
		asset,
		price,
		ctx.BlockTime().Unix(),
		ctx.BlockHeight(),
	)

	// Store the submission
	if err := k.SetValidatorSubmission(ctx, submission); err != nil {
		return err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_submitted",
			sdk.NewAttribute("validator", validatorAddr.String()),
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("price", price.String()),
			sdk.NewAttribute("timestamp", fmt.Sprintf("%d", submission.Timestamp)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", submission.BlockHeight)),
		),
	)

	k.Logger(ctx).Info(
		"Price submitted",
		"validator", validatorAddr.String(),
		"asset", asset,
		"price", price.String(),
	)

	return nil
}

// IsActiveValidator checks if a validator is active (bonded)
func (k Keeper) IsActiveValidator(ctx sdk.Context, validatorAddr sdk.ValAddress) bool {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return false
	}

	return validator.IsBonded()
}

// checkRateLimit ensures validators don't submit prices too frequently
func (k Keeper) checkRateLimit(ctx sdk.Context, validatorAddr sdk.ValAddress, asset string, updateInterval uint64) error {
	// Get last submission
	submission, found := k.GetValidatorSubmission(ctx, asset, validatorAddr.String())
	if !found {
		return nil // First submission, no rate limit
	}

	// Check if enough time has passed
	lastSubmissionTime := time.Unix(submission.Timestamp, 0)
	timeSinceLastSubmission := ctx.BlockTime().Sub(lastSubmissionTime)
	minInterval := time.Duration(updateInterval) * time.Second

	if timeSinceLastSubmission < minInterval {
		return fmt.Errorf(
			"rate limit exceeded: must wait %s between submissions, only %s has passed",
			minInterval,
			timeSinceLastSubmission,
		)
	}

	return nil
}

// GetActiveValidators returns all active (bonded) validators
func (k Keeper) GetActiveValidators(ctx sdk.Context) ([]stakingtypes.Validator, error) {
	var activeValidators []stakingtypes.Validator

	err := k.stakingKeeper.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		val, ok := validator.(stakingtypes.Validator)
		if ok {
			activeValidators = append(activeValidators, val)
		}
		return false
	})

	if err != nil {
		return nil, err
	}

	return activeValidators, nil
}

// GetValidatorPower returns the voting power of a validator
func (k Keeper) GetValidatorPower(ctx sdk.Context, validatorAddr sdk.ValAddress) (int64, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return 0, err
	}

	return validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)), nil
}

// HasMinimumSubmissions checks if an asset has received minimum required submissions
func (k Keeper) HasMinimumSubmissions(ctx sdk.Context, asset string) bool {
	submissions := k.GetValidatorSubmissions(ctx, asset)
	params := k.GetParams(ctx)

	// Filter out stale submissions
	currentTime := ctx.BlockTime()
	validSubmissions := 0

	for _, sub := range submissions {
		if !sub.IsStale(currentTime, params.ExpiryDuration) {
			validSubmissions++
		}
	}

	return validSubmissions >= int(params.MinValidators)
}

// GetValidSubmissions returns all non-stale submissions for an asset
func (k Keeper) GetValidSubmissions(ctx sdk.Context, asset string) []types.ValidatorPriceSubmission {
	submissions := k.GetValidatorSubmissions(ctx, asset)
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime()

	var validSubmissions []types.ValidatorPriceSubmission
	for _, sub := range submissions {
		if !sub.IsStale(currentTime, params.ExpiryDuration) {
			validSubmissions = append(validSubmissions, sub)
		}
	}

	return validSubmissions
}

// ValidateSubmissionBounds checks if a price is within reasonable bounds
// This prevents extreme outliers from being submitted
func (k Keeper) ValidateSubmissionBounds(ctx sdk.Context, asset string, price math.LegacyDec) error {
	// Get current price feed if exists
	priceFeed, found := k.GetPriceFeed(ctx, asset)
	if !found {
		// No existing price, accept any positive price
		return nil
	}

	// Check if price is within 50% of current median
	// This is a basic sanity check to prevent extreme manipulation
	maxDeviation := math.LegacyNewDec(50) // 50%
	deviation := types.PriceDeviation(price, priceFeed.Price)

	if deviation.GT(maxDeviation) {
		return fmt.Errorf(
			"price deviation too large: %.2f%% (max: %.2f%%)",
			deviation.MustFloat64(),
			maxDeviation.MustFloat64(),
		)
	}

	return nil
}

// GetValidatorSubmissionCount returns the number of submissions by a validator
func (k Keeper) GetValidatorSubmissionCount(ctx sdk.Context, validatorAddr string) uint64 {
	accuracy := k.GetValidatorAccuracy(ctx, validatorAddr)
	return accuracy.TotalSubmissions
}

// GetValidatorAccuracyRate returns the accuracy rate of a validator
func (k Keeper) GetValidatorAccuracyRate(ctx sdk.Context, validatorAddr string) math.LegacyDec {
	accuracy := k.GetValidatorAccuracy(ctx, validatorAddr)
	return accuracy.AccuracyRate()
}

// PruneOldSubmissions removes submissions older than a certain threshold
// This keeps storage clean and prevents unbounded growth
func (k Keeper) PruneOldSubmissions(ctx sdk.Context, maxAge uint64) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.KeyPrefix(types.ValidatorKeyPrefix)

	currentTime := ctx.BlockTime()
	maxAgeSeconds := time.Duration(maxAge) * time.Second

	// This would need proper iteration implementation
	// For now, we rely on the CleanupStaleSubmissions per asset
	_ = store
	_ = prefix
	_ = currentTime
	_ = maxAgeSeconds
}
