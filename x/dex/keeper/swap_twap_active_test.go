package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Verify a swap marks pool active and updates TWAP cumulatives together.
func TestSwapMarksActiveAndUpdatesTWAP(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	trader := types.TestAddr()
	pool, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	// Prefund trader input denom
	keepertest.FundAccount(t, k, ctx, trader, sdk.NewCoins(sdk.NewInt64Coin("upaw", 5_000_000)))

	// Set block time for TWAP
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	_, err = k.ExecuteSwap(ctx, trader, pool.Id, "upaw", "uusdc", math.NewInt(100_000), math.NewInt(1))
	require.NoError(t, err)

	// Active key exists
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	require.NotNil(t, store.Get(keeper.ActivePoolKey(pool.Id)))

	// TWAP record created/updated
	record, found, err := k.GetPoolTWAP(ctx, pool.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, ctx.BlockTime().Unix(), record.LastTimestamp)
}
