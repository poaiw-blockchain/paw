package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestIterateRequestsByRequester(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("req_requester_addr"))
	provider := sdk.AccAddress([]byte("req_provider_addr"))
	specs := types.ComputeSpec{CpuCores: 1, MemoryMb: 512, StorageGb: 5}

	requests := []types.Request{
		{
			Id:         1,
			Requester:  requester.String(),
			Provider:   provider.String(),
			Specs:      specs,
			Status:     types.REQUEST_STATUS_ASSIGNED,
			MaxPayment: math.NewInt(100),
			CreatedAt:  sdkCtx.BlockTime(),
		},
		{
			Id:         2,
			Requester:  requester.String(),
			Provider:   provider.String(),
			Specs:      specs,
			Status:     types.REQUEST_STATUS_COMPLETED,
			MaxPayment: math.NewInt(200),
			CreatedAt:  sdkCtx.BlockTime().Add(time.Minute),
		},
	}

	for _, req := range requests {
		require.NoError(t, k.SetRequest(ctx, req))
		require.NoError(t, k.setRequestIndexes(ctx, req))
	}

	var seen []uint64
	err := k.IterateRequestsByRequester(ctx, requester, func(request types.Request) (bool, error) {
		seen = append(seen, request.Id)
		return false, nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint64{1, 2}, seen)
}

func TestIterateRequestsByProvider(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("provider_req_requester"))
	provider := sdk.AccAddress([]byte("provider_req_provider"))
	specs := types.ComputeSpec{CpuCores: 2, MemoryMb: 1024, StorageGb: 8}

	req := types.Request{
		Id:         7,
		Requester:  requester.String(),
		Provider:   provider.String(),
		Specs:      specs,
		Status:     types.REQUEST_STATUS_ASSIGNED,
		MaxPayment: math.NewInt(500),
		CreatedAt:  sdkCtx.BlockTime(),
	}
	require.NoError(t, k.SetRequest(ctx, req))
	require.NoError(t, k.setRequestIndexes(ctx, req))

	count := 0
	err := k.IterateRequestsByProvider(ctx, provider, func(request types.Request) (bool, error) {
		count++
		require.Equal(t, provider.String(), request.Provider)
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestIterateRequestsByStatus(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("status_req_requester"))
	provider := sdk.AccAddress([]byte("status_req_provider"))
	specs := types.ComputeSpec{CpuCores: 1, MemoryMb: 256, StorageGb: 4}

	pendingRequest := types.Request{
		Id:         10,
		Requester:  requester.String(),
		Provider:   provider.String(),
		Specs:      specs,
		Status:     types.REQUEST_STATUS_PENDING,
		MaxPayment: math.NewInt(50),
		CreatedAt:  sdkCtx.BlockTime(),
	}
	completedRequest := types.Request{
		Id:         11,
		Requester:  requester.String(),
		Provider:   provider.String(),
		Specs:      specs,
		Status:     types.REQUEST_STATUS_COMPLETED,
		MaxPayment: math.NewInt(75),
		CreatedAt:  sdkCtx.BlockTime().Add(2 * time.Minute),
	}

	require.NoError(t, k.SetRequest(ctx, pendingRequest))
	require.NoError(t, k.setRequestIndexes(ctx, pendingRequest))
	require.NoError(t, k.SetRequest(ctx, completedRequest))
	require.NoError(t, k.setRequestIndexes(ctx, completedRequest))

	var pendingIDs []uint64
	err := k.IterateRequestsByStatus(ctx, types.REQUEST_STATUS_PENDING, func(request types.Request) (bool, error) {
		pendingIDs = append(pendingIDs, request.Id)
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{10}, pendingIDs)
}
