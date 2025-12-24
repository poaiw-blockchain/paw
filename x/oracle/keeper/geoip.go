package keeper

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPManager manages IP geolocation using MaxMind GeoLite2 database
// This provides deterministic IP-to-region mapping using a local database
// instead of external API calls, ensuring consensus compatibility
type GeoIPManager struct {
	mu     sync.RWMutex
	reader *geoip2.Reader
	dbPath string
	cache  *GeoIPCache // LRU cache with TTL for lookup results
}

// NewGeoIPManager creates a new GeoIP manager
// dbPath should point to GeoLite2-Country.mmdb or similar MaxMind database
func NewGeoIPManager(dbPath string) (*GeoIPManager, error) {
	if dbPath == "" {
		// Default path - try common locations
		possiblePaths := []string{
			"/usr/share/GeoIP/GeoLite2-Country.mmdb",
			"/var/lib/GeoIP/GeoLite2-Country.mmdb",
			"./GeoLite2-Country.mmdb",
			os.Getenv("GEOIP_DB_PATH"),
		}

		for _, path := range possiblePaths {
			if path == "" {
				continue
			}
			if _, err := os.Stat(path); err == nil {
				dbPath = path
				break
			}
		}

		if dbPath == "" {
			return nil, fmt.Errorf("GeoIP database not found. Set GEOIP_DB_PATH or place GeoLite2-Country.mmdb in a standard location")
		}
	}

	// Verify database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("GeoIP database not found at %s", dbPath)
	}

	reader, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoIP database: %w", err)
	}

	// Initialize cache with default parameters (1000 entries, 1 hour TTL)
	// Can be configured via governance parameters
	cache := NewGeoIPCache(1000, 1*time.Hour)

	return &GeoIPManager{
		reader: reader,
		dbPath: dbPath,
		cache:  cache,
	}, nil
}

// LookupCountry returns the ISO country code for an IP address
// Uses cache if available, otherwise performs database lookup and caches result
func (g *GeoIPManager) LookupCountry(ipStr string) (string, error) {
	// Check cache first (cache handles its own locking)
	if g.cache != nil {
		if entry, found := g.cache.Get(ipStr); found {
			return entry.Country, nil
		}
	}

	// Cache miss - perform database lookup
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.reader == nil {
		return "", fmt.Errorf("GeoIP database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Handle localhost and private IPs
	if ip.IsLoopback() || ip.IsPrivate() {
		country := "private"
		// Cache the result
		if g.cache != nil {
			g.cache.Set(ipStr, "private", country)
		}
		return country, nil
	}

	record, err := g.reader.Country(ip)
	if err != nil {
		return "", fmt.Errorf("GeoIP lookup failed: %w", err)
	}

	country := record.Country.IsoCode
	if country == "" {
		country = "unknown"
	}

	// Cache the lookup result
	if g.cache != nil {
		// We need the region too for cache, so derive it here
		region := "unknown"
		if record.Continent.Code != "" {
			regionMap := map[string]string{
				"NA": "north_america",
				"SA": "south_america",
				"EU": "europe",
				"AS": "asia",
				"AF": "africa",
				"OC": "oceania",
				"AN": "antarctica",
			}
			if r, ok := regionMap[record.Continent.Code]; ok {
				region = r
			}
		}
		g.cache.Set(ipStr, region, country)
	}

	return country, nil
}

// LookupContinent returns the continent code for an IP address
// Uses cache if available, otherwise performs database lookup and caches result
func (g *GeoIPManager) LookupContinent(ipStr string) (string, error) {
	// Check cache first - derive continent from region
	if g.cache != nil {
		if entry, found := g.cache.Get(ipStr); found {
			// Convert region back to continent code
			reverseMap := map[string]string{
				"north_america": "NA",
				"south_america": "SA",
				"europe":        "EU",
				"asia":          "AS",
				"africa":        "AF",
				"oceania":       "OC",
				"antarctica":    "AN",
				"private":       "private",
				"unknown":       "unknown",
			}
			if continent, ok := reverseMap[entry.Region]; ok {
				return continent, nil
			}
		}
	}

	// Cache miss - perform database lookup
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.reader == nil {
		return "", fmt.Errorf("GeoIP database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Handle localhost and private IPs
	if ip.IsLoopback() || ip.IsPrivate() {
		continent := "private"
		// Cache the result
		if g.cache != nil {
			g.cache.Set(ipStr, "private", "private")
		}
		return continent, nil
	}

	record, err := g.reader.Country(ip)
	if err != nil {
		return "", fmt.Errorf("GeoIP lookup failed: %w", err)
	}

	continent := record.Continent.Code
	if continent == "" {
		continent = "unknown"
	}

	// Cache the result
	if g.cache != nil {
		region := "unknown"
		regionMap := map[string]string{
			"NA": "north_america",
			"SA": "south_america",
			"EU": "europe",
			"AS": "asia",
			"AF": "africa",
			"OC": "oceania",
			"AN": "antarctica",
		}
		if r, ok := regionMap[continent]; ok {
			region = r
		}
		country := record.Country.IsoCode
		if country == "" {
			country = "unknown"
		}
		g.cache.Set(ipStr, region, country)
	}

	return continent, nil
}

// GetRegion returns a normalized region identifier for an IP address
// Maps continent codes to our standard region names
// Uses cache if available, otherwise performs database lookup and caches result
func (g *GeoIPManager) GetRegion(ipStr string) (string, error) {
	// Check cache first
	if g.cache != nil {
		if entry, found := g.cache.Get(ipStr); found {
			return entry.Region, nil
		}
	}

	// Cache miss - use LookupContinent which will also cache the result
	continent, err := g.LookupContinent(ipStr)
	if err != nil {
		return "", err
	}

	// Map MaxMind continent codes to our standard region names
	regionMap := map[string]string{
		"NA": "north_america", // North America
		"SA": "south_america", // South America
		"EU": "europe",        // Europe
		"AS": "asia",          // Asia
		"AF": "africa",        // Africa
		"OC": "oceania",       // Oceania
		"AN": "antarctica",    // Antarctica (unlikely for validators)
	}

	if region, ok := regionMap[continent]; ok {
		return region, nil
	}

	// Handle special cases
	if continent == "private" || continent == "unknown" {
		return continent, nil
	}

	return "unknown", nil
}

// VerifyIPMatchesRegion checks if an IP address matches a claimed region
// Returns true if the IP is in the claimed region, false otherwise
func (g *GeoIPManager) VerifyIPMatchesRegion(ipStr string, claimedRegion string) (bool, error) {
	actualRegion, err := g.GetRegion(ipStr)
	if err != nil {
		return false, err
	}

	// Exact match
	if actualRegion == claimedRegion {
		return true, nil
	}

	// Handle special cases
	// Allow "unknown" region if we can't determine location
	if actualRegion == "unknown" && claimedRegion == "unknown" {
		return true, nil
	}

	// Private IPs should not be used for validators in production
	if actualRegion == "private" {
		return false, fmt.Errorf("validator cannot use private IP address")
	}

	return false, nil
}

// Close closes the GeoIP database reader
func (g *GeoIPManager) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.reader != nil {
		return g.reader.Close()
	}
	return nil
}

// ClearCache clears all cached GeoIP entries
func (g *GeoIPManager) ClearCache() {
	if g.cache != nil {
		g.cache.Clear()
	}
}

// PruneCacheExpired removes expired entries from the cache
// Returns the number of entries pruned
func (g *GeoIPManager) PruneCacheExpired() int {
	if g.cache != nil {
		return g.cache.PruneExpired()
	}
	return 0
}

// GetCacheStats returns cache statistics (hits, misses, size, hit rate)
func (g *GeoIPManager) GetCacheStats() (hits, misses uint64, size int, hitRate float64) {
	if g.cache != nil {
		return g.cache.GetStats()
	}
	return 0, 0, 0, 0.0
}

// GetCacheSize returns the current number of entries in the cache
func (g *GeoIPManager) GetCacheSize() int {
	if g.cache != nil {
		return g.cache.Size()
	}
	return 0
}

// SetCacheMaxEntries updates the maximum cache size
func (g *GeoIPManager) SetCacheMaxEntries(maxEntries int) {
	if g.cache != nil {
		g.cache.SetMaxEntries(maxEntries)
	}
}

// SetCacheTTL updates the cache TTL duration
func (g *GeoIPManager) SetCacheTTL(ttl time.Duration) {
	if g.cache != nil {
		g.cache.SetTTL(ttl)
	}
}

// GetCacheTTL returns the current cache TTL
func (g *GeoIPManager) GetCacheTTL() time.Duration {
	if g.cache != nil {
		return g.cache.GetTTL()
	}
	return 0
}

// IsValidIP checks if a string is a valid IP address
func IsValidIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil
}

// IsPublicIP checks if an IP address is publicly routable
func IsPublicIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return false
	}

	// Check for link-local addresses
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	return true
}

// GetCountryFromRegion extracts country code from IP (for detailed verification)
func (g *GeoIPManager) GetCountryFromRegion(ipStr string) (string, string, error) {
	country, err := g.LookupCountry(ipStr)
	if err != nil {
		return "", "", err
	}

	region, err := g.GetRegion(ipStr)
	if err != nil {
		return "", "", err
	}

	return country, region, nil
}
