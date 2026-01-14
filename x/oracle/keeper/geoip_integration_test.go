package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGeoIPManagerCacheIntegration tests cache integration with GeoIPManager
func TestGeoIPManagerCacheIntegration(t *testing.T) {
	// Note: These tests verify cache behavior without requiring actual GeoIP database

	t.Run("cache management methods", func(t *testing.T) {
		manager := &GeoIPManager{
			cache: NewGeoIPCache(100, 1*time.Hour),
		}

		// Test GetCacheStats
		hits, misses, size, hitRate := manager.GetCacheStats()
		require.Equal(t, uint64(0), hits)
		require.Equal(t, uint64(0), misses)
		require.Equal(t, 0, size)
		require.Equal(t, 0.0, hitRate)

		// Test GetCacheSize
		require.Equal(t, 0, manager.GetCacheSize())

		// Test GetCacheTTL
		require.Equal(t, 1*time.Hour, manager.GetCacheTTL())

		// Test SetCacheTTL
		manager.SetCacheTTL(30 * time.Minute)
		require.Equal(t, 30*time.Minute, manager.GetCacheTTL())

		// Test SetCacheMaxEntries
		manager.SetCacheMaxEntries(50)

		// Test ClearCache
		manager.cache.Set("1.1.1.1", "north_america", "US")
		require.Equal(t, 1, manager.GetCacheSize())
		manager.ClearCache()
		require.Equal(t, 0, manager.GetCacheSize())

		// Test PruneCacheExpired
		manager.cache.Set("2.2.2.2", "europe", "GB")
		pruned := manager.PruneCacheExpired()
		require.Equal(t, 0, pruned) // Nothing expired yet
	})

	t.Run("cache methods with nil cache", func(t *testing.T) {
		manager := &GeoIPManager{
			cache: nil, // No cache
		}

		// All methods should handle nil gracefully
		hits, misses, size, hitRate := manager.GetCacheStats()
		require.Equal(t, uint64(0), hits)
		require.Equal(t, uint64(0), misses)
		require.Equal(t, 0, size)
		require.Equal(t, 0.0, hitRate)

		require.Equal(t, 0, manager.GetCacheSize())
		require.Equal(t, time.Duration(0), manager.GetCacheTTL())
		manager.SetCacheTTL(1 * time.Hour) // Should not panic
		manager.SetCacheMaxEntries(100)    // Should not panic
		manager.ClearCache()               // Should not panic
		require.Equal(t, 0, manager.PruneCacheExpired())
	})
}

// TestGeoIPCachePerformance benchmarks cache hit vs miss performance
func TestGeoIPCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cache := NewGeoIPCache(10000, 1*time.Hour)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		ip := genTestIP(i)
		cache.Set(ip, "north_america", "US")
	}

	// Measure cache hits
	start := time.Now()
	for i := 0; i < 10000; i++ {
		ip := genTestIP(i % 1000) // Cycle through cached IPs
		cache.Get(ip)
	}
	hitDuration := time.Since(start)

	// Measure cache misses
	start = time.Now()
	for i := 0; i < 10000; i++ {
		ip := genTestIP(i + 1000) // Different IPs (not cached)
		cache.Get(ip)
	}
	missDuration := time.Since(start)

	t.Logf("Cache hit duration: %v (10000 ops)", hitDuration)
	t.Logf("Cache miss duration: %v (10000 ops)", missDuration)
	t.Logf("Average hit latency: %v", hitDuration/10000)
	t.Logf("Average miss latency: %v", missDuration/10000)

	// Hits should be fast (both are fast since no actual DB lookup)
	// This mainly verifies no performance regression in cache logic
	require.True(t, hitDuration < 100*time.Millisecond, "Cache hits too slow")
	require.True(t, missDuration < 100*time.Millisecond, "Cache misses too slow")
}

// genTestIP generates a test IP address from an integer
func genTestIP(n int) string {
	a := (n >> 24) & 0xFF
	b := (n >> 16) & 0xFF
	c := (n >> 8) & 0xFF
	d := n & 0xFF
	return formatIP(a, b, c, d)
}

// formatIP formats IP components into a string (custom to avoid fmt overhead in tests)
func formatIP(a, b, c, d int) string {
	return string([]byte{
		byte('0' + (a/100)%10), byte('0' + (a/10)%10), byte('0' + a%10), '.',
		byte('0' + (b/100)%10), byte('0' + (b/10)%10), byte('0' + b%10), '.',
		byte('0' + (c/100)%10), byte('0' + (c/10)%10), byte('0' + c%10), '.',
		byte('0' + (d/100)%10), byte('0' + (d/10)%10), byte('0' + d%10),
	})
}

// TestGeoIPCacheMemoryFootprint estimates cache memory usage
func TestGeoIPCacheMemoryFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cache := NewGeoIPCache(10000, 1*time.Hour)

	// Fill cache with realistic data
	for i := 0; i < 1000; i++ {
		ip := genTestIP(i)
		cache.Set(ip, "north_america", "US")
	}

	// Estimate per-entry size
	// Each entry contains:
	// - IP string (~15 bytes)
	// - Region string (~13 bytes avg)
	// - Country string (~2 bytes)
	// - Timestamp (24 bytes)
	// - Map overhead (~40 bytes)
	// - List overhead (~40 bytes)
	// Total: ~134 bytes per entry
	estimatedSize := cache.Size() * 134

	t.Logf("Cache size: %d entries", cache.Size())
	t.Logf("Estimated memory: ~%d KB", estimatedSize/1024)
	t.Logf("Memory per entry: ~134 bytes")

	// With 1000 validators, cache should use <200KB
	require.True(t, estimatedSize < 200*1024, "Cache memory footprint too large")
}

// TestGeoIPCacheConcurrentHitRate tests hit rate under concurrent load
func TestGeoIPCacheConcurrentHitRate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	cache := NewGeoIPCache(1000, 1*time.Hour)

	// Pre-populate with 100 IPs
	for i := 0; i < 100; i++ {
		ip := genTestIP(i)
		cache.Set(ip, "north_america", "US")
	}

	// Concurrent access (80% hits, 20% misses)
	done := make(chan bool, 10)
	for worker := 0; worker < 10; worker++ {
		go func() {
			for i := 0; i < 1000; i++ {
				var ip string
				if i%5 == 0 {
					// 20% misses
					ip = genTestIP(i + 100)
				} else {
					// 80% hits
					ip = genTestIP(i % 100)
				}
				cache.Get(ip)
			}
			done <- true
		}()
	}

	// Wait for all workers
	for i := 0; i < 10; i++ {
		<-done
	}

	hits, misses, _, hitRate := cache.GetStats()
	t.Logf("Hits: %d, Misses: %d, Hit Rate: %.2f%%", hits, misses, hitRate*100)

	// Should have ~80% hit rate
	require.True(t, hitRate > 0.75 && hitRate < 0.85, "Hit rate should be ~80%%: got %.2f%%", hitRate*100)
}
