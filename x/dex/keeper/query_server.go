package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/paw-chain/paw/x/dex/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the dex QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Params returns the module parameters
func (qs queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
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
		return nil, ErrInvalidRequest
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
		return nil, ErrInvalidRequest
	}

	var pools []types.Pool
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
		return nil, ErrInvalidRequest
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
		return nil, ErrInvalidRequest
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
		return nil, ErrInvalidRequest
	}

	amountOut, err := qs.Keeper.SimulateSwap(goCtx, req.PoolId, req.TokenIn, req.TokenOut, req.AmountIn)
	if err != nil {
		return nil, err
	}

	return &types.QuerySimulateSwapResponse{
		AmountOut: amountOut,
	}, nil
}
