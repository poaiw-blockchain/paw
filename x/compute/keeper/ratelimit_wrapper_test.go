package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/paw-chain/paw/x/compute/types"
)

type recordingQueryServer struct {
	types.UnimplementedQueryServer
	called []string
}

func (m *recordingQueryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	m.called = append(m.called, "Params")
	return &types.QueryParamsResponse{}, nil
}

func (m *recordingQueryServer) Provider(ctx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	m.called = append(m.called, "Provider")
	return &types.QueryProviderResponse{}, nil
}

func (m *recordingQueryServer) Providers(ctx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	m.called = append(m.called, "Providers")
	return &types.QueryProvidersResponse{}, nil
}

func (m *recordingQueryServer) ActiveProviders(ctx context.Context, req *types.QueryActiveProvidersRequest) (*types.QueryActiveProvidersResponse, error) {
	m.called = append(m.called, "ActiveProviders")
	return &types.QueryActiveProvidersResponse{}, nil
}

func (m *recordingQueryServer) Request(ctx context.Context, req *types.QueryRequestRequest) (*types.QueryRequestResponse, error) {
	m.called = append(m.called, "Request")
	return &types.QueryRequestResponse{}, nil
}

func (m *recordingQueryServer) Requests(ctx context.Context, req *types.QueryRequestsRequest) (*types.QueryRequestsResponse, error) {
	m.called = append(m.called, "Requests")
	return &types.QueryRequestsResponse{}, nil
}

func (m *recordingQueryServer) RequestsByRequester(ctx context.Context, req *types.QueryRequestsByRequesterRequest) (*types.QueryRequestsByRequesterResponse, error) {
	m.called = append(m.called, "RequestsByRequester")
	return &types.QueryRequestsByRequesterResponse{}, nil
}

func (m *recordingQueryServer) RequestsByProvider(ctx context.Context, req *types.QueryRequestsByProviderRequest) (*types.QueryRequestsByProviderResponse, error) {
	m.called = append(m.called, "RequestsByProvider")
	return &types.QueryRequestsByProviderResponse{}, nil
}

func (m *recordingQueryServer) RequestsByStatus(ctx context.Context, req *types.QueryRequestsByStatusRequest) (*types.QueryRequestsByStatusResponse, error) {
	m.called = append(m.called, "RequestsByStatus")
	return &types.QueryRequestsByStatusResponse{}, nil
}

func (m *recordingQueryServer) Result(ctx context.Context, req *types.QueryResultRequest) (*types.QueryResultResponse, error) {
	m.called = append(m.called, "Result")
	return &types.QueryResultResponse{}, nil
}

func (m *recordingQueryServer) EstimateCost(ctx context.Context, req *types.QueryEstimateCostRequest) (*types.QueryEstimateCostResponse, error) {
	m.called = append(m.called, "EstimateCost")
	return &types.QueryEstimateCostResponse{}, nil
}

func TestRateLimitedQueryServer_DelegatesWhenAllowed(t *testing.T) {
	base := &recordingQueryServer{}
	limiter := NewRateLimiter(100, 16)
	server := NewRateLimitedQueryServer(base, limiter)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-client-id", "rlqs-allowed"))

	testCases := []struct {
		name   string
		invoke func(context.Context) error
	}{
		{"Params", func(c context.Context) error { _, err := server.Params(c, &types.QueryParamsRequest{}); return err }},
		{"Provider", func(c context.Context) error { _, err := server.Provider(c, &types.QueryProviderRequest{}); return err }},
		{"Providers", func(c context.Context) error {
			_, err := server.Providers(c, &types.QueryProvidersRequest{})
			return err
		}},
		{"ActiveProviders", func(c context.Context) error {
			_, err := server.ActiveProviders(c, &types.QueryActiveProvidersRequest{})
			return err
		}},
		{"Request", func(c context.Context) error { _, err := server.Request(c, &types.QueryRequestRequest{}); return err }},
		{"Requests", func(c context.Context) error { _, err := server.Requests(c, &types.QueryRequestsRequest{}); return err }},
		{"RequestsByRequester", func(c context.Context) error {
			_, err := server.RequestsByRequester(c, &types.QueryRequestsByRequesterRequest{})
			return err
		}},
		{"RequestsByProvider", func(c context.Context) error {
			_, err := server.RequestsByProvider(c, &types.QueryRequestsByProviderRequest{})
			return err
		}},
		{"RequestsByStatus", func(c context.Context) error {
			_, err := server.RequestsByStatus(c, &types.QueryRequestsByStatusRequest{})
			return err
		}},
		{"Result", func(c context.Context) error { _, err := server.Result(c, &types.QueryResultRequest{}); return err }},
		{"EstimateCost", func(c context.Context) error {
			_, err := server.EstimateCost(c, &types.QueryEstimateCostRequest{})
			return err
		}},
	}

	for _, tc := range testCases {
		require.NoError(t, tc.invoke(ctx), tc.name)
	}

	require.Len(t, base.called, len(testCases))
	require.ElementsMatch(t, []string{
		"Params", "Provider", "Providers", "ActiveProviders",
		"Request", "Requests", "RequestsByRequester", "RequestsByProvider",
		"RequestsByStatus", "Result", "EstimateCost",
	}, base.called)
}

func TestRateLimitedQueryServer_BlocksWhenRateExceeded(t *testing.T) {
	base := &recordingQueryServer{}
	limiter := NewRateLimiter(0, 0)
	server := NewRateLimitedQueryServer(base, limiter)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-client-id", "rlqs-blocked"))

	_, err := server.Params(ctx, &types.QueryParamsRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate limit exceeded")
	require.Empty(t, base.called)
}
