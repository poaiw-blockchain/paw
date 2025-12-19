package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
	"github.com/paw-chain/paw/x/shared/nonce"
)

// computeErrorProvider implements nonce.ErrorProvider for the compute module.
// It wraps shared nonce errors with compute-specific error types.
type computeErrorProvider struct{}

// InvalidNonceError returns compute module's invalid nonce error with the given message
func (p computeErrorProvider) InvalidNonceError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidNonce, msg)
}

// InvalidPacketError returns compute module's invalid packet error with the given message
func (p computeErrorProvider) InvalidPacketError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidPacket, msg)
}

// nonceManager returns the shared nonce manager instance for the compute module
func (k Keeper) nonceManager() *nonce.Manager {
	return nonce.NewManager(k.storeKey, computeErrorProvider{}, types.ModuleName)
}

// ValidateIncomingPacketNonce validates packet nonce and timestamp to prevent replay attacks.
// It enforces:
// 1. Timestamp must be within 24 hours of current block time (prevents old packet replay)
// 2. Nonce must be monotonically increasing per channel/sender pair
// 3. Stores the new nonce after successful validation
//
// This method delegates to the shared nonce manager while maintaining the compute module's
// public API and error types.
func (k Keeper) ValidateIncomingPacketNonce(ctx sdk.Context, channel, sender string, packetNonce uint64, timestamp int64) error {
	return k.nonceManager().ValidateIncomingPacketNonce(ctx, channel, sender, packetNonce, timestamp)
}

// NextOutboundNonce generates the next monotonically increasing nonce for outgoing packets.
// It atomically increments and returns the next nonce for the given channel/sender pair.
//
// This method delegates to the shared nonce manager while maintaining the compute module's
// public API.
func (k Keeper) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	return k.nonceManager().NextOutboundNonce(ctx, channel, sender)
}
