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

func setupMultiHopEnv(t *testing.T) (*keeper.Keeper, sdk.Context, sdk.AccAddress, []keeper.SwapHop, uint64, uint64) {
	t.Helper()

	k, _, ctx := keepertest.DexKeeperWithBank(t)

	// Build two connected pools: uosmo -> uatom -> upaw
	poolOsmoAtom := keepertest.CreateTestPool(t, k, ctx, "uosmo", "uatom", math.NewInt(5_000_000), math.NewInt(5_000_000))
	poolAtomPaw := keepertest.CreateTestPool(t, k, ctx, "uatom", "upaw", math.NewInt(5_000_000), math.NewInt(5_000_000))

	hops := []keeper.SwapHop{
		{PoolID: poolOsmoAtom, TokenIn: "uosmo", TokenOut: "uatom"},
		{PoolID: poolAtomPaw, TokenIn: "uatom", TokenOut: "upaw"},
	}

	trader := sdk.AccAddress([]byte("trader_multihop____"))
	keepertest.FundAccount(t, k, ctx, trader, sdk.NewCoins(sdk.NewInt64Coin("uosmo", 10_000_000)))

	return k, ctx, trader, hops, poolOsmoAtom, poolAtomPaw
}

func TestExecuteMultiHopSwap_Succeeds(t *testing.T) {
	k, ctx, trader, hops, _, _ := setupMultiHopEnv(t)

	startBalance := k.BankKeeper().GetBalance(ctx, trader, "upaw").Amount

	res, err := k.ExecuteMultiHopSwap(ctx, trader, hops, math.NewInt(300_000), math.NewInt(50_000))
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.AmountOut.GT(math.ZeroInt()))
	require.True(t, res.TotalFees.GT(math.ZeroInt()))

	endBalance := k.BankKeeper().GetBalance(ctx, trader, "upaw").Amount
	require.True(t, endBalance.GT(startBalance), "trader should receive output tokens")
}

func TestExecuteMultiHopSwap_Errors(t *testing.T) {
	k, ctx, trader, hops, poolOsmoAtom, _ := setupMultiHopEnv(t)

	// Broken hop chain
	badHops := []keeper.SwapHop{
		{PoolID: poolOsmoAtom, TokenIn: "uosmo", TokenOut: "uatom"},
		{PoolID: poolOsmoAtom, TokenIn: "uosmo", TokenOut: "upaw"}, // token out != next token in
	}
	_, err := k.ExecuteMultiHopSwap(ctx, trader, badHops, math.NewInt(1000), math.OneInt())
	require.ErrorIs(t, err, types.ErrInvalidInput)

	// Empty route
	_, err = k.ExecuteMultiHopSwap(ctx, trader, []keeper.SwapHop{}, math.NewInt(1000), math.OneInt())
	require.ErrorIs(t, err, types.ErrInvalidInput)

	// Slippage too high
	_, err = k.ExecuteMultiHopSwap(ctx, trader, hops, math.NewInt(500_000), math.NewInt(9_999_999_999))
	require.ErrorIs(t, err, types.ErrSlippageTooHigh)
}

func TestFindRoutesAndBestRoute(t *testing.T) {
	k, ctx, _, hops, poolOsmoAtom, poolAtomPaw := setupMultiHopEnv(t)

	// Add direct pool to verify best-route prefers 1-hop path
	poolDirect := keepertest.CreateTestPool(t, k, ctx, "uosmo", "upaw", math.NewInt(8_000_000), math.NewInt(8_000_000))

	routes, err := k.FindAllRoutes(ctx, "uosmo", "upaw", 3)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Ensure multi-hop route is discoverable
	foundMultiHop := false
	for _, route := range routes {
		if len(route) == len(hops) && route[0].PoolID == poolOsmoAtom && route[1].PoolID == poolAtomPaw {
			foundMultiHop = true
			break
		}
	}
	require.True(t, foundMultiHop, "expected multi-hop route to be returned")

	bestRoute, err := k.FindBestRoute(ctx, "uosmo", "upaw", math.NewInt(1_000_000), 3)
	require.NoError(t, err)
	require.Len(t, bestRoute, 1, "direct pool should be preferred when available")
	require.Equal(t, poolDirect, bestRoute[0].PoolID)

	_, err = k.FindAllRoutes(ctx, "uosmo", "unknown", 3)
	require.ErrorIs(t, err, types.ErrPoolNotFound)
}

func TestCalculateImpermanentLoss(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	lp := types.TestAddr()

	pool, err := k.CreatePool(ctx, lp, "upaw", "uusdc", math.NewInt(2_000_000), math.NewInt(2_000_000))
	require.NoError(t, err)
	require.NoError(t, k.SetLiquidityShares(ctx, pool.Id, lp, pool.TotalShares))

	// Accumulate LP fees for both tokens to ensure non-zero fee component
	_, _, err = k.CollectSwapFees(ctx, pool.Id, "upaw", math.NewInt(1_000_000))
	require.NoError(t, err)
	_, _, err = k.CollectSwapFees(ctx, pool.Id, "uusdc", math.NewInt(500_000))
	require.NoError(t, err)

	info, err := k.CalculateImpermanentLoss(ctx, pool.Id, lp, math.LegacyOneDec(), math.LegacyOneDec())
	require.NoError(t, err)
	require.True(t, info.FeesEarned.GT(math.ZeroInt()))
	require.False(t, info.NetProfitLoss.IsZero(), "net P&L should account for earned fees")

	_, err = k.CalculateImpermanentLoss(ctx, pool.Id, sdk.AccAddress([]byte("no_liquidity_______")), math.LegacyOneDec(), math.LegacyOneDec())
	require.ErrorIs(t, err, types.ErrInsufficientShares)
}
