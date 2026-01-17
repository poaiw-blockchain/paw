package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Ensure CancelSwapCommitment removes trader index entry.
func TestCancelSwapCommitment_RemovesTraderIndex(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	// Create pool and commit
	_, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(1, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, 1, commitHash))

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	traderKey := keeper.SwapCommitmentByTraderKey(trader, commitHash)
	require.NotNil(t, store.Get(traderKey))

	require.NoError(t, k.CancelSwapCommitment(ctx, trader, commitHash))

	require.Nil(t, store.Get(traderKey))
	require.Nil(t, store.Get(keeper.SwapCommitmentKey(commitHash)))
}
