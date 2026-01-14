package integration

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
)

// OracleIntegration provides integration with the Oracle module
type OracleIntegration struct {
	keeper *oraclekeeper.Keeper
}

// NewOracleIntegration creates a new Oracle integration
func NewOracleIntegration(keeper *oraclekeeper.Keeper) *OracleIntegration {
	return &OracleIntegration{
		keeper: keeper,
	}
}

// Pause pauses all Oracle operations
func (o *OracleIntegration) Pause(ctx sdk.Context, actor, reason string) error {
	return o.keeper.OpenCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// Resume resumes all Oracle operations
func (o *OracleIntegration) Resume(ctx sdk.Context, actor, reason string) error {
	return o.keeper.CloseCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// IsBlocked checks if Oracle operations are blocked
func (o *OracleIntegration) IsBlocked(ctx sdk.Context) bool {
	return o.keeper.IsCircuitBreakerOpen(sdk.WrapSDKContext(ctx))
}

// GetState retrieves the circuit breaker state
func (o *OracleIntegration) GetState(ctx sdk.Context) (bool, string, string) {
	return o.keeper.GetCircuitBreakerState(sdk.WrapSDKContext(ctx))
}

// OverridePrice sets an emergency price override
func (o *OracleIntegration) OverridePrice(ctx sdk.Context, pair string, price *big.Int, durationSecs int64, actor, reason string) error {
	return o.keeper.SetPriceOverride(sdk.WrapSDKContext(ctx), pair, price, durationSecs, actor, reason)
}

// ClearPriceOverride removes a price override
func (o *OracleIntegration) ClearPriceOverride(ctx sdk.Context, pair string) {
	o.keeper.ClearPriceOverride(sdk.WrapSDKContext(ctx), pair)
}

// GetPriceWithOverride retrieves price with override check
func (o *OracleIntegration) GetPriceWithOverride(ctx sdk.Context, pair string) (*big.Int, bool) {
	return o.keeper.GetPriceWithOverride(sdk.WrapSDKContext(ctx), pair)
}

// DisableSlashing temporarily disables slashing
func (o *OracleIntegration) DisableSlashing(ctx sdk.Context, actor, reason string) error {
	return o.keeper.DisableSlashing(sdk.WrapSDKContext(ctx), actor, reason)
}

// EnableSlashing re-enables slashing
func (o *OracleIntegration) EnableSlashing(ctx sdk.Context, actor, reason string) error {
	return o.keeper.EnableSlashing(sdk.WrapSDKContext(ctx), actor, reason)
}

// IsSlashingDisabled checks if slashing is disabled
func (o *OracleIntegration) IsSlashingDisabled(ctx sdk.Context) bool {
	return o.keeper.IsSlashingDisabled(sdk.WrapSDKContext(ctx))
}

// PauseFeed pauses a specific feed type
func (o *OracleIntegration) PauseFeed(ctx sdk.Context, feedType, actor, reason string) error {
	return o.keeper.OpenFeedCircuitBreaker(sdk.WrapSDKContext(ctx), feedType, actor, reason)
}

// ResumeFeed resumes a specific feed type
func (o *OracleIntegration) ResumeFeed(ctx sdk.Context, feedType, actor, reason string) error {
	return o.keeper.CloseFeedCircuitBreaker(sdk.WrapSDKContext(ctx), feedType, actor, reason)
}

// IsFeedBlocked checks if a feed is blocked
func (o *OracleIntegration) IsFeedBlocked(ctx sdk.Context, feedType string) bool {
	return o.keeper.IsFeedCircuitBreakerOpen(sdk.WrapSDKContext(ctx), feedType)
}

// GetActivePriceOverrides retrieves all active price overrides
func (o *OracleIntegration) GetActivePriceOverrides(ctx sdk.Context) (map[string]*big.Int, error) {
	// This would need to iterate through all pairs and check for overrides
	// Implementation depends on how pairs are stored in the Oracle module
	overrides := make(map[string]*big.Int)

	// Example pairs - in production, this would fetch from the module
	pairs := []string{"BTC/USD", "ETH/USD", "ATOM/USD"}

	for _, pair := range pairs {
		if price, hasOverride := o.keeper.GetPriceOverride(sdk.WrapSDKContext(ctx), pair); hasOverride {
			overrides[pair] = price
		}
	}

	return overrides, nil
}

// EmergencyPriceFreeze freezes price updates for a pair
func (o *OracleIntegration) EmergencyPriceFreeze(ctx sdk.Context, pair, actor, reason string) error {
	// Pause the specific feed
	if err := o.PauseFeed(ctx, pair, actor, reason); err != nil {
		return fmt.Errorf("failed to freeze price feed: %w", err)
	}

	// Get current price and set as override indefinitely
	currentPrice, err := o.keeper.GetPrice(sdk.WrapSDKContext(ctx), pair)
	if err != nil {
		return fmt.Errorf("current price not found for pair %q: %w", pair, err)
	}

	// Set override for a very long duration (e.g., 1 year)
	durationSecs := int64(365 * 24 * 60 * 60)
	if err := o.keeper.SetPriceOverride(sdk.WrapSDKContext(ctx), pair, currentPrice.Price.BigInt(), durationSecs, actor, "emergency freeze: "+reason); err != nil {
		return fmt.Errorf("failed to set price override: %w", err)
	}

	return nil
}

// ValidateFeedHealth checks feed health before operations
func (o *OracleIntegration) ValidateFeedHealth(ctx sdk.Context, feedType string) error {
	// Check if feed operations are allowed
	if err := o.keeper.CheckFeedCircuitBreaker(sdk.WrapSDKContext(ctx), feedType); err != nil {
		return err
	}

	// Additional health checks could go here
	// For example: validator participation, price deviation, etc.

	return nil
}
