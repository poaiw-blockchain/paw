package keeper_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	keeperpkg "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestOnTimeoutPacketRefundsEscrow(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-timeout-1"
	requester := sdk.AccAddress(bytes.Repeat([]byte{0x21}, 20))
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20))
	escrowAmount := sdkmath.NewInt(2_500_000)
	channelID := "channel-7"
	sequence := uint64(42)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	job := keeperpkg.CrossChainComputeJob{
		JobID:        jobID,
		Requester:    requester.String(),
		Provider:     provider.String(),
		Status:       "pending",
		Progress:     10,
		SubmittedAt:  ctx.BlockTime(),
		Requirements: keeperpkg.JobRequirements{},
		EscrowAmount: escrowAmount,
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("job_%s", jobID)), job)

	escrow := keeperpkg.CrossChainEscrow{
		JobID:     jobID,
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    escrowAmount,
		Status:    "locked",
		LockedAt:  ctx.BlockTime(),
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("escrow_%s", jobID)), escrow)
	store.Set([]byte(fmt.Sprintf("pending_job_%d", sequence)), []byte(jobID))
	recordPendingOperation(t, store, channelID, sequence, keeperpkg.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: keeperpkg.PacketTypeSubmitJob,
		JobID:      jobID,
	})

	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowAmount))
	require.NoError(t, getBankKeeper(t, k).MintCoins(ctx, types.ModuleName, coins))

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, keeperpkg.PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	before := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.True(t, before.IsZero())

	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	after := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.Equal(t, escrowAmount, after.Amount)

	var stored keeperpkg.CrossChainComputeJob
	require.NoError(t, json.Unmarshal(store.Get([]byte(fmt.Sprintf("job_%s", jobID))), &stored))
	require.Equal(t, "timeout", stored.Status)

	events := ctx.EventManager().Events()
	hasTimeout := false
	hasRefund := false
	for _, evt := range events {
		switch evt.Type {
		case "job_submission_timeout":
			hasTimeout = true
		case "escrow_refunded":
			hasRefund = true
		}
	}

	require.True(t, hasTimeout, "expected job timeout event")
	require.True(t, hasRefund, "expected escrow refund event")
}

func TestOnTimeoutPacketSubmitJobMissingJob(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-9"
	sequence := uint64(77)

	// No job or escrow exists - simulates already processed or invalid submission

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, keeperpkg.PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should not error even with missing job
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))
}

func TestOnTimeoutPacketDiscoverProviders(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-2"
	sequence := uint64(88)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	// Record pending discovery operation
	recordPendingOperation(t, store, channelID, sequence, keeperpkg.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: keeperpkg.PacketTypeDiscoverProviders,
		JobID:      "",
	})

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s","requirements":{"cpu":4,"memory":8000}}`, keeperpkg.PacketTypeDiscoverProviders)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Verify pending operation exists
	opKey := getPendingOperationKey(channelID, sequence)
	require.NotNil(t, store.Get(opKey))

	// Process timeout - should remove pending discovery
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify pending operation was removed
	require.Nil(t, store.Get(opKey))
}

func TestOnTimeoutPacketJobStatus(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-status-query-1"
	channelID := "channel-4"
	sequence := uint64(123)

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s","job_id":"%s"}`, keeperpkg.PacketTypeJobStatus, jobID)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Status query timeout is non-critical - should succeed without error
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// No state changes expected for status query timeout
	// This is a read-only operation, so timeout just means we don't get the status
}

func TestOnTimeoutPacketJobSubmissionRefundStateConsistency(t *testing.T) {
	// This test verifies that state remains consistent after a job submission timeout
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	jobID := "job-consistency-check"
	requester := sdk.AccAddress(bytes.Repeat([]byte{0x77}, 20))
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x88}, 20))
	escrowAmount := sdkmath.NewInt(5_000_000)
	channelID := "channel-11"
	sequence := uint64(200)

	storeKey := getStoreKey(t, k)
	store := ctx.KVStore(storeKey)

	// Create job with specific status
	job := keeperpkg.CrossChainComputeJob{
		JobID:        jobID,
		Requester:    requester.String(),
		Provider:     provider.String(),
		Status:       "pending",
		Progress:     0,
		SubmittedAt:  ctx.BlockTime(),
		Requirements: keeperpkg.JobRequirements{},
		EscrowAmount: escrowAmount,
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("job_%s", jobID)), job)

	// Create locked escrow
	escrow := keeperpkg.CrossChainEscrow{
		JobID:     jobID,
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    escrowAmount,
		Status:    "locked",
		LockedAt:  ctx.BlockTime(),
	}
	mustSetJSON(t, store, []byte(fmt.Sprintf("escrow_%s", jobID)), escrow)

	// Track pending submission
	store.Set([]byte(fmt.Sprintf("pending_job_%d", sequence)), []byte(jobID))
	recordPendingOperation(t, store, channelID, sequence, keeperpkg.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: keeperpkg.PacketTypeSubmitJob,
		JobID:      jobID,
	})

	// Fund module for refund
	coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowAmount))
	require.NoError(t, getBankKeeper(t, k).MintCoins(ctx, types.ModuleName, coins))

	packet := channeltypes.Packet{
		Data:             []byte(fmt.Sprintf(`{"type":"%s"}`, keeperpkg.PacketTypeSubmitJob)),
		Sequence:         sequence,
		SourcePort:       types.PortID,
		SourceChannel:    channelID,
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Process timeout
	require.NoError(t, k.OnTimeoutPacket(ctx, packet))

	// Verify state consistency:

	// 1. Job status should be "timeout"
	var storedJob keeperpkg.CrossChainComputeJob
	require.NoError(t, json.Unmarshal(store.Get([]byte(fmt.Sprintf("job_%s", jobID))), &storedJob))
	require.Equal(t, "timeout", storedJob.Status)

	// 2. Escrow should be refunded to requester
	balance := getBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.Equal(t, escrowAmount, balance.Amount)

	// 3. Module account should be drained
	moduleBalance := getBankKeeper(t, k).GetBalance(ctx, sdk.AccAddress(types.ModuleName), "upaw")
	require.True(t, moduleBalance.IsZero())

	// 4. Pending job tracking should be removed
	require.Nil(t, store.Get([]byte(fmt.Sprintf("pending_job_%d", sequence))))

	// 5. Pending operation should be removed
	opKey := getPendingOperationKey(channelID, sequence)
	require.Nil(t, store.Get(opKey))

	// 6. Events should be emitted
	events := ctx.EventManager().Events()
	hasTimeoutEvent := false
	hasRefundEvent := false
	for _, evt := range events {
		switch evt.Type {
		case "job_submission_timeout":
			hasTimeoutEvent = true
		case "escrow_refunded":
			hasRefundEvent = true
		}
	}
	require.True(t, hasTimeoutEvent, "expected job_submission_timeout event")
	require.True(t, hasRefundEvent, "expected escrow_refunded event")
}

func TestOnTimeoutPacketInvalidPacketType(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{"type":"unknown_compute_packet"}`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for unknown packet type
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown packet type")
}

func TestOnTimeoutPacketMalformedJSON(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{invalid json payload`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for malformed JSON
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal packet data")
}

func TestOnTimeoutPacketMissingType(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	packet := channeltypes.Packet{
		Data:             []byte(`{"job_id":"some-job"}`),
		Sequence:         1,
		SourcePort:       types.PortID,
		SourceChannel:    "channel-0",
		TimeoutTimestamp: uint64(time.Now().Add(time.Minute).UnixNano()),
	}

	// Should return error for missing type field
	err := k.OnTimeoutPacket(ctx, packet)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing packet type")
}

func getStoreKey(t *testing.T, k *keeperpkg.Keeper) storetypes.StoreKey {
	t.Helper()
	val := reflect.ValueOf(k).Elem().FieldByName("storeKey")
	return reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem().Interface().(storetypes.StoreKey)
}

func getBankKeeper(t *testing.T, k *keeperpkg.Keeper) bankkeeper.Keeper {
	t.Helper()
	val := reflect.ValueOf(k).Elem().FieldByName("bankKeeper")
	return reflect.NewAt(val.Type(), unsafe.Pointer(val.UnsafeAddr())).Elem().Interface().(bankkeeper.Keeper)
}

func mustSetJSON(t *testing.T, store storetypes.KVStore, key []byte, value interface{}) {
	t.Helper()
	bz, err := json.Marshal(value)
	require.NoError(t, err)
	store.Set(key, bz)
}

func recordPendingOperation(t *testing.T, store storetypes.KVStore, channelID string, sequence uint64, op keeperpkg.ChannelOperation) {
	t.Helper()
	bz, err := json.Marshal(op)
	require.NoError(t, err)
	prefix := []byte(fmt.Sprintf("compute_pending/%s/", channelID))
	seqBz := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBz, sequence)
	store.Set(append(prefix, seqBz...), bz)
}

func getPendingOperationKey(channelID string, sequence uint64) []byte {
	prefix := []byte(fmt.Sprintf("compute_pending/%s/", channelID))
	seqBz := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBz, sequence)
	return append(prefix, seqBz...)
}
