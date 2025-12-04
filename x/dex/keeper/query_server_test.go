package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func setupDexQueryServer(t *testing.T) (types.QueryServer, *keeper.Keeper, sdk.Context) {
	t.Helper()

	k, ctx := keepertest.DexKeeper(t)
	return keeper.NewQueryServerImpl(*k), k, ctx
}

func TestQueryServer_Params(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)

	params := types.DefaultParams()
	params.SwapFee = sdkmath.LegacyMustNewDecFromStr("0.0125")
	require.NoError(t, k.SetParams(ctx, params))

	resp, err := server.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, resp.Params)

	_, err = server.Params(sdk.WrapSDKContext(ctx), nil)
	require.Error(t, err)
}

func TestQueryServer_PoolEndpoints(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolOne := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))
	keepertest.CreateTestPool(t, k, ctx, "uatom", "usdc", sdkmath.NewInt(2_000_000), sdkmath.NewInt(3_000_000))
	keepertest.CreateTestPool(t, k, ctx, "osmo", "tokenA", sdkmath.NewInt(3_000_000), sdkmath.NewInt(1_500_000))

	t.Run("pool by id", func(t *testing.T) {
		resp, err := server.Pool(sdk.WrapSDKContext(ctx), &types.QueryPoolRequest{PoolId: poolOne})
		require.NoError(t, err)
		require.Equal(t, poolOne, resp.Pool.Id)

		_, err = server.Pool(sdk.WrapSDKContext(ctx), &types.QueryPoolRequest{PoolId: 9999})
		require.Error(t, err)
	})

	t.Run("pool by tokens", func(t *testing.T) {
		resp, err := server.PoolByTokens(sdk.WrapSDKContext(ctx), &types.QueryPoolByTokensRequest{
			TokenA: "upaw",
			TokenB: "uusdt",
		})
		require.NoError(t, err)
		require.Equal(t, poolOne, resp.Pool.Id)
	})

	t.Run("pools pagination", func(t *testing.T) {
		resp, err := server.Pools(sdk.WrapSDKContext(ctx), &types.QueryPoolsRequest{
			Pagination: &query.PageRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, resp.Pools, 2)
		require.NotNil(t, resp.Pagination)
		require.NotNil(t, resp.Pagination.NextKey)
	})
}

func TestQueryServer_LiquidityAndSimulation(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	provider := types.TestAddr()
	shares := sdkmath.NewInt(250_000)
	require.NoError(t, k.SetLiquidity(sdk.WrapSDKContext(ctx), poolID, provider, shares))

	liquidityResp, err := server.Liquidity(sdk.WrapSDKContext(ctx), &types.QueryLiquidityRequest{
		PoolId:   poolID,
		Provider: provider.String(),
	})
	require.NoError(t, err)
	require.True(t, liquidityResp.Shares.Equal(shares))

	swapResp, err := server.SimulateSwap(sdk.WrapSDKContext(ctx), &types.QuerySimulateSwapRequest{
		PoolId:   poolID,
		TokenIn:  "upaw",
		TokenOut: "uusdt",
		AmountIn: sdkmath.NewInt(50_000),
	})
	require.NoError(t, err)
	require.True(t, swapResp.AmountOut.GT(sdkmath.ZeroInt()))
}
