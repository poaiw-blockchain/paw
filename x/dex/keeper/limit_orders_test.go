package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// LimitOrderTestSuite tests the limit order engine
type LimitOrderTestSuite struct {
	suite.Suite
	keeper     *keeper.Keeper
	bankKeeper bankkeeper.Keeper
	ctx        sdk.Context
	creator    sdk.AccAddress
	trader1    sdk.AccAddress
	trader2    sdk.AccAddress
	poolID     uint64
}

func (s *LimitOrderTestSuite) SetupTest() {
	s.keeper, s.bankKeeper, s.ctx = keepertest.DexKeeperWithBank(s.T())
	// Use addresses that match the prefunded accounts in testutil/keeper/dex.go
	s.creator = types.TestAddr()
	s.trader1 = sdk.AccAddress([]byte("regular_user_1_____"))
	s.trader2 = sdk.AccAddress([]byte("regular_user_2_____"))

	// Create a pool for testing
	pool, err := s.keeper.CreatePool(s.ctx, s.creator, "upaw", "uusdt", math.NewInt(1000000), math.NewInt(2000000))
	s.Require().NoError(err)
	s.poolID = pool.Id
}

func (s *LimitOrderTestSuite) fundAccount(addr sdk.AccAddress, denom string, amount math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	s.Require().NoError(s.bankKeeper.MintCoins(s.ctx, types.ModuleName, coins))
	s.Require().NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.ModuleName, addr, coins))
}

func TestLimitOrderSuite(t *testing.T) {
	suite.Run(t, new(LimitOrderTestSuite))
}

// ============================================================================
// PlaceLimitOrder Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_BuyOrder() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)

	s.Require().NoError(err)
	s.Require().NotNil(order)
	s.Require().Equal(s.trader1.String(), order.Owner)
	s.Require().Equal(s.poolID, order.PoolID)
	s.Require().Equal(keeper.OrderTypeBuy, order.OrderType)
	s.Require().Equal("uusdt", order.TokenIn)
	s.Require().Equal("upaw", order.TokenOut)
	s.Require().Equal(math.NewInt(100000), order.AmountIn)
	s.Require().Equal(keeper.OrderStatusOpen, order.Status)
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_SellOrder() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeSell,
		"upaw",
		"uusdt",
		math.NewInt(50000),
		math.LegacyNewDecWithPrec(2, 0),
		time.Hour*24,
	)

	s.Require().NoError(err)
	s.Require().NotNil(order)
	s.Require().Equal(keeper.OrderTypeSell, order.OrderType)
	s.Require().Equal("upaw", order.TokenIn)
	s.Require().Equal("uusdt", order.TokenOut)
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_InvalidPool() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		999999,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "pool not found")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_WrongTokens() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uatom",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "tokens do not match pool")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_ZeroAmount() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.ZeroInt(),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid amount")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_NegativeAmount() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(-100),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid amount")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_ZeroPrice() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyZeroDec(),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid limit price")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_NegativePrice() {
	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDec(-1),
		time.Hour,
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "invalid limit price")
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_NoExpiry() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		0,
	)

	s.Require().NoError(err)
	s.Require().True(order.ExpiresAt.IsZero())
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_UniqueIDs() {
	var orderIDs []uint64
	for i := 0; i < 5; i++ {
		order, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
		orderIDs = append(orderIDs, order.ID)
	}

	idSet := make(map[uint64]bool)
	for _, id := range orderIDs {
		s.Require().False(idSet[id], "Duplicate order ID: %d", id)
		idSet[id] = true
	}
}

// ============================================================================
// CancelLimitOrder Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestCancelLimitOrder_Success() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order.ID)
	s.Require().NoError(err)

	cancelled, err := s.keeper.GetLimitOrder(s.ctx, order.ID)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusCancelled, cancelled.Status)
}

func (s *LimitOrderTestSuite) TestCancelLimitOrder_WrongOwner() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader2, order.ID)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not order owner")
}

func (s *LimitOrderTestSuite) TestCancelLimitOrder_NonExistent() {
	err := s.keeper.CancelLimitOrder(s.ctx, s.trader1, 999999)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")
}

func (s *LimitOrderTestSuite) TestCancelLimitOrder_AlreadyCancelled() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)
	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order.ID)
	s.Require().NoError(err)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order.ID)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot be cancelled")
}

// ============================================================================
// GetLimitOrder Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestGetLimitOrder_Exists() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	retrieved, err := s.keeper.GetLimitOrder(s.ctx, order.ID)
	s.Require().NoError(err)
	s.Require().Equal(order.ID, retrieved.ID)
	s.Require().Equal(order.Owner, retrieved.Owner)
	s.Require().Equal(order.PoolID, retrieved.PoolID)
	s.Require().Equal(order.AmountIn, retrieved.AmountIn)
}

func (s *LimitOrderTestSuite) TestGetLimitOrder_NotFound() {
	_, err := s.keeper.GetLimitOrder(s.ctx, 999999)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")
}

// ============================================================================
// Query Functions Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestGetOrdersByOwner() {
	for i := 0; i < 3; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	for i := 0; i < 2; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader2,
			s.poolID,
			keeper.OrderTypeSell,
			"upaw",
			"uusdt",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(2, 0),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	orders, err := s.keeper.GetOrdersByOwner(s.ctx, s.trader1)
	s.Require().NoError(err)
	s.Require().Len(orders, 3)
	for _, order := range orders {
		s.Require().Equal(s.trader1.String(), order.Owner)
	}

	orders, err = s.keeper.GetOrdersByOwner(s.ctx, s.trader2)
	s.Require().NoError(err)
	s.Require().Len(orders, 2)
	for _, order := range orders {
		s.Require().Equal(s.trader2.String(), order.Owner)
	}
}

func (s *LimitOrderTestSuite) TestGetOrdersByOwnerPaginated() {
	for i := 0; i < 15; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	orders, nextKey, total, err := s.keeper.GetOrdersByOwnerPaginated(s.ctx, s.trader1, nil, 5)
	s.Require().NoError(err)
	s.Require().Len(orders, 5)
	s.Require().NotNil(nextKey)
	s.Require().Equal(uint64(15), total)

	orders2, nextKey2, _, err := s.keeper.GetOrdersByOwnerPaginated(s.ctx, s.trader1, nextKey, 5)
	s.Require().NoError(err)
	s.Require().Len(orders2, 5)
	s.Require().NotNil(nextKey2)

	orderIDs := make(map[uint64]bool)
	for _, o := range orders {
		orderIDs[o.ID] = true
	}
	for _, o := range orders2 {
		s.Require().False(orderIDs[o.ID], "Duplicate order in second page")
	}
}

func (s *LimitOrderTestSuite) TestGetOrdersByPool() {
	pool2, err := s.keeper.CreatePool(s.ctx, s.creator, "uatom", "uusdt", math.NewInt(500000), math.NewInt(1000000))
	s.Require().NoError(err)

	s.fundAccount(s.trader1, "uatom", math.NewInt(10000000))

	for i := 0; i < 3; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	for i := 0; i < 2; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			pool2.Id,
			keeper.OrderTypeBuy,
			"uusdt",
			"uatom",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	orders, err := s.keeper.GetOrdersByPool(s.ctx, s.poolID)
	s.Require().NoError(err)
	s.Require().Len(orders, 3)
	for _, order := range orders {
		s.Require().Equal(s.poolID, order.PoolID)
	}

	orders, err = s.keeper.GetOrdersByPool(s.ctx, pool2.Id)
	s.Require().NoError(err)
	s.Require().Len(orders, 2)
}

func (s *LimitOrderTestSuite) TestGetOpenOrders() {
	order1, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(10000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	order2, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(10000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order1.ID)
	s.Require().NoError(err)

	openOrders, err := s.keeper.GetOpenOrders(s.ctx)
	s.Require().NoError(err)

	foundOrder2 := false
	for _, o := range openOrders {
		s.Require().NotEqual(order1.ID, o.ID, "Cancelled order should not be in open orders")
		if o.ID == order2.ID {
			foundOrder2 = true
		}
	}
	s.Require().True(foundOrder2, "Open order should be in list")
}

func (s *LimitOrderTestSuite) TestGetOrderBook() {
	for i := 0; i < 3; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	for i := 0; i < 2; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeSell,
			"upaw",
			"uusdt",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(2, 0),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	buyOrders, sellOrders, err := s.keeper.GetOrderBook(s.ctx, s.poolID)
	s.Require().NoError(err)
	s.Require().Len(buyOrders, 3)
	s.Require().Len(sellOrders, 2)

	for _, o := range buyOrders {
		s.Require().Equal(keeper.OrderTypeBuy, o.OrderType)
	}
	for _, o := range sellOrders {
		s.Require().Equal(keeper.OrderTypeSell, o.OrderType)
	}
}

// ============================================================================
// ProcessExpiredOrders Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestProcessExpiredOrders_ExpiredOrder() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Minute,
	)
	s.Require().NoError(err)

	newCtx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))

	err = s.keeper.ProcessExpiredOrders(newCtx)
	s.Require().NoError(err)

	expired, err := s.keeper.GetLimitOrder(newCtx, order.ID)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusExpired, expired.Status)
}

func (s *LimitOrderTestSuite) TestProcessExpiredOrders_NotExpired() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour*24,
	)
	s.Require().NoError(err)

	err = s.keeper.ProcessExpiredOrders(s.ctx)
	s.Require().NoError(err)

	notExpired, err := s.keeper.GetLimitOrder(s.ctx, order.ID)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, notExpired.Status)
}

func (s *LimitOrderTestSuite) TestProcessExpiredOrders_NoExpiryOrder() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		0,
	)
	s.Require().NoError(err)

	newCtx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour * 24 * 365))

	err = s.keeper.ProcessExpiredOrders(newCtx)
	s.Require().NoError(err)

	stillOpen, err := s.keeper.GetLimitOrder(newCtx, order.ID)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, stillOpen.Status)
}

// ============================================================================
// MatchLimitOrder Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestMatchLimitOrder_BuyOrderPriceNotMet() {
	pool, err := s.keeper.GetPool(s.ctx, s.poolID)
	s.Require().NoError(err)
	currentPrice := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))

	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		currentPrice.Mul(math.LegacyNewDecWithPrec(5, 1)),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, order.Status)
	s.Require().True(order.FilledAmount.IsZero())
}

func (s *LimitOrderTestSuite) TestMatchLimitOrder_SellOrderPriceNotMet() {
	pool, err := s.keeper.GetPool(s.ctx, s.poolID)
	s.Require().NoError(err)
	currentPrice := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))

	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeSell,
		"upaw",
		"uusdt",
		math.NewInt(50000),
		currentPrice.Mul(math.LegacyNewDec(2)),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, order.Status)
	s.Require().True(order.FilledAmount.IsZero())
}

// ============================================================================
// Key Function Tests
// ============================================================================

func TestLimitOrderKey(t *testing.T) {
	key := keeper.LimitOrderKey(12345)
	require.NotEmpty(t, key)
	require.True(t, len(key) > 8)
}

func TestLimitOrderByOwnerKey(t *testing.T) {
	owner := sdk.AccAddress([]byte("test_owner_address__"))
	key := keeper.LimitOrderByOwnerKey(owner, 12345)
	require.NotEmpty(t, key)
	require.True(t, len(key) > len(owner)+8)
}

func TestLimitOrderByPoolKey(t *testing.T) {
	key := keeper.LimitOrderByPoolKey(1, 12345)
	require.NotEmpty(t, key)
	require.True(t, len(key) > 16)
}

func TestLimitOrderByPriceKey(t *testing.T) {
	price := math.LegacyNewDecWithPrec(123456789, 8)
	key := keeper.LimitOrderByPriceKey(1, keeper.OrderTypeBuy, price, 12345)
	require.NotEmpty(t, key)
}

func TestLimitOrderOpenKey(t *testing.T) {
	key := keeper.LimitOrderOpenKey(12345)
	require.NotEmpty(t, key)
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_VerySmallAmount() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(1),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(1), order.AmountIn)
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_VerySmallPrice() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(1, 18),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().True(order.LimitPrice.IsPositive())
}

func (s *LimitOrderTestSuite) TestPlaceLimitOrder_VeryLargePrice() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDec(1e15),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().True(order.LimitPrice.GT(math.LegacyNewDec(1e14)))
}

func (s *LimitOrderTestSuite) TestDeleteLimitOrder() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	err = s.keeper.DeleteLimitOrder(s.ctx, order)
	s.Require().NoError(err)

	_, err = s.keeper.GetLimitOrder(s.ctx, order.ID)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")
}

func (s *LimitOrderTestSuite) TestMatchAllOrders() {
	for i := 0; i < 5; i++ {
		_, err := s.keeper.PlaceLimitOrder(
			s.ctx,
			s.trader1,
			s.poolID,
			keeper.OrderTypeBuy,
			"uusdt",
			"upaw",
			math.NewInt(10000),
			math.LegacyNewDecWithPrec(5, 1),
			time.Hour,
		)
		s.Require().NoError(err)
	}

	err := s.keeper.MatchAllOrders(s.ctx)
	s.Require().NoError(err)
}

// ============================================================================
// Concurrent Order Tests (Simulated)
// ============================================================================

func (s *LimitOrderTestSuite) TestMultipleTraderOrders() {
	traders := []sdk.AccAddress{
		sdk.AccAddress([]byte("trader_a____________")),
		sdk.AccAddress([]byte("trader_b____________")),
		sdk.AccAddress([]byte("trader_c____________")),
	}

	for _, trader := range traders {
		s.fundAccount(trader, "uusdt", math.NewInt(10000000))
	}

	for _, trader := range traders {
		for i := 0; i < 3; i++ {
			_, err := s.keeper.PlaceLimitOrder(
				s.ctx,
				trader,
				s.poolID,
				keeper.OrderTypeBuy,
				"uusdt",
				"upaw",
				math.NewInt(10000),
				math.LegacyNewDecWithPrec(5, 1),
				time.Hour,
			)
			s.Require().NoError(err)
		}
	}

	for _, trader := range traders {
		orders, err := s.keeper.GetOrdersByOwner(s.ctx, trader)
		s.Require().NoError(err)
		s.Require().Len(orders, 3)
	}
}

func (s *LimitOrderTestSuite) TestOrderAcrossMultiplePools() {
	// Create additional pools using prefunded tokens from testutil/keeper/dex.go
	pool2, err := s.keeper.CreatePool(s.ctx, s.creator, "uatom", "uusdt", math.NewInt(500000), math.NewInt(1000000))
	s.Require().NoError(err)
	pool3, err := s.keeper.CreatePool(s.ctx, s.creator, "tokenA", "tokenB", math.NewInt(300000), math.NewInt(600000))
	s.Require().NoError(err)

	// Place orders in the first pool (upaw/uusdt)
	_, err = s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(10000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	// Place orders in the second pool (uatom/uusdt)
	_, err = s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		pool2.Id,
		keeper.OrderTypeBuy,
		"uusdt",
		"uatom",
		math.NewInt(10000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	// Place orders in the third pool (tokenA/tokenB)
	_, err = s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		pool3.Id,
		keeper.OrderTypeBuy,
		"tokenB",
		"tokenA",
		math.NewInt(10000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	pools := []uint64{s.poolID, pool2.Id, pool3.Id}
	for _, poolID := range pools {
		orders, err := s.keeper.GetOrdersByPool(s.ctx, poolID)
		s.Require().NoError(err)
		s.Require().GreaterOrEqual(len(orders), 1)
	}
}

// ============================================================================
// Refund Tests
// ============================================================================

func (s *LimitOrderTestSuite) TestCancelLimitOrder_RefundsTokens() {
	initialBalance := s.bankKeeper.GetBalance(s.ctx, s.trader1, "uusdt")
	orderAmount := math.NewInt(100000)

	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		orderAmount,
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)

	afterPlaceBalance := s.bankKeeper.GetBalance(s.ctx, s.trader1, "uusdt")
	s.Require().Equal(initialBalance.Amount.Sub(orderAmount), afterPlaceBalance.Amount)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order.ID)
	s.Require().NoError(err)

	afterCancelBalance := s.bankKeeper.GetBalance(s.ctx, s.trader1, "uusdt")
	s.Require().Equal(initialBalance.Amount, afterCancelBalance.Amount)
}

func (s *LimitOrderTestSuite) TestProcessExpiredOrders_RefundsTokens() {
	initialBalance := s.bankKeeper.GetBalance(s.ctx, s.trader1, "uusdt")
	orderAmount := math.NewInt(100000)

	_, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		orderAmount,
		math.LegacyNewDecWithPrec(5, 1),
		time.Minute,
	)
	s.Require().NoError(err)

	afterPlaceBalance := s.bankKeeper.GetBalance(s.ctx, s.trader1, "uusdt")
	s.Require().Equal(initialBalance.Amount.Sub(orderAmount), afterPlaceBalance.Amount)

	newCtx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))
	err = s.keeper.ProcessExpiredOrders(newCtx)
	s.Require().NoError(err)

	afterExpireBalance := s.bankKeeper.GetBalance(newCtx, s.trader1, "uusdt")
	s.Require().Equal(initialBalance.Amount, afterExpireBalance.Amount)
}

// ============================================================================
// Order State Transitions
// ============================================================================

func (s *LimitOrderTestSuite) TestOrderStateTransitions() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Hour,
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, order.Status)

	err = s.keeper.CancelLimitOrder(s.ctx, s.trader1, order.ID)
	s.Require().NoError(err)

	cancelled, _ := s.keeper.GetLimitOrder(s.ctx, order.ID)
	s.Require().Equal(keeper.OrderStatusCancelled, cancelled.Status)
}

func (s *LimitOrderTestSuite) TestOrderExpiryStateTransition() {
	order, err := s.keeper.PlaceLimitOrder(
		s.ctx,
		s.trader1,
		s.poolID,
		keeper.OrderTypeBuy,
		"uusdt",
		"upaw",
		math.NewInt(100000),
		math.LegacyNewDecWithPrec(5, 1),
		time.Minute,
	)
	s.Require().NoError(err)
	s.Require().Equal(keeper.OrderStatusOpen, order.Status)

	newCtx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))
	err = s.keeper.ProcessExpiredOrders(newCtx)
	s.Require().NoError(err)

	expired, _ := s.keeper.GetLimitOrder(newCtx, order.ID)
	s.Require().Equal(keeper.OrderStatusExpired, expired.Status)
}
