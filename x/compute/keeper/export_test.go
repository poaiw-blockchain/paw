package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/compute/types"
)

// RecordNonceUsageForTesting exports recordNonceUsage for testing (accessible from external test packages)
func (k Keeper) RecordNonceUsageForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	k.recordNonceUsage(ctx, provider, nonce)
}

// CheckReplayAttackForTesting exports checkReplayAttack for testing (accessible from external test packages)
func (k Keeper) CheckReplayAttackForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	return k.checkReplayAttack(ctx, provider, nonce)
}

// GetAuthority returns the authority address for testing
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetDisputeForTesting exports getDispute for testing
func (k Keeper) GetDisputeForTesting(ctx context.Context, id uint64) (*types.Dispute, error) {
	return k.getDispute(ctx, id)
}
