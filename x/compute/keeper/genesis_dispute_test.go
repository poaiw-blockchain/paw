package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Ensure dispute/appeal state survives export/import and indexes remain valid.
func TestGenesisExportImport_DisputeAppealState(t *testing.T) {
	k, sdkCtx := keepertest.ComputeKeeper(t)

	requester := createTestRequester(t)
	provider := createTestProvider(t)

	now := time.Now().UTC()

	genesis := types.GenesisState{
		Params:           types.DefaultParams(),
		GovernanceParams: types.DefaultGovernanceParams(),
		Providers: []types.Provider{
			{
				Address: provider.String(),
				Stake:   math.NewInt(1_000_000),
				Active:  true,
			},
		},
		Requests: []types.Request{
			{
				Id:             5,
				Requester:      requester.String(),
				Provider:       provider.String(),
				Status:         types.REQUEST_STATUS_PROCESSING,
				EscrowedAmount: math.NewInt(500_000),
				MaxPayment:     math.NewInt(600_000),
				CreatedAt:      now,
			},
		},
		Disputes: []types.Dispute{
			{
				Id:             7,
				RequestId:      5,
				Requester:      requester.String(),
				Provider:       provider.String(),
				Reason:         "provider_fault",
				Status:         types.DISPUTE_STATUS_VOTING,
				Deposit:        math.NewInt(100_000),
				CreatedAt:      now,
				EvidenceEndsAt: now.Add(time.Hour),
				VotingEndsAt:   now.Add(2 * time.Hour),
			},
		},
		SlashRecords: []types.SlashRecord{
			{
				Id:        3,
				Provider:  provider.String(),
				RequestId: 5,
				DisputeId: 7,
				Amount:    math.NewInt(50_000),
				Reason:    "provider_fault",
				SlashedAt: now,
			},
		},
		Appeals: []types.Appeal{
			{
				Id:            2,
				SlashId:       3,
				Provider:      provider.String(),
				Status:        types.APPEAL_STATUS_PENDING,
				Deposit:       math.NewInt(25_000),
				CreatedAt:     now,
				VotingEndsAt:  now.Add(time.Hour),
				Justification: "evidence contradicts slash",
			},
		},
		NextRequestId: 6,
		NextDisputeId: 8,
		NextSlashId:   4,
		NextAppealId:  3,
	}

	require.NoError(t, k.InitGenesis(sdk.WrapSDKContext(sdkCtx), genesis))

	// Invariants should hold after genesis init.
	msg, broken := keeper.DisputeIndexInvariant(*k)(sdkCtx)
	require.False(t, broken, msg)
	msg, broken = keeper.AppealIndexInvariant(*k)(sdkCtx)
	require.False(t, broken, msg)

	exported, err := k.ExportGenesis(sdk.WrapSDKContext(sdkCtx))
	require.NoError(t, err)

	require.Equal(t, genesis.NextRequestId, exported.NextRequestId)
	require.Equal(t, genesis.NextDisputeId, exported.NextDisputeId)
	require.Equal(t, genesis.NextSlashId, exported.NextSlashId)
	require.Equal(t, genesis.NextAppealId, exported.NextAppealId)
	require.Len(t, exported.Disputes, 1)
	require.Len(t, exported.SlashRecords, 1)
	require.Len(t, exported.Appeals, 1)

	// Re-import into a fresh keeper and ensure invariants still pass.
	k2, sdkCtx2 := keepertest.ComputeKeeper(t)
	require.NoError(t, k2.InitGenesis(sdk.WrapSDKContext(sdkCtx2), *exported))

	msg, broken = keeper.DisputeIndexInvariant(*k2)(sdkCtx2)
	require.False(t, broken, msg)
	msg, broken = keeper.AppealIndexInvariant(*k2)(sdkCtx2)
	require.False(t, broken, msg)

	exported2, err := k2.ExportGenesis(sdk.WrapSDKContext(sdkCtx2))
	require.NoError(t, err)
	require.Equal(t, exported.Disputes, exported2.Disputes)
	require.Equal(t, exported.SlashRecords, exported2.SlashRecords)
	require.Equal(t, exported.Appeals, exported2.Appeals)
	require.Equal(t, exported.NextDisputeId, exported2.NextDisputeId)
	require.Equal(t, exported.NextAppealId, exported2.NextAppealId)
}
