package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RecordNonceUsageForTesting exports recordNonceUsage for testing (accessible from external test packages)
func (k Keeper) RecordNonceUsageForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	k.recordNonceUsage(ctx, provider, nonce)
}

// CheckReplayAttackForTesting exports checkReplayAttack for testing (accessible from external test packages)
func (k Keeper) CheckReplayAttackForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	return k.checkReplayAttack(ctx, provider, nonce)
}
