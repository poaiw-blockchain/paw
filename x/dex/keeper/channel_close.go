package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
)

// ChannelOperation is an alias to the shared IBC ChannelOperation type.
// This enables a unified struct across all modules (dex, oracle, compute).
type ChannelOperation = sharedibc.ChannelOperation

func pendingOperationPrefix(channelID string) []byte {
	return []byte(fmt.Sprintf("pending_op/%s/", channelID))
}

func pendingOperationKey(channelID string, sequence uint64) []byte {
	key := pendingOperationPrefix(channelID)
	seq := make([]byte, 8)
	binary.BigEndian.PutUint64(seq, sequence)
	return append(key, seq...)
}

func (k Keeper) trackPendingOperation(ctx sdk.Context, channelID, packetType string, sequence uint64) {
	if channelID == "" || packetType == "" || sequence == 0 {
		return
	}

	store := ctx.KVStore(k.storeKey)
	record := ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: packetType,
	}
	bz, err := json.Marshal(record)
	if err != nil {
		return
	}
	store.Set(pendingOperationKey(channelID, sequence), bz)
}

func (k Keeper) clearPendingOperation(ctx sdk.Context, channelID string, sequence uint64) {
	if channelID == "" || sequence == 0 {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(pendingOperationKey(channelID, sequence))
}

// GetPendingOperations returns all pending IBC operations that originated from the given channel.
func (k Keeper) GetPendingOperations(ctx sdk.Context, channelID string) []ChannelOperation {
	store := ctx.KVStore(k.storeKey)
	prefix := pendingOperationPrefix(channelID)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var ops []ChannelOperation
	for ; iterator.Valid(); iterator.Next() {
		var op ChannelOperation
		if err := json.Unmarshal(iterator.Value(), &op); err != nil {
			continue
		}
		ops = append(ops, op)
	}

	return ops
}

// RefundOnChannelClose refunds user funds or cleans up state for a pending operation when a channel closes.
func (k Keeper) RefundOnChannelClose(ctx sdk.Context, op ChannelOperation) error {
	defer k.clearPendingOperation(ctx, op.ChannelID, op.Sequence)

	switch op.PacketType {
	case PacketTypeExecuteSwap:
		if err := k.refundSwap(ctx, op.Sequence, "ibc_channel_closed"); err != nil {
			return err
		}
	case PacketTypeQueryPools:
		k.removePendingQuery(ctx, op.ChannelID, op.Sequence)
	default:
		// No action required for unknown packet types; ensure entry is cleared.
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_channel_cleanup",
			sdk.NewAttribute(types.AttributeKeyChannelID, op.ChannelID),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", op.Sequence)),
			sdk.NewAttribute(types.AttributeKeyPacketType, op.PacketType),
		),
	)

	return nil
}

// TrackPendingOperationForTest exposes trackPendingOperation for tests.
func TrackPendingOperationForTest(k *Keeper, ctx sdk.Context, channelID, packetType string, sequence uint64) {
	k.trackPendingOperation(ctx, channelID, packetType, sequence)
}
