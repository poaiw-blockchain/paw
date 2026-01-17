package keeper_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Helper to seed a cached remote pool entry that QueryCrossChainPools can read.
func seedCachedPool(t *testing.T, k *keeper.Keeper, ctx sdk.Context, pool keeper.CrossChainPoolInfo) {
	t.Helper()
	store := ctx.KVStore(k.GetStoreKey())
	bz, err := json.Marshal(pool)
	require.NoError(t, err)
	key := []byte(
		// Matches format in handleQueryPoolsAck/cache writer
		// cached_pool_<chain>_<pool>_<tokenA>_<tokenB>
		"cached_pool_" + pool.ChainID + "_" + pool.PoolID + "_" + pool.TokenA + "_" + pool.TokenB,
	)
	store.Set(key, bz)
}

// Helper to seed a local pool used by getLocalPools/executeLocalSwap.
func seedLocalPool(t *testing.T, k *keeper.Keeper, ctx sdk.Context, poolID, tokenA, tokenB string, reserveA, reserveB math.Int, fee math.LegacyDec) {
	t.Helper()
	store := ctx.KVStore(k.GetStoreKey())
	payload := struct {
		PoolID   string         `json:"pool_id"`
		TokenA   string         `json:"token_a"`
		TokenB   string         `json:"token_b"`
		ReserveA math.Int       `json:"reserve_a"`
		ReserveB math.Int       `json:"reserve_b"`
		SwapFee  math.LegacyDec `json:"swap_fee"`
	}{
		PoolID:   poolID,
		TokenA:   tokenA,
		TokenB:   tokenB,
		ReserveA: reserveA,
		ReserveB: reserveB,
		SwapFee:  fee,
	}

	bz, err := json.Marshal(payload)
	require.NoError(t, err)
	store.Set([]byte("pool_"+poolID), bz)
}

// fundPoolAddress ensures the synthetic pool account has the reserves needed for swaps.
func fundPoolAddress(t *testing.T, bk bankkeeper.Keeper, ctx sdk.Context, poolID, tokenA, tokenB string, reserveA, reserveB math.Int) {
	t.Helper()
	poolAddr := sdk.AccAddress([]byte("pool_" + poolID))
	coins := sdk.NewCoins(sdk.NewCoin(tokenA, reserveA), sdk.NewCoin(tokenB, reserveB))
	require.NoError(t, bk.MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, poolAddr, coins))
}

func TestQueryCrossChainPools_UsesFreshCache(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now())

	fresh := keeper.CrossChainPoolInfo{
		ChainID:     "osmosis-1",
		PoolID:      "1",
		TokenA:      "upaw",
		TokenB:      "uosmo",
		ReserveA:    math.NewInt(1_000_000),
		ReserveB:    math.NewInt(2_000_000),
		SwapFee:     math.LegacyMustNewDecFromStr("0.003"),
		LastUpdated: ctx.BlockTime(),
	}
	stale := fresh
	stale.PoolID = "2"
	stale.LastUpdated = ctx.BlockTime().Add(-6 * time.Minute) // older than 5m threshold

	seedCachedPool(t, k, ctx, fresh)
	seedCachedPool(t, k, ctx, stale)

	pools, err := k.QueryCrossChainPools(ctx, "upaw", "uosmo", nil)
	require.NoError(t, err)
	require.Len(t, pools, 1, "stale pools should be filtered out")
	require.Equal(t, fresh.PoolID, pools[0].PoolID)
}

func TestQueryCrossChainPools_ContinuesOnMissingConnection(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now())

	cached := keeper.CrossChainPoolInfo{
		ChainID:     "osmosis-1",
		PoolID:      "1",
		TokenA:      "upaw",
		TokenB:      "uosmo",
		ReserveA:    math.NewInt(500_000),
		ReserveB:    math.NewInt(800_000),
		SwapFee:     math.LegacyMustNewDecFromStr("0.0025"),
		LastUpdated: ctx.BlockTime(),
	}
	seedCachedPool(t, k, ctx, cached)

	// Unknown chain triggers getIBCConnection error, but function should still return cached pools without panicking.
	pools, err := k.QueryCrossChainPools(ctx, "upaw", "uosmo", []string{"unknown-chain"})
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, cached.PoolID, pools[0].PoolID)
}

func TestQueryCrossChainPools_CacheBoundaryFiveMinutes(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	now := time.Now()
	ctx = ctx.WithBlockTime(now)

	boundary := keeper.CrossChainPoolInfo{
		ChainID:     "osmosis-1",
		PoolID:      "3",
		TokenA:      "upaw",
		TokenB:      "uosmo",
		ReserveA:    math.NewInt(100),
		ReserveB:    math.NewInt(200),
		SwapFee:     math.LegacyMustNewDecFromStr("0.003"),
		LastUpdated: now.Add(-5 * time.Minute), // equality should be treated as stale
	}
	seedCachedPool(t, k, ctx, boundary)

	pools, err := k.QueryCrossChainPools(ctx, "upaw", "uosmo", nil)
	require.NoError(t, err)
	require.Len(t, pools, 0, "cache at exactly 5m should be considered stale")
}

func TestFindBestCrossChainRoute_PrefersLocalOnRemoteFailure(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithChainID("paw-mvp-1")

	reserveA := math.NewInt(1_000_000)
	reserveB := math.NewInt(900_000)
	seedLocalPool(t, k, ctx, "local-1", "upaw", "uosmo", reserveA, reserveB, math.LegacyMustNewDecFromStr("0.003"))

	route, err := k.FindBestCrossChainRoute(ctx, "upaw", "uosmo", math.NewInt(100_000), []string{"unknown-chain"})
	require.NoError(t, err)
	require.NotNil(t, route)
	require.Len(t, route.Steps, 1)
	require.Equal(t, "paw-mvp-1", route.Steps[0].ChainID)
	require.True(t, route.Steps[0].MinAmountOut.GT(math.ZeroInt()))
}

func TestExecuteCrossChainSwap_EmptyRoute(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now())

	_, err := k.ExecuteCrossChainSwap(ctx, types.TestAddr(), keeper.CrossChainSwapRoute{}, math.LegacyMustNewDecFromStr("0.05"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty swap route")
}

func TestExecuteCrossChainSwap_LocalHappyPath(t *testing.T) {
	k, bk, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithChainID("paw-mvp-1")

	reserveA := math.NewInt(1_000_000)
	reserveB := math.NewInt(1_200_000)
	poolID := "route-1"

	seedLocalPool(t, k, ctx, poolID, "upaw", "uosmo", reserveA, reserveB, math.LegacyMustNewDecFromStr("0.003"))
	fundPoolAddress(t, bk, ctx, poolID, "upaw", "uosmo", reserveA, reserveB)

	route := keeper.CrossChainSwapRoute{
		Steps: []keeper.SwapStep{
			{
				ChainID:      ctx.ChainID(),
				PoolID:       poolID,
				TokenIn:      "upaw",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(100_000),
				MinAmountOut: math.NewInt(105_000),
			},
		},
	}

	result, err := k.ExecuteCrossChainSwap(ctx, types.TestAddr(), route, math.LegacyMustNewDecFromStr("0.50"))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.AmountOut.GT(math.ZeroInt()))
	require.True(t, result.Slippage.LTE(math.LegacyMustNewDecFromStr("0.10")))
	require.Contains(t, result.Route, poolID)
}

func TestExecuteCrossChainSwap_SlippageExceeded(t *testing.T) {
	k, bk, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Now()).WithChainID("paw-mvp-1")

	reserveA := math.NewInt(800_000)
	reserveB := math.NewInt(900_000)
	poolID := "route-2"

	seedLocalPool(t, k, ctx, poolID, "upaw", "uosmo", reserveA, reserveB, math.LegacyMustNewDecFromStr("0.003"))
	fundPoolAddress(t, bk, ctx, poolID, "upaw", "uosmo", reserveA, reserveB)

	route := keeper.CrossChainSwapRoute{
		Steps: []keeper.SwapStep{
			{
				ChainID:      ctx.ChainID(),
				PoolID:       poolID,
				TokenIn:      "upaw",
				TokenOut:     "uosmo",
				AmountIn:     math.NewInt(500_000),
				MinAmountOut: math.NewInt(495_000), // Unrealistically tight, should trip slippage
			},
		},
	}

	_, err := k.ExecuteCrossChainSwap(ctx, types.TestAddr(), route, math.LegacyMustNewDecFromStr("0.01"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")
}
