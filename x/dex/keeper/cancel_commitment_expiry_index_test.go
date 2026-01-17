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

// Ensure CancelSwapCommitment removes expiry index as well as the main record.
func TestCancelSwapCommitment_RemovesExpiryIndex(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	_, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash := keeper.ComputeSwapCommitmentHash(1, "upaw", "uusdc", math.NewInt(1_000), math.NewInt(1), []byte("salt_expiry"), trader)
	require.NoError(t, k.CommitSwap(ctx, trader, 1, commitHash))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.GetStoreKey())

	// Load stored commitment to get expiry block
	bz := store.Get(keeper.SwapCommitmentKey(commitHash))
	require.NotNil(t, bz)

	var commitment keeper.SwapCommitment
	require.NoError(t, json.Unmarshal(bz, &commitment))

	expiryKey := keeper.SwapCommitmentByExpiryKey(commitment.ExpiryBlock, commitHash)
	require.NotNil(t, store.Get(expiryKey), "expiry index should exist before cancel")

	require.NoError(t, k.CancelSwapCommitment(ctx, trader, commitHash))

	require.Nil(t, store.Get(expiryKey), "expiry index should be removed after cancel")
	require.Nil(t, store.Get(keeper.SwapCommitmentKey(commitHash)), "commitment should be removed after cancel")
}
