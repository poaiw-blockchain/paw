package types

import (
	"context"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
)

// QueryClient is the client API for Query service.
type QueryClient interface {
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	Provider(ctx context.Context, in *QueryProviderRequest, opts ...grpc.CallOption) (*QueryProviderResponse, error)
	Providers(ctx context.Context, in *QueryProvidersRequest, opts ...grpc.CallOption) (*QueryProvidersResponse, error)
	ActiveProviders(ctx context.Context, in *QueryActiveProvidersRequest, opts ...grpc.CallOption) (*QueryActiveProvidersResponse, error)
	Request(ctx context.Context, in *QueryRequestRequest, opts ...grpc.CallOption) (*QueryRequestResponse, error)
	Requests(ctx context.Context, in *QueryRequestsRequest, opts ...grpc.CallOption) (*QueryRequestsResponse, error)
	RequestsByRequester(ctx context.Context, in *QueryRequestsByRequesterRequest, opts ...grpc.CallOption) (*QueryRequestsByRequesterResponse, error)
	RequestsByProvider(ctx context.Context, in *QueryRequestsByProviderRequest, opts ...grpc.CallOption) (*QueryRequestsByProviderResponse, error)
	RequestsByStatus(ctx context.Context, in *QueryRequestsByStatusRequest, opts ...grpc.CallOption) (*QueryRequestsByStatusResponse, error)
	Result(ctx context.Context, in *QueryResultRequest, opts ...grpc.CallOption) (*QueryResultResponse, error)
	EstimateCost(ctx context.Context, in *QueryEstimateCostRequest, opts ...grpc.CallOption) (*QueryEstimateCostResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Provider(ctx context.Context, in *QueryProviderRequest, opts ...grpc.CallOption) (*QueryProviderResponse, error) {
	out := new(QueryProviderResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Provider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Providers(ctx context.Context, in *QueryProvidersRequest, opts ...grpc.CallOption) (*QueryProvidersResponse, error) {
	out := new(QueryProvidersResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Providers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ActiveProviders(ctx context.Context, in *QueryActiveProvidersRequest, opts ...grpc.CallOption) (*QueryActiveProvidersResponse, error) {
	out := new(QueryActiveProvidersResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/ActiveProviders", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Request(ctx context.Context, in *QueryRequestRequest, opts ...grpc.CallOption) (*QueryRequestResponse, error) {
	out := new(QueryRequestResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Request", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Requests(ctx context.Context, in *QueryRequestsRequest, opts ...grpc.CallOption) (*QueryRequestsResponse, error) {
	out := new(QueryRequestsResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Requests", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) RequestsByRequester(ctx context.Context, in *QueryRequestsByRequesterRequest, opts ...grpc.CallOption) (*QueryRequestsByRequesterResponse, error) {
	out := new(QueryRequestsByRequesterResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/RequestsByRequester", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) RequestsByProvider(ctx context.Context, in *QueryRequestsByProviderRequest, opts ...grpc.CallOption) (*QueryRequestsByProviderResponse, error) {
	out := new(QueryRequestsByProviderResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/RequestsByProvider", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) RequestsByStatus(ctx context.Context, in *QueryRequestsByStatusRequest, opts ...grpc.CallOption) (*QueryRequestsByStatusResponse, error) {
	out := new(QueryRequestsByStatusResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/RequestsByStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Result(ctx context.Context, in *QueryResultRequest, opts ...grpc.CallOption) (*QueryResultResponse, error) {
	out := new(QueryResultResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/Result", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) EstimateCost(ctx context.Context, in *QueryEstimateCostRequest, opts ...grpc.CallOption) (*QueryEstimateCostResponse, error) {
	out := new(QueryEstimateCostResponse)
	err := c.cc.Invoke(ctx, "/paw.compute.v1.Query/EstimateCost", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
