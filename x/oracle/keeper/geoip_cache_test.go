package keeper

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGeoIPCacheBasics tests basic cache operations
func TestGeoIPCacheBasics(t *testing.T) {
	cache := NewGeoIPCache(100, 1*time.Hour)
	require.NotNil(t, cache)

	// Test initial state
	require.Equal(t, 0, cache.Size())
	require.Equal(t, 100, cache.GetMaxEntries())
	require.Equal(t, 1*time.Hour, cache.GetTTL())

	// Test cache miss
	entry, found := cache.Get("1.2.3.4")
	require.False(t, found)
	require.Equal(t, "", entry.IPAddress)

	// Test cache set and hit
	cache.Set("1.2.3.4", "north_america", "US")
	require.Equal(t, 1, cache.Size())

	entry, found = cache.Get("1.2.3.4")
	require.True(t, found)
	require.Equal(t, "1.2.3.4", entry.IPAddress)
	require.Equal(t, "north_america", entry.Region)
	require.Equal(t, "US", entry.Country)

	// Test cache update
	cache.Set("1.2.3.4", "europe", "GB")
	require.Equal(t, 1, cache.Size()) // Size should not increase

	entry, found = cache.Get("1.2.3.4")
	require.True(t, found)
	require.Equal(t, "europe", entry.Region)
	require.Equal(t, "GB", entry.Country)
}

// TestGeoIPCacheTTLExpiration tests TTL-based expiration
func TestGeoIPCacheTTLExpiration(t *testing.T) {
	// Use short TTL for testing
	cache := NewGeoIPCache(100, 100*time.Millisecond)

	// Add entry
	cache.Set("1.2.3.4", "north_america", "US")
	require.Equal(t, 1, cache.Size())

	// Verify entry exists
	entry, found := cache.Get("1.2.3.4")
	require.True(t, found)
	require.Equal(t, "US", entry.Country)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Entry should be expired and removed
	entry, found = cache.Get("1.2.3.4")
	require.False(t, found)
	require.Equal(t, 0, cache.Size())
}

// TestGeoIPCacheLRUEviction tests LRU eviction when cache is full
func TestGeoIPCacheLRUEviction(t *testing.T) {
	cache := NewGeoIPCache(3, 1*time.Hour) // Max 3 entries

	// Add 3 entries (fill cache)
	cache.Set("1.1.1.1", "north_america", "US")
	cache.Set("2.2.2.2", "europe", "GB")
	cache.Set("3.3.3.3", "asia", "JP")
	require.Equal(t, 3, cache.Size())

	// All should be present
	_, found := cache.Get("1.1.1.1")
	require.True(t, found)
	_, found = cache.Get("2.2.2.2")
	require.True(t, found)
	_, found = cache.Get("3.3.3.3")
	require.True(t, found)

	// Add 4th entry - should evict oldest (1.1.1.1)
	cache.Set("4.4.4.4", "oceania", "AU")
	require.Equal(t, 3, cache.Size())

	// 1.1.1.1 should be evicted
	_, found = cache.Get("1.1.1.1")
	require.False(t, found)

	// Others should still be present
	_, found = cache.Get("2.2.2.2")
	require.True(t, found)
	_, found = cache.Get("3.3.3.3")
	require.True(t, found)
	_, found = cache.Get("4.4.4.4")
	require.True(t, found)
}

// TestGeoIPCacheLRUOrdering tests that access updates LRU order
func TestGeoIPCacheLRUOrdering(t *testing.T) {
	cache := NewGeoIPCache(3, 1*time.Hour)

	// Add 3 entries
	cache.Set("1.1.1.1", "north_america", "US")
	cache.Set("2.2.2.2", "europe", "GB")
	cache.Set("3.3.3.3", "asia", "JP")

	// Access 1.1.1.1 to make it most recently used
	_, found := cache.Get("1.1.1.1")
	require.True(t, found)

	// Add 4th entry - should evict 2.2.2.2 (oldest)
	cache.Set("4.4.4.4", "oceania", "AU")

	// 2.2.2.2 should be evicted
	_, found = cache.Get("2.2.2.2")
	require.False(t, found)

	// 1.1.1.1 should still be present (was accessed)
	_, found = cache.Get("1.1.1.1")
	require.True(t, found)
}

// TestGeoIPCachePruneExpired tests manual pruning of expired entries
func TestGeoIPCachePruneExpired(t *testing.T) {
	cache := NewGeoIPCache(100, 100*time.Millisecond)

	// Add multiple entries
	cache.Set("1.1.1.1", "north_america", "US")
	cache.Set("2.2.2.2", "europe", "GB")
	cache.Set("3.3.3.3", "asia", "JP")
	require.Equal(t, 3, cache.Size())

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Add fresh entry (won't expire)
	cache.Set("4.4.4.4", "oceania", "AU")
	require.Equal(t, 4, cache.Size()) // All still in cache (not pruned yet)

	// Prune expired entries
	pruned := cache.PruneExpired()
	require.Equal(t, 3, pruned)
	require.Equal(t, 1, cache.Size())

	// Only fresh entry should remain
	_, found := cache.Get("4.4.4.4")
	require.True(t, found)
	_, found = cache.Get("1.1.1.1")
	require.False(t, found)
}

// TestGeoIPCacheClear tests clearing all cache entries
func TestGeoIPCacheClear(t *testing.T) {
	cache := NewGeoIPCache(100, 1*time.Hour)

	// Add entries
	for i := 1; i <= 10; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)
		cache.Set(ip, "north_america", "US")
	}
	require.Equal(t, 10, cache.Size())

	// Clear cache
	cache.Clear()
	require.Equal(t, 0, cache.Size())

	// Verify all entries are gone (don't check - would increment miss counter)
	// Stats should be reset
	hits, misses, size, hitRate := cache.GetStats()
	require.Equal(t, uint64(0), hits)
	require.Equal(t, uint64(0), misses)
	require.Equal(t, 0, size)
	require.Equal(t, 0.0, hitRate)

	// Now verify entries are actually gone
	for i := 1; i <= 10; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)
		_, found := cache.Get(ip)
		require.False(t, found)
	}
}

// TestGeoIPCacheStats tests cache statistics tracking
func TestGeoIPCacheStats(t *testing.T) {
	cache := NewGeoIPCache(100, 1*time.Hour)

	// Add entry
	cache.Set("1.1.1.1", "north_america", "US")

	// Hit
	_, found := cache.Get("1.1.1.1")
	require.True(t, found)

	// Miss
	_, found = cache.Get("2.2.2.2")
	require.False(t, found)

	// Another hit
	_, found = cache.Get("1.1.1.1")
	require.True(t, found)

	// Check stats
	hits, misses, size, hitRate := cache.GetStats()
	require.Equal(t, uint64(2), hits)
	require.Equal(t, uint64(1), misses)
	require.Equal(t, 1, size)
	require.Equal(t, 2.0/3.0, hitRate) // 2 hits out of 3 total accesses
}

// TestGeoIPCacheConcurrency tests thread-safe concurrent access
func TestGeoIPCacheConcurrency(t *testing.T) {
	cache := NewGeoIPCache(1000, 1*time.Hour)
	var wg sync.WaitGroup

	// Concurrent writes
	numWriters := 10
	writesPerWriter := 100

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				ip := fmt.Sprintf("%d.%d.%d.%d", writerID, j, 0, 0)
				cache.Set(ip, "north_america", "US")
			}
		}(i)
	}

	// Concurrent reads
	numReaders := 10
	readsPerReader := 100

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < readsPerReader; j++ {
				ip := fmt.Sprintf("%d.%d.%d.%d", readerID%numWriters, j, 0, 0)
				cache.Get(ip)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is in valid state (no crashes)
	require.True(t, cache.Size() > 0)
	require.True(t, cache.Size() <= 1000)
}

// TestGeoIPCacheSetMaxEntries tests dynamic cache size adjustment
func TestGeoIPCacheSetMaxEntries(t *testing.T) {
	cache := NewGeoIPCache(10, 1*time.Hour)

	// Fill cache
	for i := 1; i <= 10; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)
		cache.Set(ip, "north_america", "US")
	}
	require.Equal(t, 10, cache.Size())

	// Reduce max size - should evict oldest entries
	cache.SetMaxEntries(5)
	require.Equal(t, 5, cache.Size())

	// Only newest 5 should remain (6-10)
	for i := 1; i <= 5; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)
		_, found := cache.Get(ip)
		require.False(t, found, "Old entry %s should be evicted", ip)
	}
	for i := 6; i <= 10; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", i, i, i, i)
		_, found := cache.Get(ip)
		require.True(t, found, "New entry %s should remain", ip)
	}
}

// TestGeoIPCacheSetTTL tests dynamic TTL adjustment
func TestGeoIPCacheSetTTL(t *testing.T) {
	cache := NewGeoIPCache(100, 1*time.Hour)
	require.Equal(t, 1*time.Hour, cache.GetTTL())

	// Change TTL
	cache.SetTTL(30 * time.Minute)
	require.Equal(t, 30*time.Minute, cache.GetTTL())

	// Set invalid TTL (should default to 1 hour)
	cache.SetTTL(-1 * time.Second)
	require.Equal(t, 1*time.Hour, cache.GetTTL())

	cache.SetTTL(0)
	require.Equal(t, 1*time.Hour, cache.GetTTL())
}

// TestGeoIPCacheGetAllEntries tests retrieving all cache entries
func TestGeoIPCacheGetAllEntries(t *testing.T) {
	cache := NewGeoIPCache(100, 1*time.Hour)

	// Add entries
	cache.Set("1.1.1.1", "north_america", "US")
	cache.Set("2.2.2.2", "europe", "GB")
	cache.Set("3.3.3.3", "asia", "JP")

	// Get all entries
	entries := cache.GetAllEntries()
	require.Equal(t, 3, len(entries))

	// Verify entries (order is most recent first due to LRU)
	require.Equal(t, "3.3.3.3", entries[0].IPAddress)
	require.Equal(t, "2.2.2.2", entries[1].IPAddress)
	require.Equal(t, "1.1.1.1", entries[2].IPAddress)
}

// TestGeoIPCacheString tests string representation
func TestGeoIPCacheString(t *testing.T) {
	cache := NewGeoIPCache(1000, 1*time.Hour)

	// Add some entries
	cache.Set("1.1.1.1", "north_america", "US")
	cache.Set("2.2.2.2", "europe", "GB")

	// Access to generate hits/misses
	cache.Get("1.1.1.1") // hit
	cache.Get("3.3.3.3") // miss

	str := cache.String()
	require.Contains(t, str, "size=2/1000")
	require.Contains(t, str, "hits=1")
	require.Contains(t, str, "misses=1")
	require.Contains(t, str, "hitRate=50.00%")
	require.Contains(t, str, "ttl=1h")
}

// TestGeoIPCacheDefaultParameters tests default parameter handling
func TestGeoIPCacheDefaultParameters(t *testing.T) {
	// Test with invalid max entries (should default to 1000)
	cache := NewGeoIPCache(0, 1*time.Hour)
	require.Equal(t, 1000, cache.GetMaxEntries())

	cache = NewGeoIPCache(-10, 1*time.Hour)
	require.Equal(t, 1000, cache.GetMaxEntries())

	// Test with invalid TTL (should default to 1 hour)
	cache = NewGeoIPCache(100, 0)
	require.Equal(t, 1*time.Hour, cache.GetTTL())

	cache = NewGeoIPCache(100, -1*time.Second)
	require.Equal(t, 1*time.Hour, cache.GetTTL())
}
