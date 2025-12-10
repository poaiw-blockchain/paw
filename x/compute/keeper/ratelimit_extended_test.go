package keeper

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

type mockQueryServer struct {
	types.UnimplementedQueryServer
	paramsCalls int
}

func (m *mockQueryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	m.paramsCalls++
	return &types.QueryParamsResponse{}, nil
}

func TestGetClientID(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-client-id", "client123"))
	require.Equal(t, "client123", getClientID(ctx))
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	require.True(t, rl.Allow("client"))
	require.False(t, rl.Allow("client"))
}

func TestRateLimitedQueryServer_CheckRateLimit(t *testing.T) {
	base := &mockQueryServer{}
	rlqs := NewRateLimitedQueryServer(base, NewRateLimiter(1, 1))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-client-id", "clientA"))

	_, err := rlqs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	_, err = rlqs.Params(ctx, &types.QueryParamsRequest{})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.ResourceExhausted, st.Code())
	require.Equal(t, 1, base.paramsCalls)
}
