package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Test helper functions

func createTestProvider(t *testing.T) sdk.AccAddress {
	return sdk.AccAddress([]byte("test_provider_addr_"))
}

func createTestProviderWithIndex(t *testing.T, index int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("test_provider_"))
	addr[19] = byte(index)
	return sdk.AccAddress(addr)
}

func createValidComputeSpec() types.ComputeSpec {
	return types.ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "nvidia-t4",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
}

func createValidPricing() types.Pricing {
	return types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(10),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}
}

// TestRegisterProvider_Valid tests successful provider registration
func TestRegisterProvider_Valid(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	// Fund provider account
	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Verify provider was stored
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.NotNil(t, storedProvider)
	require.Equal(t, provider.String(), storedProvider.Address)
	require.Equal(t, "TestProvider", storedProvider.Moniker)
	require.Equal(t, "https://test.github.com", storedProvider.Endpoint)
	require.Equal(t, specs, storedProvider.AvailableSpecs)
	require.Equal(t, pricing, storedProvider.Pricing)
	require.Equal(t, stake, storedProvider.Stake)
	require.GreaterOrEqual(t, storedProvider.Reputation, params.MinReputationScore)
	require.LessOrEqual(t, storedProvider.Reputation, uint32(100))
	require.True(t, storedProvider.Active)
	require.Equal(t, uint64(0), storedProvider.TotalRequestsCompleted)
	require.Equal(t, uint64(0), storedProvider.TotalRequestsFailed)
}

// TestRegisterProvider_Duplicate tests rejection of duplicate provider registration
func TestRegisterProvider_Duplicate(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider first time
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Attempt to register again
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider2", "https://test2.github.com", specs, pricing, stake)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
}

// TestRegisterProvider_InsufficientStake tests rejection of insufficient stake
func TestRegisterProvider_InsufficientStake(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()

	// Use stake less than minimum
	insufficientStake := params.MinProviderStake.Sub(math.NewInt(1))

	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, insufficientStake)
	require.Error(t, err)
	require.Contains(t, err.Error(), "less than minimum required")
}

// TestRegisterProvider_EmptyMoniker tests handling of empty moniker
func TestRegisterProvider_EmptyMoniker(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Empty moniker should succeed (it's optional)
	err = k.RegisterProvider(sdkCtx, provider, "", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, "", storedProvider.Moniker)
}

// TestRegisterProvider_InvalidSpecs tests rejection of invalid compute specs
func TestRegisterProvider_InvalidSpecs(t *testing.T) {
	tests := []struct {
		name          string
		specs         types.ComputeSpec
		errorContains string
	}{
		{
			name: "zero CPU",
			specs: types.ComputeSpec{
				CpuCores:  0,
				MemoryMb:  8192,
				GpuCount:  0,
				StorageGb: 100,
			},
			errorContains: "invalid compute specs",
		},
		{
			name: "zero Memory",
			specs: types.ComputeSpec{
				CpuCores:  4,
				MemoryMb:  0,
				GpuCount:  0,
				StorageGb: 100,
			},
			errorContains: "invalid compute specs",
		},
		{
			name: "zero Storage",
			specs: types.ComputeSpec{
				CpuCores:  4,
				MemoryMb:  8192,
				GpuCount:  0,
				StorageGb: 0,
			},
			errorContains: "invalid compute specs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sdkCtx, _ := newComputeKeeperCtx(t)
			provider := createTestProvider(t)

			params, err := k.GetParams(sdkCtx)
			require.NoError(t, err)

			pricing := createValidPricing()
			stake := params.MinProviderStake

			err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", tt.specs, pricing, stake)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

// TestRegisterProvider_InvalidPricing tests rejection of invalid pricing
func TestRegisterProvider_InvalidPricing(t *testing.T) {
	tests := []struct {
		name          string
		pricing       types.Pricing
		errorContains string
	}{
		{
			name: "zero CPU price",
			pricing: types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyZeroDec(),
				MemoryPricePerMbHour:  math.LegacyNewDec(100),
				GpuPricePerHour:       math.LegacyNewDec(10000),
				StoragePricePerGbHour: math.LegacyNewDec(10),
			},
			errorContains: "invalid pricing",
		},
		{
			name: "negative Memory price",
			pricing: types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(1),
				MemoryPricePerMbHour:  math.LegacyNewDec(-1),
				GpuPricePerHour:       math.LegacyNewDec(10000),
				StoragePricePerGbHour: math.LegacyNewDec(10),
			},
			errorContains: "invalid pricing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sdkCtx, _ := newComputeKeeperCtx(t)
			provider := createTestProvider(t)

			params, err := k.GetParams(sdkCtx)
			require.NoError(t, err)

			specs := createValidComputeSpec()
			stake := params.MinProviderStake

			err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, tt.pricing, stake)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

// TestUpdateProvider_Valid tests successful provider update
func TestUpdateProvider_Valid(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Update provider
	newSpecs := types.ComputeSpec{
		CpuCores:       8,
		MemoryMb:       16384,
		GpuCount:       2,
		GpuType:        "nvidia-a100",
		StorageGb:      200,
		TimeoutSeconds: 7200,
	}
	newPricing := types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(2),
		MemoryPricePerMbHour:  math.LegacyNewDec(2),
		GpuPricePerHour:       math.LegacyNewDec(20000),
		StoragePricePerGbHour: math.LegacyNewDec(20),
	}

	err = k.UpdateProvider(sdkCtx, provider, "UpdatedProvider", "https://updated.github.com", &newSpecs, &newPricing)
	require.NoError(t, err)

	// Verify updates
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, "UpdatedProvider", storedProvider.Moniker)
	require.Equal(t, "https://updated.github.com", storedProvider.Endpoint)
	require.Equal(t, newSpecs, storedProvider.AvailableSpecs)
	require.Equal(t, newPricing, storedProvider.Pricing)
}

// TestUpdateProvider_NotFound tests update of non-existent provider
func TestUpdateProvider_NotFound(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	specs := createValidComputeSpec()
	pricing := createValidPricing()

	err := k.UpdateProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", &specs, &pricing)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestUpdateProvider_PartialUpdate tests updating only some fields
func TestUpdateProvider_PartialUpdate(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Update only moniker (pass empty endpoint)
	err = k.UpdateProvider(sdkCtx, provider, "NewMoniker", "", nil, nil)
	require.NoError(t, err)

	// Verify only moniker changed
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, "NewMoniker", storedProvider.Moniker)
	require.Equal(t, "https://test.github.com", storedProvider.Endpoint)
	require.Equal(t, specs, storedProvider.AvailableSpecs)
	require.Equal(t, pricing, storedProvider.Pricing)
}

// TestDeactivateProvider_Valid tests successful provider deactivation
func TestDeactivateProvider_Valid(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Deactivate provider
	err = k.DeactivateProvider(sdkCtx, provider)
	require.NoError(t, err)

	// Verify deactivation
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.False(t, storedProvider.Active)
}

// TestDeactivateProvider_NotFound tests deactivation of non-existent provider
func TestDeactivateProvider_NotFound(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	err := k.DeactivateProvider(sdkCtx, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestGetProvider tests provider retrieval
func TestGetProvider(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	// Provider not found initially
	_, err := k.GetProvider(sdkCtx, provider)
	require.Error(t, err)

	// Register provider
	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Provider found after registration
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.NotNil(t, storedProvider)
	require.Equal(t, provider.String(), storedProvider.Address)
}

// TestIterateProviders tests provider iteration
func TestIterateProviders(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register multiple providers
	numProviders := 5
	for i := 0; i < numProviders; i++ {
		provider := createTestProviderWithIndex(t, i)
		err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
		require.NoError(t, err)
	}

	// Iterate and count
	count := 0
	err = k.IterateProviders(sdkCtx, func(provider types.Provider) (bool, error) {
		count++
		require.NotEmpty(t, provider.Address)
		require.GreaterOrEqual(t, provider.Reputation, params.MinReputationScore)
		require.LessOrEqual(t, provider.Reputation, uint32(100))
		return false, nil // continue iteration
	})
	require.NoError(t, err)
	require.Equal(t, numProviders, count)
}

// TestProviderReputation_Increment tests reputation increase
func TestProviderReputation_Increment(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Increment reputation for successful job
	err = k.UpdateProviderReputation(sdkCtx, provider, true)
	require.NoError(t, err)

	// Verify reputation stayed at or near 100 (can't exceed 100)
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.LessOrEqual(t, storedProvider.Reputation, uint32(100))
}

// TestProviderReputation_Decrement tests reputation decrease
func TestProviderReputation_Decrement(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Decrement reputation for failed job
	err = k.UpdateProviderReputation(sdkCtx, provider, false)
	require.NoError(t, err)

	// Verify reputation decreased
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Less(t, storedProvider.Reputation, uint32(100))
}

// TestProviderStats_UpdateOnCompletion tests statistics tracking
func TestProviderStats_UpdateOnCompletion(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Update stats for successful completion
	err = k.UpdateProviderReputation(sdkCtx, provider, true)
	require.NoError(t, err)

	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)

	// Stats should be updated
	require.Equal(t, uint64(1), storedProvider.TotalRequestsCompleted)
	require.Equal(t, uint64(0), storedProvider.TotalRequestsFailed)

	// Update stats for failed completion
	err = k.UpdateProviderReputation(sdkCtx, provider, false)
	require.NoError(t, err)

	storedProvider, err = k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, uint64(1), storedProvider.TotalRequestsCompleted)
	require.Equal(t, uint64(1), storedProvider.TotalRequestsFailed)
}

// TestProviderTimestamps tests timestamp tracking
func TestProviderTimestamps(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)
	provider := createTestProvider(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	specs := createValidComputeSpec()
	pricing := createValidPricing()
	stake := params.MinProviderStake

	// Set a specific block time
	blockTime := time.Now().UTC()
	sdkCtx = sdkCtx.WithBlockTime(blockTime)

	// Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)

	// Verify timestamps
	require.Equal(t, blockTime.Unix(), storedProvider.RegisteredAt.Unix())
	require.Equal(t, blockTime.Unix(), storedProvider.LastActiveAt.Unix())
}

// TestFindSuitableProvider tests provider matching logic
func TestFindSuitableProvider(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	stake := params.MinProviderStake

	// Register provider with specific specs
	provider := createTestProvider(t)
	specs := types.ComputeSpec{
		CpuCores:       8,
		MemoryMb:       16384,
		GpuCount:       2,
		GpuType:        "nvidia-a100",
		StorageGb:      200,
		TimeoutSeconds: 7200,
	}
	pricing := createValidPricing()

	err = k.RegisterProvider(sdkCtx, provider, "HighEndProvider", "https://test.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Request matching specs (should succeed)
	requestSpecs := types.ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		GpuCount:       1,
		GpuType:        "nvidia-a100",
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	foundProvider, err := k.FindSuitableProvider(sdkCtx, requestSpecs, "")
	require.NoError(t, err)
	require.Equal(t, provider.String(), foundProvider.String())

	// Request exceeding specs (should fail)
	excessiveSpecs := types.ComputeSpec{
		CpuCores:       16,
		MemoryMb:       32768,
		GpuCount:       4,
		GpuType:        "nvidia-a100",
		StorageGb:      500,
		TimeoutSeconds: 3600,
	}

	_, err = k.FindSuitableProvider(sdkCtx, excessiveSpecs, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no suitable provider")
}

// TestFindSuitableProvider_PreferredProvider tests preferred provider selection
func TestFindSuitableProvider_PreferredProvider(t *testing.T) {
	k, sdkCtx, _ := newComputeKeeperCtx(t)

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	stake := params.MinProviderStake
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	// Register two providers
	provider1 := createTestProviderWithIndex(t, 1)
	provider2 := createTestProviderWithIndex(t, 2)

	err = k.RegisterProvider(sdkCtx, provider1, "Provider1", "https://test1.github.com", specs, pricing, stake)
	require.NoError(t, err)

	err = k.RegisterProvider(sdkCtx, provider2, "Provider2", "https://test2.github.com", specs, pricing, stake)
	require.NoError(t, err)

	// Request with preferred provider
	requestSpecs := types.ComputeSpec{
		CpuCores:  2,
		MemoryMb:  4096,
		GpuCount:  0,
		StorageGb: 50,
	}

	foundProvider, err := k.FindSuitableProvider(sdkCtx, requestSpecs, provider2.String())
	require.NoError(t, err)
	require.Equal(t, provider2.String(), foundProvider.String())
}
