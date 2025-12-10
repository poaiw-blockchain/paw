package keeper

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestQueryServer_Params(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Params(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns params", func(t *testing.T) {
		resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Params)
	})
}

func TestQueryServer_Provider(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	// Register a provider first
	providerAddr := sdk.AccAddress([]byte("test_provider_addr__"))
	provider := types.Provider{
		Address: providerAddr.String(),
		Stake:   math.NewInt(10000),
		AvailableSpecs: types.ComputeSpec{
			CpuCores:  2,
			MemoryMb:  1024,
			StorageGb: 10,
		},
		Active:     true,
		Reputation: 100,
	}
	err := k.SetProvider(ctx, provider)
	require.NoError(t, err)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Provider(ctx, nil)
		require.Error(t, err)
	})

	t.Run("empty address returns error", func(t *testing.T) {
		_, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: ""})
		require.Error(t, err)
	})

	t.Run("invalid address returns error", func(t *testing.T) {
		_, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: "invalid"})
		require.Error(t, err)
	})

	t.Run("non-existent provider returns not found", func(t *testing.T) {
		nonExistent := sdk.AccAddress([]byte("non_existent_addr___"))
		_, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: nonExistent.String()})
		require.Error(t, err)
	})

	t.Run("existing provider returns provider", func(t *testing.T) {
		resp, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: providerAddr.String()})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, providerAddr.String(), resp.Provider.Address)
	})
}

func TestQueryServer_Providers(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	// Register multiple providers
	for i := 0; i < 5; i++ {
		addr := sdk.AccAddress([]byte("test_provider_addr" + string(rune('0'+i)) + "_"))
		provider := types.Provider{
			Address: addr.String(),
			Stake:   math.NewInt(10000),
			AvailableSpecs: types.ComputeSpec{
				CpuCores:  2,
				MemoryMb:  1024,
				StorageGb: 20,
			},
			Active:     true,
			Reputation: uint32(100 + i),
		}
		err := k.SetProvider(ctx, provider)
		require.NoError(t, err)
	}

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Providers(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns providers", func(t *testing.T) {
		resp, err := qs.Providers(ctx, &types.QueryProvidersRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.GreaterOrEqual(t, len(resp.Providers), 5)
	})

	t.Run("pagination works", func(t *testing.T) {
		resp, err := qs.Providers(ctx, &types.QueryProvidersRequest{
			Pagination: &query.PageRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.LessOrEqual(t, len(resp.Providers), 2)
	})
}

func TestQueryServer_ActiveProviders(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.ActiveProviders(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns active providers", func(t *testing.T) {
		resp, err := qs.ActiveProviders(ctx, &types.QueryActiveProvidersRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_Request(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Request(ctx, nil)
		require.Error(t, err)
	})

	t.Run("non-existent request returns not found", func(t *testing.T) {
		_, err := qs.Request(ctx, &types.QueryRequestRequest{Id: 99999})
		require.Error(t, err)
	})
}

func TestQueryServer_Requests(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Requests(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns requests list", func(t *testing.T) {
		resp, err := qs.Requests(ctx, &types.QueryRequestsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_RequestsByRequester(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.RequestsByRequester(ctx, nil)
		require.Error(t, err)
	})

	t.Run("empty requester returns error", func(t *testing.T) {
		_, err := qs.RequestsByRequester(ctx, &types.QueryRequestsByRequesterRequest{Requester: ""})
		require.Error(t, err)
	})

	t.Run("invalid requester address returns error", func(t *testing.T) {
		_, err := qs.RequestsByRequester(ctx, &types.QueryRequestsByRequesterRequest{Requester: "invalid"})
		require.Error(t, err)
	})
}

func TestQueryServer_Result(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Result(ctx, nil)
		require.Error(t, err)
	})

	t.Run("non-existent result returns not found", func(t *testing.T) {
		_, err := qs.Result(ctx, &types.QueryResultRequest{RequestId: 99999})
		require.Error(t, err)
	})
}

func TestQueryServer_EstimateCost(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)
	// Register a provider able to satisfy the spec
	providerAddr := sdk.AccAddress([]byte("cost_provider_addr"))
	provider := types.Provider{
		Address: providerAddr.String(),
		Stake:   math.NewInt(10000),
		AvailableSpecs: types.ComputeSpec{
			CpuCores:  4,
			MemoryMb:  4096,
			StorageGb: 50,
		},
		Pricing: types.Pricing{
			CpuPricePerMcoreHour:     math.LegacyNewDec(1),
			MemoryPricePerMbHour:     math.LegacyNewDec(1),
			GpuPricePerHour:          math.LegacyNewDec(1),
			StoragePricePerGbHour:    math.LegacyNewDec(1),
		},
		Active: true,
	}
	require.NoError(t, k.SetProvider(ctx, provider))
	require.NoError(t, k.setActiveProviderIndex(ctx, providerAddr, true))

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.EstimateCost(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns estimate", func(t *testing.T) {
		resp, err := qs.EstimateCost(ctx, &types.QueryEstimateCostRequest{
			Specs: types.ComputeSpec{
				CpuCores:  1,
				MemoryMb:  512,
				StorageGb: 5,
			},
			ProviderAddress: providerAddr.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_Disputes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Disputes(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns disputes", func(t *testing.T) {
		resp, err := qs.Disputes(ctx, &types.QueryDisputesRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_DisputesByRequest(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.DisputesByRequest(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns disputes for request", func(t *testing.T) {
		resp, err := qs.DisputesByRequest(ctx, &types.QueryDisputesByRequestRequest{RequestId: 1})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_DisputesByStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.DisputesByStatus(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns disputes by status", func(t *testing.T) {
		resp, err := qs.DisputesByStatus(ctx, &types.QueryDisputesByStatusRequest{
			Status: types.DISPUTE_STATUS_EVIDENCE_SUBMISSION,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_Evidence(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Evidence(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns evidence", func(t *testing.T) {
		resp, err := qs.Evidence(ctx, &types.QueryEvidenceRequest{DisputeId: 1})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_Appeal(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Appeal(ctx, nil)
		require.Error(t, err)
	})

	t.Run("non-existent appeal returns not found", func(t *testing.T) {
		_, err := qs.Appeal(ctx, &types.QueryAppealRequest{AppealId: 99999})
		require.Error(t, err)
	})
}

func TestQueryServer_Appeals(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.Appeals(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns appeals", func(t *testing.T) {
		resp, err := qs.Appeals(ctx, &types.QueryAppealsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_AppealsByStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.AppealsByStatus(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns appeals by status", func(t *testing.T) {
		resp, err := qs.AppealsByStatus(ctx, &types.QueryAppealsByStatusRequest{
			Status: types.APPEAL_STATUS_PENDING,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryServer_GovernanceParams(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	t.Run("nil request returns error", func(t *testing.T) {
		_, err := qs.GovernanceParams(ctx, nil)
		require.Error(t, err)
	})

	t.Run("valid request returns governance params", func(t *testing.T) {
		resp, err := qs.GovernanceParams(ctx, &types.QueryGovernanceParamsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}
