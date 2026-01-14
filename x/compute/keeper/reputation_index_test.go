package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestReputationIndexPerformance verifies that provider selection uses the reputation index
func TestReputationIndexPerformance(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)

	params, _ := k.GetParams(sdkCtx)

	// Register 10 providers with different reputations
	providers := make([]sdk.AccAddress, 10)
	for i := 0; i < 10; i++ {
		// Create unique provider for each iteration
		addr := make([]byte, 20)
		copy(addr, []byte("test_provider_"))
		addr[19] = byte(i)
		providers[i] = sdk.AccAddress(addr)

		// Fund the provider account
		fundTestAccount(t, k, sdkCtx, providers[i], "upaw", params.MinProviderStake.MulRaw(2))

		specs := createValidComputeSpec()
		pricing := createValidPricing()

		// Register provider
		err := k.RegisterProvider(goCtx, providers[i], "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
		require.NoError(t, err)

		// Update reputation to different values
		for j := 0; j < i*10; j++ {
			_ = k.UpdateProviderReputation(goCtx, providers[i], true)
		}
	}

	// Find suitable provider - should return the one with highest reputation
	// Use smaller specs than what providers offer
	specs := types.ComputeSpec{
		CpuCores:       2,    // Providers have 4
		MemoryMb:       4096, // Providers have 8192
		StorageGb:      50,   // Providers have 100
		GpuCount:       0,    // No GPU needed
		TimeoutSeconds: 60,
	}

	bestProvider, err := k.FindSuitableProvider(goCtx, specs, "")
	require.NoError(t, err)
	require.NotNil(t, bestProvider)

	// Verify it's the provider with highest reputation
	providerRecord, err := k.GetProvider(goCtx, bestProvider)
	require.NoError(t, err)

	// Check all other providers have lower or equal reputation
	for _, addr := range providers {
		other, _ := k.GetProvider(goCtx, addr)
		require.LessOrEqual(t, other.Reputation, providerRecord.Reputation)
	}
}
