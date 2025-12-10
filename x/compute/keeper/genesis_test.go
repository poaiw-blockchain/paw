package keeper_test

import (
	"bytes"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestComputeGenesisRoundTrip(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	params := types.DefaultParams()
	params.MinProviderStake = sdkmath.NewInt(5_000_000_000)
	params.AuthorizedChannels = []types.AuthorizedChannel{
		{PortId: types.PortID, ChannelId: "channel-9"},
	}

	govParams := types.DefaultGovernanceParams()
	govParams.DisputeDeposit = sdkmath.NewInt(2_500_000)
	govParams.SlashPercentage = sdkmath.LegacyMustNewDecFromStr("0.2")

	providerAddr := sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20)).String()
	requesterAddr := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20)).String()
	baseTime := time.Unix(1_700_000_000, 0).UTC()

	provider := types.Provider{
		Address:    providerAddr,
		Moniker:    "provider-one",
		Endpoint:   "https://provider-one",
		Stake:      sdkmath.NewInt(10_000_000_000),
		Reputation: 95,
		AvailableSpecs: types.ComputeSpec{
			CpuCores:       2000,
			MemoryMb:       16_384,
			GpuCount:       1,
			GpuType:        "A100",
			StorageGb:      500,
			TimeoutSeconds: 900,
		},
		Pricing: types.Pricing{
			CpuPricePerMcoreHour:  sdkmath.LegacyMustNewDecFromStr("0.0001"),
			MemoryPricePerMbHour:  sdkmath.LegacyMustNewDecFromStr("0.00005"),
			GpuPricePerHour:       sdkmath.LegacyMustNewDecFromStr("0.5"),
			StoragePricePerGbHour: sdkmath.LegacyMustNewDecFromStr("0.00001"),
		},
		TotalRequestsCompleted: 12,
		TotalRequestsFailed:    1,
		Active:                 true,
		RegisteredAt:           baseTime,
		LastActiveAt:           baseTime.Add(2 * time.Minute),
	}

	assignedAt := baseTime.Add(3 * time.Minute)
	completedAt := baseTime.Add(10 * time.Minute)
	request := types.Request{
		Id:             1,
		Requester:      requesterAddr,
		Provider:       providerAddr,
		Specs:          provider.AvailableSpecs,
		ContainerImage: "docker.io/paw/test:latest",
		Command:        []string{"run", "--fast"},
		EnvVars:        map[string]string{"ENV": "testnet"},
		Status:         types.REQUEST_STATUS_COMPLETED,
		MaxPayment:     sdkmath.NewInt(1_000_000),
		EscrowedAmount: sdkmath.NewInt(750_000),
		CreatedAt:      baseTime,
		AssignedAt:     &assignedAt,
		CompletedAt:    &completedAt,
		ResultHash:     "hash123",
		ResultUrl:      "ipfs://result",
		ErrorMessage:   "",
	}

	result := types.Result{
		RequestId:         request.Id,
		Provider:          providerAddr,
		OutputHash:        "hash123",
		OutputUrl:         "ipfs://result",
		ExitCode:          0,
		LogsUrl:           "ipfs://logs",
		VerificationProof: []byte{0x01, 0x02},
		SubmittedAt:       baseTime.Add(11 * time.Minute),
		Verified:          true,
		VerificationScore: 98,
	}

	disputeResolvedAt := baseTime.Add(24 * time.Hour)
	dispute := types.Dispute{
		Id:             1,
		RequestId:      request.Id,
		Requester:      requesterAddr,
		Provider:       providerAddr,
		Reason:         "incorrect output",
		Status:         types.DISPUTE_STATUS_RESOLVED,
		Deposit:        sdkmath.NewInt(2_500_000),
		CreatedAt:      baseTime.Add(12 * time.Minute),
		EvidenceEndsAt: baseTime.Add(13 * time.Minute),
		VotingEndsAt:   baseTime.Add(14 * time.Minute),
		Resolution:     types.DISPUTE_RESOLUTION_PARTIAL_REFUND,
		ResolvedAt:     &disputeResolvedAt,
	}

	slash := types.SlashRecord{
		Id:        1,
		Provider:  providerAddr,
		RequestId: request.Id,
		DisputeId: dispute.Id,
		Amount:    sdkmath.NewInt(100_000),
		Reason:    "failed verification",
		SlashedAt: baseTime.Add(15 * time.Minute),
		Appealed:  true,
		AppealId:  1,
	}

	appealResolvedAt := baseTime.Add(48 * time.Hour)
	appeal := types.Appeal{
		Id:            1,
		SlashId:       slash.Id,
		Provider:      providerAddr,
		Justification: "hardware fault resolved",
		Status:        types.APPEAL_STATUS_RESOLVED,
		Deposit:       sdkmath.NewInt(500_000),
		CreatedAt:     baseTime.Add(16 * time.Minute),
		VotingEndsAt:  baseTime.Add(17 * time.Minute),
		Approved:      true,
		ResolvedAt:    &appealResolvedAt,
	}

	// Create escrow states - one LOCKED, one RELEASED
	lockedEscrow := types.EscrowState{
		RequestId:       2,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          sdkmath.NewInt(500_000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        baseTime.Add(20 * time.Minute),
		ExpiresAt:       baseTime.Add(24 * time.Hour),
		ReleasedAt:      nil,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 0,
		Nonce:           1,
	}

	releasedAt := baseTime.Add(30 * time.Minute)
	releasedEscrow := types.EscrowState{
		RequestId:       request.Id,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          request.EscrowedAmount,
		Status:          types.ESCROW_STATUS_RELEASED,
		LockedAt:        baseTime.Add(3 * time.Minute),
		ExpiresAt:       baseTime.Add(12 * time.Hour),
		ReleasedAt:      &releasedAt,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 1,
		Nonce:           2,
	}

	genesis := types.GenesisState{
		Params:           params,
		GovernanceParams: govParams,
		Providers:        []types.Provider{provider},
		Requests:         []types.Request{request},
		Results:          []types.Result{result},
		Disputes:         []types.Dispute{dispute},
		SlashRecords:     []types.SlashRecord{slash},
		Appeals:          []types.Appeal{appeal},
		EscrowStates:     []types.EscrowState{lockedEscrow, releasedEscrow},
		NextRequestId:    2,
		NextDisputeId:    3,
		NextSlashId:      4,
		NextAppealId:     5,
		NextEscrowNonce:  3,
	}

	require.NoError(t, k.InitGenesis(ctx, genesis))

	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.Equal(t, genesis.Params, exported.Params)
	require.Equal(t, genesis.GovernanceParams, exported.GovernanceParams)
	require.Equal(t, genesis.Providers, exported.Providers)
	require.Equal(t, genesis.Requests, exported.Requests)
	require.Equal(t, genesis.Results, exported.Results)
	require.Equal(t, genesis.Disputes, exported.Disputes)
	require.Equal(t, genesis.SlashRecords, exported.SlashRecords)
	require.Equal(t, genesis.Appeals, exported.Appeals)
	// EscrowStates may be in different order due to KV store iteration
	require.ElementsMatch(t, genesis.EscrowStates, exported.EscrowStates)
	require.Equal(t, genesis.NextRequestId, exported.NextRequestId)
	require.Equal(t, genesis.NextDisputeId, exported.NextDisputeId)
	require.Equal(t, genesis.NextSlashId, exported.NextSlashId)
	require.Equal(t, genesis.NextAppealId, exported.NextAppealId)
	require.Equal(t, genesis.NextEscrowNonce, exported.NextEscrowNonce)

	// Verify timeout index was restored for LOCKED escrow
	// We can verify this by checking the escrow can be retrieved
	retrievedLockedEscrow, err := k.GetEscrowState(ctx, lockedEscrow.RequestId)
	require.NoError(t, err)
	require.Equal(t, lockedEscrow.Status, retrievedLockedEscrow.Status)
	require.Equal(t, lockedEscrow.Amount, retrievedLockedEscrow.Amount)

	// Verify released escrow
	retrievedReleasedEscrow, err := k.GetEscrowState(ctx, releasedEscrow.RequestId)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_RELEASED, retrievedReleasedEscrow.Status)
}

// TestEscrowGenesisTimeoutIndexRestoration verifies that timeout indexes are properly restored
func TestEscrowGenesisTimeoutIndexRestoration(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	baseTime := time.Unix(1_700_000_000, 0).UTC()
	providerAddr := sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20)).String()
	requesterAddr := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20)).String()

	// Create multiple escrow states with different statuses
	lockedEscrow1 := types.EscrowState{
		RequestId:       1,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          sdkmath.NewInt(500_000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        baseTime,
		ExpiresAt:       baseTime.Add(1 * time.Hour),
		ReleasedAt:      nil,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 0,
		Nonce:           1,
	}

	challengedEscrow := types.EscrowState{
		RequestId:       2,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          sdkmath.NewInt(600_000),
		Status:          types.ESCROW_STATUS_CHALLENGED,
		LockedAt:        baseTime,
		ExpiresAt:       baseTime.Add(2 * time.Hour),
		ReleasedAt:      nil,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 1,
		Nonce:           2,
	}

	releasedAt := baseTime.Add(30 * time.Minute)
	releasedEscrow := types.EscrowState{
		RequestId:       3,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          sdkmath.NewInt(700_000),
		Status:          types.ESCROW_STATUS_RELEASED,
		LockedAt:        baseTime,
		ExpiresAt:       baseTime.Add(3 * time.Hour),
		ReleasedAt:      &releasedAt,
		RefundedAt:      nil,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 1,
		Nonce:           3,
	}

	refundedAt := baseTime.Add(45 * time.Minute)
	refundedEscrow := types.EscrowState{
		RequestId:       4,
		Requester:       requesterAddr,
		Provider:        providerAddr,
		Amount:          sdkmath.NewInt(800_000),
		Status:          types.ESCROW_STATUS_REFUNDED,
		LockedAt:        baseTime,
		ExpiresAt:       baseTime.Add(4 * time.Hour),
		ReleasedAt:      nil,
		RefundedAt:      &refundedAt,
		DisputeId:       0,
		ChallengeEndsAt: nil,
		ReleaseAttempts: 0,
		Nonce:           4,
	}

	genesis := types.GenesisState{
		Params:           types.DefaultParams(),
		GovernanceParams: types.DefaultGovernanceParams(),
		Providers:        []types.Provider{},
		Requests:         []types.Request{},
		Results:          []types.Result{},
		Disputes:         []types.Dispute{},
		SlashRecords:     []types.SlashRecord{},
		Appeals:          []types.Appeal{},
		EscrowStates: []types.EscrowState{
			lockedEscrow1,
			challengedEscrow,
			releasedEscrow,
			refundedEscrow,
		},
		NextRequestId:   1,
		NextDisputeId:   1,
		NextSlashId:     1,
		NextAppealId:    1,
		NextEscrowNonce: 5,
	}

	require.NoError(t, k.InitGenesis(ctx, genesis))

	// Verify all escrow states were imported
	retrieved1, err := k.GetEscrowState(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, retrieved1.Status)

	retrieved2, err := k.GetEscrowState(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_CHALLENGED, retrieved2.Status)

	retrieved3, err := k.GetEscrowState(ctx, 3)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_RELEASED, retrieved3.Status)

	retrieved4, err := k.GetEscrowState(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_REFUNDED, retrieved4.Status)

	// Verify timeout indexes were created for LOCKED and CHALLENGED escrows only
	// We test this by iterating over timeouts and checking we get the right ones
	var foundRequestIDs []uint64
	err = k.IterateEscrowTimeouts(ctx, baseTime.Add(5*time.Hour), func(requestID uint64, expiresAt time.Time) (bool, error) {
		foundRequestIDs = append(foundRequestIDs, requestID)
		return false, nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint64{1, 2}, foundRequestIDs, "Only LOCKED and CHALLENGED escrows should have timeout indexes")

	// Verify nonce counter was set correctly
	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(5), exported.NextEscrowNonce)
}
