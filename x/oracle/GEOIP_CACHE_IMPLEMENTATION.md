# GeoIP Cache Implementation (P2-PERF-1)

## Problem
No caching for GeoIP lookups causing file I/O on every validation. With 100 validators, each aggregation required 100 separate database file reads, causing significant performance overhead.

## Solution
Implemented in-memory LRU cache with TTL for GeoIP lookup results.

### Files Modified
- `/x/oracle/keeper/geoip.go` - Integrated cache into GeoIPManager
- `/x/oracle/keeper/keeper.go` - Added cache config update method
- `/x/oracle/keeper/abci.go` - Added cache pruning to BeginBlocker
- `/proto/paw/oracle/v1/oracle.proto` - Added cache governance params
- `/x/oracle/types/params.go` - Added default cache parameters

### Files Created
- `/x/oracle/keeper/geoip_cache.go` - Cache implementation
- `/x/oracle/keeper/geoip_cache_test.go` - Comprehensive unit tests
- `/x/oracle/keeper/geoip_integration_test.go` - Integration tests

## Architecture

### GeoIPCache Structure
```go
type GeoIPCache struct {
    mu         sync.RWMutex
    entries    map[string]*list.Element  // IP -> list element
    accessList *list.List                // LRU ordering
    maxEntries int                       // Max cache size
    ttl        time.Duration             // Entry TTL
    hits       uint64                    // Hit counter
    misses     uint64                    // Miss counter
}
```

### Cache Entry
```go
type GeoIPCacheEntry struct {
    IPAddress string
    Region    string    // e.g., "north_america"
    Country   string    // e.g., "US"
    Timestamp time.Time // For TTL expiration
}
```

## Features

### 1. LRU Eviction
- Cache maintains max size (default: 1000 entries)
- Least recently used entries evicted when full
- Access updates LRU position

### 2. TTL Expiration
- Entries expire after configurable TTL (default: 1 hour)
- Lazy expiration on access
- Periodic pruning via BeginBlocker (every 100 blocks)

### 3. Thread-Safe
- Uses RWMutex for concurrent access
- Safe for multiple goroutines
- Separate locks for cache vs GeoIP database reader

### 4. Governance Parameters
```proto
// geoip_cache_ttl_seconds - cache entry TTL
uint64 geoip_cache_ttl_seconds = 19;

// geoip_cache_max_entries - max cached IPs
uint64 geoip_cache_max_entries = 20;
```

Default values:
- TTL: 3600 seconds (1 hour)
- Max entries: 1000

### 5. Cache Metrics
- Hit/miss counters
- Hit rate calculation
- Size tracking
- String representation for debugging

## Performance Impact

### Before (No Cache)
- 100 validators × 1 aggregation = 100 file I/O operations
- Each lookup: ~1-5ms (disk I/O dependent)
- Total per aggregation: ~100-500ms

### After (With Cache)
- First lookup: ~1-5ms (cache miss, file I/O)
- Subsequent lookups: ~50-100ns (cache hit, in-memory)
- Total per aggregation (hot cache): ~5-10μs

**Improvement: ~10,000x faster for cached lookups**

### Memory Footprint
- Per entry: ~134 bytes
  - IP string: ~15 bytes
  - Region string: ~13 bytes
  - Country string: ~2 bytes
  - Timestamp: 24 bytes
  - Map overhead: ~40 bytes
  - List overhead: ~40 bytes

- 1000 entries: ~130 KB
- Negligible impact on validator memory

## Cache Management

### Automatic Pruning
BeginBlocker prunes expired entries every 100 blocks (~10 minutes):
```go
if sdkCtx.BlockHeight()%100 == 0 {
    pruned := k.geoIPManager.PruneCacheExpired()
    // Emit event for monitoring
}
```

### Manual Management
```go
// Clear entire cache
k.geoIPManager.ClearCache()

// Prune expired entries
pruned := k.geoIPManager.PruneCacheExpired()

// Get cache statistics
hits, misses, size, hitRate := k.geoIPManager.GetCacheStats()

// Update cache config
k.geoIPManager.SetCacheTTL(30 * time.Minute)
k.geoIPManager.SetCacheMaxEntries(2000)
```

### Governance Updates
Cache configuration can be updated via governance:
```go
func (k Keeper) UpdateGeoIPCacheConfig(ctx context.Context) error {
    params, _ := k.GetParams(ctx)
    k.geoIPManager.SetCacheTTL(time.Duration(params.GeoipCacheTtlSeconds) * time.Second)
    k.geoIPManager.SetCacheMaxEntries(int(params.GeoipCacheMaxEntries))
    return nil
}
```

## Testing

### Unit Tests (13 tests, all passing)
- `TestGeoIPCacheBasics` - Set, get, update
- `TestGeoIPCacheTTLExpiration` - TTL-based expiration
- `TestGeoIPCacheLRUEviction` - LRU eviction when full
- `TestGeoIPCacheLRUOrdering` - Access updates LRU
- `TestGeoIPCachePruneExpired` - Manual pruning
- `TestGeoIPCacheClear` - Cache clearing
- `TestGeoIPCacheStats` - Hit/miss tracking
- `TestGeoIPCacheConcurrency` - Thread safety
- `TestGeoIPCacheSetMaxEntries` - Dynamic size adjustment
- `TestGeoIPCacheSetTTL` - Dynamic TTL adjustment
- `TestGeoIPCacheGetAllEntries` - Entry enumeration
- `TestGeoIPCacheString` - String representation
- `TestGeoIPCacheDefaultParameters` - Default handling

### Integration Tests
- `TestGeoIPManagerCacheIntegration` - Cache integration
- `TestGeoIPCachePerformance` - Performance benchmarks
- `TestGeoIPCacheMemoryFootprint` - Memory usage
- `TestGeoIPCacheConcurrentHitRate` - Concurrent hit rate

## Security Considerations

### 1. Cache Poisoning Prevention
- Cache stores deterministic database results only
- No external API calls cached
- Cache can be cleared if database is updated

### 2. DoS Prevention
- Bounded cache size prevents memory exhaustion
- LRU eviction ensures oldest entries removed
- TTL prevents stale data accumulation

### 3. Race Condition Protection
- RWMutex ensures thread safety
- Separate cache lock from GeoIP reader lock
- Atomic operations for stats updates

## Monitoring

### Events Emitted
```go
sdk.NewEvent(
    "geoip_cache_pruned",
    sdk.NewAttribute("count", fmt.Sprintf("%d", pruned)),
    sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
)
```

### Metrics (via GetCacheStats)
- `hits` - Total cache hits
- `misses` - Total cache misses
- `size` - Current entries
- `hitRate` - Hit rate percentage

## Future Enhancements

1. **ASN Caching**: Cache ASN lookups separately
2. **Warm-up**: Pre-populate cache with known validator IPs
3. **Persistence**: Optional cache persistence across restarts
4. **Metrics Export**: Prometheus metrics for monitoring
5. **Smart Eviction**: Evict based on access frequency, not just recency
