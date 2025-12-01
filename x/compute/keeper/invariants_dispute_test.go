package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestDisputeAndAppealInvariants(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)

	// Create request + dispute
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, sdkCtx)
	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(5_000_000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	govParams, err := k.GetGovernanceParams(ctx)
	require.NoError(t, err)
	disputeID, err := k.CreateDispute(ctx, requester, requestID, "provider_fault", []byte("initial evidence"), govParams.DisputeDeposit)
	require.NoError(t, err)

	// Cast votes to trigger slash + appeal path
	val := []byte("validator_address_1")
	require.NoError(t, k.VoteOnDispute(ctx, val, disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "insufficient proof"))
	require.NoError(t, k.VoteOnDispute(ctx, []byte("validator_address_2"), disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "timeout breach"))

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	resolution, err := k.ResolveDispute(ctx, authority, disputeID)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_RESOLUTION_SLASH_PROVIDER, resolution)
	require.NoError(t, k.SettleDisputeOutcome(ctx, disputeID, resolution))

	// Create an appeal for the slash record to exercise appeal index
	qs := keeper.NewQueryServerImpl(*k)
	slashResp, err := qs.SlashRecords(ctx, &types.QuerySlashRecordsRequest{})
	require.NoError(t, err)
	require.Len(t, slashResp.SlashRecords, 1)
	slashRecord := slashResp.SlashRecords[0]

	appealDeposit := govParams.AppealDepositPercentage.MulInt(slashRecord.Amount).TruncateInt()
	if appealDeposit.LT(govParams.DisputeDeposit) {
		appealDeposit = govParams.DisputeDeposit
	}
	appealID, err := k.CreateAppeal(ctx, provider, slashRecord.Id, "slash incorrect", appealDeposit)
	require.NoError(t, err)
	require.NoError(t, k.VoteOnAppeal(ctx, val, appealID, true, "agree to refund"))
	require.NoError(t, k.VoteOnAppeal(ctx, []byte("validator_address_2"), appealID, true, "agree"))
	approved, err := k.ResolveAppeal(ctx, authority, appealID)
	require.NoError(t, err)
	require.True(t, approved)
	require.NoError(t, k.ApplyAppealOutcome(ctx, appealID, approved))

	// Run invariants
	msg, broken := keeper.DisputeIndexInvariant(*k)(sdkCtx)
	require.False(t, broken, msg)

	msg, broken = keeper.AppealIndexInvariant(*k)(sdkCtx)
	require.False(t, broken, msg)
}
