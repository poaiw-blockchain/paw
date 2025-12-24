package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// ProviderCacheMetadata stores metadata about the provider cache
type ProviderCacheMetadata struct {
	LastRefreshBlock int64  `json:"last_refresh_block"`
	CacheSize        uint32 `json:"cache_size"`
	Enabled          bool   `json:"enabled"`
}

// GetProviderCacheMetadata retrieves the cache metadata
func (k Keeper) GetProviderCacheMetadata(ctx context.Context) (*ProviderCacheMetadata, error) {
	store := k.getStore(ctx)
	bz := store.Get(ProviderCacheMetadataKey)

	if bz == nil {
		// Return default metadata if not found
		return &ProviderCacheMetadata{
			LastRefreshBlock: 0,
			CacheSize:        0,
			Enabled:          false,
		}, nil
	}

	var metadata ProviderCacheMetadata
	if err := json.Unmarshal(bz, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache metadata: %w", err)
	}

	return &metadata, nil
}

// SetProviderCacheMetadata stores the cache metadata
func (k Keeper) SetProviderCacheMetadata(ctx context.Context, metadata *ProviderCacheMetadata) error {
	store := k.getStore(ctx)

	bz, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal cache metadata: %w", err)
	}

	store.Set(ProviderCacheMetadataKey, bz)
	return nil
}

// GetCachedProvider retrieves a cached provider by index
func (k Keeper) GetCachedProvider(ctx context.Context, index uint32) (*types.CachedProvider, error) {
	store := k.getStore(ctx)
	bz := store.Get(ProviderCacheKey(index))

	if bz == nil {
		return nil, fmt.Errorf("cached provider not found at index %d", index)
	}

	var cached types.CachedProvider
	if err := k.cdc.Unmarshal(bz, &cached); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached provider: %w", err)
	}

	return &cached, nil
}

// SetCachedProvider stores a cached provider at index
func (k Keeper) SetCachedProvider(ctx context.Context, index uint32, cached types.CachedProvider) error {
	store := k.getStore(ctx)

	bz, err := k.cdc.Marshal(&cached)
	if err != nil {
		return fmt.Errorf("failed to marshal cached provider: %w", err)
	}

	store.Set(ProviderCacheKey(index), bz)
	return nil
}

// ClearProviderCache clears all cached provider entries
func (k Keeper) ClearProviderCache(ctx context.Context) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ProviderCacheKeyPrefix)
	defer iterator.Close()

	keysToDelete := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keysToDelete = append(keysToDelete, iterator.Key())
	}

	for _, key := range keysToDelete {
		store.Delete(key)
	}

	// Reset metadata
	metadata := &ProviderCacheMetadata{
		LastRefreshBlock: 0,
		CacheSize:        0,
		Enabled:          false,
	}
	return k.SetProviderCacheMetadata(ctx, metadata)
}

// RefreshProviderCache refreshes the provider cache with top N providers by reputation
// This is called periodically in BeginBlocker to maintain an up-to-date cache
func (k Keeper) RefreshProviderCache(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get params to check if cache is enabled
	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	if !params.UseProviderCache {
		// Cache is disabled, clear it if it exists
		metadata, _ := k.GetProviderCacheMetadata(ctx)
		if metadata != nil && metadata.Enabled {
			return k.ClearProviderCache(ctx)
		}
		return nil
	}

	// Collect all active providers with their reputations
	type providerReputation struct {
		address    sdk.AccAddress
		reputation uint32
	}

	providers := []providerReputation{}

	err = k.IterateActiveProviders(ctx, func(provider types.Provider) (bool, error) {
		addr, err := sdk.AccAddressFromBech32(provider.Address)
		if err != nil {
			return false, nil // Skip invalid address
		}

		// Only include providers meeting minimum reputation
		if provider.Reputation >= params.MinReputationScore {
			providers = append(providers, providerReputation{
				address:    addr,
				reputation: provider.Reputation,
			})
		}

		return false, nil // Continue iterating
	})

	if err != nil {
		return fmt.Errorf("failed to iterate providers: %w", err)
	}

	// Sort providers by reputation (descending)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].reputation > providers[j].reputation
	})

	// Determine cache size
	cacheSize := params.ProviderCacheSize
	if cacheSize == 0 {
		cacheSize = 10 // Default
	}

	// Limit cache to actual number of providers
	if uint32(len(providers)) < cacheSize {
		cacheSize = uint32(len(providers))
	}

	// Clear old cache entries
	if err := k.ClearProviderCache(ctx); err != nil {
		return fmt.Errorf("failed to clear old cache: %w", err)
	}

	// Store top N providers in cache
	for i := uint32(0); i < cacheSize && i < uint32(len(providers)); i++ {
		cached := types.CachedProvider{
			Provider:       providers[i].address.String(),
			Reputation:     providers[i].reputation,
			CachedAtBlock:  currentHeight,
		}

		if err := k.SetCachedProvider(ctx, i, cached); err != nil {
			return fmt.Errorf("failed to cache provider at index %d: %w", i, err)
		}
	}

	// Update metadata
	metadata := &ProviderCacheMetadata{
		LastRefreshBlock: currentHeight,
		CacheSize:        cacheSize,
		Enabled:          true,
	}

	if err := k.SetProviderCacheMetadata(ctx, metadata); err != nil {
		return fmt.Errorf("failed to update cache metadata: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_cache_refreshed",
			sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
			sdk.NewAttribute("cache_size", fmt.Sprintf("%d", cacheSize)),
			sdk.NewAttribute("total_eligible_providers", fmt.Sprintf("%d", len(providers))),
		),
	)

	return nil
}

// InvalidateProviderCache invalidates the cache by clearing it
// This should be called when provider state changes significantly
func (k Keeper) InvalidateProviderCache(ctx context.Context) error {
	return k.ClearProviderCache(ctx)
}

// ShouldRefreshCache determines if the cache should be refreshed based on params
func (k Keeper) ShouldRefreshCache(ctx context.Context) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}

	if !params.UseProviderCache {
		return false, nil
	}

	metadata, err := k.GetProviderCacheMetadata(ctx)
	if err != nil {
		return false, err
	}

	// Refresh if cache is not initialized
	if !metadata.Enabled || metadata.CacheSize == 0 {
		return true, nil
	}

	// Refresh based on interval
	refreshInterval := params.ProviderCacheRefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = 100 // Default
	}

	blocksSinceRefresh := currentHeight - metadata.LastRefreshBlock
	return blocksSinceRefresh >= refreshInterval, nil
}

// FindSuitableProviderFromCache attempts to find a suitable provider from the cache
// Returns nil if no suitable cached provider found (caller should fall back to full iteration)
func (k Keeper) FindSuitableProviderFromCache(ctx context.Context, specs types.ComputeSpec) (sdk.AccAddress, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Check if cache is enabled
	if !params.UseProviderCache {
		return nil, fmt.Errorf("cache disabled")
	}

	metadata, err := k.GetProviderCacheMetadata(ctx)
	if err != nil {
		return nil, err
	}

	if !metadata.Enabled || metadata.CacheSize == 0 {
		return nil, fmt.Errorf("cache not initialized")
	}

	// Iterate through cached providers (already sorted by reputation descending)
	for i := uint32(0); i < metadata.CacheSize; i++ {
		cached, err := k.GetCachedProvider(ctx, i)
		if err != nil {
			continue // Skip on error
		}

		providerAddr, err := sdk.AccAddressFromBech32(cached.Provider)
		if err != nil {
			continue // Skip invalid address
		}

		// Get full provider record to check specs
		provider, err := k.GetProvider(ctx, providerAddr)
		if err != nil {
			continue // Skip if provider not found
		}

		// Verify provider is still active and meets minimum reputation
		if !provider.Active {
			continue
		}

		if provider.Reputation < params.MinReputationScore {
			continue
		}

		// Check if provider can handle the requested specs
		if k.canProviderHandleSpecs(*provider, specs) {
			return providerAddr, nil
		}
	}

	// No suitable provider found in cache
	return nil, fmt.Errorf("no suitable provider in cache")
}

// IterateCachedProviders iterates over all cached providers
func (k Keeper) IterateCachedProviders(ctx context.Context, cb func(index uint32, cached types.CachedProvider) (stop bool, err error)) error {
	metadata, err := k.GetProviderCacheMetadata(ctx)
	if err != nil {
		return err
	}

	if !metadata.Enabled {
		return nil
	}

	for i := uint32(0); i < metadata.CacheSize; i++ {
		cached, err := k.GetCachedProvider(ctx, i)
		if err != nil {
			continue // Skip on error
		}

		stop, err := cb(i, *cached)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// GetCacheStats returns statistics about the provider cache
func (k Keeper) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	metadata, err := k.GetProviderCacheMetadata(ctx)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	stats := map[string]interface{}{
		"enabled":             metadata.Enabled,
		"cache_size":          metadata.CacheSize,
		"last_refresh_block":  metadata.LastRefreshBlock,
		"current_block":       currentHeight,
		"blocks_since_refresh": currentHeight - metadata.LastRefreshBlock,
	}

	return stats, nil
}
