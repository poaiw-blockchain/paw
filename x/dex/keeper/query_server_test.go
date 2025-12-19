package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

	resp, err := server.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, resp.Params)

	_, err = server.Params(ctx, nil)
	require.Error(t, err)
}

func TestQueryServer_PoolEndpoints(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolOne := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))
	keepertest.CreateTestPool(t, k, ctx, "uatom", "usdc", sdkmath.NewInt(2_000_000), sdkmath.NewInt(3_000_000))
	keepertest.CreateTestPool(t, k, ctx, "osmo", "tokenA", sdkmath.NewInt(3_000_000), sdkmath.NewInt(1_500_000))

	t.Run("pool by id", func(t *testing.T) {
		resp, err := server.Pool(ctx, &types.QueryPoolRequest{PoolId: poolOne})
		require.NoError(t, err)
		require.Equal(t, poolOne, resp.Pool.Id)

		_, err = server.Pool(ctx, &types.QueryPoolRequest{PoolId: 9999})
		require.Error(t, err)
	})

	t.Run("pool by tokens", func(t *testing.T) {
		resp, err := server.PoolByTokens(ctx, &types.QueryPoolByTokensRequest{
			TokenA: "upaw",
			TokenB: "uusdt",
		})
		require.NoError(t, err)
		require.Equal(t, poolOne, resp.Pool.Id)
	})

	t.Run("pools pagination", func(t *testing.T) {
		resp, err := server.Pools(ctx, &types.QueryPoolsRequest{
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
	require.NoError(t, k.SetLiquidity(ctx, poolID, provider, shares))

	liquidityResp, err := server.Liquidity(ctx, &types.QueryLiquidityRequest{
		PoolId:   poolID,
		Provider: provider.String(),
	})
	require.NoError(t, err)
	require.True(t, liquidityResp.Shares.Equal(shares))

	swapResp, err := server.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
		PoolId:   poolID,
		TokenIn:  "upaw",
		TokenOut: "uusdt",
		AmountIn: sdkmath.NewInt(50_000),
	})
	require.NoError(t, err)
	require.True(t, swapResp.AmountOut.GT(sdkmath.ZeroInt()))
}

func TestQueryServer_LimitOrder(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	trader := sdk.AccAddress([]byte("test_trader_address"))

	t.Run("query existing order", func(t *testing.T) {
		order, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(100_000),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)

		resp, err := server.LimitOrder(ctx, &types.QueryLimitOrderRequest{
			OrderId: order.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, order.ID, resp.Order.Id)
		require.Equal(t, trader.String(), resp.Order.Owner)
		require.Equal(t, poolID, resp.Order.PoolId)
		require.Equal(t, "upaw", resp.Order.TokenIn)
		require.Equal(t, "uusdt", resp.Order.TokenOut)
		require.True(t, resp.Order.AmountIn.Equal(sdkmath.NewInt(100_000)))
		require.Equal(t, types.OrderStatus_ORDER_STATUS_OPEN, resp.Order.Status)
	})

	t.Run("nil request error", func(t *testing.T) {
		_, err := server.LimitOrder(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("order not found error", func(t *testing.T) {
		_, err := server.LimitOrder(ctx, &types.QueryLimitOrderRequest{
			OrderId: 99999,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("verify order type conversion - buy", func(t *testing.T) {
		buyOrder, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(50_000),
			sdkmath.LegacyMustNewDecFromStr("2.5"),
			0,
		)
		require.NoError(t, err)

		resp, err := server.LimitOrder(ctx, &types.QueryLimitOrderRequest{
			OrderId: buyOrder.ID,
		})
		require.NoError(t, err)
		require.Equal(t, types.OrderType_ORDER_TYPE_BUY, resp.Order.OrderType)
	})

	t.Run("verify order type conversion - sell", func(t *testing.T) {
		sellOrder, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID,
			keeper.OrderTypeSell,
			"uusdt",
			"upaw",
			sdkmath.NewInt(50_000),
			sdkmath.LegacyMustNewDecFromStr("0.4"),
			0,
		)
		require.NoError(t, err)

		resp, err := server.LimitOrder(ctx, &types.QueryLimitOrderRequest{
			OrderId: sellOrder.ID,
		})
		require.NoError(t, err)
		require.Equal(t, types.OrderType_ORDER_TYPE_SELL, resp.Order.OrderType)
	})
}

func TestQueryServer_LimitOrders(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	trader1 := sdk.AccAddress([]byte("trader1____________"))
	trader2 := sdk.AccAddress([]byte("trader2____________"))

	t.Run("empty orders list", func(t *testing.T) {
		resp, err := server.LimitOrders(ctx, &types.QueryLimitOrdersRequest{})
		require.NoError(t, err)
		require.Empty(t, resp.Orders)
		require.NotNil(t, resp.Pagination)
	})

	// Create multiple orders
	for i := 0; i < 5; i++ {
		trader := trader1
		if i%2 == 0 {
			trader = trader2
		}
		_, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(10_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	t.Run("query all orders without pagination", func(t *testing.T) {
		resp, err := server.LimitOrders(ctx, &types.QueryLimitOrdersRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 5)
		require.NotNil(t, resp.Pagination)
	})

	t.Run("query with pagination limit", func(t *testing.T) {
		resp, err := server.LimitOrders(ctx, &types.QueryLimitOrdersRequest{
			Pagination: &query.PageRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 2)
		require.NotNil(t, resp.Pagination)
		require.NotNil(t, resp.Pagination.NextKey)

		// Query next page
		resp2, err := server.LimitOrders(ctx, &types.QueryLimitOrdersRequest{
			Pagination: &query.PageRequest{
				Key:   resp.Pagination.NextKey,
				Limit: 2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp2.Orders, 2)
	})

	t.Run("nil request error", func(t *testing.T) {
		_, err := server.LimitOrders(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("verify order details in list", func(t *testing.T) {
		resp, err := server.LimitOrders(ctx, &types.QueryLimitOrdersRequest{
			Pagination: &query.PageRequest{Limit: 1},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 1)
		order := resp.Orders[0]
		require.Greater(t, order.Id, uint64(0))
		require.NotEmpty(t, order.Owner)
		require.Equal(t, poolID, order.PoolId)
		require.NotEmpty(t, order.TokenIn)
		require.NotEmpty(t, order.TokenOut)
		require.True(t, order.AmountIn.GT(sdkmath.ZeroInt()))
	})
}

func TestQueryServer_LimitOrdersByOwner(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID1 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))
	poolID2 := keepertest.CreateTestPool(t, k, ctx, "uatom", "usdc", sdkmath.NewInt(500_000), sdkmath.NewInt(1_000_000))

	trader1 := sdk.AccAddress([]byte("trader1____________"))
	trader2 := sdk.AccAddress([]byte("trader2____________"))

	t.Run("empty orders for new owner", func(t *testing.T) {
		resp, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader1.String(),
		})
		require.NoError(t, err)
		require.Empty(t, resp.Orders)
		require.NotNil(t, resp.Pagination)
	})

	// Create orders for trader1 across multiple pools
	for i := 0; i < 3; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader1,
			poolID1,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(10_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	// Create order for trader1 in different pool
	_, err := k.PlaceLimitOrder(
		ctx,
		trader1,
		poolID2,
		keeper.OrderTypeBuy,
		"uatom",
		"usdc",
		sdkmath.NewInt(5_000),
		sdkmath.LegacyMustNewDecFromStr("1.5"),
		0,
	)
	require.NoError(t, err)

	// Create orders for trader2
	for i := 0; i < 2; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader2,
			poolID1,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(5_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	t.Run("query all orders for owner", func(t *testing.T) {
		resp, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader1.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 4) // 3 from pool1 + 1 from pool2
		require.NotNil(t, resp.Pagination)

		// Verify all orders belong to trader1
		for _, order := range resp.Orders {
			require.Equal(t, trader1.String(), order.Owner)
		}
	})

	t.Run("query different owner", func(t *testing.T) {
		resp, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader2.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 2)

		// Verify all orders belong to trader2
		for _, order := range resp.Orders {
			require.Equal(t, trader2.String(), order.Owner)
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		resp, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader1.String(),
			Pagination: &query.PageRequest{
				Limit: 2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 2)
		require.NotNil(t, resp.Pagination)
		require.NotNil(t, resp.Pagination.NextKey)

		// Query next page
		resp2, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader1.String(),
			Pagination: &query.PageRequest{
				Key:   resp.Pagination.NextKey,
				Limit: 2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp2.Orders, 2)
	})

	t.Run("nil request error", func(t *testing.T) {
		_, err := server.LimitOrdersByOwner(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("invalid owner address error", func(t *testing.T) {
		_, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: "invalid_address",
		})
		require.Error(t, err)
	})

	t.Run("default pagination limit", func(t *testing.T) {
		resp, err := server.LimitOrdersByOwner(ctx, &types.QueryLimitOrdersByOwnerRequest{
			Owner: trader1.String(),
			Pagination: &query.PageRequest{
				Limit: 0, // Should use default limit of 100
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 4) // Less than default 100
	})
}

func TestQueryServer_LimitOrdersByPool(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID1 := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))
	poolID2 := keepertest.CreateTestPool(t, k, ctx, "uatom", "usdc", sdkmath.NewInt(500_000), sdkmath.NewInt(1_000_000))

	trader1 := sdk.AccAddress([]byte("trader1____________"))
	trader2 := sdk.AccAddress([]byte("trader2____________"))
	trader3 := sdk.AccAddress([]byte("trader3____________"))

	t.Run("empty orders for new pool", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID1,
		})
		require.NoError(t, err)
		require.Empty(t, resp.Orders)
		require.NotNil(t, resp.Pagination)
	})

	// Create orders for pool1 from multiple traders
	for i := 0; i < 3; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader1,
			poolID1,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(10_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader2,
			poolID1,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(5_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	// Create orders for pool2
	for i := 0; i < 4; i++ {
		trader := trader2
		if i%2 == 0 {
			trader = trader3
		}
		_, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID2,
			keeper.OrderTypeBuy,
			"uatom",
			"usdc",
			sdkmath.NewInt(int64(3_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("1.8"),
			0,
		)
		require.NoError(t, err)
	}

	t.Run("query all orders for pool1", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID1,
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 5) // 3 from trader1 + 2 from trader2
		require.NotNil(t, resp.Pagination)

		// Verify all orders belong to pool1
		for _, order := range resp.Orders {
			require.Equal(t, poolID1, order.PoolId)
		}
	})

	t.Run("query all orders for pool2", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID2,
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 4)

		// Verify all orders belong to pool2
		for _, order := range resp.Orders {
			require.Equal(t, poolID2, order.PoolId)
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID1,
			Pagination: &query.PageRequest{
				Limit: 3,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 3)
		require.NotNil(t, resp.Pagination)
		require.NotNil(t, resp.Pagination.NextKey)

		// Query next page
		resp2, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID1,
			Pagination: &query.PageRequest{
				Key:   resp.Pagination.NextKey,
				Limit: 3,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp2.Orders, 2) // Remaining orders
	})

	t.Run("nil request error", func(t *testing.T) {
		_, err := server.LimitOrdersByPool(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("non-existent pool", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: 99999,
		})
		// Should return empty list, not error
		require.NoError(t, err)
		require.Empty(t, resp.Orders)
	})

	t.Run("default pagination limit", func(t *testing.T) {
		resp, err := server.LimitOrdersByPool(ctx, &types.QueryLimitOrdersByPoolRequest{
			PoolId: poolID2,
			Pagination: &query.PageRequest{
				Limit: 0, // Should use default limit of 100
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Orders, 4) // Less than default 100
	})
}

func TestQueryServer_OrderBook(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	trader1 := sdk.AccAddress([]byte("trader1____________"))
	trader2 := sdk.AccAddress([]byte("trader2____________"))
	trader3 := sdk.AccAddress([]byte("trader3____________"))

	t.Run("empty order book", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
		})
		require.NoError(t, err)
		require.Empty(t, resp.BuyOrders)
		require.Empty(t, resp.SellOrders)
	})

	// Place multiple buy orders (buying uusdt with upaw) with limit prices that won't match
	// Pool has 1M upaw and 2M uusdt, so current price is 2.0 uusdt per upaw
	// Use limit price of 1.5 for buy orders so they won't execute immediately
	for i := 0; i < 3; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader1,
			poolID,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(int64(10_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("1.5"),
			0,
		)
		require.NoError(t, err)
	}

	// Place multiple sell orders (selling upaw for uusdt) with limit prices that won't match
	// Use limit price of 2.5 for sell orders so they won't execute immediately
	for i := 0; i < 4; i++ {
		_, err := k.PlaceLimitOrder(
			ctx,
			trader2,
			poolID,
			keeper.OrderTypeSell,
			"uusdt",
			"upaw",
			sdkmath.NewInt(int64(8_000*(i+1))),
			sdkmath.LegacyMustNewDecFromStr("2.5"),
			0,
		)
		require.NoError(t, err)
	}

	t.Run("query order book with orders", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
		})
		require.NoError(t, err)
		require.Len(t, resp.BuyOrders, 3)
		require.Len(t, resp.SellOrders, 4)

		// Verify buy orders
		for _, order := range resp.BuyOrders {
			require.Equal(t, poolID, order.PoolId)
			require.Equal(t, types.OrderType_ORDER_TYPE_BUY, order.OrderType)
		}

		// Verify sell orders
		for _, order := range resp.SellOrders {
			require.Equal(t, poolID, order.PoolId)
			require.Equal(t, types.OrderType_ORDER_TYPE_SELL, order.OrderType)
		}
	})

	t.Run("query with custom limit", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
			Limit:  2,
		})
		require.NoError(t, err)
		require.Len(t, resp.BuyOrders, 2)  // Limited to 2
		require.Len(t, resp.SellOrders, 2) // Limited to 2
	})

	t.Run("query with zero limit uses default", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
			Limit:  0, // Should use default limit of 50
		})
		require.NoError(t, err)
		require.Len(t, resp.BuyOrders, 3)  // All buy orders (less than 50)
		require.Len(t, resp.SellOrders, 4) // All sell orders (less than 50)
	})

	t.Run("query with large limit", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
			Limit:  100,
		})
		require.NoError(t, err)
		require.Len(t, resp.BuyOrders, 3)  // All available
		require.Len(t, resp.SellOrders, 4) // All available
	})

	t.Run("nil request error", func(t *testing.T) {
		_, err := server.OrderBook(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("non-existent pool returns empty", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: 99999,
		})
		// GetOrdersByPool returns empty list for non-existent pool, not an error
		require.NoError(t, err)
		require.Empty(t, resp.BuyOrders)
		require.Empty(t, resp.SellOrders)
	})

	// Create more orders to test limit enforcement
	for i := 0; i < 60; i++ {
		trader := trader3
		_, err := k.PlaceLimitOrder(
			ctx,
			trader,
			poolID,
			keeper.OrderTypeBuy,
			"upaw",
			"uusdt",
			sdkmath.NewInt(1_000),
			sdkmath.LegacyMustNewDecFromStr("2.0"),
			0,
		)
		require.NoError(t, err)
	}

	t.Run("limit enforcement with many orders", func(t *testing.T) {
		resp, err := server.OrderBook(ctx, &types.QueryOrderBookRequest{
			PoolId: poolID,
			Limit:  10,
		})
		require.NoError(t, err)
		require.Len(t, resp.BuyOrders, 10) // Should be limited to 10
	})
}

func TestQueryServer_NilRequestHandling(t *testing.T) {
	server, _, ctx := setupDexQueryServer(t)

	t.Run("Pool nil request", func(t *testing.T) {
		_, err := server.Pool(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("Pools nil request", func(t *testing.T) {
		_, err := server.Pools(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("PoolByTokens nil request", func(t *testing.T) {
		_, err := server.PoolByTokens(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("Liquidity nil request", func(t *testing.T) {
		_, err := server.Liquidity(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})

	t.Run("SimulateSwap nil request", func(t *testing.T) {
		_, err := server.SimulateSwap(ctx, nil)
		require.Error(t, err)
		require.Equal(t, sdkerrors.ErrInvalidRequest, err)
	})
}

func TestQueryServer_ErrorCases(t *testing.T) {
	server, k, ctx := setupDexQueryServer(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	t.Run("Liquidity - invalid address", func(t *testing.T) {
		_, err := server.Liquidity(ctx, &types.QueryLiquidityRequest{
			PoolId:   poolID,
			Provider: "invalid_bech32_address",
		})
		require.Error(t, err)
	})

	t.Run("Liquidity - non-existent pool", func(t *testing.T) {
		provider := types.TestAddr()
		resp, err := server.Liquidity(ctx, &types.QueryLiquidityRequest{
			PoolId:   99999,
			Provider: provider.String(),
		})
		// Non-existent pool returns zero shares, not an error
		require.NoError(t, err)
		require.True(t, resp.Shares.IsZero())
	})

	t.Run("SimulateSwap - non-existent pool", func(t *testing.T) {
		_, err := server.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
			PoolId:   99999,
			TokenIn:  "upaw",
			TokenOut: "uusdt",
			AmountIn: sdkmath.NewInt(1_000),
		})
		require.Error(t, err)
	})

	t.Run("SimulateSwap - invalid token pair", func(t *testing.T) {
		_, err := server.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
			PoolId:   poolID,
			TokenIn:  "invalid",
			TokenOut: "uusdt",
			AmountIn: sdkmath.NewInt(1_000),
		})
		require.Error(t, err)
	})

	t.Run("PoolByTokens - non-existent pair", func(t *testing.T) {
		_, err := server.PoolByTokens(ctx, &types.QueryPoolByTokensRequest{
			TokenA: "nonexistent1",
			TokenB: "nonexistent2",
		})
		require.Error(t, err)
	})
}
