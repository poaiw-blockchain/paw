package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// CircuitBreakerConfig holds the configuration for circuit breaker thresholds
type CircuitBreakerConfig struct {
	// Price change thresholds for different time windows
	Threshold1Min  math.LegacyDec `json:"threshold_1min"`  // e.g., 10%
	Threshold5Min  math.LegacyDec `json:"threshold_5min"`  // e.g., 20%
	Threshold15Min math.LegacyDec `json:"threshold_15min"` // e.g., 25%
	Threshold1Hour math.LegacyDec `json:"threshold_1hour"` // e.g., 30%

	// Cooldown period before trading can resume (in seconds)
	CooldownPeriod int64 `json:"cooldown_period"` // e.g., 600 seconds (10 minutes)

	// Gradual resume settings
	EnableGradualResume bool           `json:"enable_gradual_resume"` // Whether to limit volume on resume
	ResumeVolumeFactor  math.LegacyDec `json:"resume_volume_factor"`  // e.g., 0.5 = 50% of normal volume allowed initially
}

// CircuitBreakerState tracks the state of the circuit breaker for a pool
type CircuitBreakerState struct {
	PoolId          uint64         `json:"pool_id"`
	IsTripped       bool           `json:"is_tripped"`
	TripReason      string         `json:"trip_reason"`
	TrippedAt       int64          `json:"tripped_at"`        // Block height when tripped
	TrippedAtTime   int64          `json:"tripped_at_time"`   // Unix timestamp when tripped
	PriceAtTrip     math.LegacyDec `json:"price_at_trip"`     // Price when circuit breaker tripped
	CanResumeAt     int64          `json:"can_resume_at"`     // Unix timestamp when trading can resume
	GradualResume   bool           `json:"gradual_resume"`    // Whether in gradual resume mode
	ResumeStartedAt int64          `json:"resume_started_at"` // Unix timestamp when gradual resume started
}

// PriceSnapshot stores a snapshot of price at a specific time for volatility detection
type PriceSnapshot struct {
	Timestamp int64          `json:"timestamp"`
	Price     math.LegacyDec `json:"price"`
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Threshold1Min:       math.LegacyNewDecWithPrec(10, 2), // 10%
		Threshold5Min:       math.LegacyNewDecWithPrec(20, 2), // 20%
		Threshold15Min:      math.LegacyNewDecWithPrec(25, 2), // 25%
		Threshold1Hour:      math.LegacyNewDecWithPrec(30, 2), // 30%
		CooldownPeriod:      600,                              // 10 minutes
		EnableGradualResume: true,
		ResumeVolumeFactor:  math.LegacyNewDecWithPrec(5, 1), // 50%
	}
}

// GetCircuitBreakerConfig retrieves the circuit breaker configuration
func (k Keeper) GetCircuitBreakerConfig(ctx sdk.Context) CircuitBreakerConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CircuitBreakerConfigKey)
	if bz == nil {
		return DefaultCircuitBreakerConfig()
	}

	var config CircuitBreakerConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		k.Logger(ctx).Error("failed to unmarshal circuit breaker config", "error", err)
		return DefaultCircuitBreakerConfig()
	}
	return config
}

// SetCircuitBreakerConfig sets the circuit breaker configuration
func (k Keeper) SetCircuitBreakerConfig(ctx sdk.Context, config CircuitBreakerConfig) error {
	// Validate config
	if err := validateCircuitBreakerConfig(config); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal circuit breaker config: %w", err)
	}
	store.Set(types.CircuitBreakerConfigKey, bz)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"circuit_breaker_config_updated",
			sdk.NewAttribute("threshold_1min", config.Threshold1Min.String()),
			sdk.NewAttribute("threshold_5min", config.Threshold5Min.String()),
			sdk.NewAttribute("threshold_15min", config.Threshold15Min.String()),
			sdk.NewAttribute("threshold_1hour", config.Threshold1Hour.String()),
			sdk.NewAttribute("cooldown_period", fmt.Sprintf("%d", config.CooldownPeriod)),
		),
	)

	return nil
}

// GetCircuitBreakerState retrieves the circuit breaker state for a pool
func (k Keeper) GetCircuitBreakerState(ctx sdk.Context, poolId uint64) CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetCircuitBreakerStateKey(poolId))
	if bz == nil {
		return CircuitBreakerState{
			PoolId:    poolId,
			IsTripped: false,
		}
	}

	var state CircuitBreakerState
	if err := json.Unmarshal(bz, &state); err != nil {
		k.Logger(ctx).Error("failed to unmarshal circuit breaker state", "pool_id", poolId, "error", err)
		return CircuitBreakerState{
			PoolId:    poolId,
			IsTripped: false,
		}
	}
	return state
}

// SetCircuitBreakerState sets the circuit breaker state for a pool
func (k Keeper) SetCircuitBreakerState(ctx sdk.Context, state CircuitBreakerState) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(&state)
	if err != nil {
		// This should never happen with simple struct
		panic(fmt.Sprintf("failed to marshal circuit breaker state: %v", err))
	}
	store.Set(types.GetCircuitBreakerStateKey(state.PoolId), bz)
}

// CheckCircuitBreaker checks if the circuit breaker should trip for a pool
// This should be called before every swap operation
func (k Keeper) CheckCircuitBreaker(ctx sdk.Context, poolId uint64) error {
	// Check if circuit breaker is already tripped
	state := k.GetCircuitBreakerState(ctx, poolId)
	if state.IsTripped {
		// Check if cooldown period has passed
		currentTime := ctx.BlockTime().Unix()
		if currentTime < state.CanResumeAt {
			remainingTime := state.CanResumeAt - currentTime
			return types.ErrCircuitBreakerTripped.Wrapf(
				"circuit breaker is active for pool %d. Reason: %s. Can resume in %d seconds",
				poolId, state.TripReason, remainingTime,
			)
		}

		// Cooldown period has passed, check if we should resume
		// This will be handled in the gradual resume logic
	}

	// Check for price volatility
	if err := k.DetectPriceVolatility(ctx, poolId); err != nil {
		return err
	}

	return nil
}

// DetectPriceVolatility detects excessive price movements and trips circuit breaker if needed
func (k Keeper) DetectPriceVolatility(ctx sdk.Context, poolId uint64) error {
	config := k.GetCircuitBreakerConfig(ctx)
	currentTime := ctx.BlockTime().Unix()

	// Get current price
	currentPrice, err := k.GetSpotPrice(ctx, poolId)
	if err != nil {
		// If we can't get price, allow the operation (might be new pool)
		return nil
	}

	// Get price observations
	observations := k.GetPriceObservations(ctx, poolId)
	if len(observations) < 2 {
		// Not enough data to detect volatility
		return nil
	}

	// Check volatility over different time windows
	type window struct {
		seconds   int64
		threshold math.LegacyDec
		name      string
	}

	windows := []window{
		{60, config.Threshold1Min, "1 minute"},     // 1 minute
		{300, config.Threshold5Min, "5 minutes"},   // 5 minutes
		{900, config.Threshold15Min, "15 minutes"}, // 15 minutes
		{3600, config.Threshold1Hour, "1 hour"},    // 1 hour
	}

	for _, w := range windows {
		startTime := currentTime - w.seconds

		// Find the earliest observation in this window
		var earliestPrice math.LegacyDec
		var foundPrice bool
		for i := len(observations) - 1; i >= 0; i-- {
			if observations[i].Timestamp >= startTime {
				earliestPrice = observations[i].Price
				foundPrice = true
			} else {
				break
			}
		}

		if !foundPrice {
			// No observations in this window, skip
			continue
		}

		// Calculate price change percentage
		if earliestPrice.IsZero() {
			continue
		}

		priceChange := currentPrice.Sub(earliestPrice).Quo(earliestPrice).Abs()

		// Check if threshold exceeded
		if priceChange.GT(w.threshold) {
			// Trip the circuit breaker!
			k.TripCircuitBreaker(ctx, poolId, fmt.Sprintf(
				"Price volatility exceeded %s threshold: %.2f%% change in %s (threshold: %.2f%%)",
				w.name,
				priceChange.MulInt64(100).MustFloat64(),
				w.name,
				w.threshold.MulInt64(100).MustFloat64(),
			), currentPrice)

			return types.ErrCircuitBreakerTripped.Wrapf(
				"circuit breaker tripped for pool %d: %s",
				poolId,
				fmt.Sprintf("%.2f%% price change in %s", priceChange.MulInt64(100).MustFloat64(), w.name),
			)
		}
	}

	return nil
}

// TripCircuitBreaker trips the circuit breaker for a pool
func (k Keeper) TripCircuitBreaker(ctx sdk.Context, poolId uint64, reason string, priceAtTrip math.LegacyDec) {
	config := k.GetCircuitBreakerConfig(ctx)
	currentTime := ctx.BlockTime().Unix()

	state := CircuitBreakerState{
		PoolId:        poolId,
		IsTripped:     true,
		TripReason:    reason,
		TrippedAt:     ctx.BlockHeight(),
		TrippedAtTime: currentTime,
		PriceAtTrip:   priceAtTrip,
		CanResumeAt:   currentTime + config.CooldownPeriod,
		GradualResume: config.EnableGradualResume,
	}

	k.SetCircuitBreakerState(ctx, state)

	// Emit critical event for monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerTripped,
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("tripped_at_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("tripped_at_time", fmt.Sprintf("%d", currentTime)),
			sdk.NewAttribute("price_at_trip", priceAtTrip.String()),
			sdk.NewAttribute("can_resume_at", fmt.Sprintf("%d", state.CanResumeAt)),
			sdk.NewAttribute("severity", "critical"),
		),
	)

	// Log critical message
	k.Logger(ctx).Error(
		"Circuit breaker tripped",
		"pool_id", poolId,
		"reason", reason,
		"height", ctx.BlockHeight(),
		"price", priceAtTrip.String(),
	)
}

// ResumeTrading resumes trading for a pool (can be called by governance or after cooldown)
func (k Keeper) ResumeTrading(ctx sdk.Context, poolId uint64, isGovernanceOverride bool) error {
	state := k.GetCircuitBreakerState(ctx, poolId)

	if !state.IsTripped {
		return types.ErrInvalidParams.Wrap("circuit breaker is not tripped for this pool")
	}

	currentTime := ctx.BlockTime().Unix()

	// If not governance override, check cooldown period
	if !isGovernanceOverride && currentTime < state.CanResumeAt {
		remainingTime := state.CanResumeAt - currentTime
		return types.ErrInvalidParams.Wrapf(
			"cooldown period not elapsed. Can resume in %d seconds", remainingTime,
		)
	}

	// Reset circuit breaker state
	config := k.GetCircuitBreakerConfig(ctx)
	newState := CircuitBreakerState{
		PoolId:          poolId,
		IsTripped:       false,
		GradualResume:   config.EnableGradualResume,
		ResumeStartedAt: currentTime,
	}

	k.SetCircuitBreakerState(ctx, newState)

	// Emit event
	eventType := types.EventTypeCircuitBreakerResumed
	attributes := []sdk.Attribute{
		sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
		sdk.NewAttribute("resumed_at_height", fmt.Sprintf("%d", ctx.BlockHeight())),
		sdk.NewAttribute("resumed_at_time", fmt.Sprintf("%d", currentTime)),
		sdk.NewAttribute("governance_override", fmt.Sprintf("%t", isGovernanceOverride)),
	}

	if config.EnableGradualResume {
		attributes = append(attributes, sdk.NewAttribute("gradual_resume", "enabled"))
		attributes = append(attributes, sdk.NewAttribute("volume_factor", config.ResumeVolumeFactor.String()))
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(eventType, attributes...))

	// Log
	k.Logger(ctx).Info(
		"Trading resumed",
		"pool_id", poolId,
		"governance_override", isGovernanceOverride,
		"gradual_resume", config.EnableGradualResume,
	)

	return nil
}

// CheckSwapVolumeLimit checks if a swap respects gradual resume volume limits
func (k Keeper) CheckSwapVolumeLimit(ctx sdk.Context, poolId uint64, amountIn math.Int) error {
	state := k.GetCircuitBreakerState(ctx, poolId)

	// If circuit breaker is tripped, reject
	if state.IsTripped {
		return types.ErrCircuitBreakerTripped.Wrap("circuit breaker is currently active")
	}

	// If not in gradual resume mode, allow all swaps
	if !state.GradualResume || state.ResumeStartedAt == 0 {
		return nil
	}

	config := k.GetCircuitBreakerConfig(ctx)
	currentTime := ctx.BlockTime().Unix()

	// Check if gradual resume period is over (e.g., 1 hour after resume)
	gradualResumePeriod := int64(3600) // 1 hour
	if currentTime > state.ResumeStartedAt+gradualResumePeriod {
		// Gradual resume period is over, disable it
		state.GradualResume = false
		k.SetCircuitBreakerState(ctx, state)
		return nil
	}

	// Get pool to check reserves
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	// Calculate maximum allowed swap amount (percentage of pool reserves)
	// Use the smaller reserve as the base
	var baseReserve math.Int
	if pool.ReserveA.LT(pool.ReserveB) {
		baseReserve = pool.ReserveA
	} else {
		baseReserve = pool.ReserveB
	}

	// Maximum swap size = baseReserve * resumeVolumeFactor
	maxSwapAmount := math.LegacyNewDecFromInt(baseReserve).Mul(config.ResumeVolumeFactor).TruncateInt()

	if amountIn.GT(maxSwapAmount) {
		return types.ErrInvalidAmount.Wrapf(
			"swap amount exceeds gradual resume limit: %s > %s (%.0f%% of pool reserve)",
			amountIn.String(),
			maxSwapAmount.String(),
			config.ResumeVolumeFactor.MulInt64(100).MustFloat64(),
		)
	}

	return nil
}

// validateCircuitBreakerConfig validates circuit breaker configuration
func validateCircuitBreakerConfig(config CircuitBreakerConfig) error {
	// Validate thresholds are positive and reasonable
	if config.Threshold1Min.IsNegative() || config.Threshold1Min.IsZero() {
		return types.ErrInvalidParams.Wrap("1-minute threshold must be positive")
	}
	if config.Threshold5Min.IsNegative() || config.Threshold5Min.IsZero() {
		return types.ErrInvalidParams.Wrap("5-minute threshold must be positive")
	}
	if config.Threshold15Min.IsNegative() || config.Threshold15Min.IsZero() {
		return types.ErrInvalidParams.Wrap("15-minute threshold must be positive")
	}
	if config.Threshold1Hour.IsNegative() || config.Threshold1Hour.IsZero() {
		return types.ErrInvalidParams.Wrap("1-hour threshold must be positive")
	}

	// Validate thresholds are in ascending order (longer windows should have higher thresholds)
	if config.Threshold5Min.LT(config.Threshold1Min) {
		return types.ErrInvalidParams.Wrap("5-minute threshold should be >= 1-minute threshold")
	}
	if config.Threshold15Min.LT(config.Threshold5Min) {
		return types.ErrInvalidParams.Wrap("15-minute threshold should be >= 5-minute threshold")
	}
	if config.Threshold1Hour.LT(config.Threshold15Min) {
		return types.ErrInvalidParams.Wrap("1-hour threshold should be >= 15-minute threshold")
	}

	// Validate thresholds are not too high (>100% is unreasonable)
	maxThreshold := math.LegacyNewDec(1) // 100%
	if config.Threshold1Hour.GT(maxThreshold) {
		return types.ErrInvalidParams.Wrap("thresholds cannot exceed 100%")
	}

	// Validate cooldown period
	if config.CooldownPeriod <= 0 {
		return types.ErrInvalidParams.Wrap("cooldown period must be positive")
	}
	if config.CooldownPeriod > 86400 { // Max 24 hours
		return types.ErrInvalidParams.Wrap("cooldown period cannot exceed 24 hours")
	}

	// Validate resume volume factor
	if config.EnableGradualResume {
		if config.ResumeVolumeFactor.IsNegative() || config.ResumeVolumeFactor.IsZero() {
			return types.ErrInvalidParams.Wrap("resume volume factor must be positive")
		}
		if config.ResumeVolumeFactor.GT(math.LegacyOneDec()) {
			return types.ErrInvalidParams.Wrap("resume volume factor cannot exceed 100%")
		}
	}

	return nil
}

// IsCircuitBreakerTripped checks if circuit breaker is currently tripped for a pool
func (k Keeper) IsCircuitBreakerTripped(ctx sdk.Context, poolId uint64) bool {
	state := k.GetCircuitBreakerState(ctx, poolId)

	if !state.IsTripped {
		return false
	}

	// Check if cooldown period has naturally expired
	currentTime := ctx.BlockTime().Unix()
	if currentTime >= state.CanResumeAt {
		// Auto-resume after cooldown
		_ = k.ResumeTrading(ctx, poolId, false)
		return false
	}

	return true
}

// GetAllCircuitBreakerStates returns all circuit breaker states (for queries/monitoring)
func (k Keeper) GetAllCircuitBreakerStates(ctx sdk.Context) []CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.CircuitBreakerStateKeyPrefix)
	defer iterator.Close()

	states := []CircuitBreakerState{}
	for ; iterator.Valid(); iterator.Next() {
		var state CircuitBreakerState
		if err := json.Unmarshal(iterator.Value(), &state); err != nil {
			k.Logger(ctx).Error("failed to unmarshal circuit breaker state", "error", err)
			continue
		}
		states = append(states, state)
	}

	return states
}
