package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// ChannelOperation tracks outbound oracle packet state so we can clean up when the channel closes.
type ChannelOperation struct {
	ChannelID  string `json:"channel_id"`
	ChainID    string `json:"chain_id"`
	Sequence   uint64 `json:"sequence"`
	PacketType string `json:"packet_type"`
}

func channelOpPrefix(channelID string) []byte {
	return []byte(fmt.Sprintf("oracle_pending_op/%s/", channelID))
}

func channelOpKey(channelID string, sequence uint64) []byte {
	prefix := channelOpPrefix(channelID)
	seq := make([]byte, 8)
	binary.BigEndian.PutUint64(seq, sequence)
	return append(prefix, seq...)
}

func (k Keeper) trackPendingOperation(ctx sdk.Context, channelID, chainID, packetType string, sequence uint64) {
	if channelID == "" || packetType == "" || sequence == 0 {
		return
	}

	record := ChannelOperation{
		ChannelID:  channelID,
		ChainID:    chainID,
		Sequence:   sequence,
		PacketType: packetType,
	}

	bz, err := json.Marshal(record)
	if err != nil {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(channelOpKey(channelID, sequence), bz)
}

func (k Keeper) clearPendingOperation(ctx sdk.Context, channelID string, sequence uint64) {
	if channelID == "" || sequence == 0 {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(channelOpKey(channelID, sequence))
}

// GetPendingOperations returns all unresolved oracle IBC operations for a channel.
func (k Keeper) GetPendingOperations(ctx sdk.Context, channelID string) []ChannelOperation {
	store := ctx.KVStore(k.storeKey)
	it := storetypes.KVStorePrefixIterator(store, channelOpPrefix(channelID))
	defer it.Close()

	var ops []ChannelOperation
	for ; it.Valid(); it.Next() {
		var op ChannelOperation
		if err := json.Unmarshal(it.Value(), &op); err != nil {
			continue
		}
		ops = append(ops, op)
	}

	return ops
}

// RefundOnChannelClose removes pending state and penalizes the remote oracle when a channel closes.
func (k Keeper) RefundOnChannelClose(ctx sdk.Context, op ChannelOperation) error {
	defer k.clearPendingOperation(ctx, op.ChannelID, op.Sequence)

	switch op.PacketType {
	case PacketTypeSubscribePrices:
		k.removePendingSubscription(ctx, op.ChannelID, op.Sequence)
		k.penalizeOracleSource(ctx, op.ChainID, "ibc_channel_closed")
	case PacketTypeQueryPrice:
		k.removePendingPriceQuery(ctx, op.ChannelID, op.Sequence)
		k.penalizeOracleSource(ctx, op.ChainID, "ibc_channel_closed")
	default:
		// Nothing to do.
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_channel_cleanup",
			sdk.NewAttribute(types.AttributeKeyChannelID, op.ChannelID),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", op.Sequence)),
			sdk.NewAttribute(types.AttributeKeyPacketType, op.PacketType),
		),
	)

	return nil
}

// TrackPendingOperationForTest exposes the pending operation tracker to tests.
func TrackPendingOperationForTest(k *Keeper, ctx sdk.Context, channelID, chainID, packetType string, sequence uint64) {
	k.trackPendingOperation(ctx, channelID, chainID, packetType, sequence)
}
