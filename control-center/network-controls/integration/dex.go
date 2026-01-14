package integration

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
)

// DEXIntegration provides integration with the DEX module
type DEXIntegration struct {
	keeper *dexkeeper.Keeper
}

// NewDEXIntegration creates a new DEX integration
func NewDEXIntegration(keeper *dexkeeper.Keeper) *DEXIntegration {
	return &DEXIntegration{
		keeper: keeper,
	}
}

// Pause pauses all DEX operations
func (d *DEXIntegration) Pause(ctx sdk.Context, actor, reason string) error {
	return d.keeper.OpenCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// Resume resumes all DEX operations
func (d *DEXIntegration) Resume(ctx sdk.Context, actor, reason string) error {
	return d.keeper.CloseCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// IsBlocked checks if DEX operations are blocked
func (d *DEXIntegration) IsBlocked(ctx sdk.Context) bool {
	return d.keeper.IsCircuitBreakerOpen(sdk.WrapSDKContext(ctx))
}

// GetState retrieves the circuit breaker state
func (d *DEXIntegration) GetState(ctx sdk.Context) (bool, string, string) {
	return d.keeper.GetCircuitBreakerState(sdk.WrapSDKContext(ctx))
}

// PausePool pauses a specific pool
func (d *DEXIntegration) PausePool(ctx sdk.Context, poolID uint64, actor, reason string) error {
	return d.keeper.OpenPoolCircuitBreaker(sdk.WrapSDKContext(ctx), poolID, actor, reason)
}

// ResumePool resumes a specific pool
func (d *DEXIntegration) ResumePool(ctx sdk.Context, poolID uint64, actor, reason string) error {
	return d.keeper.ClosePoolCircuitBreaker(sdk.WrapSDKContext(ctx), poolID, actor, reason)
}

// IsPoolBlocked checks if a pool is blocked
func (d *DEXIntegration) IsPoolBlocked(ctx sdk.Context, poolID uint64) bool {
	return d.keeper.IsPoolCircuitBreakerOpen(sdk.WrapSDKContext(ctx), poolID)
}

// GetAllPools retrieves all pools (for status reporting)
func (d *DEXIntegration) GetAllPools(ctx sdk.Context) ([]uint64, error) {
	pools, err := d.keeper.GetAllPools(sdk.WrapSDKContext(ctx))
	if err != nil {
		return nil, err
	}
	poolIDs := make([]uint64, len(pools))
	for i, pool := range pools {
		poolIDs[i] = pool.Id
	}
	return poolIDs, nil
}

// EmergencyLiquidityProtection implements emergency liquidity protection
func (d *DEXIntegration) EmergencyLiquidityProtection(ctx sdk.Context, poolID uint64, actor, reason string) error {
	// Pause the pool
	if err := d.PausePool(ctx, poolID, actor, reason); err != nil {
		return fmt.Errorf("failed to pause pool: %w", err)
	}

	// Additional liquidity protection logic could go here
	// For example: preventing withdrawals, limiting swaps, etc.

	return nil
}

// ValidatePoolHealth checks pool health before operations
func (d *DEXIntegration) ValidatePoolHealth(ctx sdk.Context, poolID uint64) error {
	// Check if pool operations are allowed
	if err := d.keeper.CheckPoolCircuitBreaker(sdk.WrapSDKContext(ctx), poolID); err != nil {
		return err
	}

	// Additional health checks could go here
	// For example: liquidity thresholds, price impact limits, etc.

	return nil
}
