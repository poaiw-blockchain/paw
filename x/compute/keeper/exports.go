package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app/ibcutil"
	"github.com/paw-chain/paw/x/compute/types"
)

// This file contains test helper exports that need to be accessible from external test packages.
// Functions here are exported (capitalized) and can be used from both keeper_test and compute_test packages.
// Internal-only test helpers should remain in export_test.go.

// TrackPendingOperationForTest exposes trackPendingOperation for white-box tests.
// This function is accessible from external test packages (e.g., compute_test).
func TrackPendingOperationForTest(k *Keeper, ctx sdk.Context, op ChannelOperation) {
	k.trackPendingOperation(ctx, op)
}

// RecordNonceUsageForTesting exports recordNonceUsage for testing (accessible from external test packages).
func (k Keeper) RecordNonceUsageForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	k.recordNonceUsage(ctx, provider, nonce)
}

// CheckReplayAttackForTesting exports checkReplayAttack for testing (accessible from external test packages).
func (k Keeper) CheckReplayAttackForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	return k.checkReplayAttack(ctx, provider, nonce)
}

// AuthorizeComputeChannelForTest authorizes a channel for compute operations in tests.
// Accessible from external test packages.
func (k *Keeper) AuthorizeComputeChannelForTest(ctx sdk.Context, channelID string) {
	channels, _ := k.GetAuthorizedChannels(ctx)
	channels = append(channels, ibcutil.AuthorizedChannel{PortId: types.PortID, ChannelId: channelID})
	_ = k.SetAuthorizedChannels(ctx, channels)
}

// GetOrCreateCrossChainJobForTest gets or creates a job for testing.
// Accessible from external test packages.
func (k *Keeper) GetOrCreateCrossChainJobForTest(ctx sdk.Context, jobID string) *CrossChainComputeJob {
	job := k.GetCrossChainJob(ctx, jobID)
	if job == nil {
		return &CrossChainComputeJob{JobID: jobID}
	}
	return job
}

// GetAuthority returns the authority address for testing.
// Accessible from external test packages.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetDisputeForTesting exports getDispute for testing.
// Accessible from external test packages.
func (k Keeper) GetDisputeForTesting(ctx context.Context, id uint64) (*types.Dispute, error) {
	return k.getDispute(ctx, id)
}

// GetBankKeeper exports bankKeeper for testing.
// Accessible from external test packages.
func (k Keeper) GetBankKeeper() types.BankKeeper {
	return k.bankKeeper
}
