package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	// Set next pool ID
	if genState.NextPoolId == 0 {
		genState.NextPoolId = 1
	}
	k.SetNextPoolId(ctx, genState.NextPoolId)

	// Initialize pools
	for _, pool := range genState.Pools {
		k.SetPool(ctx, pool)
		k.SetPoolByTokens(ctx, pool.TokenA, pool.TokenB, pool.Id)
	}
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.DefaultGenesis()

	// Export pools
	// TODO: Implement pool iteration
	// genesis.Pools = k.GetAllPools(ctx)
	genesis.NextPoolId = k.GetNextPoolId(ctx)

	return genesis
}
