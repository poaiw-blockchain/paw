package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Helper function to fund an account with sufficient tokens for testing
func fundPaginationTestAccount(t *testing.T, k *keeper.Keeper, ctx sdk.Context, addr sdk.AccAddress) {
	// Mint enough coins to cover provider stake and request payments
	fundAmount := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1_000_000_000))
	bankKeeper := getBankKeeper(t, k)
	err := bankKeeper.MintCoins(ctx, types.ModuleName, fundAmount)
	require.NoError(t, err)
	err = bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, fundAmount)
	require.NoError(t, err)
}

// Helper function to register a test provider for pagination tests
func registerPaginationTestProvider(t *testing.T, k *keeper.Keeper, ctx sdk.Context, index int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("provider"))
	addr[19] = byte(index)
	providerAddr := sdk.AccAddress(addr)

	// Fund the provider account
	fundPaginationTestAccount(t, k, ctx, providerAddr)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	specs := types.ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "nvidia-t4",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	pricing := types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}

	err = k.RegisterProvider(ctx, providerAddr, "TestProvider", "http://example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	return providerAddr
}

// TestQueryRequestsByProviderPagination tests pagination for RequestsByProvider query
func TestQueryRequestsByProviderPagination(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	queryServer := keeper.NewQueryServerImpl(*k)

	// Create a test provider
	providerAddr := registerPaginationTestProvider(t, k, ctx, 1)

	specs := types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      50,
		TimeoutSeconds: 1800,
	}
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash", "-c", "echo hello"}
	envVars := map[string]string{"TEST": "value"}
	maxPayment := math.NewInt(10000000)

	// Create 175 requests for this provider
	for i := 0; i < 175; i++ {
		addr := make([]byte, 20)
		copy(addr, []byte("req"))
		addr[19] = byte(i)
		requesterAddr := sdk.AccAddress(addr)

		// Fund the requester account
		fundPaginationTestAccount(t, k, ctx, requesterAddr)

		_, err := k.SubmitRequest(ctx, requesterAddr, specs, containerImage, command, envVars, maxPayment, providerAddr.String())
		require.NoError(t, err)
	}

	// Test default pagination (100 items per page)
	resp, err := queryServer.RequestsByProvider(ctx, &types.QueryRequestsByProviderRequest{
		Provider: providerAddr.String(),
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Requests, 100)
	require.NotNil(t, resp.Pagination)
	require.NotNil(t, resp.Pagination.NextKey)

	// Verify all requests belong to the provider
	for _, req := range resp.Requests {
		require.Equal(t, providerAddr.String(), req.Provider)
	}

	// Test second page
	resp2, err := queryServer.RequestsByProvider(ctx, &types.QueryRequestsByProviderRequest{
		Provider: providerAddr.String(),
		Pagination: &query.PageRequest{
			Key:   resp.Pagination.NextKey,
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Requests, 75) // Remaining 75

	// Verify second page also belongs to provider
	for _, req := range resp2.Requests {
		require.Equal(t, providerAddr.String(), req.Provider)
	}

	// Test custom page size
	resp3, err := queryServer.RequestsByProvider(ctx, &types.QueryRequestsByProviderRequest{
		Provider: providerAddr.String(),
		Pagination: &query.PageRequest{
			Limit: 50,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp3.Requests, 50)
	require.NotNil(t, resp3.Pagination.NextKey)
}

// TestQueryRequestsByStatusPagination tests pagination for RequestsByStatus query
func TestQueryRequestsByStatusPagination(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	queryServer := keeper.NewQueryServerImpl(*k)

	// Create a test provider
	providerAddr := registerPaginationTestProvider(t, k, ctx, 1)

	specs := types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      50,
		TimeoutSeconds: 1800,
	}
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash", "-c", "echo hello"}
	envVars := map[string]string{"TEST": "value"}
	maxPayment := math.NewInt(10000000)

	// Create 130 requests (they will be ASSIGNED status by default)
	for i := 0; i < 130; i++ {
		addr := make([]byte, 20)
		copy(addr, []byte("req"))
		addr[19] = byte(i)
		requesterAddr := sdk.AccAddress(addr)

		// Fund the requester account
		fundPaginationTestAccount(t, k, ctx, requesterAddr)

		_, err := k.SubmitRequest(ctx, requesterAddr, specs, containerImage, command, envVars, maxPayment, providerAddr.String())
		require.NoError(t, err)
	}

	// Test pagination for ASSIGNED status
	resp, err := queryServer.RequestsByStatus(ctx, &types.QueryRequestsByStatusRequest{
		Status: types.REQUEST_STATUS_ASSIGNED,
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Requests, 100)
	require.NotNil(t, resp.Pagination)
	require.NotNil(t, resp.Pagination.NextKey)

	// Verify all requests have ASSIGNED status
	for _, req := range resp.Requests {
		require.Equal(t, types.REQUEST_STATUS_ASSIGNED, req.Status)
	}

	// Test second page
	resp2, err := queryServer.RequestsByStatus(ctx, &types.QueryRequestsByStatusRequest{
		Status: types.REQUEST_STATUS_ASSIGNED,
		Pagination: &query.PageRequest{
			Key:   resp.Pagination.NextKey,
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Requests, 30) // Remaining 30

	// Verify second page has correct status
	for _, req := range resp2.Requests {
		require.Equal(t, types.REQUEST_STATUS_ASSIGNED, req.Status)
	}

	// Test custom page size
	resp3, err := queryServer.RequestsByStatus(ctx, &types.QueryRequestsByStatusRequest{
		Status: types.REQUEST_STATUS_ASSIGNED,
		Pagination: &query.PageRequest{
			Limit: 50,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp3.Requests, 50)

	// Verify all requests have ASSIGNED status
	for _, req := range resp3.Requests {
		require.Equal(t, types.REQUEST_STATUS_ASSIGNED, req.Status)
	}
}

// TestQueryPaginationEmpty tests pagination with no results
func TestQueryPaginationEmpty(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	queryServer := keeper.NewQueryServerImpl(*k)

	// Create a test provider but no requests
	providerAddr := registerPaginationTestProvider(t, k, ctx, 1)

	// Query requests for provider with no requests
	resp, err := queryServer.RequestsByProvider(ctx, &types.QueryRequestsByProviderRequest{
		Provider: providerAddr.String(),
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Requests)
	require.NotNil(t, resp.Pagination)
	require.Nil(t, resp.Pagination.NextKey)

	// Query by status when no requests exist
	resp2, err := queryServer.RequestsByStatus(ctx, &types.QueryRequestsByStatusRequest{
		Status: types.REQUEST_STATUS_PENDING,
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	})
	require.NoError(t, err)
	require.Empty(t, resp2.Requests)
	require.NotNil(t, resp2.Pagination)
	require.Nil(t, resp2.Pagination.NextKey)
}
