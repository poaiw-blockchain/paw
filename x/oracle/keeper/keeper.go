package keeper

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
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
	metrics        *OracleMetrics
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
		metrics:        NewOracleMetrics(),
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

// IsAuthorizedChannel returns nil error if the provided IBC port/channel pair is whitelisted.
// Returns an error if the channel is not authorized or if params cannot be loaded.
func (k Keeper) IsAuthorizedChannel(ctx sdk.Context, portID, channelID string) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		ctx.Logger().Error("failed to load oracle params for channel authorization", "error", err)
		return err
	}

	for _, ch := range params.AuthorizedChannels {
		if ch.PortId == portID && ch.ChannelId == channelID {
			return nil // Authorized
		}
	}

	return types.ErrUnauthorizedChannel
}

// AuthorizeChannel appends a new port/channel pair to the allowed list, preventing duplicates.
func (k Keeper) AuthorizeChannel(ctx sdk.Context, portID, channelID string) error {
	portID = strings.TrimSpace(portID)
	channelID = strings.TrimSpace(channelID)
	if portID == "" || channelID == "" {
		return errorsmod.Wrap(types.ErrInvalidAsset, "port_id and channel_id must be non-empty")
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	for _, ch := range params.AuthorizedChannels {
		if ch.PortId == portID && ch.ChannelId == channelID {
			return nil
		}
	}

	params.AuthorizedChannels = append(params.AuthorizedChannels, types.AuthorizedChannel{
		PortId:    portID,
		ChannelId: channelID,
	})
	return k.SetParams(ctx, params)
}

// SetAuthorizedChannels replaces the currently authorized ports/channels.
func (k Keeper) SetAuthorizedChannels(ctx sdk.Context, channels []types.AuthorizedChannel) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	normalized := make([]types.AuthorizedChannel, 0, len(channels))
	seen := make(map[string]struct{}, len(channels))
	for _, ch := range channels {
		portID := strings.TrimSpace(ch.PortId)
		channelID := strings.TrimSpace(ch.ChannelId)
		if portID == "" || channelID == "" {
			return errorsmod.Wrap(types.ErrInvalidAsset, "port_id and channel_id must be non-empty")
		}
		key := portID + "/" + channelID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, types.AuthorizedChannel{
			PortId:    portID,
			ChannelId: channelID,
		})
	}

	params.AuthorizedChannels = normalized
	return k.SetParams(ctx, params)
}
