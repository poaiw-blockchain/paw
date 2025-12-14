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

	"github.com/paw-chain/paw/app/ibcutil"
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
	metrics        *OracleMetrics
	geoIPManager   *GeoIPManager
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
	// Initialize GeoIP manager (non-fatal if database not available)
	// This allows the chain to start even without GeoIP database
	// Location verification will be disabled until database is loaded
	geoIPManager, err := NewGeoIPManager("")
	if err != nil {
		// Log warning but don't fail - GeoIP is optional during initialization
		// Validators should configure GEOIP_DB_PATH for production
	}

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
		metrics:        NewOracleMetrics(),
		geoIPManager:   geoIPManager,
	}
}

// getStore returns the KVStore for the oracle module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
}

// GetStoreKey returns the store key for testing purposes
func (k Keeper) GetStoreKey() storetypes.StoreKey {
	return k.storeKey
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
func (k *Keeper) BindPort(ctx sdk.Context) error {
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
	moduleChannels := make([]types.AuthorizedChannel, len(channels))
	for i, ch := range channels {
		moduleChannels[i] = types.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}

	params.AuthorizedChannels = moduleChannels
	return k.SetParams(ctx, params)
}

// IsAuthorizedChannel returns nil error if the provided IBC port/channel pair is whitelisted.
// Returns an error if the channel is not authorized or if params cannot be loaded.
func (k Keeper) IsAuthorizedChannel(ctx sdk.Context, portID, channelID string) error {
	if ibcutil.IsAuthorizedChannel(ctx, k, portID, channelID) {
		return nil
	}
	return types.ErrUnauthorizedChannel
}

// AuthorizeChannel appends a new port/channel pair to the allowed list, preventing duplicates.
func (k Keeper) AuthorizeChannel(ctx sdk.Context, portID, channelID string) error {
	return ibcutil.AuthorizeChannel(ctx, k, portID, channelID)
}

// SetAuthorizedChannelsWithValidation replaces the currently authorized ports/channels with validation.
func (k Keeper) SetAuthorizedChannelsWithValidation(ctx sdk.Context, channels []types.AuthorizedChannel) error {
	// Convert module-specific type to shared type
	ibcChannels := make([]ibcutil.AuthorizedChannel, len(channels))
	for i, ch := range channels {
		ibcChannels[i] = ibcutil.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}

	return ibcutil.SetAuthorizedChannelsWithValidation(ctx, k, ibcChannels)
}
