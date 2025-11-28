package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Helper functions for request tests

func createTestRequester(t *testing.T) sdk.AccAddress {
	return sdk.AccAddress([]byte("test_requester_addr"))
}

func setupProviderForRequests(t *testing.T, k *keeper.Keeper, ctx sdk.Context) sdk.AccAddress {
	provider := createTestProvider(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	err = k.RegisterProvider(ctx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	return provider
}

// TestSubmitRequest_Valid tests successful request submission
func TestSubmitRequest_Valid(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := types.ComputeSpec{
		Cpu:     2,
		Memory:  4096,
		Gpu:     0,
		Storage: 50,
	}

	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash", "-c", "echo hello"}
	envVars := map[string]string{"TEST": "value"}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)
	require.Greater(t, requestID, uint64(0))

	// Verify request was stored
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.NotNil(t, request)
	require.Equal(t, requester.String(), request.Requester)
	require.Equal(t, containerImage, request.ContainerImage)
	require.Equal(t, command, request.Command)
	require.Equal(t, envVars, request.EnvVars)
	require.Equal(t, maxPayment, request.MaxPayment)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_ASSIGNED, request.Status)
}

// TestSubmitRequest_InsufficientEscrow tests rejection when payment is too low
func TestSubmitRequest_InsufficientEscrow(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash", "-c", "echo hello"}
	envVars := map[string]string{}

	// Very low payment that won't cover costs
	insufficientPayment := math.NewInt(1)

	_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, insufficientPayment, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "less than estimated cost")
}

// TestSubmitRequest_InvalidProvider tests handling when no suitable provider exists
func TestSubmitRequest_InvalidProvider(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)

	// Don't register any provider

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash", "-c", "echo hello"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "suitable provider")
}

// TestSubmitRequest_ZeroPayment tests rejection of zero payment
func TestSubmitRequest_ZeroPayment(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	zeroPayment := math.NewInt(0)

	_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, zeroPayment, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be greater than zero")
}

// TestSubmitRequest_NegativePayment tests rejection of negative payment
func TestSubmitRequest_NegativePayment(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	negativePayment := math.NewInt(-1000)

	_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, negativePayment, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be greater than zero")
}

// TestSubmitRequest_EmptyContainerImage tests rejection of empty container image
func TestSubmitRequest_EmptyContainerImage(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := ""
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "container image is required")
}

// TestSubmitRequest_InvalidSpecs tests rejection of invalid compute specs
func TestSubmitRequest_InvalidSpecs(t *testing.T) {
	tests := []struct {
		name          string
		specs         types.ComputeSpec
		errorContains string
	}{
		{
			name: "zero CPU",
			specs: types.ComputeSpec{
				Cpu:     0,
				Memory:  4096,
				Gpu:     0,
				Storage: 50,
			},
			errorContains: "invalid compute specs",
		},
		{
			name: "negative Memory",
			specs: types.ComputeSpec{
				Cpu:     2,
				Memory:  -1,
				Gpu:     0,
				Storage: 50,
			},
			errorContains: "invalid compute specs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := keepertest.ComputeKeeper(t)
			requester := createTestRequester(t)
			_ = setupProviderForRequests(t, k, ctx)

			containerImage := "ubuntu:22.04"
			command := []string{"/bin/bash"}
			envVars := map[string]string{}
			maxPayment := math.NewInt(10000000)

			_, err := k.SubmitRequest(ctx, requester, tt.specs, containerImage, command, envVars, maxPayment, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

// TestSubmitRequest_PreferredProvider tests request with preferred provider
func TestSubmitRequest_PreferredProvider(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, ctx)

	specs := types.ComputeSpec{
		Cpu:     2,
		Memory:  4096,
		Gpu:     0,
		Storage: 50,
	}

	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, provider.String())
	require.NoError(t, err)

	// Verify request assigned to preferred provider
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, provider.String(), request.Provider)
}

// TestCancelRequest_Valid tests successful request cancellation
func TestCancelRequest_Valid(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Cancel request
	err = k.CancelRequest(ctx, requestID, requester)
	require.NoError(t, err)

	// Verify request is cancelled
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_CANCELLED, request.Status)
}

// TestCancelRequest_NotFound tests cancellation of non-existent request
func TestCancelRequest_NotFound(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)

	err := k.CancelRequest(ctx, 99999, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestCancelRequest_NotOwner tests that only requester can cancel
func TestCancelRequest_NotOwner(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Try to cancel as different user
	otherUser := sdk.AccAddress([]byte("other_user_address_"))
	err = k.CancelRequest(ctx, requestID, otherUser)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

// TestCancelRequest_AlreadyProcessing tests cancellation of processing request
func TestCancelRequest_AlreadyProcessing(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Manually update status to processing
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Status = types.RequestStatus_REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Try to cancel
	err = k.CancelRequest(ctx, requestID, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot cancel")
}

// TestSubmitResult_Valid tests successful result submission
func TestSubmitResult_Valid(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Submit result
	resultData := []byte("test result data")
	resultHash := "abc123"
	proof := types.VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("trace"),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	err = k.SubmitResult(ctx, requestID, provider, resultData, resultHash, proof)
	require.NoError(t, err)

	// Verify result stored
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_COMPLETED, request.Status)
	require.NotNil(t, request.CompletedAt)
}

// TestSubmitResult_NotFound tests result submission for non-existent request
func TestSubmitResult_NotFound(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	provider := setupProviderForRequests(t, k, ctx)

	resultData := []byte("test result data")
	resultHash := "abc123"
	proof := types.VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("trace"),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	err := k.SubmitResult(ctx, 99999, provider, resultData, resultHash, proof)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestSubmitResult_WrongProvider tests result submission by wrong provider
func TestSubmitResult_WrongProvider(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Different provider tries to submit result
	wrongProvider := sdk.AccAddress([]byte("wrong_provider_addr_"))
	resultData := []byte("test result data")
	resultHash := "abc123"
	proof := types.VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("trace"),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	err = k.SubmitResult(ctx, requestID, wrongProvider, resultData, resultHash, proof)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

// TestGetRequest tests request retrieval
func TestGetRequest(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Get request
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.NotNil(t, request)
	require.Equal(t, requestID, request.Id)
	require.Equal(t, requester.String(), request.Requester)
}

// TestIterateRequests tests request iteration
func TestIterateRequests(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	// Submit multiple requests
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err := k.IterateRequests(ctx, func(request types.Request) bool {
		count++
		require.NotEmpty(t, request.Requester)
		require.Greater(t, request.Id, uint64(0))
		return false // continue iteration
	})
	require.NoError(t, err)
	require.Equal(t, numRequests, count)
}

// TestRequestTimestamps tests timestamp tracking for requests
func TestRequestTimestamps(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, ctx)

	blockTime := time.Now().UTC()
	ctx = ctx.WithBlockTime(blockTime)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)

	// Verify timestamps
	require.Equal(t, blockTime.Unix(), request.CreatedAt.Unix())
	require.NotNil(t, request.AssignedAt)
	require.Equal(t, blockTime.Unix(), request.AssignedAt.Unix())
}

// TestEstimateCost tests cost estimation logic
func TestEstimateCost(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	provider := setupProviderForRequests(t, k, ctx)

	specs := types.ComputeSpec{
		Cpu:     4,
		Memory:  8192,
		Gpu:     1,
		GpuType: "nvidia-t4",
		Storage: 100,
	}

	cost, breakdown, err := k.EstimateCost(ctx, provider, specs)
	require.NoError(t, err)
	require.True(t, cost.IsPositive())
	require.NotNil(t, breakdown)

	// Cost should include CPU, memory, GPU, and storage components
	require.True(t, breakdown.CpuCost.IsPositive())
	require.True(t, breakdown.MemoryCost.IsPositive())
	require.True(t, breakdown.GpuCost.IsPositive())
	require.True(t, breakdown.StorageCost.IsPositive())
}

// TestRequestStatusTransitions tests valid status transitions
func TestRequestStatusTransitions(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, ctx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Initial status should be ASSIGNED
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_ASSIGNED, request.Status)

	// Transition to PROCESSING
	request.Status = types.RequestStatus_REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Transition to COMPLETED
	resultData := []byte("result")
	resultHash := "hash"
	proof := types.VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("trace"),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	err = k.SubmitResult(ctx, requestID, provider, resultData, resultHash, proof)
	require.NoError(t, err)

	request, err = k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.RequestStatus_REQUEST_STATUS_COMPLETED, request.Status)
}
