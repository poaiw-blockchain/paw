package keeper_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestTraderCommitmentsIndexing(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(10)
	ctx = sdkCtx
	trader := sdk.AccAddress("trader1____________")

	commitHash := keeper.ComputeSwapCommitmentHash(1, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)

	// Manually set commitment and index
	commitment := keeper.SwapCommitment{
		CommitmentHash: commitHash,
		Trader:         trader.String(),
		PoolID:         1,
		CommitBlock:    sdkCtx.BlockHeight(),
		ExpiryBlock:    sdkCtx.BlockHeight() + 10,
		DepositAmount:  math.NewInt(100),
		DepositDenom:   "upaw",
	}
	bz, _ := json.Marshal(commitment)
	store := sdkCtx.KVStore(k.GetStoreKey())
	store.Set(keeper.SwapCommitmentKey(commitHash), bz)
	store.Set(keeper.SwapCommitmentByTraderKey(trader, commitHash), commitHash)

	commitments, err := k.GetTraderCommitments(ctx, trader)
	require.NoError(t, err)
	require.Len(t, commitments, 1)
	require.Equal(t, commitment.CommitmentHash, commitments[0].CommitmentHash)
}

func TestDuplicateCommitSwapRejected(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	commitHash := []byte("dup_hash")
	require.NoError(t, k.CommitSwap(ctx, trader, 1, commitHash))

	err := k.CommitSwap(ctx, trader, 1, commitHash)
	require.ErrorIs(t, err, types.ErrDuplicateCommitment)
}
