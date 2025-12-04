package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// ChannelOperation captures metadata about pending IBC packets so we can refund escrow on channel close.
type ChannelOperation struct {
	ChannelID   string `json:"channel_id"`
	Sequence    uint64 `json:"sequence"`
	PacketType  string `json:"packet_type"`
	JobID       string `json:"job_id,omitempty"`
	TargetChain string `json:"target_chain,omitempty"`
}

func computePendingPrefix(channelID string) []byte {
	return []byte(fmt.Sprintf("compute_pending/%s/", channelID))
}

func computePendingKey(channelID string, sequence uint64) []byte {
	prefix := computePendingPrefix(channelID)
	seq := make([]byte, 8)
	binary.BigEndian.PutUint64(seq, sequence)
	return append(prefix, seq...)
}

func (k Keeper) trackPendingOperation(ctx sdk.Context, op ChannelOperation) {
	if op.ChannelID == "" || op.Sequence == 0 || op.PacketType == "" {
		return
	}

	bz, err := json.Marshal(op)
	if err != nil {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(computePendingKey(op.ChannelID, op.Sequence), bz)
}

func (k Keeper) clearPendingOperation(ctx sdk.Context, channelID string, sequence uint64) {
	if channelID == "" || sequence == 0 {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(computePendingKey(channelID, sequence))
}

// GetPendingOperations collects pending compute operations for the provided channel.
func (k Keeper) GetPendingOperations(ctx sdk.Context, channelID string) []ChannelOperation {
	store := ctx.KVStore(k.storeKey)
	it := storetypes.KVStorePrefixIterator(store, computePendingPrefix(channelID))
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

// RefundOnChannelClose refunds escrow or cleans pending state when the channel closes unexpectedly.
func (k Keeper) RefundOnChannelClose(ctx sdk.Context, op ChannelOperation) error {
	defer k.clearPendingOperation(ctx, op.ChannelID, op.Sequence)

	switch op.PacketType {
	case PacketTypeSubmitJob:
		if op.JobID != "" {
			if err := k.refundJobOnChannelClose(ctx, op.JobID); err != nil {
				return err
			}
		}
		k.removePendingJobSubmission(ctx, op.ChannelID, op.Sequence)
	case PacketTypeDiscoverProviders:
		k.removePendingDiscovery(ctx, op.ChannelID, op.Sequence)
	default:
		// Nothing else to clean up.
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_channel_cleanup",
			sdk.NewAttribute(types.AttributeKeyChannelID, op.ChannelID),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", op.Sequence)),
			sdk.NewAttribute(types.AttributeKeyPacketType, op.PacketType),
		),
	)

	return nil
}

func (k Keeper) refundJobOnChannelClose(ctx sdk.Context, jobID string) error {
	job := k.getJob(ctx, jobID)
	if job != nil && job.Status != "completed" && job.Status != "failed" {
		job.Status = "channel_closed"
		job.Progress = progressForStatus("timeout", job.Progress)
		k.storeJob(ctx, jobID, job)
		if err := k.TrackCrossChainJobStatus(ctx, jobID, "channel_closed", "ibc_channel_closed"); err != nil {
			ctx.Logger().Error("failed to mark job channel closure", "job_id", jobID, "error", err)
		}
	}

	if err := k.RefundEscrowOnTimeout(ctx, jobID, "ibc_channel_closed"); err != nil {
		return err
	}

	return nil
}

// TrackPendingOperationForTest exposes trackPendingOperation for white-box tests.
func TrackPendingOperationForTest(k *Keeper, ctx sdk.Context, op ChannelOperation) {
	k.trackPendingOperation(ctx, op)
}
