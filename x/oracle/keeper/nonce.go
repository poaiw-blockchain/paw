package keeper

import (
	"encoding/binary"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/oracle/types"
)

const (
	incomingNoncePrefix = "nonce"
	sendNoncePrefix     = "nonce_send"
)

func encodeNonce(n uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	return bz
}

func decodeNonce(bz []byte) uint64 {
	if len(bz) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func normalizeSender(sender string) string {
	if sender == "" {
		return types.ModuleName
	}
	return sender
}

func (k Keeper) incomingNonceKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", incomingNoncePrefix, channel, normalizeSender(sender)))
}

func (k Keeper) sendNonceKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", sendNoncePrefix, channel, normalizeSender(sender)))
}

func (k Keeper) getIncomingNonce(ctx sdk.Context, channel, sender string) uint64 {
	store := ctx.KVStore(k.storeKey)
	return decodeNonce(store.Get(k.incomingNonceKey(channel, sender)))
}

func (k Keeper) setIncomingNonce(ctx sdk.Context, channel, sender string, nonce uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(k.incomingNonceKey(channel, sender), encodeNonce(nonce))
}

func (k Keeper) ValidateIncomingPacketNonce(ctx sdk.Context, channel, sender string, packetNonce uint64) error {
	if packetNonce == 0 {
		return errorsmod.Wrap(types.ErrInvalidNonce, "nonce must be greater than zero")
	}
	if channel == "" {
		return errorsmod.Wrap(types.ErrInvalidPacket, "source channel missing")
	}

	stored := k.getIncomingNonce(ctx, channel, sender)
	if packetNonce <= stored {
		return errorsmod.Wrapf(types.ErrInvalidNonce, "packet nonce %d not greater than stored %d", packetNonce, stored)
	}
	k.setIncomingNonce(ctx, channel, sender, packetNonce)
	return nil
}

func (k Keeper) getSendNonce(ctx sdk.Context, channel, sender string) uint64 {
	store := ctx.KVStore(k.storeKey)
	return decodeNonce(store.Get(k.sendNonceKey(channel, sender)))
}

func (k Keeper) setSendNonce(ctx sdk.Context, channel, sender string, nonce uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(k.sendNonceKey(channel, sender), encodeNonce(nonce))
}

func (k Keeper) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	if channel == "" {
		channel = "unknown"
	}
	next := k.getSendNonce(ctx, channel, sender) + 1
	k.setSendNonce(ctx, channel, sender, next)
	return next
}
