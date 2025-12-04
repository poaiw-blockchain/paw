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
