package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
	"github.com/paw-chain/paw/x/shared/nonce"
)

// oracleErrorProvider implements nonce.ErrorProvider for the oracle module.
// It wraps errors using oracle-specific error types while delegating nonce
// validation logic to the shared nonce manager.
type oracleErrorProvider struct{}

// InvalidNonceError returns oracle module's invalid nonce error
func (oracleErrorProvider) InvalidNonceError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidNonce, msg)
}

// InvalidPacketError returns oracle module's invalid packet error
func (oracleErrorProvider) InvalidPacketError(msg string) error {
	return errorsmod.Wrap(types.ErrInvalidPacket, msg)
}

// ValidateIncomingPacketNonce validates packet nonce and timestamp to prevent replay attacks.
// It enforces:
// 1. Timestamp must be within 24 hours of current block time (prevents old packet replay)
// 2. Nonce must be monotonically increasing per channel/sender pair
// 3. Stores the new nonce after successful validation
//
// This method delegates to the shared nonce manager while maintaining the same
// external API for the oracle keeper.
func (k Keeper) ValidateIncomingPacketNonce(ctx sdk.Context, channel, sender string, packetNonce uint64, timestamp int64) error {
	manager := nonce.NewManager(k.storeKey, oracleErrorProvider{}, types.ModuleName)
	return manager.ValidateIncomingPacketNonce(ctx, channel, sender, packetNonce, timestamp)
}

// NextOutboundNonce generates the next monotonically increasing nonce for outgoing packets.
// It atomically increments and returns the next nonce for the given channel/sender pair.
//
// This method delegates to the shared nonce manager while maintaining the same
// external API for the oracle keeper.
func (k Keeper) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	manager := nonce.NewManager(k.storeKey, oracleErrorProvider{}, types.ModuleName)
	return manager.NextOutboundNonce(ctx, channel, sender)
}

// PruneExpiredNonces removes IBC nonces older than the configured TTL.
// This prevents unbounded state growth while maintaining replay attack protection
// within the TTL window.
//
// Returns the number of nonces pruned. Errors are logged but don't halt block production.
func (k Keeper) PruneExpiredNonces(sdkCtx sdk.Context) (int, error) {
	params, err := k.GetParams(sdkCtx)
	if err != nil {
		return 0, errorsmod.Wrap(err, "failed to get params for nonce pruning")
	}

	manager := nonce.NewManager(k.storeKey, oracleErrorProvider{}, types.ModuleName)

	// Use module parameter for TTL, with safety limits
	ttlSeconds := int64(params.NonceTtlSeconds)
	if ttlSeconds <= 0 {
		ttlSeconds = nonce.DefaultNonceTTLSeconds
	}

	// Prune up to 100 nonces per block to prevent gas spikes
	const maxPrunePerBlock = 100

	return manager.PruneExpiredNonces(sdkCtx, ttlSeconds, maxPrunePerBlock)
}
