package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestProviderCacheMetadata tests cache metadata storage and retrieval
func TestProviderCacheMetadata(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Get default metadata
	metadata, err := k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.NotNil(t, metadata)
	require.Equal(t, int64(0), metadata.LastRefreshBlock)
	require.Equal(t, uint32(0), metadata.CacheSize)
	require.False(t, metadata.Enabled)

	// Update metadata
	newMetadata := &keeper.ProviderCacheMetadata{
		LastRefreshBlock: 100,
		CacheSize:        10,
		Enabled:          true,
	}
	err = k.SetProviderCacheMetadata(ctx, newMetadata)
	require.NoError(t, err)

	// Retrieve updated metadata
	retrieved, err := k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(100), retrieved.LastRefreshBlock)
	require.Equal(t, uint32(10), retrieved.CacheSize)
	require.True(t, retrieved.Enabled)
}

// TestCachedProviderStorageRetrieval tests storing and retrieving cached providers
func TestCachedProviderStorageRetrieval(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test cached provider
	providerAddr := sdk.AccAddress([]byte("provider1"))
	cached := types.CachedProvider{
		Provider:      providerAddr.String(),
		Reputation:    85,
		CachedAtBlock: sdkCtx.BlockHeight(),
	}

	// Store cached provider
	err := k.SetCachedProvider(ctx, 0, cached)
	require.NoError(t, err)

	// Retrieve cached provider
	retrieved, err := k.GetCachedProvider(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, cached.Provider, retrieved.Provider)
	require.Equal(t, cached.Reputation, retrieved.Reputation)
	require.Equal(t, cached.CachedAtBlock, retrieved.CachedAtBlock)

	// Try to retrieve non-existent cache entry
	_, err = k.GetCachedProvider(ctx, 10)
	require.Error(t, err)
}

// TestRefreshProviderCache tests the cache refresh mechanism
func TestRefreshProviderCache(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test providers with varying reputations
	providers := []struct {
		addr       sdk.AccAddress
		moniker    string
		reputation uint32
		stake      math.Int
	}{
		{sdk.AccAddress([]byte("provider1")), "Provider 1", 95, math.NewInt(2000000)},
		{sdk.AccAddress([]byte("provider2")), "Provider 2", 85, math.NewInt(1500000)},
		{sdk.AccAddress([]byte("provider3")), "Provider 3", 75, math.NewInt(1200000)},
		{sdk.AccAddress([]byte("provider4")), "Provider 4", 65, math.NewInt(1100000)},
		{sdk.AccAddress([]byte("provider5")), "Provider 5", 55, math.NewInt(1050000)},
		{sdk.AccAddress([]byte("provider6")), "Provider 6", 50, math.NewInt(1000000)},
	}

	// Register providers
	for _, p := range providers {
		// Fund the provider account before registration
		fundTestAccount(t, k, sdkCtx, p.addr, "upaw", p.stake.MulRaw(2))

		specs := types.ComputeSpec{
			CpuCores:       4000,
			MemoryMb:       8192,
			StorageGb:      100,
			TimeoutSeconds: 3600,
		}
		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		}

		err := k.RegisterProvider(ctx, p.addr, p.moniker, "http://localhost:8080", specs, pricing, p.stake)
		require.NoError(t, err)

		// Manually set reputation for testing
		provider, err := k.GetProvider(ctx, p.addr)
		require.NoError(t, err)
		provider.Reputation = p.reputation
		err = k.SetProvider(ctx, *provider)
		require.NoError(t, err)

		// Update reputation index
		err = k.UpdateProviderReputation(ctx, p.addr, true)
		require.NoError(t, err)
	}

	// Set cache parameters
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UseProviderCache = true
	params.ProviderCacheSize = 3 // Cache top 3 providers
	params.ProviderCacheRefreshInterval = 10
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Refresh cache
	err = k.RefreshProviderCache(ctx)
	require.NoError(t, err)

	// Verify metadata
	metadata, err := k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.True(t, metadata.Enabled)
	require.Equal(t, uint32(3), metadata.CacheSize)
	require.Equal(t, sdkCtx.BlockHeight(), metadata.LastRefreshBlock)

	// Verify cached providers are top 3 by reputation
	cached0, err := k.GetCachedProvider(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, providers[0].addr.String(), cached0.Provider) // 95 reputation
	require.Equal(t, uint32(95), cached0.Reputation)

	cached1, err := k.GetCachedProvider(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, providers[1].addr.String(), cached1.Provider) // 85 reputation
	require.Equal(t, uint32(85), cached1.Reputation)

	cached2, err := k.GetCachedProvider(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, providers[2].addr.String(), cached2.Provider) // 75 reputation
	require.Equal(t, uint32(75), cached2.Reputation)

	// Verify 4th entry doesn't exist
	_, err = k.GetCachedProvider(ctx, 3)
	require.Error(t, err)
}

// TestShouldRefreshCache tests the cache refresh interval logic
func TestShouldRefreshCache(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set params
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UseProviderCache = true
	params.ProviderCacheRefreshInterval = 100
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Should refresh when cache is not initialized
	shouldRefresh, err := k.ShouldRefreshCache(ctx)
	require.NoError(t, err)
	require.True(t, shouldRefresh)

	// Set metadata to simulate initialized cache
	metadata := &keeper.ProviderCacheMetadata{
		LastRefreshBlock: sdkCtx.BlockHeight(),
		CacheSize:        10,
		Enabled:          true,
	}
	err = k.SetProviderCacheMetadata(ctx, metadata)
	require.NoError(t, err)

	// Should not refresh immediately
	shouldRefresh, err = k.ShouldRefreshCache(ctx)
	require.NoError(t, err)
	require.False(t, shouldRefresh)

	// Advance block height past refresh interval
	newSdkCtx := sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 101)
	newCtx := sdk.WrapSDKContext(newSdkCtx)

	// Should refresh now
	shouldRefresh, err = k.ShouldRefreshCache(newCtx)
	require.NoError(t, err)
	require.True(t, shouldRefresh)

	// Disable cache
	params.UseProviderCache = false
	err = k.SetParams(newCtx, params)
	require.NoError(t, err)

	// Should not refresh when disabled
	shouldRefresh, err = k.ShouldRefreshCache(newCtx)
	require.NoError(t, err)
	require.False(t, shouldRefresh)
}

// TestFindSuitableProviderFromCache tests finding providers from cache
func TestFindSuitableProviderFromCache(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Create test providers
	providers := []struct {
		addr       sdk.AccAddress
		reputation uint32
		cpuCores   uint64
		memoryMb   uint64
	}{
		{sdk.AccAddress([]byte("provider1")), 95, 8000, 16384},
		{sdk.AccAddress([]byte("provider2")), 85, 4000, 8192},
		{sdk.AccAddress([]byte("provider3")), 75, 2000, 4096},
	}

	// Register providers
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, p := range providers {
		// Fund the provider account before registration
		fundTestAccount(t, k, sdkCtx, p.addr, "upaw", math.NewInt(2000000))

		specs := types.ComputeSpec{
			CpuCores:       p.cpuCores,
			MemoryMb:       p.memoryMb,
			StorageGb:      100,
			TimeoutSeconds: 3600,
		}
		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		}

		err := k.RegisterProvider(ctx, p.addr, "Provider", "http://localhost:8080", specs, pricing, math.NewInt(1000000))
		require.NoError(t, err)

		// Set reputation
		provider, err := k.GetProvider(ctx, p.addr)
		require.NoError(t, err)
		provider.Reputation = p.reputation
		err = k.SetProvider(ctx, *provider)
		require.NoError(t, err)
	}

	// Enable cache and refresh
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UseProviderCache = true
	params.ProviderCacheSize = 3
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	err = k.RefreshProviderCache(ctx)
	require.NoError(t, err)

	// Test finding provider with small specs (should match provider3)
	requestSpecs := types.ComputeSpec{
		CpuCores:       1000,
		MemoryMb:       2048,
		StorageGb:      50,
		TimeoutSeconds: 3600,
	}

	addr, err := k.FindSuitableProviderFromCache(ctx, requestSpecs)
	require.NoError(t, err)
	require.NotNil(t, addr)
	// Should select highest reputation provider that can handle specs (provider1)
	require.Equal(t, providers[0].addr.String(), addr.String())

	// Test finding provider with medium specs (should match provider2 or provider1)
	requestSpecs = types.ComputeSpec{
		CpuCores:       3000,
		MemoryMb:       6144,
		StorageGb:      50,
		TimeoutSeconds: 3600,
	}

	addr, err = k.FindSuitableProviderFromCache(ctx, requestSpecs)
	require.NoError(t, err)
	require.NotNil(t, addr)
	// Should select highest reputation provider that can handle specs (provider1)
	require.Equal(t, providers[0].addr.String(), addr.String())

	// Test finding provider with large specs (only provider1 can handle)
	requestSpecs = types.ComputeSpec{
		CpuCores:       6000,
		MemoryMb:       12288,
		StorageGb:      80,
		TimeoutSeconds: 3600,
	}

	addr, err = k.FindSuitableProviderFromCache(ctx, requestSpecs)
	require.NoError(t, err)
	require.NotNil(t, addr)
	require.Equal(t, providers[0].addr.String(), addr.String())

	// Test finding provider with specs too large (no provider can handle)
	requestSpecs = types.ComputeSpec{
		CpuCores:       10000,
		MemoryMb:       32768,
		StorageGb:      200,
		TimeoutSeconds: 3600,
	}

	_, err = k.FindSuitableProviderFromCache(ctx, requestSpecs)
	require.Error(t, err) // No suitable provider
}

// TestCacheInvalidation tests cache invalidation triggers
func TestCacheInvalidation(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Register a provider
	providerAddr := sdk.AccAddress([]byte("provider1"))

	// Fund the provider account before registration
	fundTestAccount(t, k, sdkCtx, providerAddr, "upaw", math.NewInt(2000000))

	specs := types.ComputeSpec{
		CpuCores:       4000,
		MemoryMb:       8192,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	pricing := types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(100),
		MemoryPricePerMbHour:  math.LegacyNewDec(10),
		GpuPricePerHour:       math.LegacyNewDec(1000),
		StoragePricePerGbHour: math.LegacyNewDec(5),
	}

	err := k.RegisterProvider(ctx, providerAddr, "Provider 1", "http://localhost:8080", specs, pricing, math.NewInt(1000000))
	require.NoError(t, err)

	// Enable cache and refresh
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UseProviderCache = true
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	err = k.RefreshProviderCache(ctx)
	require.NoError(t, err)

	// Verify cache is enabled
	metadata, err := k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.True(t, metadata.Enabled)

	// Update provider reputation - should invalidate cache
	err = k.UpdateProviderReputation(ctx, providerAddr, true)
	require.NoError(t, err)

	// Verify cache was invalidated
	metadata, err = k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.False(t, metadata.Enabled)

	// Re-enable cache
	err = k.RefreshProviderCache(ctx)
	require.NoError(t, err)

	// Deactivate provider - should invalidate cache
	err = k.DeactivateProvider(ctx, providerAddr)
	require.NoError(t, err)

	// Verify cache was invalidated
	metadata, err = k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.False(t, metadata.Enabled)
}

// TestClearProviderCache tests clearing the cache
func TestClearProviderCache(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create cached entries
	for i := uint32(0); i < 5; i++ {
		providerAddr := sdk.AccAddress([]byte{byte(i)})
		cached := types.CachedProvider{
			Provider:      providerAddr.String(),
			Reputation:    90 - i*5,
			CachedAtBlock: sdkCtx.BlockHeight(),
		}
		err := k.SetCachedProvider(ctx, i, cached)
		require.NoError(t, err)
	}

	// Set metadata
	metadata := &keeper.ProviderCacheMetadata{
		LastRefreshBlock: sdkCtx.BlockHeight(),
		CacheSize:        5,
		Enabled:          true,
	}
	err := k.SetProviderCacheMetadata(ctx, metadata)
	require.NoError(t, err)

	// Clear cache
	err = k.ClearProviderCache(ctx)
	require.NoError(t, err)

	// Verify all entries are deleted
	for i := uint32(0); i < 5; i++ {
		_, err := k.GetCachedProvider(ctx, i)
		require.Error(t, err)
	}

	// Verify metadata is reset
	metadata, err = k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.False(t, metadata.Enabled)
	require.Equal(t, uint32(0), metadata.CacheSize)
}

// TestIterateCachedProviders tests iterating over cached providers
func TestIterateCachedProviders(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create cached entries
	expectedProviders := []types.CachedProvider{}
	for i := uint32(0); i < 3; i++ {
		providerAddr := sdk.AccAddress([]byte{byte(i)})
		cached := types.CachedProvider{
			Provider:      providerAddr.String(),
			Reputation:    90 - i*5,
			CachedAtBlock: sdkCtx.BlockHeight(),
		}
		err := k.SetCachedProvider(ctx, i, cached)
		require.NoError(t, err)
		expectedProviders = append(expectedProviders, cached)
	}

	// Set metadata
	metadata := &keeper.ProviderCacheMetadata{
		LastRefreshBlock: sdkCtx.BlockHeight(),
		CacheSize:        3,
		Enabled:          true,
	}
	err := k.SetProviderCacheMetadata(ctx, metadata)
	require.NoError(t, err)

	// Iterate and verify
	count := 0
	err = k.IterateCachedProviders(ctx, func(index uint32, cached types.CachedProvider) (bool, error) {
		require.Equal(t, expectedProviders[index].Provider, cached.Provider)
		require.Equal(t, expectedProviders[index].Reputation, cached.Reputation)
		count++
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

// TestGetCacheStats tests retrieving cache statistics
func TestGetCacheStats(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set metadata
	metadata := &keeper.ProviderCacheMetadata{
		LastRefreshBlock: sdkCtx.BlockHeight() - 50,
		CacheSize:        10,
		Enabled:          true,
	}
	err := k.SetProviderCacheMetadata(ctx, metadata)
	require.NoError(t, err)

	// Get stats
	stats, err := k.GetCacheStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Equal(t, true, stats["enabled"])
	require.Equal(t, uint32(10), stats["cache_size"])
	require.Equal(t, sdkCtx.BlockHeight()-50, stats["last_refresh_block"])
	require.Equal(t, sdkCtx.BlockHeight(), stats["current_block"])
	require.Equal(t, int64(50), stats["blocks_since_refresh"])
}

// TestCacheWithMinReputationFilter tests that cache only includes providers above min reputation
func TestCacheWithMinReputationFilter(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create providers with varying reputations (some below min threshold)
	providers := []struct {
		addr       sdk.AccAddress
		reputation uint32
	}{
		{sdk.AccAddress([]byte("provider1")), 95},
		{sdk.AccAddress([]byte("provider2")), 75},
		{sdk.AccAddress([]byte("provider3")), 55},
		{sdk.AccAddress([]byte("provider4")), 45}, // Below min (50)
		{sdk.AccAddress([]byte("provider5")), 30}, // Below min (50)
	}

	// Register providers
	for _, p := range providers {
		// Fund the provider account before registration
		fundTestAccount(t, k, sdkCtx, p.addr, "upaw", math.NewInt(2000000))

		specs := types.ComputeSpec{
			CpuCores:       4000,
			MemoryMb:       8192,
			StorageGb:      100,
			TimeoutSeconds: 3600,
		}
		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		}

		err := k.RegisterProvider(ctx, p.addr, "Provider", "http://localhost:8080", specs, pricing, math.NewInt(1000000))
		require.NoError(t, err)

		// Set reputation
		provider, err := k.GetProvider(ctx, p.addr)
		require.NoError(t, err)
		provider.Reputation = p.reputation
		err = k.SetProvider(ctx, *provider)
		require.NoError(t, err)
	}

	// Set params with min reputation score
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.UseProviderCache = true
	params.ProviderCacheSize = 10
	params.MinReputationScore = 50
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Refresh cache
	err = k.RefreshProviderCache(ctx)
	require.NoError(t, err)

	// Verify only 3 providers are cached (those above min reputation)
	metadata, err := k.GetProviderCacheMetadata(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(3), metadata.CacheSize)

	// Verify cached providers have reputation >= 50
	for i := uint32(0); i < metadata.CacheSize; i++ {
		cached, err := k.GetCachedProvider(ctx, i)
		require.NoError(t, err)
		require.GreaterOrEqual(t, cached.Reputation, uint32(50))
	}
}

// BenchmarkProviderSelectionCache benchmarks provider selection with cache
func BenchmarkProviderSelectionCache(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Create 100 providers
	for i := 0; i < 100; i++ {
		providerAddr := sdk.AccAddress([]byte{byte(i)})
		specs := types.ComputeSpec{
			CpuCores:       4000,
			MemoryMb:       8192,
			StorageGb:      100,
			TimeoutSeconds: 3600,
		}
		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		}

		_ = k.RegisterProvider(ctx, providerAddr, "Provider", "http://localhost:8080", specs, pricing, math.NewInt(1000000))
	}

	// Enable cache
	params, _ := k.GetParams(ctx)
	params.UseProviderCache = true
	params.ProviderCacheSize = 10
	_ = k.SetParams(ctx, params)
	_ = k.RefreshProviderCache(ctx)

	requestSpecs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		StorageGb:      50,
		TimeoutSeconds: 3600,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = k.FindSuitableProvider(ctx, requestSpecs, "")
	}
}

// BenchmarkProviderSelectionNoCache benchmarks provider selection without cache
func BenchmarkProviderSelectionNoCache(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Create 100 providers
	for i := 0; i < 100; i++ {
		providerAddr := sdk.AccAddress([]byte{byte(i)})
		specs := types.ComputeSpec{
			CpuCores:       4000,
			MemoryMb:       8192,
			StorageGb:      100,
			TimeoutSeconds: 3600,
		}
		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		}

		_ = k.RegisterProvider(ctx, providerAddr, "Provider", "http://localhost:8080", specs, pricing, math.NewInt(1000000))
	}

	// Disable cache
	params, _ := k.GetParams(ctx)
	params.UseProviderCache = false
	_ = k.SetParams(ctx, params)

	requestSpecs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		StorageGb:      50,
		TimeoutSeconds: 3600,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = k.FindSuitableProvider(ctx, requestSpecs, "")
	}
}
