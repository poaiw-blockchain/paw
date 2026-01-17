package keeper

import (
	"context"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/paw-chain/paw/app/ibcutil"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// channelSender abstracts the subset of ChannelKeeper we need for sending packets (test override).
type channelSender interface {
	SendPacket(ctx sdk.Context,
		channelCap *capabilitytypes.Capability,
		sourcePort string,
		sourceChannel string,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		data []byte,
	) (uint64, error)
}

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
	channelSender      channelSender  // test override for SendPacket

	// PERF-10: Token graph cache to avoid rebuilding on every route search
	// The cache is invalidated when pools are created or deleted by incrementing poolVersion
	tokenGraphCache   *tokenGraph // Cached token graph for route finding
	tokenGraphVersion uint64      // Version when cache was built

	// ARCH-2: Hooks for cross-module notifications
	hooks dextypes.DexHooks

	// test-only: optional override for sending IBC packets; nil in production.
	sendPacketFn func(ctx sdk.Context, connectionID, channelID string, data []byte, timeout time.Duration) (uint64, error)
}

// kvStoreProvider is an interface for types that can provide a KVStore.
// CODE-10: This allows getStore() to work with both sdk.Context and direct store providers.
type kvStoreProvider interface {
	KVStore(key storetypes.StoreKey) storetypes.KVStore
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

// getStore returns the KVStore for the dex module.
// CODE-10: Uses defensive pattern to handle both sdk.Context and direct kvStoreProvider.
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	if provider, ok := ctx.(kvStoreProvider); ok {
		return provider.KVStore(k.storeKey)
	}

	unwrapped := sdk.UnwrapSDKContext(ctx)
	return unwrapped.KVStore(k.storeKey)
}

// GetStoreKey returns the store key for testing purposes
func (k Keeper) GetStoreKey() storetypes.StoreKey {
	return k.storeKey
}

// GetAuthority returns the module authority for testing purposes
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetHooks sets the DEX hooks.
// ARCH-2: Enables cross-module notifications for DEX events.
func (k *Keeper) SetHooks(hooks dextypes.DexHooks) {
	if k.hooks != nil {
		panic("cannot set dex hooks twice")
	}
	k.hooks = hooks
}

// SetChannelSender overrides the channel send path for testing.
func (k *Keeper) SetChannelSender(sender channelSender) {
	k.channelSender = sender
}

// GetHooks returns the DEX hooks.
func (k Keeper) GetHooks() dextypes.DexHooks {
	return k.hooks
}

// ScopedKeeper returns the capability scoped keeper (testing only).
func (k Keeper) ScopedKeeper() capabilitykeeper.ScopedKeeper {
	return k.scopedKeeper
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
		return fmt.Errorf("BindPort: claim port capability: %w", err)
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
		return nil, fmt.Errorf("GetAuthorizedChannels: get params: %w", err)
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
		return fmt.Errorf("SetAuthorizedChannels: get params: %w", err)
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
