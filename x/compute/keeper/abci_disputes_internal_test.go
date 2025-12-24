package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// =============================================================================
// ABCI Disputes Internal Tests
// These tests require access to internal keeper functions and must use package keeper
// =============================================================================

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

func TestCalculateDisputeScorePenaltyAndDefaults(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("nil stats returns perfect score", func(t *testing.T) {
		require.Equal(t, uint32(100), k.calculateDisputeScore(ctx, "missing"))
	})

	t.Run("loss rate and volume penalties applied", func(t *testing.T) {
		provider := "prov-disputes"
		stats := &ProviderStats{
			TotalDisputes: 20,
			DisputesLost:  5, // 25% loss -> base 75
		}
		require.NoError(t, k.SetProviderStats(ctx, provider, stats))

		score := k.calculateDisputeScore(ctx, provider)
		// Base 75 minus volume penalty ((20-10)*2 = 20) = 55
		require.Equal(t, uint32(55), score)
	})
}

func TestCalculateUptimeScoreBoundaries(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	now := time.Now()

	prov := types.Provider{
		Address:      sdk.AccAddress([]byte("addr")).String(),
		RegisteredAt: now.Add(-2 * time.Hour),
		LastActiveAt: now.Add(-30 * time.Minute),
	}
	require.Equal(t, uint32(90), k.calculateUptimeScore(ctx, prov, now))

	prov.LastActiveAt = now.Add(-9 * time.Hour)
	require.Equal(t, uint32(30), k.calculateUptimeScore(ctx, prov, now))

	prov.RegisteredAt = now.Add(time.Minute)
	require.Equal(t, uint32(100), k.calculateUptimeScore(ctx, prov, now))
}
