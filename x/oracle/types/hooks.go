package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
)

// OracleHooks defines the interface for oracle module callbacks.
// ARCH-2: Enables cross-module notifications for price updates.
type OracleHooks interface {
	// AfterPriceAggregated is called when a new price has been aggregated for an asset.
	// Subscribers can react to price updates (e.g., DEX can update price-dependent logic).
	AfterPriceAggregated(ctx context.Context, asset string, price sdkmath.LegacyDec, blockHeight int64) error

	// AfterPriceSubmitted is called when a validator submits a price.
	// Useful for monitoring and analytics hooks.
	AfterPriceSubmitted(ctx context.Context, validator string, asset string, price sdkmath.LegacyDec) error

	// OnCircuitBreakerTriggered is called when the oracle circuit breaker activates.
	// Dependent modules should pause price-sensitive operations.
	OnCircuitBreakerTriggered(ctx context.Context, reason string) error
}

// MultiOracleHooks combines multiple oracle hooks into a single hook that calls all of them.
type MultiOracleHooks []OracleHooks

// NewMultiOracleHooks creates a new MultiOracleHooks from a list of hooks.
func NewMultiOracleHooks(hooks ...OracleHooks) MultiOracleHooks {
	return hooks
}

// AfterPriceAggregated calls AfterPriceAggregated on all registered hooks.
func (h MultiOracleHooks) AfterPriceAggregated(ctx context.Context, asset string, price sdkmath.LegacyDec, blockHeight int64) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterPriceAggregated(ctx, asset, price, blockHeight); err != nil {
			return err
		}
	}
	return nil
}

// AfterPriceSubmitted calls AfterPriceSubmitted on all registered hooks.
func (h MultiOracleHooks) AfterPriceSubmitted(ctx context.Context, validator string, asset string, price sdkmath.LegacyDec) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.AfterPriceSubmitted(ctx, validator, asset, price); err != nil {
			return err
		}
	}
	return nil
}

// OnCircuitBreakerTriggered calls OnCircuitBreakerTriggered on all registered hooks.
func (h MultiOracleHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	for _, hook := range h {
		if hook == nil {
			continue
		}
		if err := hook.OnCircuitBreakerTriggered(ctx, reason); err != nil {
			return err
		}
	}
	return nil
}
