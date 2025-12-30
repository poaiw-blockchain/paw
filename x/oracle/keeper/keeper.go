package keeper

import (
	"context"
	"fmt"
	"time"

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

	// ARCH-2: Hooks for cross-module notifications
	hooks types.OracleHooks
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
	// Initialize GeoIP manager (non-fatal if database not available during keeper construction)
	// Final validation happens in InitGenesis based on RequireGeographicDiversity parameter
	// This allows the chain to start for testing without GeoIP database
	geoIPManager, err := NewGeoIPManager("")
	if err != nil {
		// Warning will be logged, but genesis validation will enforce if required
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

// ValidateGeoIPAvailability checks if GeoIP database is available and functional
func (k Keeper) ValidateGeoIPAvailability() error {
	if k.geoIPManager == nil {
		return fmt.Errorf("GeoIP manager not initialized")
	}

	// Test with a known public IP to verify database functionality
	testIP := "8.8.8.8" // Google DNS - should resolve to US
	_, err := k.geoIPManager.GetRegion(testIP)
	if err != nil {
		return fmt.Errorf("GeoIP database validation failed: %w", err)
	}

	return nil
}

// UpdateGeoIPCacheConfig updates the GeoIP cache configuration from params
func (k Keeper) UpdateGeoIPCacheConfig(ctx context.Context) error {
	if k.geoIPManager == nil {
		return nil // No GeoIP manager, nothing to configure
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Update cache TTL if configured
	if params.GeoipCacheTtlSeconds > 0 {
		ttl := time.Duration(params.GeoipCacheTtlSeconds) * time.Second
		k.geoIPManager.SetCacheTTL(ttl)
	}

	// Update cache max entries if configured
	if params.GeoipCacheMaxEntries > 0 {
		k.geoIPManager.SetCacheMaxEntries(int(params.GeoipCacheMaxEntries))
	}

	return nil
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

// GetAuthority returns the authority address for governance
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetHooks sets the oracle hooks.
// ARCH-2: Enables cross-module notifications for price updates.
func (k *Keeper) SetHooks(hooks types.OracleHooks) {
	if k.hooks != nil {
		panic("cannot set oracle hooks twice")
	}
	k.hooks = hooks
}

// GetHooks returns the oracle hooks.
func (k Keeper) GetHooks() types.OracleHooks {
	return k.hooks
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

