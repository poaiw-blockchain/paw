package types

import (
	"context"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
)

// QueryClient is the client API for Query service.
type QueryClient interface {
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	Price(ctx context.Context, in *QueryPriceRequest, opts ...grpc.CallOption) (*QueryPriceResponse, error)
	Prices(ctx context.Context, in *QueryPricesRequest, opts ...grpc.CallOption) (*QueryPricesResponse, error)
	Validator(ctx context.Context, in *QueryValidatorRequest, opts ...grpc.CallOption) (*QueryValidatorResponse, error)
	Validators(ctx context.Context, in *QueryValidatorsRequest, opts ...grpc.CallOption) (*QueryValidatorsResponse, error)
	ValidatorPrice(ctx context.Context, in *QueryValidatorPriceRequest, opts ...grpc.CallOption) (*QueryValidatorPriceResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Price(ctx context.Context, in *QueryPriceRequest, opts ...grpc.CallOption) (*QueryPriceResponse, error) {
	out := new(QueryPriceResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/Price", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Prices(ctx context.Context, in *QueryPricesRequest, opts ...grpc.CallOption) (*QueryPricesResponse, error) {
	out := new(QueryPricesResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/Prices", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Validator(ctx context.Context, in *QueryValidatorRequest, opts ...grpc.CallOption) (*QueryValidatorResponse, error) {
	out := new(QueryValidatorResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/Validator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Validators(ctx context.Context, in *QueryValidatorsRequest, opts ...grpc.CallOption) (*QueryValidatorsResponse, error) {
	out := new(QueryValidatorsResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/Validators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ValidatorPrice(ctx context.Context, in *QueryValidatorPriceRequest, opts ...grpc.CallOption) (*QueryValidatorPriceResponse, error) {
	out := new(QueryValidatorPriceResponse)
	err := c.cc.Invoke(ctx, "/paw.oracle.v1.Query/ValidatorPrice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
