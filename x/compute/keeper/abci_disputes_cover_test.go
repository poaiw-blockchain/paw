package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestProcessPendingDisputesTransitionsStatuses(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()

	// Evidence submission -> voting transition
	disputeEvidence := types.Dispute{
		Id:             1,
		RequestId:      10,
		Status:         types.DISPUTE_STATUS_EVIDENCE_SUBMISSION,
		EvidenceEndsAt: now.Add(-time.Hour),
	}
	require.NoError(t, k.setDispute(sdkCtx, disputeEvidence))

	// Voting expired path (ResolveDispute may fail but should not panic)
	disputeVoting := types.Dispute{
		Id:           2,
		RequestId:    20,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: now.Add(-time.Hour),
	}
	require.NoError(t, k.setDispute(sdkCtx, disputeVoting))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	d1, err := k.getDispute(sdkCtx, 1)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_VOTING, d1.Status)
}

func TestProcessPendingDisputesAutoResolve(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Lower consensus threshold to zero to allow resolution with one vote
	gov, err := k.GetGovernanceParams(sdkCtx)
	require.NoError(t, err)
	gov.ConsensusThreshold = math.LegacyZeroDec()
	require.NoError(t, k.SetGovernanceParams(sdkCtx, gov))

	voter := sdk.AccAddress([]byte("voter_address"))
	dispute := types.Dispute{
		Id:           3,
		RequestId:    30,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: sdkCtx.BlockTime().Add(-time.Minute),
		Votes: []types.DisputeVote{
			{
				Validator:   voter.String(),
				Option:      types.DISPUTE_VOTE_PROVIDER_FAULT,
				VotingPower: math.NewInt(1),
			},
		},
	}
	require.NoError(t, k.setDispute(sdkCtx, dispute))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	resolved, err := k.getDispute(sdkCtx, dispute.Id)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_RESOLVED, resolved.Status)
}

func TestProcessPendingDisputesSkipsInvalidAuthority(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Corrupt authority so resolution is skipped
	k.authority = "invalid"

	dispute := types.Dispute{
		Id:           4,
		RequestId:    40,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: sdkCtx.BlockTime().Add(-time.Minute),
	}
	require.NoError(t, k.setDispute(sdkCtx, dispute))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	stored, err := k.getDispute(sdkCtx, dispute.Id)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_VOTING, stored.Status)
}
