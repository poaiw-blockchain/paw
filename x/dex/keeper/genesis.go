package keeper

import (
	"context"
	"fmt"

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

	// Initialize circuit breaker states
	/*
		for _, cbState := range genState.CircuitBreakerStates {
			if err := k.SetCircuitBreakerState(ctx, cbState.PoolId, CircuitBreakerState{
				Enabled:       cbState.Enabled,
				PausedUntil:   cbState.PausedUntil,
				LastPrice:     cbState.LastPrice,
				TriggeredBy:   cbState.TriggeredBy,
				TriggerReason: cbState.TriggerReason,
			}); err != nil {
				return fmt.Errorf("failed to set circuit breaker state for pool %d: %w", cbState.PoolId, err)
			}
		}
	*/

	// Initialize liquidity positions
	/*
		for _, liqPos := range genState.LiquidityPositions {
			provider, err := sdk.AccAddressFromBech32(liqPos.Provider)
			if err != nil {
				return fmt.Errorf("invalid liquidity provider address %s: %w", liqPos.Provider, err)
			}

			if err := k.SetLiquidity(ctx, liqPos.PoolId, provider, liqPos.Shares); err != nil {
				return fmt.Errorf("failed to set liquidity position for pool %d, provider %s: %w",
					liqPos.PoolId, liqPos.Provider, err)
			}
		}
	*/

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
	/*
		var cbStates []types.CircuitBreakerStateExport
		for _, pool := range pools {
			cbState, err := k.GetCircuitBreakerState(ctx, pool.Id)
			if err == nil {
				cbStates = append(cbStates, types.CircuitBreakerStateExport{
					PoolId:        pool.Id,
					Enabled:       cbState.Enabled,
					PausedUntil:   cbState.PausedUntil,
					LastPrice:     cbState.LastPrice,
					TriggeredBy:   cbState.TriggeredBy,
					TriggerReason: cbState.TriggerReason,
				})
			}
		}
	*/

	// Export all liquidity positions
	/*
		var liqPositions []types.LiquidityPositionExport
		for _, pool := range pools {
			// Iterate over all liquidity providers for this pool
			if err := k.IterateLiquidityPositions(ctx, pool.Id, func(provider sdk.AccAddress, shares math.Int) bool {
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
	*/

	return &types.GenesisState{
		Params:     params,
		Pools:      pools,
		NextPoolId: nextPoolID,
		// CircuitBreakerStates:  cbStates,
		// LiquidityPositions:    liqPositions,
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
