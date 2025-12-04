package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestHandleRequestTimeoutPenalizesReputation(t *testing.T) {
	k, sdkCtx, ctx := newComputeKeeperCtx(t)
	requester := createTestRequester(t)
	provider := setupProviderForRequests(t, k, sdkCtx)

	specs := createValidComputeSpec()
	containerImage := "ubuntu:22.04"
	command := []string{"/bin/bash"}
	envVars := map[string]string{}
	maxPayment := math.NewInt(10_000_000)

	requestID, err := k.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, "")
	require.NoError(t, err)

	request, err := k.GetRequest(ctx, requestID)
	require.NoError(t, err)
	request.Provider = provider.String()
	request.Status = types.REQUEST_STATUS_PROCESSING
	require.NoError(t, k.SetRequest(ctx, *request))

	now := sdkCtx.BlockTime()
	initialRep := types.ProviderReputation{
		Provider:               provider.String(),
		OverallScore:           100,
		ReliabilityScore:       1.0,
		SpeedScore:             1.0,
		AccuracyScore:          1.0,
		AvailabilityScore:      1.0,
		LastDecayTimestamp:     now,
		LastUpdateTimestamp:    now,
		PerformanceHistory:     nil,
		TotalRequests:          0,
		SuccessfulRequests:     0,
		FailedRequests:         0,
		TotalVerificationScore: 0,
		AverageResponseTime:    0,
	}
	require.NoError(t, k.SetProviderReputation(ctx, initialRep))

	providerRecord, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	providerRecord.Reputation = 100
	require.NoError(t, k.SetProvider(ctx, *providerRecord))

	err = k.HandleRequestTimeout(ctx, requestID)
	require.NoError(t, err)

	updatedRep, err := k.GetProviderReputation(ctx, provider)
	require.NoError(t, err)
	require.InDelta(t, 0.9, updatedRep.ReliabilityScore, 0.0001)
	require.Equal(t, uint64(1), updatedRep.FailedRequests)
	require.Equal(t, uint32(96), updatedRep.OverallScore)

	updatedProvider, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, uint32(96), updatedProvider.Reputation)
}
