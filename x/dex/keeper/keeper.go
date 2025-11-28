package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

// Keeper of the dex store
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	bankKeeper bankkeeper.Keeper
	ibcKeeper  *ibckeeper.Keeper
}

// NewKeeper creates a new dex Keeper instance
func NewKeeper(cdc codec.BinaryCodec, key storetypes.StoreKey, bankKeeper bankkeeper.Keeper, ibcKeeper *ibckeeper.Keeper) *Keeper {
	return &Keeper{
		storeKey:   key,
		cdc:        cdc,
		bankKeeper: bankKeeper,
		ibcKeeper:  ibcKeeper,
	}
}

// getStore returns the KVStore for the dex module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
}
