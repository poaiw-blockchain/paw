package dex

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	"github.com/paw-chain/paw/app/ibcutil"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// keeperAdapter adapts the keeper to implement shared IBC interfaces.
// This avoids modifying the keeper directly and keeps the adapter logic centralized.
type keeperAdapter struct {
	keeper *keeper.Keeper
}

// newKeeperAdapter creates a new keeper adapter.
func newKeeperAdapter(k *keeper.Keeper) *keeperAdapter {
	return &keeperAdapter{keeper: k}
}

// IsAuthorizedChannel implements the ChannelAuthorizer interface.
// Converts ibcutil's bool result to an error for the interface.
func (ka *keeperAdapter) IsAuthorizedChannel(ctx sdk.Context, sourcePort, sourceChannel string) error {
	if ibcutil.IsAuthorizedChannel(ctx, ka.keeper, sourcePort, sourceChannel) {
		return nil
	}
	return types.ErrUnauthorizedChannel
}

// ValidateIncomingPacketNonce implements the NonceValidator interface.
// This is a direct pass-through to the keeper method.
func (ka *keeperAdapter) ValidateIncomingPacketNonce(ctx sdk.Context, sourceChannel, sender string, nonce uint64, timestamp int64) error {
	return ka.keeper.ValidateIncomingPacketNonce(ctx, sourceChannel, sender, nonce, timestamp)
}

// ClaimCapability implements the CapabilityClaimer interface.
// This is a direct pass-through to the keeper method.
func (ka *keeperAdapter) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return ka.keeper.ClaimCapability(ctx, cap, name)
}
