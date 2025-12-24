package keeper

import (
	"context"
	"errors"
	"sync"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	portkeeper "github.com/cosmos/ibc-go/v8/modules/core/05-port/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/paw-chain/paw/app/ibcutil"
	computetypes "github.com/paw-chain/paw/x/compute/types"
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
	portKeeper     *portkeeper.Keeper
	authority      string
	scopedKeeper   capabilitykeeper.ScopedKeeper

	// circuitManager handles ZK circuit operations for compute verification.
	// It is lazily initialized on first use to avoid expensive circuit compilation at startup.
	circuitManager *CircuitManager

	metrics *ComputeMetrics

	// authorizedChannelsCache caches the authorized IBC channels to avoid
	// repeated GetParams() calls on every IBC packet. The cache is invalidated
	// when params are updated via SetParams or SetAuthorizedChannels.
	channelCacheMu          sync.RWMutex
	authorizedChannelsCache map[string]struct{}
	channelCacheValid       bool
}

type kvStoreProvider interface {
	KVStore(key storetypes.StoreKey) storetypes.KVStore
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
	portKeeper *portkeeper.Keeper,
	authority string,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	return &Keeper{
		storeKey:                key,
		cdc:                     cdc,
		bankKeeper:              bankKeeper,
		accountKeeper:           accountKeeper,
		stakingKeeper:           stakingKeeper,
		slashingKeeper:          slashingKeeper,
		ibcKeeper:               ibcKeeper,
		portKeeper:              portKeeper,
		authority:               authority,
		scopedKeeper:            scopedKeeper,
		metrics:                 NewComputeMetrics(),
		authorizedChannelsCache: make(map[string]struct{}),
	}
}

// getStore returns the KVStore for the compute module
func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
	if provider, ok := ctx.(kvStoreProvider); ok {
		return provider.KVStore(k.storeKey)
	}

	unwrapped := sdk.UnwrapSDKContext(ctx)
	return unwrapped.KVStore(k.storeKey)
}

// ClaimCapability claims a channel capability for the compute module.
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// GetChannelCapability retrieves a previously claimed channel capability.
func (k Keeper) GetChannelCapability(ctx sdk.Context, portID, channelID string) (*capabilitytypes.Capability, bool) {
	return k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(portID, channelID))
}

// BindPort binds the compute module's IBC port and claims the capability.
func (k Keeper) BindPort(ctx sdk.Context) error {
	if k.portKeeper.IsBound(ctx, computetypes.PortID) {
		return nil
	}

	portCap := k.portKeeper.BindPort(ctx, computetypes.PortID)
	if err := k.scopedKeeper.ClaimCapability(ctx, portCap, host.PortPath(computetypes.PortID)); err != nil {
		if errors.Is(err, capabilitytypes.ErrOwnerClaimed) {
			return nil
		}
		return err
	}
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

// InvalidateChannelCache invalidates the authorized channels cache.
// This must be called whenever params.AuthorizedChannels is modified.
func (k *Keeper) InvalidateChannelCache() {
	k.channelCacheMu.Lock()
	k.channelCacheValid = false
	k.channelCacheMu.Unlock()
}

// SetAuthorizedChannels implements ibcutil.ChannelStore.
// It persists the updated list of authorized IBC channels to module params.
func (k *Keeper) SetAuthorizedChannels(ctx context.Context, channels []ibcutil.AuthorizedChannel) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Convert shared type to module-specific type
	moduleChannels := make([]computetypes.AuthorizedChannel, len(channels))
	for i, ch := range channels {
		moduleChannels[i] = computetypes.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}

	params.AuthorizedChannels = moduleChannels
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	// Invalidate cache after successful param update
	k.InvalidateChannelCache()
	return nil
}

// IsAuthorizedChannel checks whether incoming packets from the given port/channel are allowed.
// Uses an in-memory cache to avoid repeated GetParams() calls on every IBC packet.
func (k *Keeper) IsAuthorizedChannel(ctx sdk.Context, portID, channelID string) bool {
	cacheKey := portID + "/" + channelID

	// Fast path: check cache with read lock
	k.channelCacheMu.RLock()
	if k.channelCacheValid {
		_, authorized := k.authorizedChannelsCache[cacheKey]
		k.channelCacheMu.RUnlock()
		return authorized
	}
	k.channelCacheMu.RUnlock()

	// Cache miss: populate cache with write lock
	k.channelCacheMu.Lock()
	defer k.channelCacheMu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have populated it)
	if k.channelCacheValid {
		_, authorized := k.authorizedChannelsCache[cacheKey]
		return authorized
	}

	// Load channels from params and populate cache
	params, err := k.GetParams(ctx)
	if err != nil {
		ctx.Logger().Error("failed to load params for channel authorization", "error", err)
		return false
	}

	// Clear and rebuild cache
	k.authorizedChannelsCache = make(map[string]struct{}, len(params.AuthorizedChannels))
	for _, ch := range params.AuthorizedChannels {
		k.authorizedChannelsCache[ch.PortId+"/"+ch.ChannelId] = struct{}{}
	}
	k.channelCacheValid = true

	_, authorized := k.authorizedChannelsCache[cacheKey]
	return authorized
}

// AuthorizeChannel appends a port/channel pair to the allowlist when governance approves it.
func (k *Keeper) AuthorizeChannel(ctx sdk.Context, portID, channelID string) error {
	if err := ibcutil.AuthorizeChannel(ctx, k, portID, channelID); err != nil {
		return err
	}
	// Invalidate cache after successful update
	k.InvalidateChannelCache()
	return nil
}

// SetAuthorizedChannelsWithValidation replaces the allowlist with the provided slice, with validation.
func (k *Keeper) SetAuthorizedChannelsWithValidation(ctx sdk.Context, channels []computetypes.AuthorizedChannel) error {
	// Convert module-specific type to shared type
	ibcChannels := make([]ibcutil.AuthorizedChannel, len(channels))
	for i, ch := range channels {
		ibcChannels[i] = ibcutil.AuthorizedChannel{
			PortId:    ch.PortId,
			ChannelId: ch.ChannelId,
		}
	}

	if err := ibcutil.SetAuthorizedChannelsWithValidation(ctx, k, ibcChannels); err != nil {
		return err
	}
	// Invalidate cache after successful update
	k.InvalidateChannelCache()
	return nil
}

// GetCircuitManager returns the circuit manager, lazily initializing it if needed.
// The circuit manager handles ZK-SNARK proof verification for compute results.
func (k *Keeper) GetCircuitManager() *CircuitManager {
	if k.circuitManager == nil {
		k.circuitManager = NewCircuitManager(k)
	}
	return k.circuitManager
}

// InitializeCircuits initializes all ZK circuits for the compute module.
// This is an expensive operation that compiles circuits and generates proving/verifying keys.
// It should be called during genesis initialization or module setup.
func (k *Keeper) InitializeCircuits(ctx context.Context) error {
	cm := k.GetCircuitManager()
	return cm.Initialize(ctx)
}

// VerifyComputeProofWithCircuitManager verifies a compute proof using the circuit manager.
// This provides a higher-level interface for ZK proof verification.
func (k *Keeper) VerifyComputeProofWithCircuitManager(
	ctx sdk.Context,
	proofData []byte,
	requestID uint64,
	resultCommitment interface{},
	providerCommitment interface{},
	resourceCommitment interface{},
) (bool, error) {
	cm := k.GetCircuitManager()

	if !cm.IsInitialized() {
		// Lazy initialization - this will be slow first time but cached subsequently
		if err := cm.Initialize(ctx); err != nil {
			return false, err
		}
	}

	return cm.VerifyComputeProof(ctx, proofData, &ComputePublicInputs{
		RequestID:          requestID,
		ResultCommitment:   resultCommitment,
		ProviderCommitment: providerCommitment,
		ResourceCommitment: resourceCommitment,
	})
}

// VerifyEscrowProofWithCircuitManager verifies an escrow release proof.
func (k *Keeper) VerifyEscrowProofWithCircuitManager(
	ctx sdk.Context,
	proofData []byte,
	requestID uint64,
	escrowAmount uint64,
	requesterCommitment interface{},
	providerCommitment interface{},
	completionCommitment interface{},
) (bool, error) {
	cm := k.GetCircuitManager()

	if !cm.IsInitialized() {
		if err := cm.Initialize(ctx); err != nil {
			return false, err
		}
	}

	return cm.VerifyEscrowProof(ctx, proofData, &EscrowPublicInputs{
		RequestID:            requestID,
		EscrowAmount:         escrowAmount,
		RequesterCommitment:  requesterCommitment,
		ProviderCommitment:   providerCommitment,
		CompletionCommitment: completionCommitment,
	})
}

// VerifyResultProofWithCircuitManager verifies a result correctness proof.
func (k *Keeper) VerifyResultProofWithCircuitManager(
	ctx sdk.Context,
	proofData []byte,
	requestID uint64,
	resultRootHash interface{},
	inputRootHash interface{},
	programHash interface{},
) (bool, error) {
	cm := k.GetCircuitManager()

	if !cm.IsInitialized() {
		if err := cm.Initialize(ctx); err != nil {
			return false, err
		}
	}

	return cm.VerifyResultProof(ctx, proofData, &ResultPublicInputs{
		RequestID:      requestID,
		ResultRootHash: resultRootHash,
		InputRootHash:  inputRootHash,
		ProgramHash:    programHash,
	})
}
