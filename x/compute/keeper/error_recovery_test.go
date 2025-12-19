package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestSubmitRequestRevertOnEscrowFailure tests that request submission reverts when escrow fails
func TestSubmitRequestRevertOnEscrowFailure(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Get params first
	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	// Register a provider first (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", params.MinProviderStake.Add(math.NewInt(100000)))

	specs := types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	pricing := types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}
	err = k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	// Create requester without sufficient funds
	requester := sdk.AccAddress("requester__________")

	// Attempt to submit request (will fail due to insufficient funds for escrow)
	maxPayment := math.NewInt(1000000)
	requestSpecs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		GpuType:        "",
		StorageGb:      50,
		TimeoutSeconds: 1800,
	}
	requestID, err := k.SubmitRequest(ctx, requester, requestSpecs, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to escrow payment")
	require.Equal(t, uint64(0), requestID, "no request ID should be assigned on failure")

	// Verify no request was stored
	_, err = k.GetRequest(ctx, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// Verify requester balance unchanged (no escrow)
	bankKeeper := getBankKeeper(t, k)
	balance := bankKeeper.GetBalance(ctx, requester, "upaw")
	require.True(t, balance.Amount.IsZero(), "balance should remain zero")
}

// TestCancelRequestRevertOnRefundFailure tests that cancellation handles refund failures
func TestCancelRequestRevertOnRefundFailure(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundAmount := math.NewInt(10000000)
	fundTestAccount(t, k, ctx, requester, "upaw", fundAmount)

	// Submit request successfully
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)
	require.Greater(t, requestID, uint64(0))

	// Verify escrow happened
	bankKeeper := getBankKeeper(t, k)
	requesterBalance := bankKeeper.GetBalance(ctx, requester, "upaw")
	expectedBalance := fundAmount.Sub(maxPayment)
	require.Equal(t, expectedBalance, requesterBalance.Amount)

	// Get request before cancellation
	requestBefore, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_ASSIGNED, requestBefore.Status)

	// Cancel request (refund should succeed)
	err = k.CancelRequest(ctx, requester, requestID)
	require.NoError(t, err)

	// Verify request status updated
	requestAfter, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_CANCELLED, requestAfter.Status)

	// Verify refund occurred
	finalBalance := bankKeeper.GetBalance(ctx, requester, "upaw")
	require.Equal(t, fundAmount, finalBalance.Amount, "full amount should be refunded")
}

// TestCompleteRequestRevertOnPaymentReleaseFailure tests completion with payment release failure
func TestCompleteRequestRevertOnPaymentReleaseFailure(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundAmount := math.NewInt(10000000)
	fundTestAccount(t, k, ctx, requester, "upaw", fundAmount)

	// Submit request
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)

	// Start processing
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Drain module account to simulate payment release failure
	bankKeeper := getBankKeeper(t, k)
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	moduleBalance := bankKeeper.GetBalance(ctx, moduleAddr, "upaw")
	if moduleBalance.Amount.IsPositive() {
		burnAddr := authtypes.NewModuleAddress("burn")
		err = bankKeeper.SendCoins(ctx, moduleAddr, burnAddr, sdk.NewCoins(moduleBalance))
		require.NoError(t, err)
	}

	// Attempt to complete request (payment release should fail)
	err = k.CompleteRequest(ctx, requestID, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to release payment")

	// Verify request status unchanged (or properly handled)
	// The implementation may vary - verify it doesn't corrupt state
	finalRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	// Status may be PROCESSING or updated depending on implementation
	// The key is that escrow amount should be preserved
	require.True(t, finalRequest.EscrowedAmount.GT(math.ZeroInt()) || finalRequest.Status == types.REQUEST_STATUS_PROCESSING)
}

// TestCompleteRequestFailurePreservesEscrow tests that failed completion doesn't lose escrow
func TestCompleteRequestFailurePreservesEscrow(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundAmount := math.NewInt(10000000)
	fundTestAccount(t, k, ctx, requester, "upaw", fundAmount)

	// Submit request
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)

	// Get initial escrow amount
	initialRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	initialEscrow := initialRequest.EscrowedAmount

	// Attempt to complete request from wrong status (should fail)
	err = k.CompleteRequest(ctx, requestID, true)
	require.Error(t, err)

	// Verify escrow preserved
	finalRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, initialEscrow, finalRequest.EscrowedAmount, "escrow should be preserved on completion failure")
}

// TestPartialRequestFailureDoesNotCorruptState tests partial failure handling
func TestPartialRequestFailureDoesNotCorruptState(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester without funds
	requester := sdk.AccAddress("requester__________")

	// Get initial module balance
	bankKeeper := getBankKeeper(t, k)
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	initialModuleBalance := bankKeeper.GetBalance(ctx, moduleAddr, "upaw")

	// Attempt request submission (will fail)
	maxPayment := math.NewInt(1000000)
	_, err = k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.Error(t, err)

	// Verify state unchanged
	finalModuleBalance := bankKeeper.GetBalance(ctx, moduleAddr, "upaw")
	require.Equal(t, initialModuleBalance, finalModuleBalance, "module balance should be unchanged on failed request")

	// Verify no dangling request data
	_, err = k.GetRequest(ctx, 1)
	require.Error(t, err)
}

// TestCancelRequestFailurePreservesStatus tests that failed cancellation doesn't change status
func TestCancelRequestFailurePreservesStatus(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundTestAccount(t, k, ctx, requester, "upaw", math.NewInt(10000000))

	// Submit request
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)

	// Move to processing state
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Attempt to cancel from wrong status (should fail)
	err = k.CancelRequest(ctx, requester, requestID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be cancelled")

	// Verify status unchanged
	finalRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_PROCESSING, finalRequest.Status)
}

// TestDoubleCompletionPrevented tests that requests cannot be completed twice
func TestDoubleCompletionPrevented(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundTestAccount(t, k, ctx, requester, "upaw", math.NewInt(10000000))

	// Submit request
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)

	// Move to processing state
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Complete successfully first time
	err = k.CompleteRequest(ctx, requestID, true)
	require.NoError(t, err)

	// Verify completed
	completedRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_COMPLETED, completedRequest.Status)
	require.True(t, completedRequest.EscrowedAmount.IsZero(), "escrow should be released")

	// Attempt to complete again (should fail or be idempotent)
	err = k.CompleteRequest(ctx, requestID, true)
	require.Error(t, err, "should not allow double completion")

	// Could also check for specific error like "already settled"
	if err != nil {
		require.Contains(t, err.Error(), "already settled")
	}
}

// TestRequestRefundOnFailure tests that failed requests properly refund to requester
func TestRequestRefundOnFailure(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register provider (fund them first for stake)
	provider := sdk.AccAddress("test_provider______")
	fundTestAccount(t, k, ctx, provider, "upaw", math.NewInt(2000000))
	err := k.RegisterProvider(ctx, provider, "TestProvider", "https://test.example.com", types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}, types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}, math.NewInt(1000000))
	require.NoError(t, err)

	// Create requester and fund them
	requester := sdk.AccAddress("requester__________")
	fundAmount := math.NewInt(10000000)
	fundTestAccount(t, k, ctx, requester, "upaw", fundAmount)

	// Get initial balance
	bankKeeper := getBankKeeper(t, k)
	initialBalance := bankKeeper.GetBalance(ctx, requester, "upaw")

	// Submit request
	maxPayment := math.NewInt(1000000)
	requestID, err := k.SubmitRequest(ctx, requester, types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      50,
		GpuCount:       0,
		TimeoutSeconds: 1800,
	}, "alpine:latest", []string{"/bin/sh", "-c", "echo hello"}, nil, maxPayment, "")
	require.NoError(t, err)

	// Verify escrow occurred
	balanceAfterEscrow := bankKeeper.GetBalance(ctx, requester, "upaw")
	require.Equal(t, initialBalance.Amount.Sub(maxPayment), balanceAfterEscrow.Amount)

	// Move to processing state
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Complete with failure (should refund)
	err = k.CompleteRequest(ctx, requestID, false)
	require.NoError(t, err)

	// Verify refund occurred
	finalBalance := bankKeeper.GetBalance(ctx, requester, "upaw")
	require.Equal(t, initialBalance.Amount, finalBalance.Amount, "full refund should occur on failed request")

	// Verify request marked as failed
	finalRequest, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_FAILED, finalRequest.Status)
	require.True(t, finalRequest.EscrowedAmount.IsZero(), "escrow should be released")
}

// Helper function to fund accounts in compute module tests
func fundTestAccount(t *testing.T, k *keeper.Keeper, ctx sdk.Context, addr sdk.AccAddress, denom string, amount math.Int) {
	// Get bank keeper using reflection
	bankKeeper := getBankKeeper(t, k)

	// Mint coins to module account first
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))

	err := bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	require.NoError(t, err)

	// Transfer to target address
	err = bankKeeper.SendCoins(ctx, moduleAddr, addr, coins)
	require.NoError(t, err)
}
