package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
	"github.com/paw-chain/paw/x/shared/nonce"
)

// dexErrorProvider implements nonce.ErrorProvider using dex module error types
type dexErrorProvider struct{}

// InvalidNonceError returns dex-specific invalid nonce error
func (dexErrorProvider) InvalidNonceError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidNonce, msg)
}

// InvalidPacketError returns dex-specific invalid packet error
func (dexErrorProvider) InvalidPacketError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidPacket, msg)
}

// getnonceManager creates a nonce manager instance for the keeper.
// We create it on-the-fly rather than storing it as a field to avoid
// adding dependencies to the Keeper struct initialization.
func (k Keeper) getnonceManager() *nonce.Manager {
	return nonce.NewManager(k.storeKey, dexErrorProvider{}, types.ModuleName)
}

// ValidateIncomingPacketNonce validates packet nonce and timestamp to prevent replay attacks.
// It enforces:
// 1. Timestamp must be within 24 hours of current block time (prevents old packet replay)
// 2. Nonce must be monotonically increasing per channel/sender pair
// 3. Stores the new nonce after successful validation
//
// This method delegates to the shared nonce manager while maintaining the same public API.
func (k Keeper) ValidateIncomingPacketNonce(ctx sdk.Context, channel, sender string, packetNonce uint64, timestamp int64) error {
	return k.getnonceManager().ValidateIncomingPacketNonce(ctx, channel, sender, packetNonce, timestamp)
}

// NextOutboundNonce generates the next monotonically increasing nonce for outgoing packets.
// It atomically increments and returns the next nonce for the given channel/sender pair.
//
// This method delegates to the shared nonce manager while maintaining the same public API.
func (k Keeper) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	return k.getnonceManager().NextOutboundNonce(ctx, channel, sender)
}
