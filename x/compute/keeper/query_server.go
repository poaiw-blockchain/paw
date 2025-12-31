package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	storeprefix "cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/paw-chain/paw/x/compute/types"
)

var _ types.QueryServer = queryServer{}

const (
	defaultPaginationLimit = 100
	maxPaginationLimit     = 1000
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// sanitizePagination enforces default and max limits to prevent unbounded queries.
func sanitizePagination(p *query.PageRequest) *query.PageRequest {
	if p == nil {
		return &query.PageRequest{Limit: defaultPaginationLimit}
	}

	if p.Limit == 0 {
		p.Limit = defaultPaginationLimit
	}

	if p.Limit > maxPaginationLimit {
		p.Limit = maxPaginationLimit
	}

	return p
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

	return &types.QueryProviderResponse{Provider: provider}, nil
}

// Providers returns a list of all registered providers with pagination
func (qs queryServer) Providers(goCtx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	providerStore := storeprefix.NewStore(store, ProviderKeyPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	providers := make([]types.Provider, 0, sanitized.Limit)
	pageRes, err := query.Paginate(providerStore, sanitized, func(key []byte, value []byte) error {
		var provider types.Provider
		if err := qs.Keeper.cdc.Unmarshal(value, &provider); err != nil {
			return fmt.Errorf("unmarshal provider: %w", err)
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
	activeProviderStore := storeprefix.NewStore(store, ActiveProvidersPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	providers := make([]types.Provider, 0, sanitized.Limit)
	pageRes, err := query.Paginate(activeProviderStore, sanitized, func(key []byte, value []byte) error {
		// PERF-12: Try to unmarshal full provider from cached value first
		// Falls back to GetProvider if value is just an address (legacy format)
		var provider types.Provider
		if len(value) > 32 { // Full provider data is larger than just an address
			if err := qs.Keeper.cdc.Unmarshal(value, &provider); err == nil {
				providers = append(providers, provider)
				return nil
			}
		}

		// Fallback: value is just address bytes (legacy format), fetch full provider
		providerAddr := sdk.AccAddress(key)
		providerData, err := qs.Keeper.GetProvider(ctx, providerAddr)
		if err != nil {
			return fmt.Errorf("get provider %s: %w", providerAddr, err)
		}
		providers = append(providers, *providerData)
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

	return &types.QueryRequestResponse{Request: request}, nil
}

// Requests returns a list of all compute requests with pagination
func (qs queryServer) Requests(goCtx context.Context, req *types.QueryRequestsRequest) (*types.QueryRequestsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	requestStore := storeprefix.NewStore(store, RequestKeyPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	requests := make([]types.Request, 0, sanitized.Limit)
	pageRes, err := query.Paginate(requestStore, sanitized, func(key []byte, value []byte) error {
		var request types.Request
		if err := qs.Keeper.cdc.Unmarshal(value, &request); err != nil {
			return fmt.Errorf("unmarshal request: %w", err)
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
	requesterStore := storeprefix.NewStore(store, requesterPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	requests := make([]types.Request, 0, sanitized.Limit)
	pageRes, err := query.Paginate(requesterStore, sanitized, func(key []byte, value []byte) error {
		// Extract request ID from key and fetch full request
		requestID := sdk.BigEndianToUint64(key)
		request, err := qs.Keeper.GetRequest(ctx, requestID)
		if err != nil {
			return fmt.Errorf("get request %d: %w", requestID, err)
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

// RequestsByProvider returns all requests assigned to a specific provider with pagination
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

	store := qs.Keeper.getStore(ctx)
	providerPrefix := append(RequestsByProviderPrefix, providerAddr.Bytes()...)
	providerStore := storeprefix.NewStore(store, providerPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	requests := make([]types.Request, 0, sanitized.Limit)
	pageRes, err := query.Paginate(providerStore, sanitized, func(key []byte, value []byte) error {
		// Extract request ID from key and fetch full request
		requestID := sdk.BigEndianToUint64(key)
		request, err := qs.Keeper.GetRequest(ctx, requestID)
		if err != nil {
			return fmt.Errorf("get request %d: %w", requestID, err)
		}
		requests = append(requests, *request)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsByProviderResponse{
		Requests:   requests,
		Pagination: pageRes,
	}, nil
}

// RequestsByStatus returns all requests with a specific status with pagination
func (qs queryServer) RequestsByStatus(goCtx context.Context, req *types.QueryRequestsByStatusRequest) (*types.QueryRequestsByStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := qs.Keeper.getStore(ctx)
	statusBz := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBz, types.SaturateInt64ToUint32(int64(req.Status)))
	statusPrefix := append(RequestsByStatusPrefix, statusBz...)
	statusStore := storeprefix.NewStore(store, statusPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	requests := make([]types.Request, 0, sanitized.Limit)
	pageRes, err := query.Paginate(statusStore, sanitized, func(key []byte, value []byte) error {
		// Extract request ID from key and fetch full request
		requestID := sdk.BigEndianToUint64(key)
		request, err := qs.Keeper.GetRequest(ctx, requestID)
		if err != nil {
			return fmt.Errorf("get request %d: %w", requestID, err)
		}
		requests = append(requests, *request)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRequestsByStatusResponse{
		Requests:   requests,
		Pagination: pageRes,
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

	return &types.QueryResultResponse{Result: result}, nil
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
func (qs queryServer) Dispute(goCtx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, err := qs.Keeper.getDispute(ctx, req.DisputeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryDisputeResponse{Dispute: dispute}, nil
}

// Disputes queries all disputes with pagination
func (qs queryServer) Disputes(goCtx context.Context, req *types.QueryDisputesRequest) (*types.QueryDisputesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	view := storeprefix.NewStore(store, DisputeKeyPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	disputes := make([]types.Dispute, 0, sanitized.Limit)
	pageRes, err := query.Paginate(view, sanitized, func(key []byte, value []byte) error {
		var dispute types.Dispute
		if err := qs.Keeper.cdc.Unmarshal(value, &dispute); err != nil {
			return fmt.Errorf("unmarshal dispute: %w", err)
		}
		disputes = append(disputes, dispute)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesResponse{Disputes: disputes, Pagination: pageRes}, nil
}

// DisputesByRequest queries disputes associated with a request
func (qs queryServer) DisputesByRequest(goCtx context.Context, req *types.QueryDisputesByRequestRequest) (*types.QueryDisputesByRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	prefixKey := DisputeByRequestKey(req.RequestId, 0)[:len(DisputesByRequestPrefix)+8]
	view := storeprefix.NewStore(store, prefixKey)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	disputes := make([]types.Dispute, 0, sanitized.Limit)
	pageRes, err := query.Paginate(view, sanitized, func(key []byte, value []byte) error {
		if len(key) < 8 {
			return nil
		}
		disputeID := binary.BigEndian.Uint64(key[len(key)-8:])
		dispute, err := qs.Keeper.getDispute(ctx, disputeID)
		if err != nil {
			return fmt.Errorf("get dispute %d: %w", disputeID, err)
		}
		disputes = append(disputes, *dispute)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesByRequestResponse{Disputes: disputes, Pagination: pageRes}, nil
}

// DisputesByStatus queries disputes filtered by status
func (qs queryServer) DisputesByStatus(goCtx context.Context, req *types.QueryDisputesByStatusRequest) (*types.QueryDisputesByStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	statusPrefix := DisputeByStatusKey(types.SaturateInt64ToUint32(int64(req.Status)), 0)[:len(DisputesByStatusPrefix)+4]
	view := storeprefix.NewStore(store, statusPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	disputes := make([]types.Dispute, 0, sanitized.Limit)
	pageRes, err := query.Paginate(view, sanitized, func(key []byte, value []byte) error {
		if len(key) < 8 {
			return nil
		}
		disputeID := binary.BigEndian.Uint64(key[len(key)-8:])
		dispute, err := qs.Keeper.getDispute(ctx, disputeID)
		if err != nil {
			return fmt.Errorf("get dispute %d: %w", disputeID, err)
		}
		disputes = append(disputes, *dispute)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDisputesByStatusResponse{Disputes: disputes, Pagination: pageRes}, nil
}

// Evidence queries all evidence for a dispute
func (qs queryServer) Evidence(goCtx context.Context, req *types.QueryEvidenceRequest) (*types.QueryEvidenceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	items, pageRes, err := qs.Keeper.ListEvidence(ctx, req.DisputeId, sanitizePagination(req.Pagination))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryEvidenceResponse{Evidence: items, Pagination: pageRes}, nil
}

// SlashRecord queries a single slash record
func (qs queryServer) SlashRecord(goCtx context.Context, req *types.QuerySlashRecordRequest) (*types.QuerySlashRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	record, err := qs.Keeper.getSlashRecord(ctx, req.SlashId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &types.QuerySlashRecordResponse{SlashRecord: record}, nil
}

// SlashRecords queries all slash records
func (qs queryServer) SlashRecords(goCtx context.Context, req *types.QuerySlashRecordsRequest) (*types.QuerySlashRecordsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	records, pageRes, err := qs.Keeper.listSlashRecords(ctx, sdk.AccAddress{}, sanitizePagination(req.Pagination))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySlashRecordsResponse{SlashRecords: records, Pagination: pageRes}, nil
}

// SlashRecordsByProvider queries slash records for a provider
func (qs queryServer) SlashRecordsByProvider(goCtx context.Context, req *types.QuerySlashRecordsByProviderRequest) (*types.QuerySlashRecordsByProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	provider, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid provider address")
	}
	records, pageRes, err := qs.Keeper.listSlashRecords(ctx, provider, sanitizePagination(req.Pagination))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySlashRecordsByProviderResponse{SlashRecords: records, Pagination: pageRes}, nil
}

// Appeal queries a single appeal
func (qs queryServer) Appeal(goCtx context.Context, req *types.QueryAppealRequest) (*types.QueryAppealResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	appeal, err := qs.Keeper.getAppeal(ctx, req.AppealId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &types.QueryAppealResponse{Appeal: appeal}, nil
}

// Appeals queries all appeals
func (qs queryServer) Appeals(goCtx context.Context, req *types.QueryAppealsRequest) (*types.QueryAppealsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	view := storeprefix.NewStore(store, AppealKeyPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	appeals := make([]types.Appeal, 0, sanitized.Limit)
	pageRes, err := query.Paginate(view, sanitized, func(key []byte, value []byte) error {
		var appeal types.Appeal
		if err := qs.Keeper.cdc.Unmarshal(value, &appeal); err != nil {
			return fmt.Errorf("unmarshal appeal: %w", err)
		}
		appeals = append(appeals, appeal)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAppealsResponse{Appeals: appeals, Pagination: pageRes}, nil
}

// AppealsByStatus filters appeals by status
func (qs queryServer) AppealsByStatus(goCtx context.Context, req *types.QueryAppealsByStatusRequest) (*types.QueryAppealsByStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	statusPrefix := AppealByStatusKey(types.SaturateInt64ToUint32(int64(req.Status)), 0)[:len(AppealsByStatusPrefix)+4]
	view := storeprefix.NewStore(store, statusPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	appeals := make([]types.Appeal, 0, sanitized.Limit)
	pageRes, err := query.Paginate(view, sanitized, func(key []byte, value []byte) error {
		if len(key) < 8 {
			return nil
		}
		appealID := binary.BigEndian.Uint64(key[len(key)-8:])
		appeal, err := qs.Keeper.getAppeal(ctx, appealID)
		if err != nil {
			return fmt.Errorf("get appeal %d: %w", appealID, err)
		}
		appeals = append(appeals, *appeal)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAppealsByStatusResponse{Appeals: appeals, Pagination: pageRes}, nil
}

// GovernanceParams queries the dispute/appeal governance parameters
func (qs queryServer) GovernanceParams(goCtx context.Context, req *types.QueryGovernanceParamsRequest) (*types.QueryGovernanceParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := qs.Keeper.GetGovernanceParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryGovernanceParamsResponse{Params: params}, nil
}

// CatastrophicFailures queries all catastrophic failure records with optional filtering
func (qs queryServer) CatastrophicFailures(goCtx context.Context, req *types.QueryCatastrophicFailuresRequest) (*types.QueryCatastrophicFailuresResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := qs.Keeper.getStore(ctx)
	failureStore := storeprefix.NewStore(store, CatastrophicFailureKeyPrefix)

	// P3-PERF-3: Pre-size with pagination limit capacity
	sanitized := sanitizePagination(req.Pagination)
	failures := make([]types.CatastrophicFailure, 0, sanitized.Limit)
	pageRes, err := query.Paginate(failureStore, sanitized, func(key []byte, value []byte) error {
		var failure types.CatastrophicFailure
		if err := qs.Keeper.cdc.Unmarshal(value, &failure); err != nil {
			return fmt.Errorf("unmarshal catastrophic failure: %w", err)
		}

		// Filter by resolved status if requested
		if req.OnlyUnresolved && failure.Resolved {
			return nil // Skip resolved failures if filter is active
		}

		failures = append(failures, failure)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCatastrophicFailuresResponse{Failures: failures, Pagination: pageRes}, nil
}

// CatastrophicFailure queries a single catastrophic failure record by ID
func (qs queryServer) CatastrophicFailure(goCtx context.Context, req *types.QueryCatastrophicFailureRequest) (*types.QueryCatastrophicFailureResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.FailureId == 0 {
		return nil, status.Error(codes.InvalidArgument, "failure ID must be greater than 0")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	failure, err := qs.Keeper.GetCatastrophicFailure(ctx, req.FailureId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("catastrophic failure not found: %s", err))
	}

	return &types.QueryCatastrophicFailureResponse{Failure: failure}, nil
}

// SimulateRequest simulates a compute request without executing it
// AGENT-2: Enables agents to preview gas/cost/providers before submitting
func (qs queryServer) SimulateRequest(goCtx context.Context, req *types.QuerySimulateRequestRequest) (*types.QuerySimulateRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Initialize response
	resp := &types.QuerySimulateRequestResponse{
		ValidationErrors:    make([]string, 0),
		AvailableProviders:  make([]string, 0),
		WillQueue:           false,
		EstimatedWaitTimeSeconds: 0,
	}

	// Validate specs
	if req.Specs.CpuCores == 0 {
		resp.ValidationErrors = append(resp.ValidationErrors, "cpu_cores must be greater than 0")
	}
	if req.Specs.MemoryMb == 0 {
		resp.ValidationErrors = append(resp.ValidationErrors, "memory_mb must be greater than 0")
	}
	if req.ContainerImage == "" {
		resp.ValidationErrors = append(resp.ValidationErrors, "container_image is required")
	}

	// If validation errors, return early
	if len(resp.ValidationErrors) > 0 {
		return resp, nil
	}

	// Find available providers matching specs
	store := qs.Keeper.getStore(ctx)
	activeProviderStore := storeprefix.NewStore(store, ActiveProvidersPrefix)

	iterator := activeProviderStore.Iterator(nil, nil)
	defer iterator.Close()

	var matchingProviders []string
	var bestProvider *types.Provider

	for ; iterator.Valid(); iterator.Next() {
		providerAddr := sdk.AccAddress(iterator.Key())
		provider, err := qs.Keeper.GetProvider(ctx, providerAddr)
		if err != nil {
			continue
		}

		// Check if provider can handle the specs
		if provider.AvailableSpecs.CpuCores >= req.Specs.CpuCores &&
			provider.AvailableSpecs.MemoryMb >= req.Specs.MemoryMb &&
			provider.AvailableSpecs.StorageGb >= req.Specs.StorageGb {

			matchingProviders = append(matchingProviders, provider.Address)

			// Track best provider (first match or preferred)
			if bestProvider == nil {
				bestProvider = provider
			}
			if req.PreferredProvider != "" && provider.Address == req.PreferredProvider {
				bestProvider = provider
			}
		}
	}

	resp.AvailableProviders = matchingProviders

	if len(matchingProviders) == 0 {
		resp.ValidationErrors = append(resp.ValidationErrors, "no providers available with matching specs")
		return resp, nil
	}

	// Set matching provider
	if bestProvider != nil {
		resp.MatchingProvider = bestProvider.Address

		// Estimate cost using the matching provider
		// FIXED CODE-1.1: Replace MustAccAddressFromBech32 with error-handling variant
		providerAddr, addrErr := sdk.AccAddressFromBech32(bestProvider.Address)
		if addrErr == nil {
			estimatedCost, _, err := qs.Keeper.EstimateCost(ctx, providerAddr, req.Specs)
			if err == nil {
				resp.EstimatedCost = estimatedCost
			}
		}
	}

	// Estimate gas (base gas + compute-specific overhead)
	// Base tx gas is around 100k, compute request adds overhead based on specs
	baseGas := uint64(100000)
	cpuGas := req.Specs.CpuCores * 10              // ~10 gas per cpu core unit
	memGas := req.Specs.MemoryMb * 5               // ~5 gas per MB
	storageGas := req.Specs.StorageGb * 1000       // ~1000 gas per GB
	resp.EstimatedGas = baseGas + cpuGas + memGas + storageGas

	// Estimate queue wait time based on pending requests
	pendingRequests := qs.countPendingRequestsForProvider(ctx, resp.MatchingProvider)
	if pendingRequests > 0 {
		resp.WillQueue = true
		// Estimate ~60 seconds per pending request
		resp.EstimatedWaitTimeSeconds = pendingRequests * 60
	}

	return resp, nil
}

// countPendingRequestsForProvider counts pending requests for a provider
func (qs queryServer) countPendingRequestsForProvider(ctx sdk.Context, providerAddr string) uint64 {
	if providerAddr == "" {
		return 0
	}

	addr, err := sdk.AccAddressFromBech32(providerAddr)
	if err != nil {
		return 0
	}

	store := qs.Keeper.getStore(ctx)
	providerPrefix := append(RequestsByProviderPrefix, addr.Bytes()...)
	providerStore := storeprefix.NewStore(store, providerPrefix)

	var count uint64
	iterator := providerStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		requestID := sdk.BigEndianToUint64(iterator.Key())
		request, err := qs.Keeper.GetRequest(ctx, requestID)
		if err != nil {
			continue
		}
		if request.Status == types.REQUEST_STATUS_PENDING ||
			request.Status == types.REQUEST_STATUS_ASSIGNED {
			count++
		}
	}

	return count
}
