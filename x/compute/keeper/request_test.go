package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
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

func buildVerificationProof(t *testing.T, requestID uint64, outputHash, containerImage string, command []string) []byte {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	execTrace := bytes.Repeat([]byte{0xAB}, 32)
	merkleSibling := bytes.Repeat([]byte{0xCD}, 32)

	currentHash := make([]byte, len(execTrace))
	copy(currentHash, execTrace)
	if len(currentHash) != 32 {
		sum := sha256.Sum256(currentHash)
		currentHash = sum[:]
	}

	merkleProof := [][]byte{merkleSibling}
	for _, sibling := range merkleProof {
		hasher := sha256.New()
		hasher.Write(currentHash)
		hasher.Write(sibling)
		currentHash = hasher.Sum(nil)
	}
	merkleRoot := currentHash

	stateHasher := sha256.New()
	stateHasher.Write([]byte(containerImage))
	for _, cmd := range command {
		stateHasher.Write([]byte(cmd))
	}
	stateHasher.Write([]byte(outputHash))
	stateHasher.Write(execTrace)
	stateCommitment := stateHasher.Sum(nil)

	proof := types.VerificationProof{
		PublicKey:       pubKey,
		MerkleRoot:      merkleRoot,
		MerkleProof:     merkleProof,
		StateCommitment: stateCommitment,
		ExecutionTrace:  execTrace,
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	message := proof.ComputeMessageHash(requestID, outputHash)
	proof.Signature = ed25519.Sign(privKey, message)

	buf := bytes.NewBuffer(nil)
	buf.Write(proof.Signature)
	buf.Write(proof.PublicKey)
	buf.Write(proof.MerkleRoot)
	buf.WriteByte(byte(len(proof.MerkleProof)))
	for _, node := range proof.MerkleProof {
		buf.Write(node)
	}
	buf.Write(proof.StateCommitment)
	buf.Write(proof.ExecutionTrace[:32])

	nonceBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBz, proof.Nonce)
	buf.Write(nonceBz)

	tsBz := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBz, uint64(proof.Timestamp))
	buf.Write(tsBz)

	return buf.Bytes()
}

// TestSubmitRequest_Valid tests successful request submission
func TestSubmitRequest_Valid(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	require.Equal(t, types.REQUEST_STATUS_ASSIGNED, request.Status)
}

// TestSubmitRequest_InsufficientEscrow tests rejection when payment is too low
func TestSubmitRequest_InsufficientEscrow(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	k, _, ctx := newComputeKeeperCtx(t)
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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
		mutateSpec    func(spec *types.ComputeSpec)
		errorContains string
	}{
		{
			name: "zero CPU",
			mutateSpec: func(spec *types.ComputeSpec) {
				spec.CpuCores = 0
			},
			errorContains: "invalid compute specs",
		},
		{
			name: "zero memory",
			mutateSpec: func(spec *types.ComputeSpec) {
				spec.MemoryMb = 0
			},
			errorContains: "invalid compute specs",
		},
		{
			name: "zero timeout",
			mutateSpec: func(spec *types.ComputeSpec) {
				spec.TimeoutSeconds = 0
			},
			errorContains: "invalid compute specs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sdkCtx, ctx := newComputeKeeperCtx(t)
			requester := createTestRequester(t)
			_ = setupProviderForRequests(t, k, sdkCtx)

			specs := createValidComputeSpec()
			if tt.mutateSpec != nil {
				tt.mutateSpec(&specs)
			}

			containerImage := "ubuntu:22.04"
			command := []string{"/bin/bash"}
			envVars := map[string]string{}
			maxPayment := math.NewInt(10000000)

			_, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

// TestSubmitRequest_PreferredProvider tests request with preferred provider
func TestSubmitRequest_PreferredProvider(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

	specs := types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      50,
		TimeoutSeconds: 1800,
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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Cancel request
	err = k.CancelRequest(ctx, requester, requestID)
	require.NoError(t, err)

	// Verify request is cancelled
	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_CANCELLED, request.Status)
}

// TestCancelRequest_NotFound tests cancellation of non-existent request
func TestCancelRequest_NotFound(t *testing.T) {
	k, _, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)

	err := k.CancelRequest(ctx, requester, 99999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestCancelRequest_NotOwner tests that only requester can cancel
func TestCancelRequest_NotOwner(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Try to cancel as different user
	otherUser := sdk.AccAddress([]byte("other_user_address_"))
	err = k.CancelRequest(ctx, otherUser, requestID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

// TestCancelRequest_AlreadyProcessing tests cancellation of processing request
func TestCancelRequest_AlreadyProcessing(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Try to cancel
	err = k.CancelRequest(ctx, requester, requestID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be cancelled")
}

// TestSubmitResult_Valid tests successful result submission
func TestSubmitResult_Valid(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(sdkCtx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Submit result
	outputHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	outputURL := "https://storage.paw/results/1"
	logsURL := "https://storage.paw/logs/1"
	proofBytes := buildVerificationProof(t, requestID, outputHash, containerImage, command)

	err = k.SubmitResult(sdkCtx, provider, requestID, outputHash, outputURL, 0, logsURL, proofBytes)
	require.NoError(t, err)

	// Verify result stored
	request, err := k.GetRequest(sdkCtx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_COMPLETED, request.Status)
	require.NotNil(t, request.CompletedAt)

	result, err := k.GetResult(sdkCtx, requestID)
	require.NoError(t, err)
	require.True(t, result.Verified)
	require.Equal(t, outputHash, result.OutputHash)
	require.Equal(t, outputURL, result.OutputUrl)
}

// TestSubmitResult_NotFound tests result submission for non-existent request
func TestSubmitResult_NotFound(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

	outputHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	outputURL := "https://storage.paw/results/missing"
	proofBytes := buildVerificationProof(t, 99999, outputHash, "ubuntu:22.04", []string{"/bin/bash"})

	err := k.SubmitResult(ctx, provider, 99999, outputHash, outputURL, 0, "", proofBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestSubmitResult_WrongProvider tests result submission by wrong provider
func TestSubmitResult_WrongProvider(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10000000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	// Different provider tries to submit result
	wrongProvider := sdk.AccAddress([]byte("wrong_provider_addr_"))
	outputHash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	outputURL := "https://storage.paw/results/wrong-provider"
	proofBytes := buildVerificationProof(t, requestID, outputHash, containerImage, command)

	err = k.SubmitResult(ctx, wrongProvider, requestID, outputHash, outputURL, 0, "", proofBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

// TestGetRequest tests request retrieval
func TestGetRequest(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

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
	err := k.IterateRequests(ctx, func(request types.Request) (bool, error) {
		count++
		require.NotEmpty(t, request.Requester)
		require.Greater(t, request.Id, uint64(0))
		return false, nil // continue iteration
	})
	require.NoError(t, err)
	require.Equal(t, numRequests, count)
}

// TestRequestTimestamps tests timestamp tracking for requests
func TestRequestTimestamps(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	_ = setupProviderForRequests(t, k, sdkCtx)

	blockTime := time.Now().UTC()
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

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
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

	specs := types.ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "nvidia-t4",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	cost, costPerHour, err := k.EstimateCost(ctx, provider, specs)
	require.NoError(t, err)
	require.True(t, cost.IsPositive())
	require.True(t, costPerHour.IsPositive())

	expectedPerHour := math.LegacyNewDec(4).
		Add(math.LegacyNewDec(8192)).
		Add(math.LegacyNewDec(10)).
		Add(math.LegacyNewDec(100))
	require.True(t, costPerHour.Equal(expectedPerHour))

	hours := math.LegacyNewDec(int64(specs.TimeoutSeconds)).QuoInt64(3600)
	expectedTotal := expectedPerHour.Mul(hours).Ceil().TruncateInt()
	require.True(t, cost.Equal(expectedTotal))
}

// TestRequestStatusTransitions tests valid status transitions
func TestRequestStatusTransitions(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

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
	require.Equal(t, types.REQUEST_STATUS_ASSIGNED, request.Status)

	// Transition to PROCESSING
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = k.SetRequest(ctx, *request)
	require.NoError(t, err)

	// Transition to COMPLETED
	outputHash := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	outputURL := "https://storage.paw/results/status"
	logsURL := "https://storage.paw/logs/status"
	proofBytes := buildVerificationProof(t, requestID, outputHash, containerImage, command)

	err = k.SubmitResult(ctx, provider, requestID, outputHash, outputURL, 0, logsURL, proofBytes)
	require.NoError(t, err)

	request, err = k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.REQUEST_STATUS_COMPLETED, request.Status)
}
