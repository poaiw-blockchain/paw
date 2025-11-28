package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/paw-chain/paw/x/compute/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Params returns the module parameters
func (qs queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := qs.Keeper.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

// Provider returns information about a specific provider
func (qs queryServer) Provider(goCtx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "provider address cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	providerAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid provider address: %s", err))
	}

	provider, err := qs.Keeper.GetProvider(ctx, providerAddr)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("provider not found: %s", err))
	}

	return &types.QueryProviderResponse{Provider: *provider}, nil
}

// Providers returns a list of all registered providers with pagination
func (qs queryServer) Providers(goCtx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	providerStore := prefix.NewStore(store, ProviderKeyPrefix)

	var providers []types.Provider
	pageRes, err := query.Paginate(providerStore, req.Pagination, func(key []byte, value []byte) error {
		var provider types.Provider
		if err := qs.Keeper.cdc.Unmarshal(value, &provider); err != nil {
			return err
		}
		providers = append(providers, provider)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryProvidersResponse{
		Providers:  providers,
		Pagination: pageRes,
	}, nil
}

// ActiveProviders returns a list of all active providers with pagination
func (qs queryServer) ActiveProviders(goCtx context.Context, req *types.QueryActiveProvidersRequest) (*types.QueryActiveProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	activeProviderStore := prefix.NewStore(store, ActiveProvidersPrefix)

	var providers []types.Provider
	pageRes, err := query.Paginate(activeProviderStore, req.Pagination, func(key []byte, value []byte) error {
		// The active provider index stores addresses, need to fetch full provider
		providerAddr := sdk.AccAddress(key)
		provider, err := qs.Keeper.GetProvider(ctx, providerAddr)
		if err != nil {
			return err
		}
		providers = append(providers, *provider)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryActiveProvidersResponse{
		Providers:  providers,
		Pagination: pageRes,
	}, nil
}

// Request returns information about a specific compute request
func (qs queryServer) Request(goCtx context.Context, req *types.QueryRequestRequest) (*types.QueryRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	request, err := qs.Keeper.GetRequest(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("request not found: %s", err))
	}

	return &types.QueryRequestResponse{Request: *request}, nil
}

// Requests returns a list of all compute requests with pagination
func (qs queryServer) Requests(goCtx context.Context, req *types.QueryRequestsRequest) (*types.QueryRequestsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	requestStore := prefix.NewStore(store, RequestKeyPrefix)

	var requests []types.Request
	pageRes, err := query.Paginate(requestStore, req.Pagination, func(key []byte, value []byte) error {
		var request types.Request
		if err := qs.Keeper.cdc.Unmarshal(value, &request); err != nil {
			return err
		}
		requests = append(requests, request)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsResponse{
		Requests:   requests,
		Pagination: pageRes,
	}, nil
}

// RequestsByRequester returns all requests submitted by a specific requester with pagination
func (qs queryServer) RequestsByRequester(goCtx context.Context, req *types.QueryRequestsByRequesterRequest) (*types.QueryRequestsByRequesterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Requester == "" {
		return nil, status.Error(codes.InvalidArgument, "requester address cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	requesterAddr, err := sdk.AccAddressFromBech32(req.Requester)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid requester address: %s", err))
	}

	store := qs.Keeper.getStore(ctx)
	requesterPrefix := append(RequestsByRequesterPrefix, requesterAddr.Bytes()...)
	requesterStore := prefix.NewStore(store, requesterPrefix)

	var requests []types.Request
	pageRes, err := query.Paginate(requesterStore, req.Pagination, func(key []byte, value []byte) error {
		// Extract request ID from key and fetch full request
		requestID := sdk.BigEndianToUint64(key)
		request, err := qs.Keeper.GetRequest(ctx, requestID)
		if err != nil {
			return err
		}
		requests = append(requests, *request)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsByRequesterResponse{
		Requests:   requests,
		Pagination: pageRes,
	}, nil
}

// RequestsByProvider returns all requests assigned to a specific provider
func (qs queryServer) RequestsByProvider(goCtx context.Context, req *types.QueryRequestsByProviderRequest) (*types.QueryRequestsByProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Provider == "" {
		return nil, status.Error(codes.InvalidArgument, "provider address cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	providerAddr, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid provider address: %s", err))
	}

	var requests []types.Request
	err = qs.Keeper.IterateRequestsByProvider(ctx, providerAddr, func(request types.Request) (bool, error) {
		requests = append(requests, request)
		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsByProviderResponse{
		Requests: requests,
	}, nil
}

// RequestsByStatus returns all requests with a specific status
func (qs queryServer) RequestsByStatus(goCtx context.Context, req *types.QueryRequestsByStatusRequest) (*types.QueryRequestsByStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var requests []types.Request
	err := qs.Keeper.IterateRequestsByStatus(ctx, req.Status, func(request types.Request) (bool, error) {
		requests = append(requests, request)
		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsByStatusResponse{
		Requests: requests,
	}, nil
}

// Result returns the result of a specific request
func (qs queryServer) Result(goCtx context.Context, req *types.QueryResultRequest) (*types.QueryResultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	result, err := qs.Keeper.GetResult(ctx, req.RequestId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("result not found: %s", err))
	}

	return &types.QueryResultResponse{Result: *result}, nil
}

// EstimateCost estimates the cost of a compute request
func (qs queryServer) EstimateCost(goCtx context.Context, req *types.QueryEstimateCostRequest) (*types.QueryEstimateCostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// If provider address is specified, use that provider
	if req.ProviderAddress != "" {
		providerAddr, err := sdk.AccAddressFromBech32(req.ProviderAddress)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid provider address: %s", err))
		}

		estimatedCost, costPerHour, err := qs.Keeper.EstimateCost(ctx, providerAddr, req.Specs)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		return &types.QueryEstimateCostResponse{
			EstimatedCost: estimatedCost,
			CostPerHour:   costPerHour,
		}, nil
	}

	// Otherwise, find a suitable provider and estimate cost
	provider, err := qs.Keeper.FindSuitableProvider(ctx, req.Specs, "")
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("no suitable provider found: %s", err))
	}

	estimatedCost, costPerHour, err := qs.Keeper.EstimateCost(ctx, provider, req.Specs)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryEstimateCostResponse{
		EstimatedCost: estimatedCost,
		CostPerHour:   costPerHour,
	}, nil
}

// Dispute queries a single dispute by ID
