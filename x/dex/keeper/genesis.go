package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// InitGenesis initializes the dex module's state from a genesis state
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := k.BindPort(sdkCtx); err != nil {
		return fmt.Errorf("failed to bind IBC port: %w", err)
	}

	// Set parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Set next pool ID counter
	if genState.NextPoolId > 0 {
		store := k.getStore(ctx)
		poolIDBytes := make([]byte, 8)
		// Use binary encoding for the pool ID
		for i := 0; i < 8; i++ {
			poolIDBytes[7-i] = byte(genState.NextPoolId >> (8 * i))
		}
		store.Set(PoolCountKey, poolIDBytes)
	}

	// Initialize pools
	for _, pool := range genState.Pools {
		// Validate pool
		if err := validatePool(&pool); err != nil {
			return fmt.Errorf("invalid pool %d: %w", pool.Id, err)
		}

		// Set pool in store
		if err := k.SetPool(ctx, &pool); err != nil {
			return fmt.Errorf("failed to set pool %d: %w", pool.Id, err)
		}

		// Index pool by tokens
		if err := k.SetPoolByTokens(ctx, pool.TokenA, pool.TokenB, pool.Id); err != nil {
			return fmt.Errorf("failed to index pool %d: %w", pool.Id, err)
		}
	}

	for _, twap := range genState.PoolTwapRecords {
		if err := k.SetPoolTWAP(ctx, twap); err != nil {
			return fmt.Errorf("failed to set pool TWAP for pool %d: %w", twap.PoolId, err)
		}
	}

	// Initialize circuit breaker states
	// NOTE: We only import persistent configuration, NOT volatile runtime state.
	// Runtime state (PausedUntil, NotificationsSent, LastNotification, TriggeredBy, TriggerReason)
	// is intentionally reset to zero values for new chain instances.
	for _, cbState := range genState.CircuitBreakerStates {
		state := CircuitBreakerState{
			// Persistent configuration
			Enabled:        cbState.Enabled,
			LastPrice:      cbState.LastPrice,
			PersistenceKey: cbState.PersistenceKey,

			// Runtime state - always reset to zero values on import
			PausedUntil:       time.Time{}, // Zero time
			NotificationsSent: 0,
			LastNotification:  time.Time{}, // Zero time
			TriggeredBy:       "",
			TriggerReason:     "",
		}

		if err := k.SetCircuitBreakerState(ctx, cbState.PoolId, state); err != nil {
			return fmt.Errorf("failed to set circuit breaker state for pool %d: %w", cbState.PoolId, err)
		}
	}

	// Initialize liquidity positions and validate shares sum equals pool.TotalShares
	poolSharesSums := make(map[uint64]math.Int)
	for _, liqPos := range genState.LiquidityPositions {
		provider, err := sdk.AccAddressFromBech32(liqPos.Provider)
		if err != nil {
			return fmt.Errorf("invalid liquidity provider address %s: %w", liqPos.Provider, err)
		}

		if err := k.SetLiquidity(ctx, liqPos.PoolId, provider, liqPos.Shares); err != nil {
			return fmt.Errorf("failed to set liquidity position for pool %d, provider %s: %w",
				liqPos.PoolId, liqPos.Provider, err)
		}

		// Accumulate shares for validation
		if _, exists := poolSharesSums[liqPos.PoolId]; !exists {
			poolSharesSums[liqPos.PoolId] = math.ZeroInt()
		}
		poolSharesSums[liqPos.PoolId] = poolSharesSums[liqPos.PoolId].Add(liqPos.Shares)
	}

	// Validate that sum of LP shares equals pool.TotalShares for each pool
	// NOTE: We use strict equality here because LP shares represent ownership fractions
	// and must always sum exactly to TotalShares. Unlike reserves (which can accumulate
	// fees), shares are never modified by swaps - only by add/remove liquidity operations.
	//
	// Fee accumulation affects reserves, not shares. The relationship:
	// - Shares: Constant during swaps, only change on liquidity operations
	// - Reserves: Increase from swap fees, decrease from swaps
	// - k-value (reserves product): Can increase up to 10% from fees (see invariants.go)
	//
	// This validation ensures genesis data integrity - corrupted or manually-edited
	// genesis files with mismatched shares will be rejected at chain start.
	for _, pool := range genState.Pools {
		if !pool.TotalShares.IsZero() {
			sharesSum, exists := poolSharesSums[pool.Id]
			if !exists {
				sharesSum = math.ZeroInt()
			}
			if !sharesSum.Equal(pool.TotalShares) {
				return fmt.Errorf("pool %d shares mismatch: sum of LP positions (%s) != pool.TotalShares (%s)",
					pool.Id, sharesSum.String(), pool.TotalShares.String())
			}
		}
	}

	return nil
}

// ExportGenesis exports the dex module's state to a genesis state
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	// Get parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}

	// Get all pools
	pools, err := k.GetAllPools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	// Get next pool ID
	store := k.getStore(ctx)
	bz := store.Get(PoolCountKey)
	var nextPoolID uint64 = 1
	if bz != nil {
		// Decode big-endian uint64
		for i := 0; i < 8; i++ {
			nextPoolID |= uint64(bz[7-i]) << (8 * i)
		}
	}

	// Export circuit breaker states for all pools
	// NOTE: We only export persistent configuration, NOT volatile runtime state.
	// This ensures chain exports/imports don't carry over transient pause states.
	var cbStates []types.CircuitBreakerStateExport
	for _, pool := range pools {
		cbState, err := k.GetPoolCircuitBreakerState(ctx, pool.Id)
		if err == nil {
			cbStates = append(cbStates, types.CircuitBreakerStateExport{
				PoolId:         pool.Id,
				PersistenceKey: cbState.PersistenceKey,

				// Persistent configuration
				Enabled:   cbState.Enabled,
				LastPrice: cbState.LastPrice,

				// Runtime state - exported as zero values
				PausedUntil:       0, // Reset to zero (no pause on new chain)
				NotificationsSent: 0, // Reset counter
				LastNotification:  0, // Reset timestamp
				TriggeredBy:       "", // Clear transient event data
				TriggerReason:     "", // Clear transient event data
			})
		}
	}

	// Export all liquidity positions
	var liqPositions []types.LiquidityPositionExport
	for _, pool := range pools {
		// Iterate over all liquidity providers for this pool
		if err := k.IterateLiquidityByPool(ctx, pool.Id, func(provider sdk.AccAddress, shares math.Int) bool {
			liqPositions = append(liqPositions, types.LiquidityPositionExport{
				PoolId:   pool.Id,
				Provider: provider.String(),
				Shares:   shares,
			})
			return false
		}); err != nil {
			// Log but don't fail export
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Error("failed to iterate liquidity positions", "pool_id", pool.Id, "error", err)
		}
	}

	twapRecords, err := k.GetAllPoolTWAPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export pool TWAPs: %w", err)
	}

	return &types.GenesisState{
		Params:               params,
		Pools:                pools,
		NextPoolId:           nextPoolID,
		PoolTwapRecords:      twapRecords,
		CircuitBreakerStates: cbStates,
		LiquidityPositions:   liqPositions,
	}, nil
}

// validatePool validates a pool's structure
func validatePool(pool *types.Pool) error {
	if pool.Id == 0 {
		return fmt.Errorf("pool ID cannot be zero")
	}

	if pool.TokenA == "" || pool.TokenB == "" {
		return fmt.Errorf("token denoms cannot be empty")
	}

	if pool.TokenA == pool.TokenB {
		return fmt.Errorf("token denoms must be different")
	}

	if pool.TokenA > pool.TokenB {
		return fmt.Errorf("tokens must be ordered: tokenA < tokenB")
	}

	if pool.ReserveA.IsNegative() || pool.ReserveB.IsNegative() {
		return fmt.Errorf("reserves cannot be negative")
	}

	if pool.TotalShares.IsNegative() {
		return fmt.Errorf("total shares cannot be negative")
	}

	if _, err := sdk.AccAddressFromBech32(pool.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// Validate constant product invariant (allowing for some tolerance due to fees)
	if !pool.ReserveA.IsZero() && !pool.ReserveB.IsZero() && pool.TotalShares.IsZero() {
		return fmt.Errorf("pool has reserves but no shares")
	}

	return nil
}
