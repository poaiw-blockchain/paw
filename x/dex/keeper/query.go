package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

var _ types.QueryServer = Keeper{}

// Params returns the module parameters
func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Pool queries a pool by ID
func (k Keeper) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.PoolId == 0 {
		return nil, status.Error(codes.InvalidArgument, "pool id cannot be zero")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	pool := k.GetPool(ctx, req.PoolId)

	if pool == nil {
		return nil, status.Errorf(codes.NotFound, "pool %d not found", req.PoolId)
	}

	return &types.QueryPoolResponse{
		Pool: *pool,
	}, nil
}

// Pools queries all pools with pagination
func (k Keeper) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get all pools
	allPools := k.GetAllPools(ctx)

	// Apply pagination
	var pools []types.Pool
	pageRes := &query.PageResponse{}

	if req.Pagination != nil {
		// Calculate pagination
		offset := req.Pagination.Offset
		limit := req.Pagination.Limit

		// Default limit if not specified
		if limit == 0 {
			limit = 100 // Default page size
		}

		// Validate offset
		if offset >= uint64(len(allPools)) {
			offset = 0
		}

		// Calculate end index
		end := offset + limit
		if end > uint64(len(allPools)) {
			end = uint64(len(allPools))
		}

		// Slice pools
		pools = allPools[offset:end]

		// Set next key if there are more results
		if end < uint64(len(allPools)) {
			pageRes.NextKey = []byte{byte(end)}
		}
		pageRes.Total = uint64(len(allPools))
	} else {
		// No pagination, return all pools
		pools = allPools
		pageRes.Total = uint64(len(allPools))
	}

	return &types.QueryPoolsResponse{
		Pools:      pools,
		Pagination: pageRes,
	}, nil
}

// PoolByTokens queries a pool by token pair
func (k Keeper) PoolByTokens(goCtx context.Context, req *types.QueryPoolByTokensRequest) (*types.QueryPoolByTokensResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.TokenA == "" || req.TokenB == "" {
		return nil, status.Error(codes.InvalidArgument, "token denoms cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	pool := k.GetPoolByTokens(ctx, req.TokenA, req.TokenB)

	if pool == nil {
		return nil, status.Errorf(codes.NotFound, "pool not found for tokens %s/%s", req.TokenA, req.TokenB)
	}

	return &types.QueryPoolByTokensResponse{
		Pool: *pool,
	}, nil
}

// Liquidity queries a user's liquidity in a pool
func (k Keeper) Liquidity(goCtx context.Context, req *types.QueryLiquidityRequest) (*types.QueryLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.PoolId == 0 {
		return nil, status.Error(codes.InvalidArgument, "pool id cannot be zero")
	}

	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider address cannot be empty")
	}

	// Validate provider address
	if _, err := sdk.AccAddressFromBech32(req.Provider); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid provider address: %s", err)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get pool to verify it exists
	pool := k.GetPool(ctx, req.PoolId)
	if pool == nil {
		return nil, status.Errorf(codes.NotFound, "pool %d not found", req.PoolId)
	}

	// Get user's liquidity shares
	shares := k.GetLiquidity(ctx, req.PoolId, req.Provider)

	return &types.QueryLiquidityResponse{
		Shares: shares,
	}, nil
}

// SimulateSwap simulates a swap without executing it
func (k Keeper) SimulateSwap(goCtx context.Context, req *types.QuerySimulateSwapRequest) (*types.QuerySimulateSwapResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.PoolId == 0 {
		return nil, status.Error(codes.InvalidArgument, "pool id cannot be zero")
	}

	if req.TokenIn == "" || req.TokenOut == "" {
		return nil, status.Error(codes.InvalidArgument, "token denoms cannot be empty")
	}

	if req.AmountIn.IsNil() || !req.AmountIn.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount in must be positive")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get pool
	pool := k.GetPool(ctx, req.PoolId)
	if pool == nil {
		return nil, status.Errorf(codes.NotFound, "pool %d not found", req.PoolId)
	}

	// Verify tokens match pool
	if (req.TokenIn != pool.TokenA && req.TokenIn != pool.TokenB) ||
		(req.TokenOut != pool.TokenA && req.TokenOut != pool.TokenB) {
		return nil, status.Error(codes.InvalidArgument, "tokens do not match pool")
	}

	if req.TokenIn == req.TokenOut {
		return nil, status.Error(codes.InvalidArgument, "cannot swap token for itself")
	}

	// Get reserves based on token direction
	var reserveIn, reserveOut math.Int
	if req.TokenIn == pool.TokenA {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	}

	// Get params for fee calculation
	params := k.GetParams(ctx)

	// Calculate swap fee
	swapFee := params.SwapFee.Add(params.LpFee).Add(params.ProtocolFee)

	// Apply fee: amountInAfterFee = amountIn * (1 - swapFee)
	feeAmount := swapFee.MulInt(req.AmountIn).TruncateInt()
	amountInAfterFee := req.AmountIn.Sub(feeAmount)
	_ = feeAmount // Mark as used for potential future use

	// Calculate output using constant product formula: x * y = k
	// amountOut = (reserveOut * amountInAfterFee) / (reserveIn + amountInAfterFee)
	numerator := reserveOut.Mul(amountInAfterFee)
	denominator := reserveIn.Add(amountInAfterFee)
	amountOut := numerator.Quo(denominator)

	return &types.QuerySimulateSwapResponse{
		AmountOut: amountOut,
	}, nil
}
