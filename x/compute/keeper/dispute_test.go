package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keeperpkg "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestDisputeLifecycle_SlashProviderResolution(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
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

	initialDeposit := govParams.DisputeDeposit
	disputeID, err := k.CreateDispute(ctx, requester, requestID, "provider_fault", []byte("initial evidence"), initialDeposit)
	require.NoError(t, err)

	queryServer := keeperpkg.NewQueryServerImpl(*k)
	disputeResp, err := queryServer.Dispute(ctx, &types.QueryDisputeRequest{DisputeId: disputeID})
	require.NoError(t, err)
	dispute := disputeResp.Dispute
	require.Equal(t, types.DISPUTE_STATUS_EVIDENCE_SUBMISSION, dispute.Status)
	require.Equal(t, initialDeposit, dispute.Deposit)

	// Submit extra evidence and ensure pagination returns it.
	err = k.SubmitEvidence(ctx, requester, disputeID, "log", []byte("more data"), "latency spike")
	require.NoError(t, err)
	evidence, _, err := k.ListEvidence(ctx, disputeID, nil)
	require.NoError(t, err)
	require.Len(t, evidence, 2)

	val1 := sdk.ValAddress([]byte("validator_address_1"))
	val2 := sdk.ValAddress([]byte("validator_address_2"))

	require.NoError(t, k.VoteOnDispute(ctx, val1, disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "bad proof"))
	require.NoError(t, k.VoteOnDispute(ctx, val2, disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "timeout breach"))

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	resolution, err := k.ResolveDispute(ctx, authority, disputeID)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_RESOLUTION_SLASH_PROVIDER, resolution)

	// Apply settlement and ensure slashing occurred.
	storedProvider, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	stakeBefore := storedProvider.Stake

	require.NoError(t, k.SettleDisputeOutcome(ctx, disputeID, resolution))

	slashedProvider, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	require.True(t, slashedProvider.Stake.LT(stakeBefore))

	resp, err := queryServer.SlashRecordsByProvider(ctx, &types.QuerySlashRecordsByProviderRequest{
		Provider: provider.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.SlashRecords, 1)
	require.Equal(t, disputeID, resp.SlashRecords[0].DisputeId)
}

func TestAppealLifecycle_RefundsSlashOnApproval(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
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

	// Cast votes so provider is slashed.
	val1 := sdk.ValAddress([]byte("validator_address_1"))
	val2 := sdk.ValAddress([]byte("validator_address_2"))
	require.NoError(t, k.VoteOnDispute(ctx, val1, disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "bad proof"))
	require.NoError(t, k.VoteOnDispute(ctx, val2, disputeID, types.DISPUTE_VOTE_PROVIDER_FAULT, "timeout breach"))

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	resolution, err := k.ResolveDispute(ctx, authority, disputeID)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_RESOLUTION_SLASH_PROVIDER, resolution)
	require.NoError(t, k.SettleDisputeOutcome(ctx, disputeID, resolution))

	queryServer := keeperpkg.NewQueryServerImpl(*k)
	resp, err := queryServer.SlashRecordsByProvider(ctx, &types.QuerySlashRecordsByProviderRequest{
		Provider: provider.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.SlashRecords, 1)
	slashRecord := resp.SlashRecords[0]
	deposit := govParams.AppealDepositPercentage.MulInt(slashRecord.Amount).TruncateInt()
	if deposit.LT(govParams.DisputeDeposit) {
		deposit = govParams.DisputeDeposit
	}

	appealID, err := k.CreateAppeal(ctx, provider, slashRecord.Id, "incorrect slash", deposit)
	require.NoError(t, err)

	require.NoError(t, k.VoteOnAppeal(ctx, val1, appealID, true, "evidence holds"))
	require.NoError(t, k.VoteOnAppeal(ctx, val2, appealID, true, "agreed"))

	approved, err := k.ResolveAppeal(ctx, authority, appealID)
	require.NoError(t, err)
	require.True(t, approved)

	require.NoError(t, k.ApplyAppealOutcome(ctx, appealID, approved))

	recordResp, err := queryServer.SlashRecord(ctx, &types.QuerySlashRecordRequest{SlashId: slashRecord.Id})
	require.NoError(t, err)
	updatedRecord := recordResp.SlashRecord
	require.True(t, updatedRecord.Appealed)
	require.Equal(t, appealID, updatedRecord.AppealId)
}
