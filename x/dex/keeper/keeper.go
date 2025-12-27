package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/paw-chain/paw/app/ibcutil"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// Keeper of the dex store
type Keeper struct {
	storeKey           storetypes.StoreKey
	cdc                codec.BinaryCodec
	bankKeeper         bankkeeper.Keeper
	ibcKeeper          *ibckeeper.Keeper
	portKeeper         *portkeeper.Keeper
	scopedKeeper       capabilitykeeper.ScopedKeeper
	portCapability     *capabilitytypes.Capability
	authority          string
	metrics            *DEXMetrics
	moduleAddressCache sdk.AccAddress // Cached module address to avoid repeated allocations
}

// NewKeeper creates a new dex Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bankKeeper bankkeeper.Keeper,
	ibcKeeper *ibckeeper.Keeper,
	portKeeper *portkeeper.Keeper,
	authority string,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	return &Keeper{
		storeKey:           key,
		cdc:                cdc,
		bankKeeper:         bankKeeper,
		ibcKeeper:          ibcKeeper,
		portKeeper:         portKeeper,
		authority:          authority,
		scopedKeeper:       scopedKeeper,
		metrics:            NewDEXMetrics(),
		moduleAddressCache: sdk.AccAddress([]byte(dextypes.ModuleName)), // Cache module address at init
	}
}

// getStore returns the KVStore for the dex module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
}

// GetStoreKey returns the store key for testing purposes
func (k Keeper) GetStoreKey() storetypes.StoreKey {
	return k.storeKey
}

// GetAuthority returns the module authority for testing purposes
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ClaimCapability claims a channel capability for later authentication.
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// GetChannelCapability retrieves a previously claimed channel capability.
func (k Keeper) GetChannelCapability(ctx sdk.Context, portID, channelID string) (*capabilitytypes.Capability, bool) {
	return k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
}

// BindPort binds the IBC port for the dex module and claims the capability.
func (k *Keeper) BindPort(ctx sdk.Context) error {
	if k.portKeeper.IsBound(ctx, dextypes.PortID) {
		if cap, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(dextypes.PortID)); ok {
			k.portCapability = cap
		}
		return nil
	}

	portCap := k.portKeeper.BindPort(ctx, dextypes.PortID)
	if err := k.scopedKeeper.ClaimCapability(ctx, portCap, host.PortPath(dextypes.PortID)); err != nil {
		return err
	}
	k.portCapability = portCap
	return nil
}

// BankKeeper returns the underlying bank keeper so tests can inspect balances.
func (k Keeper) BankKeeper() bankkeeper.Keeper {
	return k.bankKeeper
}

// GetAuthorizedChannels implements ibcutil.ChannelStore.
// It retrieves the current list of authorized IBC channels from module params.
func (k Keeper) GetAuthorizedChannels(ctx context.Context) ([]ibcutil.AuthorizedChannel, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Convert module-specific type to shared type
	channels := make([]ibcutil.AuthorizedChannel, len(params.AuthorizedChannels))
	for i, ch := range params.AuthorizedChannels {
		channels[i] = ibcutil.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}
	return channels, nil
}

// SetAuthorizedChannels implements ibcutil.ChannelStore.
// It persists the updated list of authorized IBC channels to module params.
func (k Keeper) SetAuthorizedChannels(ctx context.Context, channels []ibcutil.AuthorizedChannel) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Convert shared type to module-specific type
	moduleChannels := make([]dextypes.AuthorizedChannel, len(channels))
	for i, ch := range channels {
		moduleChannels[i] = dextypes.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}

	params.AuthorizedChannels = moduleChannels
	return k.SetParams(ctx, params)
}

