package keeper

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestCleanupOldPacketNoncesPrunesAndEmitsEvent(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	store := sdkCtx.KVStore(k.storeKey)
	keyOld := []byte("pending_packet_channel-1_1")
	keyNew := []byte("pending_packet_channel-1_2")

	valOld := make([]byte, 8)
	binary.BigEndian.PutUint64(valOld, uint64(sdkCtx.BlockHeight()-10))
	valNew := make([]byte, 8)
	binary.BigEndian.PutUint64(valNew, uint64(sdkCtx.BlockHeight()))

	store.Set(keyOld, valOld)
	store.Set(keyNew, valNew)

	require.NoError(t, k.CleanupOldPacketNonces(sdkCtx, 5))

	require.Nil(t, store.Get(keyOld))
	require.NotNil(t, store.Get(keyNew))
	events := sdkCtx.EventManager().Events()
	require.NotEmpty(t, events)
}

func TestRefundEscrowOnTimeoutMissingEscrow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	err := k.RefundEscrowOnTimeout(sdkCtx, "unknown-job", "timeout")
	require.Error(t, err)
	require.Contains(t, err.Error(), "escrow not found")
}

func TestRefundEscrowOnTimeoutSucceeds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_refund_req"))
	amount := sdk.NewInt64Coin("upaw", 200)

	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))

	k.storeEscrow(sdkCtx, "job-refund", &CrossChainEscrow{
		JobID:     "job-refund",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	balBefore := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount
	require.NoError(t, k.RefundEscrowOnTimeout(sdkCtx, "job-refund", "timeout"))
	balAfter := k.bankKeeper.GetBalance(sdkCtx, requester, "upaw").Amount

	require.True(t, balAfter.GTE(balBefore))
}

func TestRefundEscrowOnTimeoutFailsWithInsufficientModuleFunds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_refund_req2"))
	amount := sdk.NewInt64Coin("upaw", 500)

	// Do NOT mint or lock escrow funds, leaving module account empty
	k.storeEscrow(sdkCtx, "job-refund-fail", &CrossChainEscrow{
		JobID:     "job-refund-fail",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	err := k.RefundEscrowOnTimeout(sdkCtx, "job-refund-fail", "timeout")
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient")
}

func TestGetJobStatusMissingAndExisting(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	status, err := k.GetJobStatus(sdkCtx, "missing-job")
	require.Error(t, err)
	require.Nil(t, status)

	jobStatus := JobStatus{
		JobID:        "job-status-1",
		Status:       "completed",
		Requester:    "req1",
		Provider:     "prov1",
		Progress:     90,
		UpdatedAt:    sdkCtx.BlockTime().Unix(),
		ErrorMessage: "",
	}
	require.NoError(t, k.storeJobStatus(sdkCtx, jobStatus))

	found, err := k.GetJobStatus(sdkCtx, jobStatus.JobID)
	require.NoError(t, err)
	require.Equal(t, jobStatus.JobID, found.JobID)
	require.Equal(t, jobStatus.Status, found.Status)
	require.Equal(t, jobStatus.Progress, found.Progress)
}
