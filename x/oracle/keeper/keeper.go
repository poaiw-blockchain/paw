package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/paw-chain/paw/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	bankKeeper     bankkeeper.Keeper
	stakingKeeper  *stakingkeeper.Keeper
	slashingKeeper slashingkeeper.Keeper
	ibcKeeper      *ibckeeper.Keeper
	authority      string
	scopedKeeper   capabilitykeeper.ScopedKeeper
	portKeeper     *portkeeper.Keeper
	portCapability *capabilitytypes.Capability
}

// NewKeeper creates a new oracle Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bankKeeper bankkeeper.Keeper,
	stakingKeeper *stakingkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	ibcKeeper *ibckeeper.Keeper,
	portKeeper *portkeeper.Keeper,
	authority string,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	return &Keeper{
		storeKey:       key,
		cdc:            cdc,
		bankKeeper:     bankKeeper,
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
		ibcKeeper:      ibcKeeper,
		portKeeper:     portKeeper,
		authority:      authority,
		scopedKeeper:   scopedKeeper,
	}
}

// getStore returns the KVStore for the oracle module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
}

// ClaimCapability claims a channel capability for the oracle module.
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// GetChannelCapability retrieves a channel capability by port/channel identifiers.
func (k Keeper) GetChannelCapability(ctx sdk.Context, portID, channelID string) (*capabilitytypes.Capability, bool) {
	return k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
}

// BindPort binds the oracle IBC port and claims the associated capability.
func (k Keeper) BindPort(ctx sdk.Context) error {
	if k.portKeeper.IsBound(ctx, types.PortID) {
		if cap, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(types.PortID)); ok {
			k.portCapability = cap
		}
		return nil
	}

	portCap := k.portKeeper.BindPort(ctx, types.PortID)
	if err := k.scopedKeeper.ClaimCapability(ctx, portCap, host.PortPath(types.PortID)); err != nil {
		return err
	}
	k.portCapability = portCap
	return nil
}
