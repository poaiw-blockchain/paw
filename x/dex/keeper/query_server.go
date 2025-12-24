package keeper

import (
	"context"
	"encoding/json"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/paw-chain/paw/x/dex/types"
)

type queryServer struct {
	Keeper
}

const (
	defaultPaginationLimit = 100
	maxPaginationLimit     = 1000
)

// NewQueryServerImpl returns an implementation of the dex QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Params returns the module parameters
func (qs queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	params, err := qs.Keeper.GetParams(goCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Pool returns a specific pool by ID
func (qs queryServer) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	pool, err := qs.Keeper.GetPool(goCtx, req.PoolId)
	if err != nil {
		return nil, err
	}

	return &types.QueryPoolResponse{
		Pool: *pool,
	}, nil
}

// Pools returns all pools with pagination
func (qs queryServer) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	// Enforce sane pagination defaults and caps to protect against unbounded queries.
	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{Limit: defaultPaginationLimit}
	} else {
		if req.Pagination.Limit == 0 {
			req.Pagination.Limit = defaultPaginationLimit
		}
		if req.Pagination.Limit > maxPaginationLimit {
			req.Pagination.Limit = maxPaginationLimit
		}
	}

	// P3-PERF-3: Pre-size with pagination limit capacity
	limit := uint64(defaultPaginationLimit)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > maxPaginationLimit {
			limit = maxPaginationLimit
		}
	}
	pools := make([]types.Pool, 0, int(limit))
	store := qs.Keeper.getStore(goCtx)
	poolStore := prefix.NewStore(store, PoolKeyPrefix)

	pageRes, err := query.Paginate(poolStore, req.Pagination, func(key []byte, value []byte) error {
		var pool types.Pool
		if err := qs.Keeper.cdc.Unmarshal(value, &pool); err != nil {
			return err
		}
		pools = append(pools, pool)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryPoolsResponse{
		Pools:      pools,
		Pagination: pageRes,
	}, nil
}

// PoolByTokens returns a pool by its token pair
func (qs queryServer) PoolByTokens(goCtx context.Context, req *types.QueryPoolByTokensRequest) (*types.QueryPoolByTokensResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	pool, err := qs.Keeper.GetPoolByTokens(goCtx, req.TokenA, req.TokenB)
	if err != nil {
		return nil, err
	}

	return &types.QueryPoolByTokensResponse{
		Pool: *pool,
	}, nil
}

// Liquidity returns a user's liquidity position in a pool
func (qs queryServer) Liquidity(goCtx context.Context, req *types.QueryLiquidityRequest) (*types.QueryLiquidityResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	provider, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, err
	}

	shares, err := qs.Keeper.GetLiquidity(goCtx, req.PoolId, provider)
	if err != nil {
		return nil, err
	}

	return &types.QueryLiquidityResponse{
		Shares: shares,
	}, nil
}

// SimulateSwap simulates a swap without executing it
func (qs queryServer) SimulateSwap(goCtx context.Context, req *types.QuerySimulateSwapRequest) (*types.QuerySimulateSwapResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	amountOut, err := qs.Keeper.SimulateSwap(goCtx, req.PoolId, req.TokenIn, req.TokenOut, req.AmountIn)
	if err != nil {
		return nil, err
	}

	return &types.QuerySimulateSwapResponse{
		AmountOut: amountOut,
	}, nil
}

// LimitOrder returns a specific limit order by ID
func (qs queryServer) LimitOrder(goCtx context.Context, req *types.QueryLimitOrderRequest) (*types.QueryLimitOrderResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	order, err := qs.Keeper.GetLimitOrder(goCtx, req.OrderId)
	if err != nil {
		return nil, err
	}

	// Convert internal LimitOrder to proto type
	protoOrder := convertToProtoLimitOrder(order)

	return &types.QueryLimitOrderResponse{
		Order: protoOrder,
	}, nil
}

// LimitOrders returns all limit orders with pagination
func (qs queryServer) LimitOrders(goCtx context.Context, req *types.QueryLimitOrdersRequest) (*types.QueryLimitOrdersResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{Limit: defaultPaginationLimit}
	} else {
		if req.Pagination.Limit == 0 {
			req.Pagination.Limit = defaultPaginationLimit
		}
		if req.Pagination.Limit > maxPaginationLimit {
			req.Pagination.Limit = maxPaginationLimit
		}
	}

	// P3-PERF-3: Pre-size with pagination limit capacity
	limit := uint64(defaultPaginationLimit)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > maxPaginationLimit {
			limit = maxPaginationLimit
		}
	}
	orders := make([]types.LimitOrder, 0, int(limit))
	store := qs.Keeper.getStore(goCtx)
	orderStore := prefix.NewStore(store, LimitOrderKeyPrefix)

	pageRes, err := query.Paginate(orderStore, req.Pagination, func(key []byte, value []byte) error {
		var order LimitOrder
		if err := json.Unmarshal(value, &order); err != nil {
			return err
		}
		orders = append(orders, convertToProtoLimitOrder(&order))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryLimitOrdersResponse{
		Orders:     orders,
		Pagination: pageRes,
	}, nil
}

// LimitOrdersByOwner returns limit orders for a specific owner
func (qs queryServer) LimitOrdersByOwner(goCtx context.Context, req *types.QueryLimitOrdersByOwnerRequest) (*types.QueryLimitOrdersByOwnerResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{Limit: defaultPaginationLimit}
	} else {
		if req.Pagination.Limit == 0 {
			req.Pagination.Limit = defaultPaginationLimit
		}
		if req.Pagination.Limit > maxPaginationLimit {
			req.Pagination.Limit = maxPaginationLimit
		}
	}

	owner, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, err
	}

	// Extract pagination parameters
	var pageKey []byte
	var limit uint64 = defaultPaginationLimit
	if req.Pagination != nil {
		pageKey = req.Pagination.Key
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
	}

	internalOrders, nextKey, total, err := qs.Keeper.GetOrdersByOwnerPaginated(goCtx, owner, pageKey, limit)
	if err != nil {
		return nil, err
	}

	// P3-PERF-3: Pre-size with known capacity from internal orders
	orders := make([]types.LimitOrder, 0, len(internalOrders))
	for _, order := range internalOrders {
		orders = append(orders, convertToProtoLimitOrder(order))
	}

	return &types.QueryLimitOrdersByOwnerResponse{
		Orders: orders,
		Pagination: &query.PageResponse{
			NextKey: nextKey,
			Total:   total,
		},
	}, nil
}

// LimitOrdersByPool returns limit orders for a specific pool
func (qs queryServer) LimitOrdersByPool(goCtx context.Context, req *types.QueryLimitOrdersByPoolRequest) (*types.QueryLimitOrdersByPoolResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if req.Pagination == nil {
		req.Pagination = &query.PageRequest{Limit: defaultPaginationLimit}
	} else {
		if req.Pagination.Limit == 0 {
			req.Pagination.Limit = defaultPaginationLimit
		}
		if req.Pagination.Limit > maxPaginationLimit {
			req.Pagination.Limit = maxPaginationLimit
		}
	}

	// Extract pagination parameters
	var pageKey []byte
	var limit uint64 = defaultPaginationLimit
	if req.Pagination != nil {
		pageKey = req.Pagination.Key
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
	}

	internalOrders, nextKey, total, err := qs.Keeper.GetOrdersByPoolPaginated(goCtx, req.PoolId, pageKey, limit)
	if err != nil {
		return nil, err
	}

	// P3-PERF-3: Pre-size with known capacity from internal orders
	orders := make([]types.LimitOrder, 0, len(internalOrders))
	for _, order := range internalOrders {
		orders = append(orders, convertToProtoLimitOrder(order))
	}

	return &types.QueryLimitOrdersByPoolResponse{
		Orders: orders,
		Pagination: &query.PageResponse{
			NextKey: nextKey,
			Total:   total,
		},
	}, nil
}

// OrderBook returns the order book for a pool
func (qs queryServer) OrderBook(goCtx context.Context, req *types.QueryOrderBookRequest) (*types.QueryOrderBookResponse, error) {
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	limit := int(req.Limit)
	if limit == 0 {
		limit = DefaultOrderBookLimit
	}
	if limit > MaxOrderBookLimit {
		limit = MaxOrderBookLimit
	}

	// GetOrderBook now handles limits efficiently at the storage level
	buyOrders, sellOrders, err := qs.Keeper.GetOrderBook(goCtx, req.PoolId, limit)
	if err != nil {
		return nil, err
	}

	var protoBuyOrders, protoSellOrders []types.LimitOrder
	for _, order := range buyOrders {
		protoBuyOrders = append(protoBuyOrders, convertToProtoLimitOrder(order))
	}
	for _, order := range sellOrders {
		protoSellOrders = append(protoSellOrders, convertToProtoLimitOrder(order))
	}

	return &types.QueryOrderBookResponse{
		BuyOrders:  protoBuyOrders,
		SellOrders: protoSellOrders,
	}, nil
}

// convertToProtoLimitOrder converts the internal LimitOrder to the proto types.LimitOrder
func convertToProtoLimitOrder(order *LimitOrder) types.LimitOrder {
	orderType := types.OrderType_ORDER_TYPE_BUY
	if order.OrderType == OrderTypeSell {
		orderType = types.OrderType_ORDER_TYPE_SELL
	}

	var status types.OrderStatus
	switch order.Status {
	case OrderStatusOpen:
		status = types.OrderStatus_ORDER_STATUS_OPEN
	case OrderStatusPartial:
		status = types.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED
	case OrderStatusFilled:
		status = types.OrderStatus_ORDER_STATUS_FILLED
	case OrderStatusCancelled:
		status = types.OrderStatus_ORDER_STATUS_CANCELLED
	case OrderStatusExpired:
		status = types.OrderStatus_ORDER_STATUS_EXPIRED
	default:
		status = types.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}

	return types.LimitOrder{
		Id:              order.ID,
		Owner:           order.Owner,
		PoolId:          order.PoolID,
		OrderType:       orderType,
		TokenIn:         order.TokenIn,
		TokenOut:        order.TokenOut,
		AmountIn:        order.AmountIn,
		MinAmountOut:    order.MinAmountOut,
		LimitPrice:      order.LimitPrice,
		FilledAmount:    order.FilledAmount,
		ReceivedAmount:  order.ReceivedAmount,
		Status:          status,
		CreatedAt:       order.CreatedAt.Unix(),
		ExpiresAt:       order.ExpiresAt.Unix(),
		CreatedAtHeight: order.CreatedAtHeight,
	}
}
