package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// Advanced reputation system implementing:
// - Bayesian reputation scoring
// - Time-based decay
// - Multi-dimensional reputation
// - Historical performance tracking
// - Anti-collusion mechanisms

// ReputationDimension represents different aspects of provider performance
type ReputationDimension string

const (
	DimensionReliability  ReputationDimension = "reliability"  // Completion rate
	DimensionSpeed        ReputationDimension = "speed"        // Response time
	DimensionAccuracy     ReputationDimension = "accuracy"     // Verification score
	DimensionAvailability ReputationDimension = "availability" // Uptime
)

// ProviderReputation is an alias to the protobuf type for reputation management
type ProviderReputation = types.ProviderReputation

// PerformanceRecord is an alias to the protobuf type for performance tracking
type PerformanceRecord = types.PerformanceRecord

// ProviderSelectionScore combines multiple factors for fair selection
type ProviderSelectionScore struct {
	Provider          string
	ReputationScore   float64
	StakeWeight       float64
	LoadFactor        float64
	RandomFactor      float64
	TotalScore        float64
	SelectionPriority int
}

// UpdateReputationAdvanced updates provider reputation using Bayesian inference
// This prevents gaming through sophisticated statistical modeling
func (k Keeper) UpdateReputationAdvanced(ctx context.Context, provider sdk.AccAddress, success bool, verificationScore uint32, responseTimeMs uint64, requestValue sdkmath.Int) error {
	rep, err := k.GetProviderReputation(ctx, provider)
	if err != nil {
		// Initialize new reputation
		rep = &ProviderReputation{
			Provider:               provider.String(),
			OverallScore:           50, // Start at neutral
			ReliabilityScore:       0.5,
			SpeedScore:             0.5,
			AccuracyScore:          0.5,
			AvailabilityScore:      0.5,
			TotalRequests:          0,
			SuccessfulRequests:     0,
			FailedRequests:         0,
			TotalVerificationScore: 0,
			AverageResponseTime:    0,
			LastDecayTimestamp:     sdk.UnwrapSDKContext(ctx).BlockTime(),
			LastUpdateTimestamp:    sdk.UnwrapSDKContext(ctx).BlockTime(),
			PerformanceHistory:     make([]PerformanceRecord, 0),
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// Apply time-based decay before updating
	if err := k.applyReputationDecay(rep, now); err != nil {
		return fmt.Errorf("failed to apply reputation decay: %w", err)
	}

	// Update counters
	rep.TotalRequests++
	if success {
		rep.SuccessfulRequests++
	} else {
		rep.FailedRequests++
	}

	// Add to performance history (keep last 100 records)
	record := PerformanceRecord{
		Timestamp:         now,
		Success:           success,
		VerificationScore: verificationScore,
		ResponseTimeMs:    responseTimeMs,
		RequestValue:      requestValue,
	}
	rep.PerformanceHistory = append(rep.PerformanceHistory, record)
	if len(rep.PerformanceHistory) > 100 {
		rep.PerformanceHistory = rep.PerformanceHistory[1:]
	}

	// Update reliability using Bayesian approach
	// Beta distribution parameters
	alpha := float64(rep.SuccessfulRequests + 1) // Prior: Beta(1,1)
	beta := float64(rep.FailedRequests + 1)
	rep.ReliabilityScore = alpha / (alpha + beta)

	// Update accuracy based on verification scores
	rep.TotalVerificationScore += uint64(verificationScore)
	avgVerificationScore := float64(rep.TotalVerificationScore) / float64(rep.TotalRequests)
	rep.AccuracyScore = avgVerificationScore / 100.0 // Normalize to 0-1

	// Update speed based on response times (exponential moving average)
	if rep.AverageResponseTime == 0 {
		rep.AverageResponseTime = float64(responseTimeMs)
	} else {
		alpha := 0.1 // Smoothing factor
		rep.AverageResponseTime = alpha*float64(responseTimeMs) + (1-alpha)*rep.AverageResponseTime
	}
	// Normalize speed score (lower is better, assume 60s is baseline)
	targetResponseTime := 60000.0 // 60 seconds in ms
	rep.SpeedScore = math.Max(0, math.Min(1.0, targetResponseTime/rep.AverageResponseTime))

	// Calculate overall score as weighted average
	weights := map[ReputationDimension]float64{
		DimensionReliability:  0.4,
		DimensionAccuracy:     0.3,
		DimensionSpeed:        0.2,
		DimensionAvailability: 0.1,
	}

	overallScore := weights[DimensionReliability]*rep.ReliabilityScore +
		weights[DimensionAccuracy]*rep.AccuracyScore +
		weights[DimensionSpeed]*rep.SpeedScore +
		weights[DimensionAvailability]*rep.AvailabilityScore

	rep.OverallScore = uint32(overallScore * 100)
	rep.LastUpdateTimestamp = now

	// Save updated reputation
	if err := k.SetProviderReputation(ctx, *rep); err != nil {
		return fmt.Errorf("failed to save reputation: %w", err)
	}

	// Also update simple reputation for backward compatibility
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil {
		return err
	}
	providerRecord.Reputation = rep.OverallScore
	if err := k.SetProvider(ctx, *providerRecord); err != nil {
		return err
	}

	// Emit detailed reputation event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"reputation_updated_advanced",
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("overall_score", fmt.Sprintf("%d", rep.OverallScore)),
			sdk.NewAttribute("reliability", fmt.Sprintf("%.3f", rep.ReliabilityScore)),
			sdk.NewAttribute("accuracy", fmt.Sprintf("%.3f", rep.AccuracyScore)),
			sdk.NewAttribute("speed", fmt.Sprintf("%.3f", rep.SpeedScore)),
			sdk.NewAttribute("total_requests", fmt.Sprintf("%d", rep.TotalRequests)),
			sdk.NewAttribute("success_rate", fmt.Sprintf("%.2f%%", float64(rep.SuccessfulRequests)/float64(rep.TotalRequests)*100)),
		),
	)

	return nil
}

// applyReputationDecay applies time-based decay to reputation scores
// This ensures old good behavior doesn't protect against new bad behavior
func (k Keeper) applyReputationDecay(rep *ProviderReputation, currentTime time.Time) error {
	// Calculate time elapsed since last decay
	elapsed := currentTime.Sub(rep.LastDecayTimestamp)
	if elapsed < 24*time.Hour {
		return nil // Decay once per day
	}

	// Decay factor: 1% per day towards neutral (0.5)
	decayRate := 0.01
	daysElapsed := elapsed.Hours() / 24.0

	// Decay each dimension towards 0.5 (neutral)
	rep.ReliabilityScore = decayTowardsNeutral(rep.ReliabilityScore, decayRate*daysElapsed)
	rep.AccuracyScore = decayTowardsNeutral(rep.AccuracyScore, decayRate*daysElapsed)
	rep.SpeedScore = decayTowardsNeutral(rep.SpeedScore, decayRate*daysElapsed)
	rep.AvailabilityScore = decayTowardsNeutral(rep.AvailabilityScore, decayRate*daysElapsed)

	// Update overall score
	weights := map[ReputationDimension]float64{
		DimensionReliability:  0.4,
		DimensionAccuracy:     0.3,
		DimensionSpeed:        0.2,
		DimensionAvailability: 0.1,
	}

	overallScore := weights[DimensionReliability]*rep.ReliabilityScore +
		weights[DimensionAccuracy]*rep.AccuracyScore +
		weights[DimensionSpeed]*rep.SpeedScore +
		weights[DimensionAvailability]*rep.AvailabilityScore

	rep.OverallScore = uint32(overallScore * 100)
	rep.LastDecayTimestamp = currentTime

	return nil
}

// decayTowardsNeutral moves a score towards 0.5 (neutral) by the decay amount
func decayTowardsNeutral(score float64, decayAmount float64) float64 {
	neutral := 0.5
	if score > neutral {
		return math.Max(neutral, score-decayAmount)
	} else {
		return math.Min(neutral, score+decayAmount)
	}
}

// SelectProviderAdvanced implements sophisticated provider selection with anti-collusion
// Uses weighted randomness based on reputation, stake, and load
func (k Keeper) SelectProviderAdvanced(ctx context.Context, specs types.ComputeSpec, requestID uint64, preferredProvider string) (sdk.AccAddress, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Try preferred provider first if specified
	if preferredProvider != "" {
		addr, err := sdk.AccAddressFromBech32(preferredProvider)
		if err == nil {
			if k.isProviderEligible(ctx, addr, specs, params) {
				return addr, nil
			}
		}
	}

	// Collect eligible providers and calculate selection scores
	var candidates []ProviderSelectionScore

	err = k.IterateActiveProviders(ctx, func(provider types.Provider) (bool, error) {
		addr, err := sdk.AccAddressFromBech32(provider.Address)
		if err != nil {
			return false, nil
		}

		// Check eligibility
		if !k.isProviderEligible(ctx, addr, specs, params) {
			return false, nil
		}

		// Get advanced reputation
		rep, err := k.GetProviderReputation(ctx, addr)
		if err != nil {
			// Fallback to basic reputation
			rep = &ProviderReputation{
				OverallScore: provider.Reputation,
			}
		}

		// Calculate selection score
		score := k.calculateSelectionScore(ctx, provider, rep, specs, requestID)
		candidates = append(candidates, score)

		return false, nil
	})

	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no eligible providers found")
	}

	// Select provider using weighted random selection
	selected := k.weightedRandomSelection(ctx, candidates, requestID)
	if selected == nil {
		return nil, fmt.Errorf("failed to select provider")
	}

	addr, err := sdk.AccAddressFromBech32(selected.Provider)
	if err != nil {
		return nil, err
	}

	// Emit selection event for transparency
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_selected_advanced",
			sdk.NewAttribute("provider", selected.Provider),
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("reputation_score", fmt.Sprintf("%.3f", selected.ReputationScore)),
			sdk.NewAttribute("stake_weight", fmt.Sprintf("%.3f", selected.StakeWeight)),
			sdk.NewAttribute("load_factor", fmt.Sprintf("%.3f", selected.LoadFactor)),
			sdk.NewAttribute("random_factor", fmt.Sprintf("%.3f", selected.RandomFactor)),
			sdk.NewAttribute("total_score", fmt.Sprintf("%.3f", selected.TotalScore)),
			sdk.NewAttribute("candidates", fmt.Sprintf("%d", len(candidates))),
		),
	)

	return addr, nil
}

// isProviderEligible checks if a provider meets all requirements
func (k Keeper) isProviderEligible(ctx context.Context, provider sdk.AccAddress, specs types.ComputeSpec, params types.Params) bool {
	// Get provider record
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil || !providerRecord.Active {
		return false
	}

	// Check reputation threshold
	if providerRecord.Reputation < params.MinReputationScore {
		return false
	}

	// Check specs compatibility
	if !k.canProviderHandleSpecs(*providerRecord, specs) {
		return false
	}

	// Check capacity
	if err := k.CheckProviderCapacity(ctx, provider, specs); err != nil {
		return false
	}

	return true
}

// calculateSelectionScore computes a multi-factor selection score
func (k Keeper) calculateSelectionScore(ctx context.Context, provider types.Provider, rep *ProviderReputation, specs types.ComputeSpec, requestID uint64) ProviderSelectionScore {
	// Reputation component (0-1)
	reputationScore := float64(rep.OverallScore) / 100.0

	// Stake weight (normalized)
	// Higher stake = higher trust
	// Convert math.Int to float64 by converting to string then parsing
	stakeStr := provider.Stake.String()
	stakeFloat, err := strconv.ParseFloat(stakeStr, 64)
	if err != nil {
		stakeFloat = 1000000.0 // Default to 1M tokens if conversion fails
	}
	stakeWeight := math.Log1p(stakeFloat) / 20.0 // Log scale, capped
	if stakeWeight > 1.0 {
		stakeWeight = 1.0
	}

	// Load factor (inverted - lower load is better)
	tracker, err := k.GetProviderLoadTracker(ctx, sdk.MustAccAddressFromBech32(provider.Address))
	var loadFactor float64 = 1.0
	if err == nil && tracker.MaxConcurrentRequests > 0 {
		utilization := float64(tracker.CurrentRequests) / float64(tracker.MaxConcurrentRequests)
		loadFactor = 1.0 - utilization
	}

	// Random factor for fairness (prevents deterministic gaming)
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, requestID)
	randomInt := k.GenerateSecureRandomness(ctx, seed)
	randomFactor := float64(randomInt.Mod(randomInt, big.NewInt(1000)).Int64()) / 1000.0

	// Combined score with weights
	weights := map[string]float64{
		"reputation": 0.4,
		"stake":      0.3,
		"load":       0.2,
		"random":     0.1,
	}

	totalScore := weights["reputation"]*reputationScore +
		weights["stake"]*stakeWeight +
		weights["load"]*loadFactor +
		weights["random"]*randomFactor

	return ProviderSelectionScore{
		Provider:        provider.Address,
		ReputationScore: reputationScore,
		StakeWeight:     stakeWeight,
		LoadFactor:      loadFactor,
		RandomFactor:    randomFactor,
		TotalScore:      totalScore,
	}
}

// weightedRandomSelection selects a provider using weighted probability
func (k Keeper) weightedRandomSelection(ctx context.Context, candidates []ProviderSelectionScore, requestID uint64) *ProviderSelectionScore {
	if len(candidates) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, candidate := range candidates {
		totalWeight += candidate.TotalScore
	}

	if totalWeight == 0 {
		// Fallback to first candidate
		return &candidates[0]
	}

	// Generate random value
	seed := make([]byte, 16)
	binary.BigEndian.PutUint64(seed, requestID)
	binary.BigEndian.PutUint64(seed[8:], uint64(len(candidates)))
	randomInt := k.GenerateSecureRandomness(ctx, seed)
	randomValue := float64(randomInt.Mod(randomInt, big.NewInt(1000000)).Int64()) / 1000000.0
	randomValue *= totalWeight

	// Select using cumulative distribution
	cumulativeWeight := 0.0
	for i := range candidates {
		cumulativeWeight += candidates[i].TotalScore
		if randomValue <= cumulativeWeight {
			return &candidates[i]
		}
	}

	// Fallback to last candidate
	return &candidates[len(candidates)-1]
}

// Storage functions for advanced reputation

func (k Keeper) GetProviderReputation(ctx context.Context, provider sdk.AccAddress) (*ProviderReputation, error) {
	store := k.getStore(ctx)
	bz := store.Get(ProviderReputationKey(provider))

	if bz == nil {
		return nil, fmt.Errorf("provider reputation not found")
	}

	var rep ProviderReputation
	if err := k.cdc.Unmarshal(bz, &rep); err != nil {
		return nil, err
	}

	return &rep, nil
}

func (k Keeper) SetProviderReputation(ctx context.Context, rep ProviderReputation) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&rep)
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(rep.Provider)
	if err != nil {
		return err
	}

	store.Set(ProviderReputationKey(addr), bz)
	return nil
}

// IterateProviderReputations iterates over all provider reputations
func (k Keeper) IterateProviderReputations(ctx context.Context, cb func(rep ProviderReputation) (stop bool, err error)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ProviderReputationPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var rep ProviderReputation
		if err := k.cdc.Unmarshal(iterator.Value(), &rep); err != nil {
			return err
		}

		stop, err := cb(rep)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// Additional key definitions
var (
	ProviderReputationPrefix = []byte{0x40}
)

func ProviderReputationKey(provider sdk.AccAddress) []byte {
	return append(ProviderReputationPrefix, provider.Bytes()...)
}
