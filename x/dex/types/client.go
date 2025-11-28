package types

import (
	"context"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
)

// QueryClient is the client API for Query service.
type QueryClient interface {
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	Pool(ctx context.Context, in *QueryPoolRequest, opts ...grpc.CallOption) (*QueryPoolResponse, error)
	Pools(ctx context.Context, in *QueryPoolsRequest, opts ...grpc.CallOption) (*QueryPoolsResponse, error)
	PoolByTokens(ctx context.Context, in *QueryPoolByTokensRequest, opts ...grpc.CallOption) (*QueryPoolByTokensResponse, error)
	Liquidity(ctx context.Context, in *QueryLiquidityRequest, opts ...grpc.CallOption) (*QueryLiquidityResponse, error)
	SimulateSwap(ctx context.Context, in *QuerySimulateSwapRequest, opts ...grpc.CallOption) (*QuerySimulateSwapResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Pool(ctx context.Context, in *QueryPoolRequest, opts ...grpc.CallOption) (*QueryPoolResponse, error) {
	out := new(QueryPoolResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/Pool", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Pools(ctx context.Context, in *QueryPoolsRequest, opts ...grpc.CallOption) (*QueryPoolsResponse, error) {
	out := new(QueryPoolsResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/Pools", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) PoolByTokens(ctx context.Context, in *QueryPoolByTokensRequest, opts ...grpc.CallOption) (*QueryPoolByTokensResponse, error) {
	out := new(QueryPoolByTokensResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/PoolByTokens", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Liquidity(ctx context.Context, in *QueryLiquidityRequest, opts ...grpc.CallOption) (*QueryLiquidityResponse, error) {
	out := new(QueryLiquidityResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/Liquidity", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) SimulateSwap(ctx context.Context, in *QuerySimulateSwapRequest, opts ...grpc.CallOption) (*QuerySimulateSwapResponse, error) {
	out := new(QuerySimulateSwapResponse)
	err := c.cc.Invoke(ctx, "/paw.dex.v1.Query/SimulateSwap", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
