package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestProcessPendingDisputesMultiQueueEmitsEvents(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()

	d1 := types.Dispute{Id: 11, RequestId: 101, Status: types.DISPUTE_STATUS_EVIDENCE_SUBMISSION, EvidenceEndsAt: now.Add(-time.Minute)}
	d2 := types.Dispute{Id: 12, RequestId: 102, Status: types.DISPUTE_STATUS_VOTING, VotingEndsAt: now.Add(-time.Minute)}
	require.NoError(t, k.setDispute(sdkCtx, d1))
	require.NoError(t, k.setDispute(sdkCtx, d2))

	// ensure authority is valid to attempt auto-resolution
	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	events := sdkCtx.EventManager().Events()
	require.NotEmpty(t, events)
}
