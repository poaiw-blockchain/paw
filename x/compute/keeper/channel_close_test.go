package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestTrackAndClearPendingOperation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	op := ChannelOperation{
		ChannelID:  "channel-0",
		Sequence:   7,
		PacketType: PacketTypeSubmitJob,
		JobID:      "job-track",
	}
	k.trackPendingOperation(sdkCtx, op)

	pending := k.GetPendingOperations(sdkCtx, "channel-0")
	require.Len(t, pending, 1)
	require.Equal(t, op.Sequence, pending[0].Sequence)

	k.clearPendingOperation(sdkCtx, op.ChannelID, op.Sequence)
	pending = k.GetPendingOperations(sdkCtx, "channel-0")
	require.Len(t, pending, 0)
}

func TestRefundOnChannelCloseSubmitJob(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("channel-close-req"))
	amount := sdk.NewInt64Coin("upaw", 250)
	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))

	job := &CrossChainComputeJob{
		JobID:       "job-close",
		Status:      "submitted",
		Progress:    20,
		SubmittedAt: time.Now(),
	}
	k.storeJob(sdkCtx, job.JobID, job)
	k.storeEscrow(sdkCtx, job.JobID, &CrossChainEscrow{
		JobID:     job.JobID,
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	op := ChannelOperation{
		ChannelID:  "channel-0",
		Sequence:   9,
		PacketType: PacketTypeSubmitJob,
		JobID:      job.JobID,
	}
	k.trackPendingOperation(sdkCtx, op)

	balBefore := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount
	require.NoError(t, k.RefundOnChannelClose(sdkCtx, op))
	balAfter := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount

	updated := k.getJob(sdkCtx, job.JobID)
	require.NotNil(t, updated)
	require.Equal(t, "channel_closed", updated.Status)
	require.Equal(t, uint32(0), updated.Progress)
	require.True(t, balAfter.GT(balBefore))

	require.Empty(t, k.GetPendingOperations(sdkCtx, op.ChannelID))
}

func TestRefundOnChannelCloseUnknownType(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	op := ChannelOperation{
		ChannelID:  "channel-1",
		Sequence:   3,
		PacketType: "unknown",
	}
	require.NoError(t, k.RefundOnChannelClose(sdkCtx, op))
}

func TestRefundOnChannelClosePropagatesRefundError(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	jobID := "job-close-fail"
	job := &CrossChainComputeJob{JobID: jobID, Status: "submitted", Progress: 10}
	k.storeJob(sdkCtx, jobID, job)

	// Store escrow without funding module account to force refund failure
	k.storeEscrow(sdkCtx, jobID, &CrossChainEscrow{
		JobID:     jobID,
		Requester: sdk.AccAddress([]byte("req-close-fail")).String(),
		Provider:  sdk.AccAddress([]byte("prov-close-fail")).String(),
		Amount:    math.NewInt(500),
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	op := ChannelOperation{
		ChannelID:  "channel-2",
		Sequence:   4,
		PacketType: PacketTypeSubmitJob,
		JobID:      jobID,
	}
	k.trackPendingOperation(sdkCtx, op)

	err := k.RefundOnChannelClose(sdkCtx, op)
	require.Error(t, err)

	// Pending entry should still be cleared
	require.Empty(t, k.GetPendingOperations(sdkCtx, op.ChannelID))

	// Job status should have been marked channel_closed before refund failure
	updated := k.getJob(sdkCtx, jobID)
	require.NotNil(t, updated)
	require.Equal(t, "channel_closed", updated.Status)
}
