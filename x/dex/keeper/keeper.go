package keeper

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// Keeper of the dex store
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	bankKeeper     bankkeeper.Keeper
	ibcKeeper      *ibckeeper.Keeper
	portKeeper     *portkeeper.Keeper
	scopedKeeper   capabilitykeeper.ScopedKeeper
	portCapability *capabilitytypes.Capability
	metrics        *DEXMetrics
}

// NewKeeper creates a new dex Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bankKeeper bankkeeper.Keeper,
	ibcKeeper *ibckeeper.Keeper,
	portKeeper *portkeeper.Keeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	return &Keeper{
		storeKey:     key,
		cdc:          cdc,
		bankKeeper:   bankKeeper,
		ibcKeeper:    ibcKeeper,
		portKeeper:   portKeeper,
		scopedKeeper: scopedKeeper,
		metrics:      NewDEXMetrics(),
	}
}

// getStore returns the KVStore for the dex module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.KVStore(k.storeKey)
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
func (k Keeper) BindPort(ctx sdk.Context) error {
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

// IsAuthorizedChannel returns true if the provided port/channel pair is allowed to relay packets.
func (k Keeper) IsAuthorizedChannel(ctx sdk.Context, portID, channelID string) bool {
	params, err := k.GetParams(ctx)
	if err != nil {
		ctx.Logger().Error("failed to load dex params for channel authorization", "error", err)
		return false
	}

	for _, ch := range params.AuthorizedChannels {
		if ch.PortId == portID && ch.ChannelId == channelID {
			return true
		}
	}
	return false
}

// AuthorizeChannel adds a port/channel pair to the allowlist, deduplicating entries.
func (k Keeper) AuthorizeChannel(ctx sdk.Context, portID, channelID string) error {
	portID = strings.TrimSpace(portID)
	channelID = strings.TrimSpace(channelID)
	if portID == "" || channelID == "" {
		return errorsmod.Wrap(dextypes.ErrInvalidInput, "port_id and channel_id must be non-empty")
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

	params.AuthorizedChannels = append(params.AuthorizedChannels, dextypes.AuthorizedChannel{
		PortId:    portID,
		ChannelId: channelID,
	})
	return k.SetParams(ctx, params)
}

// SetAuthorizedChannels replaces the entire allowlist with the provided port/channel pairs.
func (k Keeper) SetAuthorizedChannels(ctx sdk.Context, channels []dextypes.AuthorizedChannel) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	normalized := make([]dextypes.AuthorizedChannel, 0, len(channels))
	seen := make(map[string]struct{}, len(channels))
	for _, ch := range channels {
		portID := strings.TrimSpace(ch.PortId)
		channelID := strings.TrimSpace(ch.ChannelId)
		if portID == "" || channelID == "" {
			return errorsmod.Wrap(dextypes.ErrInvalidInput, "port_id and channel_id must be non-empty")
		}

		key := portID + "/" + channelID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, dextypes.AuthorizedChannel{
			PortId:    portID,
			ChannelId: channelID,
		})
	}

	params.AuthorizedChannels = normalized
	return k.SetParams(ctx, params)
}
