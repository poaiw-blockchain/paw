package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Governance-style commit cleanup (no module account dependencies).
func TestCleanupExpiredSwapCommits(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(120)
	ctx = sdkCtx

	expired := types.SwapCommit{
		SwapHash:     "hash_expired",
		Trader:       "paw1expired",
		CommitHeight: 10,
		ExpiryHeight: 110,
	}
	active := types.SwapCommit{
		SwapHash:     "hash_active",
		Trader:       "paw1active",
		CommitHeight: 115,
		ExpiryHeight: 200,
	}

	require.NoError(t, k.SetSwapCommit(ctx, expired))
	require.NoError(t, k.SetSwapCommit(ctx, active))

	require.NoError(t, k.CleanupExpiredSwapCommits(ctx))

	store := sdkCtx.KVStore(k.GetStoreKey())
	require.Nil(t, store.Get(keeper.SwapCommitKey(expired.SwapHash)))
	require.NotNil(t, store.Get(keeper.SwapCommitKey(active.SwapHash)))
}
