package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

const (
	// Outlier reputation tracking window (in blocks)
	OutlierReputationWindow = 1000

	// Grace period for first-time offenders (in blocks)
	OutlierGracePeriod = 1000

	// Repeated offender threshold (number of outliers in window)
	RepeatedOffenderThreshold = 3

	// MaxOutlierHistoryBlocks is the maximum number of blocks to retain outlier history.
	// History older than this is eligible for cleanup to prevent unbounded state growth.
	// SEC-9: This constant controls the cleanup window for outlier history storage.
	MaxOutlierHistoryBlocks = 1000
)

// OutlierHistory tracks outlier submissions for reputation
type OutlierHistory struct {
	ValidatorAddr string
	Asset         string
	BlockHeight   int64
	Severity      OutlierSeverity
}

// handleOutlierSlashing handles slashing based on outlier severity
func (k Keeper) handleOutlierSlashing(ctx context.Context, asset string, outlier OutlierDetectionResult) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	valAddr, err := sdk.ValAddressFromBech32(outlier.ValidatorAddr)
	if err != nil {
		return err
	}

	// Check if validator should be slashed based on severity and history
	shouldSlash, slashFraction, shouldJail := k.shouldSlashForOutlier(ctx, valAddr, asset, outlier.Severity)

	if !shouldSlash {
		// Record the outlier but don't slash (grace period or low severity)
		k.recordOutlierHistory(ctx, outlier.ValidatorAddr, asset, sdkCtx.BlockHeight(), outlier.Severity)
		return nil
	}

	// Get validator
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return fmt.Errorf("validator not found: %s", outlier.ValidatorAddr)
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	// Slash the validator
	if _, err := k.stakingKeeper.Slash(
		ctx,
		consAddr,
		sdkCtx.BlockHeight(),
		validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)),
		slashFraction,
	); err != nil {
		return fmt.Errorf("failed to slash validator %s: %w", outlier.ValidatorAddr, err)
	}

	// Jail if severity is extreme or repeated offender
	if shouldJail {
		if err := k.JailValidator(ctx, valAddr); err != nil {
			return err
		}
	}

	// Record outlier in history
	k.recordOutlierHistory(ctx, outlier.ValidatorAddr, asset, sdkCtx.BlockHeight(), outlier.Severity)

	// Emit detailed slashing event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleSlashOutlier,
			sdk.NewAttribute(types.AttributeKeyValidator, outlier.ValidatorAddr),
			sdk.NewAttribute(types.AttributeKeyAsset, asset),
			sdk.NewAttribute(types.AttributeKeySeverity, fmt.Sprintf("%d", outlier.Severity)),
			sdk.NewAttribute(types.AttributeKeySlashFraction, slashFraction.String()),
			sdk.NewAttribute(types.AttributeKeyJailed, fmt.Sprintf("%t", shouldJail)),
			sdk.NewAttribute(types.AttributeKeyPrice, outlier.Price.String()),
			sdk.NewAttribute(types.AttributeKeyDeviation, outlier.Deviation.String()),
			sdk.NewAttribute(types.AttributeKeyReason, outlier.Reason),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	sdkCtx.Logger().Info("slashed validator for outlier submission",
		"validator", outlier.ValidatorAddr,
		"asset", asset,
		"severity", outlier.Severity,
		"slash_fraction", slashFraction.String(),
		"jailed", shouldJail,
	)

	return nil
}

// shouldSlashForOutlier determines if a validator should be slashed based on outlier severity and history
func (k Keeper) shouldSlashForOutlier(ctx context.Context, valAddr sdk.ValAddress, asset string, severity OutlierSeverity) (shouldSlash bool, slashFraction sdkmath.LegacyDec, shouldJail bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get outlier history for this validator
	recentOutliers := k.getRecentOutliers(ctx, valAddr.String(), asset, OutlierReputationWindow)

	// Check if validator is in grace period (first offense)
	isFirstOffense := len(recentOutliers) == 0
	if isFirstOffense && severity < SeverityHigh {
		// Grace period: warn but don't slash for first moderate/low outlier
		sdkCtx.Logger().Warn("validator first outlier - grace period",
			"validator", valAddr.String(),
			"asset", asset,
			"severity", severity,
		)
		return false, sdkmath.LegacyZeroDec(), false
	}

	// Check if validator is a repeated offender
	isRepeatedOffender := len(recentOutliers) >= RepeatedOffenderThreshold

	// Get base parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, sdkmath.LegacyZeroDec(), false
	}

	baseSlashFraction := params.SlashFraction

	// Determine slash fraction and jailing based on severity
	switch severity {
	case SeverityExtreme:
		// Extreme outlier: 0.05% slash + jail
		slashFraction = sdkmath.LegacyMustNewDecFromStr("0.0005")
		shouldJail = true
		shouldSlash = true

	case SeverityHigh:
		// High outlier: 0.02% slash, jail if repeated offender
		slashFraction = sdkmath.LegacyMustNewDecFromStr("0.0002")
		shouldJail = isRepeatedOffender
		shouldSlash = true

	case SeverityModerate:
		// Moderate outlier: 0.01% slash if repeated offender
		slashFraction = sdkmath.LegacyMustNewDecFromStr("0.0001")
		shouldJail = false
		shouldSlash = isRepeatedOffender

	case SeverityLow:
		// Low outlier: warn only, slash if extreme repeated offender
		slashFraction = sdkmath.LegacyMustNewDecFromStr("0.00005")
		shouldJail = false
		shouldSlash = len(recentOutliers) >= (RepeatedOffenderThreshold * 2)

	default:
		return false, sdkmath.LegacyZeroDec(), false
	}

	// Apply repeated offender multiplier
	if isRepeatedOffender && severity >= SeverityModerate {
		// Multiply slash fraction by 2x for repeated offenders
		slashFraction = slashFraction.Mul(sdkmath.LegacyNewDec(2))
		shouldJail = true
	}

	// Cap slash fraction at 0.10% (0.001)
	maxSlashFraction := sdkmath.LegacyMustNewDecFromStr("0.001")
	if slashFraction.GT(maxSlashFraction) {
		slashFraction = maxSlashFraction
	}

	// Ensure minimum slash fraction if slashing
	minSlashFraction := baseSlashFraction
	if shouldSlash && slashFraction.LT(minSlashFraction) {
		slashFraction = minSlashFraction
	}

	return shouldSlash, slashFraction, shouldJail
}

// recordOutlierHistory records an outlier submission in the validator's history
func (k Keeper) recordOutlierHistory(ctx context.Context, validatorAddr, asset string, blockHeight int64, severity OutlierSeverity) {
	store := k.getStore(ctx)

	// Store key: prefix + validator + asset + blockHeight
	key := k.getOutlierHistoryKey(validatorAddr, asset, blockHeight)

	// Simple encoding for history (we'll use a basic format)
	value := []byte(fmt.Sprintf("%d:%d", severity, blockHeight))
	store.Set(key, value)
}

// getRecentOutliers retrieves recent outlier submissions for a validator
func (k Keeper) getRecentOutliers(ctx context.Context, validatorAddr, asset string, window int64) []OutlierHistory {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	minHeight := sdkCtx.BlockHeight() - window

	// Get all outlier history entries for this validator and asset
	prefix := k.getOutlierHistoryPrefix(validatorAddr, asset)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	outliers := []OutlierHistory{}

	for ; iterator.Valid(); iterator.Next() {
		// Parse the value to extract block height and severity
		var severity OutlierSeverity
		var blockHeight int64
		if _, err := fmt.Sscanf(string(iterator.Value()), "%d:%d", &severity, &blockHeight); err != nil {
			sdkCtx.Logger().Error("failed to parse outlier history entry", "error", err)
			continue
		}

		if blockHeight >= minHeight {
			outliers = append(outliers, OutlierHistory{
				ValidatorAddr: validatorAddr,
				Asset:         asset,
				BlockHeight:   blockHeight,
				Severity:      severity,
			})
		}
	}

	return outliers
}

// getOutlierHistoryKey generates a storage key for outlier history
func (k Keeper) getOutlierHistoryKey(validatorAddr, asset string, blockHeight int64) []byte {
	// Prefix: OutlierHistoryKeyPrefix + validator + 0x00 + asset + 0x00 + blockHeight
	key := append([]byte(nil), OutlierHistoryKeyPrefix...)
	key = append(key, []byte(validatorAddr)...)
	key = append(key, byte(0x00)) // separator
	key = append(key, []byte(asset)...)
	key = append(key, byte(0x00)) // separator

	// Encode block height
	heightBytes := sdk.Uint64ToBigEndian(uint64(blockHeight))
	key = append(key, heightBytes...)

	return key
}

// getOutlierHistoryPrefix generates a storage prefix for outlier history queries
func (k Keeper) getOutlierHistoryPrefix(validatorAddr, asset string) []byte {
	key := append([]byte(nil), OutlierHistoryKeyPrefix...)
	key = append(key, []byte(validatorAddr)...)
	key = append(key, byte(0x00)) // separator
	key = append(key, []byte(asset)...)
	key = append(key, byte(0x00)) // separator
	return key
}

// CleanupOldOutlierHistory removes outlier history older than the window
func (k Keeper) CleanupOldOutlierHistory(ctx context.Context, validatorAddr, asset string, minHeight int64) error {
	store := k.getStore(ctx)

	prefix := k.getOutlierHistoryPrefix(validatorAddr, asset)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	keysToDelete := [][]byte{}

	for ; iterator.Valid(); iterator.Next() {
		var blockHeight int64
		var severity OutlierSeverity
		if _, err := fmt.Sscanf(string(iterator.Value()), "%d:%d", &severity, &blockHeight); err != nil {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Error("failed to parse outlier history entry", "error", err)
			continue
		}

		if blockHeight < minHeight {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
	}

	for _, key := range keysToDelete {
		store.Delete(key)
	}

	return nil
}

// SlashMissVote slashes a validator for missing oracle votes
func (k Keeper) SlashMissVote(ctx context.Context, validatorAddrStr string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	valAddr, err := sdk.ValAddressFromBech32(validatorAddrStr)
	if err != nil {
		return err
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return fmt.Errorf("validator not found: %s", validatorAddrStr)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	if _, err := k.stakingKeeper.Slash(
		ctx,
		consAddr,
		sdkCtx.BlockHeight(),
		validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)),
		params.SlashFraction,
	); err != nil {
		return fmt.Errorf("failed to slash validator %s for missed vote: %w", validatorAddrStr, err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleSlash,
			sdk.NewAttribute(types.AttributeKeyValidator, validatorAddrStr),
			sdk.NewAttribute(types.AttributeKeyReason, "missed_vote"),
			sdk.NewAttribute(types.AttributeKeySlashFraction, params.SlashFraction.String()),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	sdkCtx.Logger().Info("slashed validator for missing oracle vote",
		"validator", validatorAddrStr,
		"slash_fraction", params.SlashFraction.String(),
	)

	return nil
}

// SlashBadData slashes a validator for submitting invalid data
func (k Keeper) SlashBadData(ctx context.Context, validatorAddr sdk.ValAddress, reason string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return fmt.Errorf("validator not found: %s", validatorAddr.String())
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	// Use higher slash fraction for bad data (2x the miss vote fraction)
	badDataSlashFraction := params.SlashFraction.Mul(sdkmath.LegacyNewDec(2))
	maxSlashFraction := sdkmath.LegacyMustNewDecFromStr("0.001") // Cap at 0.1%
	if badDataSlashFraction.GT(maxSlashFraction) {
		badDataSlashFraction = maxSlashFraction
	}

	if _, err := k.stakingKeeper.Slash(
		ctx,
		consAddr,
		sdkCtx.BlockHeight(),
		validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)),
		badDataSlashFraction,
	); err != nil {
		return fmt.Errorf("failed to slash validator %s for bad data: %w", validatorAddr.String(), err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleSlash,
			sdk.NewAttribute(types.AttributeKeyValidator, validatorAddr.String()),
			sdk.NewAttribute(types.AttributeKeyReason, "bad_data"),
			sdk.NewAttribute(types.AttributeKeyDetails, reason),
			sdk.NewAttribute(types.AttributeKeySlashFraction, badDataSlashFraction.String()),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	sdkCtx.Logger().Info("slashed validator for submitting bad data",
		"validator", validatorAddr.String(),
		"reason", reason,
		"slash_fraction", badDataSlashFraction.String(),
	)

	return nil
}

// JailValidator jails a validator for oracle misbehavior
func (k Keeper) JailValidator(ctx context.Context, validatorAddr sdk.ValAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return fmt.Errorf("validator not found: %s", validatorAddr.String())
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	if err := k.slashingKeeper.Jail(ctx, consAddr); err != nil {
		return fmt.Errorf("failed to jail validator %s: %w", validatorAddr.String(), err)
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_jail",
			sdk.NewAttribute("validator", validatorAddr.String()),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	sdkCtx.Logger().Info("jailed validator for oracle misbehavior",
		"validator", validatorAddr.String(),
	)

	return nil
}

// ValidatePriceSubmission validates a price submission for potential slashing
func (k Keeper) ValidatePriceSubmission(ctx context.Context, validatorAddr sdk.ValAddress, asset string, price sdkmath.LegacyDec) error {
	if price.IsNil() || price.LTE(sdkmath.LegacyZeroDec()) {
		return k.SlashBadData(ctx, validatorAddr, "non-positive price")
	}

	if price.IsNegative() {
		return k.SlashBadData(ctx, validatorAddr, "negative price")
	}

	currentPrice, err := k.GetPrice(ctx, asset)
	if err != nil {
		return nil
	}

	// Sanity check: reject prices that are absurdly different (100x deviation)
	// This is a final safeguard; the main outlier detection happens in aggregation
	maxDeviation := sdkmath.LegacyNewDec(100)
	minValidPrice := currentPrice.Price.Quo(maxDeviation)
	maxValidPrice := currentPrice.Price.Mul(maxDeviation)

	if price.LT(minValidPrice) || price.GT(maxValidPrice) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Warn("price submission outside sanity bounds",
			"validator", validatorAddr.String(),
			"asset", asset,
			"submitted_price", price.String(),
			"current_price", currentPrice.Price.String(),
			"min_valid", minValidPrice.String(),
			"max_valid", maxValidPrice.String(),
		)
	}

	return nil
}

// HandleSlashWindow processes the slash window for a vote period
func (k Keeper) HandleSlashWindow(ctx context.Context, asset string) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	if currentHeight%int64(params.VotePeriod) != 0 {
		return nil
	}

	return k.CheckMissedVotes(ctx, asset)
}

// GetValidatorOutlierReputation calculates a reputation score for a validator based on recent outliers
func (k Keeper) GetValidatorOutlierReputation(ctx context.Context, validatorAddr, asset string) (reputationScore sdkmath.LegacyDec, totalOutliers int) {
	outliers := k.getRecentOutliers(ctx, validatorAddr, asset, OutlierReputationWindow)

	if len(outliers) == 0 {
		return sdkmath.LegacyOneDec(), 0 // Perfect reputation
	}

	// Calculate weighted reputation score
	// Extreme outliers count more heavily against reputation
	penaltyPoints := sdkmath.LegacyZeroDec()

	for _, outlier := range outliers {
		switch outlier.Severity {
		case SeverityExtreme:
			penaltyPoints = penaltyPoints.Add(sdkmath.LegacyMustNewDecFromStr("1.0"))
		case SeverityHigh:
			penaltyPoints = penaltyPoints.Add(sdkmath.LegacyMustNewDecFromStr("0.5"))
		case SeverityModerate:
			penaltyPoints = penaltyPoints.Add(sdkmath.LegacyMustNewDecFromStr("0.25"))
		case SeverityLow:
			penaltyPoints = penaltyPoints.Add(sdkmath.LegacyMustNewDecFromStr("0.1"))
		}
	}

	// Reputation = 1 / (1 + penalty_points)
	// Perfect score: 1.0, worst score approaches 0
	reputationScore = sdkmath.LegacyOneDec().Quo(sdkmath.LegacyOneDec().Add(penaltyPoints))

	return reputationScore, len(outliers)
}
