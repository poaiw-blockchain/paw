package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// MEVProtectionManager handles MEV attack detection and prevention
type MEVProtectionManager struct {
	keeper *Keeper
}

// NewMEVProtectionManager creates a new MEV protection manager
func NewMEVProtectionManager(keeper *Keeper) *MEVProtectionManager {
	return &MEVProtectionManager{
		keeper: keeper,
	}
}

// DetectMEVAttack performs comprehensive MEV attack detection
func (mpm *MEVProtectionManager) DetectMEVAttack(
	ctx sdk.Context,
	trader string,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, amountOut math.Int,
	priceImpact math.LegacyDec,
) types.MEVDetectionResult {
	config := mpm.keeper.GetMEVProtectionConfig(ctx)

	// Initialize result
	result := types.MEVDetectionResult{
		Detected:        false,
		AttackType:      "",
		Confidence:      math.LegacyZeroDec(),
		ShouldBlock:     false,
		Reason:          "",
		RelatedTxHashes: []string{},
		SuggestedAction: "allow",
	}

	// Check 1: Sandwich Attack Detection
	if config.EnableSandwichDetection {
		sandwichResult := mpm.DetectSandwichAttack(ctx, trader, poolID, tokenIn, tokenOut, amountIn, amountOut)
		if sandwichResult.Detected {
			result = sandwichResult
			result.ShouldBlock = true // Block sandwich attacks
			return result
		}
	}

	// Check 2: Price Impact Limits
	if config.EnablePriceImpactLimits {
		impactResult := mpm.CheckPriceImpactLimit(ctx, poolID, priceImpact, config.MaxPriceImpact)
		if impactResult.ExceedsLimit {
			result.Detected = true
			result.AttackType = "excessive_price_impact"
			result.Confidence = math.LegacyOneDec()
			result.ShouldBlock = true
			result.Reason = fmt.Sprintf(
				"price impact %.4f%% exceeds maximum %.4f%%",
				priceImpact.MulInt64(100),
				config.MaxPriceImpact.MulInt64(100),
			)
			result.SuggestedAction = "reject"
			return result
		}
	}

	// Check 3: Front-running Detection
	frontRunResult := mpm.DetectFrontRunning(ctx, trader, poolID, tokenIn, tokenOut, amountIn)
	if frontRunResult.Detected {
		result = frontRunResult
		// Front-running is logged but not automatically blocked (requires governance decision)
		result.ShouldBlock = false
		result.SuggestedAction = "flag_for_review"
	}

	return result
}

// DetectSandwichAttack detects sandwich attack patterns
// Pattern: Large buy -> Victim trade -> Large sell (all same pool, short time window)
func (mpm *MEVProtectionManager) DetectSandwichAttack(
	ctx sdk.Context,
	trader string,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, amountOut math.Int,
) types.MEVDetectionResult {
	config := mpm.keeper.GetMEVProtectionConfig(ctx)

	result := types.MEVDetectionResult{
		Detected:    false,
		AttackType:  "sandwich_attack",
		Confidence:  math.LegacyZeroDec(),
		ShouldBlock: false,
	}

	// Get recent transactions in the detection window
	recentTxs := mpm.getRecentTransactionsInWindow(ctx, poolID, config.SandwichDetectionWindow)
	if len(recentTxs) < 2 {
		return result // Need at least 2 previous transactions to detect sandwich
	}

	currentTime := ctx.BlockTime().Unix()

	// Look for sandwich pattern:
	// 1. Recent large buy (same direction as current tx or opposite to victim)
	// 2. Current tx could be victim or attacker sell
	// 3. Check for buy-victim-sell pattern

	var potentialFrontRun *types.TransactionRecord
	var potentialVictim *types.TransactionRecord

	for i := len(recentTxs) - 1; i >= 0; i-- {
		tx := recentTxs[i]

		// Skip if outside time window
		if currentTime-tx.Timestamp > config.SandwichDetectionWindow {
			continue
		}

		// Skip if same trader (checking for different pattern)
		if tx.Trader == trader {
			continue
		}

		// Pattern 1: Look for a large buy followed by smaller transaction
		if tx.TokenIn == tokenIn && tx.TokenOut == tokenOut {
			// Same direction trade - could be front-run
			if isLargeTrade(tx.AmountIn, amountIn, config.SandwichMinRatio) {
				potentialFrontRun = &tx
			} else {
				potentialVictim = &tx
			}
		}
	}

	// Check if current transaction could be the back-running sell
	// Current tx should be opposite direction to the potential victim
	if potentialFrontRun != nil && potentialVictim != nil {
		// Check if there's a pattern: large buy -> small trade -> current (checking if current is sell)
		if tokenIn == potentialFrontRun.TokenOut && tokenOut == potentialFrontRun.TokenIn {
			// Current tx is opposite direction to front-run (potential back-run)

			// Calculate confidence based on:
			// 1. Size ratio
			// 2. Time proximity
			// 3. Direction matching

			sizeRatio := calculateSizeRatio(potentialFrontRun.AmountIn, potentialVictim.AmountIn)
			timeProximity := calculateTimeProximity(potentialFrontRun.Timestamp, potentialVictim.Timestamp, currentTime)
			confidence := calculateSandwichConfidence(sizeRatio, timeProximity)

			if confidence.GT(math.LegacyNewDecWithPrec(7, 1)) { // 70% confidence threshold
				result.Detected = true
				result.Confidence = confidence
				result.ShouldBlock = true
				result.Reason = fmt.Sprintf(
					"sandwich attack detected: front-run tx %s, victim tx %s, confidence %.2f%%",
					potentialFrontRun.TxHash,
					potentialVictim.TxHash,
					confidence.MulInt64(100),
				)
				result.RelatedTxHashes = []string{potentialFrontRun.TxHash, potentialVictim.TxHash}
				result.SuggestedAction = "reject"

				// Emit detailed event
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeSandwichAttack,
						sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
						sdk.NewAttribute("attacker", trader),
						sdk.NewAttribute("front_run_tx", potentialFrontRun.TxHash),
						sdk.NewAttribute("victim_tx", potentialVictim.TxHash),
						sdk.NewAttribute("confidence", confidence.String()),
						sdk.NewAttribute("blocked", "true"),
					),
				)
			}
		}
	}

	return result
}

// DetectFrontRunning detects front-running patterns
func (mpm *MEVProtectionManager) DetectFrontRunning(
	ctx sdk.Context,
	trader string,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn math.Int,
) types.MEVDetectionResult {
	result := types.MEVDetectionResult{
		Detected:    false,
		AttackType:  "front_running",
		Confidence:  math.LegacyZeroDec(),
		ShouldBlock: false,
	}

	// Get pending transactions (in mempool concept)
	// Note: In Cosmos, we don't have direct mempool access, but we can check
	// transactions that were included in the same block before this one

	recentTxs := mpm.keeper.GetRecentPoolTransactions(ctx, poolID, 5)
	if len(recentTxs) == 0 {
		return result
	}

	currentTime := ctx.BlockTime().Unix()

	// Look for pattern: large trade in same direction right before smaller trades
	for _, tx := range recentTxs {
		// Same direction trade
		if tx.TokenIn == tokenIn && tx.TokenOut == tokenOut {
			// Check if this is a large trade relative to the current one
			if isLargeTrade(tx.AmountIn, amountIn, math.LegacyNewDecWithPrec(15, 1)) { // 1.5x ratio
				// Check time proximity (very recent)
				timeDiff := currentTime - tx.Timestamp
				if timeDiff <= 10 { // Within 10 seconds

					// Calculate confidence
					sizeRatio := tx.AmountIn.ToLegacyDec().Quo(amountIn.ToLegacyDec())
					timeProximityScore := math.LegacyOneDec().Sub(
						math.LegacyNewDec(timeDiff).Quo(math.LegacyNewDec(10)),
					)
					confidence := sizeRatio.Mul(timeProximityScore).Quo(math.LegacyNewDec(2))

					if confidence.GT(math.LegacyNewDecWithPrec(5, 1)) { // 50% threshold
						result.Detected = true
						result.Confidence = confidence
						result.Reason = fmt.Sprintf(
							"potential front-running: large trade %s (%.2fx) preceded current trade by %d seconds",
							tx.TxHash,
							sizeRatio,
							timeDiff,
						)
						result.RelatedTxHashes = []string{tx.TxHash}
						result.SuggestedAction = "flag_for_review"

						// Emit event for monitoring
						ctx.EventManager().EmitEvent(
							sdk.NewEvent(
								types.EventTypeFrontRunning,
								sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
								sdk.NewAttribute("trader", trader),
								sdk.NewAttribute("suspected_front_run_tx", tx.TxHash),
								sdk.NewAttribute("confidence", confidence.String()),
							),
						)

						break
					}
				}
			}
		}
	}

	return result
}

// CheckPriceImpactLimit checks if price impact exceeds the configured limit
func (mpm *MEVProtectionManager) CheckPriceImpactLimit(
	ctx sdk.Context,
	poolID uint64,
	priceImpact math.LegacyDec,
	maxImpact math.LegacyDec,
) types.PriceImpactCheck {
	result := types.PriceImpactCheck{
		PriceImpact:  priceImpact,
		ExceedsLimit: priceImpact.GT(maxImpact),
		MaxAllowed:   maxImpact,
		PoolID:       poolID,
	}

	if result.ExceedsLimit {
		// Emit event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePriceImpactExceeded,
				sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
				sdk.NewAttribute("price_impact", priceImpact.String()),
				sdk.NewAttribute("max_allowed", maxImpact.String()),
			),
		)
	}

	return result
}

// getRecentTransactionsInWindow retrieves transactions within a time window
func (mpm *MEVProtectionManager) getRecentTransactionsInWindow(
	ctx sdk.Context,
	poolID uint64,
	windowSeconds int64,
) []types.TransactionRecord {
	allRecent := mpm.keeper.GetRecentPoolTransactions(ctx, poolID, 50)
	currentTime := ctx.BlockTime().Unix()

	var filtered []types.TransactionRecord
	for _, tx := range allRecent {
		if currentTime-tx.Timestamp <= windowSeconds {
			filtered = append(filtered, tx)
		}
	}

	return filtered
}

// Helper functions

// isLargeTrade checks if trade A is significantly larger than trade B
func isLargeTrade(amountA, amountB math.Int, minRatio math.LegacyDec) bool {
	if amountB.IsZero() {
		return false
	}
	ratio := amountA.ToLegacyDec().Quo(amountB.ToLegacyDec())
	return ratio.GTE(minRatio)
}

// calculateSizeRatio calculates the size ratio between two amounts
func calculateSizeRatio(larger, smaller math.Int) math.LegacyDec {
	if smaller.IsZero() {
		return math.LegacyZeroDec()
	}
	return larger.ToLegacyDec().Quo(smaller.ToLegacyDec())
}

// calculateTimeProximity calculates time proximity score (1.0 = same time, 0.0 = far apart)
func calculateTimeProximity(time1, time2, time3 int64) math.LegacyDec {
	// Calculate how close time2 is between time1 and time3
	if time3 <= time1 {
		return math.LegacyZeroDec()
	}

	totalWindow := time3 - time1
	if totalWindow == 0 {
		return math.LegacyOneDec()
	}

	// How far is time2 from time1 (as percentage of total window)
	time2Distance := time2 - time1
	if time2Distance < 0 {
		return math.LegacyZeroDec()
	}

	// Proximity is higher when time2 is closer to time1
	// Score: 1.0 when time2 = time1, decreasing as time2 approaches time3
	proximityScore := math.LegacyOneDec().Sub(
		math.LegacyNewDec(time2Distance).Quo(math.LegacyNewDec(totalWindow)),
	)

	if proximityScore.IsNegative() {
		return math.LegacyZeroDec()
	}

	return proximityScore
}

// calculateSandwichConfidence calculates confidence score for sandwich attack detection
func calculateSandwichConfidence(sizeRatio, timeProximity math.LegacyDec) math.LegacyDec {
	// Confidence based on:
	// 1. Size ratio (higher ratio = higher confidence, but cap at 5x)
	// 2. Time proximity (closer in time = higher confidence)

	// Normalize size ratio to 0-1 scale (assuming max meaningful ratio is 10x)
	normalizedSizeRatio := sizeRatio.Quo(math.LegacyNewDec(10))
	if normalizedSizeRatio.GT(math.LegacyOneDec()) {
		normalizedSizeRatio = math.LegacyOneDec()
	}

	// Weight: 60% size ratio, 40% time proximity
	confidence := normalizedSizeRatio.MulInt64(60).Add(timeProximity.MulInt64(40)).QuoInt64(100)

	return confidence
}

// GetMEVProtectionConfig retrieves the MEV protection configuration
func (k Keeper) GetMEVProtectionConfig(ctx sdk.Context) types.MEVProtectionConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.MEVProtectionConfigKey)
	if bz == nil {
		// Return default config if not set
		return types.DefaultMEVProtectionConfig()
	}

	var config types.MEVProtectionConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		// Return default if unmarshal fails
		return types.DefaultMEVProtectionConfig()
	}
	return config
}

// SetMEVProtectionConfig sets the MEV protection configuration
func (k Keeper) SetMEVProtectionConfig(ctx sdk.Context, config types.MEVProtectionConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(&config)
	if err != nil {
		return err
	}
	store.Set(types.MEVProtectionConfigKey, bz)
	return nil
}

// UpdateMEVMetrics updates MEV protection metrics
func (k Keeper) UpdateMEVMetrics(ctx sdk.Context, detectionResult types.MEVDetectionResult) {
	metrics := k.GetMEVMetrics(ctx)

	metrics.TotalTransactions++

	if detectionResult.Detected {
		metrics.TotalMEVDetected++

		if detectionResult.ShouldBlock {
			metrics.TotalMEVBlocked++
		}

		switch detectionResult.AttackType {
		case "sandwich_attack":
			metrics.SandwichAttacksDetected++
		case "front_running":
			metrics.FrontRunningDetected++
		case "excessive_price_impact":
			metrics.PriceImpactViolations++
		}
	}

	metrics.LastUpdated = ctx.BlockTime().Unix()

	k.SetMEVMetrics(ctx, metrics)
}

// GetMEVMetrics retrieves MEV protection metrics
func (k Keeper) GetMEVMetrics(ctx sdk.Context) types.MEVProtectionMetrics {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.MEVMetricsKey)
	if bz == nil {
		return types.MEVProtectionMetrics{}
	}

	var metrics types.MEVProtectionMetrics
	if err := json.Unmarshal(bz, &metrics); err != nil {
		return types.MEVProtectionMetrics{}
	}
	return metrics
}

// SetMEVMetrics sets MEV protection metrics
func (k Keeper) SetMEVMetrics(ctx sdk.Context, metrics types.MEVProtectionMetrics) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(&metrics)
	if err != nil {
		return
	}
	store.Set(types.MEVMetricsKey, bz)
}

// RecordSandwichPattern records a detected sandwich attack pattern
func (k Keeper) RecordSandwichPattern(ctx sdk.Context, pattern types.SandwichPattern) {
	store := ctx.KVStore(k.storeKey)

	// Generate a unique key for this pattern
	key := types.GetSandwichPatternKey(pattern.VictimTx.PoolID, ctx.BlockHeight(), pattern.VictimTx.TxHash)

	bz, err := json.Marshal(&pattern)
	if err != nil {
		return
	}
	store.Set(key, bz)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSandwichPattern,
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", pattern.VictimTx.PoolID)),
			sdk.NewAttribute("victim_tx", pattern.VictimTx.TxHash),
			sdk.NewAttribute("front_run_tx", pattern.FrontRunTx.TxHash),
			sdk.NewAttribute("confidence", pattern.ConfidenceScore.String()),
			sdk.NewAttribute("blocked", fmt.Sprintf("%t", pattern.Blocked)),
		),
	)
}

// GetSandwichPatterns retrieves detected sandwich patterns for a pool
func (k Keeper) GetSandwichPatterns(ctx sdk.Context, poolID uint64, limit int) []types.SandwichPattern {
	store := ctx.KVStore(k.storeKey)
	prefix := types.GetSandwichPatternPrefix(poolID)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var patterns []types.SandwichPattern
	count := 0

	for ; iterator.Valid() && count < limit; iterator.Next() {
		var pattern types.SandwichPattern
		if err := json.Unmarshal(iterator.Value(), &pattern); err != nil {
			continue
		}
		patterns = append(patterns, pattern)
		count++
	}

	return patterns
}
