package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

// Keeper of the compute store
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	bankKeeper     bankkeeper.Keeper
	accountKeeper  accountkeeper.AccountKeeper
	stakingKeeper  *stakingkeeper.Keeper
	slashingKeeper slashingkeeper.Keeper
	ibcKeeper      *ibckeeper.Keeper
	authority      string
}

// NewKeeper creates a new compute Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bankKeeper bankkeeper.Keeper,
	accountKeeper accountkeeper.AccountKeeper,
	stakingKeeper *stakingkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	ibcKeeper *ibckeeper.Keeper,
	authority string,
) *Keeper {
	return &Keeper{
		storeKey:       key,
		cdc:            cdc,
		bankKeeper:     bankKeeper,
		accountKeeper:  accountKeeper,
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
		ibcKeeper:      ibcKeeper,
		authority:      authority,
	}
}

// getStore returns the KVStore for the compute module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
}
