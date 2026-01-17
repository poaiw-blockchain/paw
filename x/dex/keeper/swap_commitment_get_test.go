package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Ensures GetSwapCommitment returns stored commitment and errors when missing.
func TestGetSwapCommitment(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	pool, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(pool.Id, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, pool.Id, commitHash))

	commitment, err := k.GetSwapCommitment(ctx, commitHash)
	require.NoError(t, err)
	require.Equal(t, trader.String(), commitment.Trader)
	require.Equal(t, pool.Id, commitment.PoolID)

	_, err = k.GetSwapCommitment(ctx, []byte("missing"))
	require.Error(t, err)
}
