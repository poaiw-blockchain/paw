package keeper

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// GeoIPCacheEntry stores cached GeoIP lookup result with timestamp
type GeoIPCacheEntry struct {
	IPAddress string
	Region    string
	Country   string
	Timestamp time.Time
}

// GeoIPCache implements an in-memory LRU cache with TTL for GeoIP lookups
// Thread-safe using RWMutex for concurrent access
type GeoIPCache struct {
	mu sync.RWMutex

	// LRU tracking
	entries    map[string]*list.Element // IP -> list element
	accessList *list.List               // LRU ordering (most recent at front)

	// Cache configuration
	maxEntries int           // Maximum cache entries before eviction
	ttl        time.Duration // Time-to-live for cache entries

	// Metrics
	hits   uint64 // Cache hits
	misses uint64 // Cache misses
}

// lruEntry wraps GeoIPCacheEntry for LRU list
type lruEntry struct {
	key   string
	value GeoIPCacheEntry
}

// NewGeoIPCache creates a new GeoIP cache with specified configuration
func NewGeoIPCache(maxEntries int, ttl time.Duration) *GeoIPCache {
	if maxEntries <= 0 {
		maxEntries = 1000 // Default: 1000 entries
	}
	if ttl <= 0 {
		ttl = 1 * time.Hour // Default: 1 hour TTL
	}

	return &GeoIPCache{
		entries:    make(map[string]*list.Element),
		accessList: list.New(),
		maxEntries: maxEntries,
		ttl:        ttl,
		hits:       0,
		misses:     0,
	}
}

// Get retrieves a cached entry for the given IP address
// Returns (entry, true) if found and not expired
// Returns (empty, false) if not found or expired
func (c *GeoIPCache) Get(ipAddress string) (GeoIPCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.entries[ipAddress]
	if !exists {
		c.misses++
		return GeoIPCacheEntry{}, false
	}

	// Check if entry is expired
	entry := elem.Value.(*lruEntry).value
	if time.Since(entry.Timestamp) > c.ttl {
		// Expired - remove from cache
		c.removeElement(elem)
		c.misses++
		return GeoIPCacheEntry{}, false
	}

	// Move to front (most recently used)
	c.accessList.MoveToFront(elem)
	c.hits++
	return entry, true
}

// Set stores a GeoIP lookup result in the cache
// Evicts oldest entry if cache is full
func (c *GeoIPCache) Set(ipAddress, region, country string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if entry already exists
	if elem, exists := c.entries[ipAddress]; exists {
		// Update existing entry
		entry := elem.Value.(*lruEntry)
		entry.value = GeoIPCacheEntry{
			IPAddress: ipAddress,
			Region:    region,
			Country:   country,
			Timestamp: time.Now(),
		}
		c.accessList.MoveToFront(elem)
		return
	}

	// Add new entry
	entry := &lruEntry{
		key: ipAddress,
		value: GeoIPCacheEntry{
			IPAddress: ipAddress,
			Region:    region,
			Country:   country,
			Timestamp: time.Now(),
		},
	}

	elem := c.accessList.PushFront(entry)
	c.entries[ipAddress] = elem

	// Evict oldest entry if cache is full
	if c.accessList.Len() > c.maxEntries {
		c.evictOldest()
	}
}

// evictOldest removes the least recently used entry from the cache
// Must be called with lock held
func (c *GeoIPCache) evictOldest() {
	elem := c.accessList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes a specific element from the cache
// Must be called with lock held
func (c *GeoIPCache) removeElement(elem *list.Element) {
	c.accessList.Remove(elem)
	entry := elem.Value.(*lruEntry)
	delete(c.entries, entry.key)
}

// Clear removes all entries from the cache
func (c *GeoIPCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*list.Element)
	c.accessList.Init()
	c.hits = 0
	c.misses = 0
}

// PruneExpired removes all expired entries from the cache
// Returns the number of entries removed
func (c *GeoIPCache) PruneExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	pruned := 0

	// Iterate through all entries and remove expired ones
	for elem := c.accessList.Front(); elem != nil; {
		next := elem.Next()
		entry := elem.Value.(*lruEntry).value

		if now.Sub(entry.Timestamp) > c.ttl {
			c.removeElement(elem)
			pruned++
		}

		elem = next
	}

	return pruned
}

// GetStats returns cache statistics
func (c *GeoIPCache) GetStats() (hits, misses uint64, size int, hitRate float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits = c.hits
	misses = c.misses
	size = c.accessList.Len()

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return
}

// Size returns the current number of entries in the cache
func (c *GeoIPCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessList.Len()
}

// GetMaxEntries returns the maximum number of entries allowed
func (c *GeoIPCache) GetMaxEntries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxEntries
}

// GetTTL returns the cache TTL duration
func (c *GeoIPCache) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttl
}

// SetMaxEntries updates the maximum number of entries
// Evicts oldest entries if current size exceeds new limit
func (c *GeoIPCache) SetMaxEntries(maxEntries int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if maxEntries <= 0 {
		maxEntries = 1000
	}

	c.maxEntries = maxEntries

	// Evict excess entries if current size exceeds new limit
	for c.accessList.Len() > c.maxEntries {
		c.evictOldest()
	}
}

// SetTTL updates the cache TTL duration
func (c *GeoIPCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		ttl = 1 * time.Hour
	}

	c.ttl = ttl
}

// GetAllEntries returns a snapshot of all cache entries (for debugging/testing)
func (c *GeoIPCache) GetAllEntries() []GeoIPCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]GeoIPCacheEntry, 0, c.accessList.Len())
	for elem := c.accessList.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*lruEntry).value
		entries = append(entries, entry)
	}

	return entries
}

// String returns a human-readable cache status
func (c *GeoIPCache) String() string {
	hits, misses, size, hitRate := c.GetStats()
	return fmt.Sprintf("GeoIPCache[size=%d/%d, hits=%d, misses=%d, hitRate=%.2f%%, ttl=%s]",
		size, c.maxEntries, hits, misses, hitRate*100, c.ttl)
}
