package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// InitGenesis initializes the dex module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	// Initialize pools
	for _, pool := range gs.Pools {
		k.SetPool(ctx, pool)
		k.SetPoolByTokens(ctx, pool.TokenA, pool.TokenB, pool.Id)
	}

	// Set next pool ID if provided, otherwise default to 1
	nextPoolId := gs.NextPoolId
	if nextPoolId == 0 {
		nextPoolId = 1
	}
	k.SetNextPoolId(ctx, nextPoolId)
}

// ExportGenesis returns the dex module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	// Get next pool ID
	nextPoolId := k.GetNextPoolId(ctx)

	// In a full implementation, we would iterate through all pools in the store
	// For now, return minimal genesis state
	return &types.GenesisState{
		Pools:      []types.Pool{},
		NextPoolId: nextPoolId,
	}
}
