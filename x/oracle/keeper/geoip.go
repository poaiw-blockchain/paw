package keeper

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPManager manages IP geolocation using MaxMind GeoLite2 database
// This provides deterministic IP-to-region mapping using a local database
// instead of external API calls, ensuring consensus compatibility
type GeoIPManager struct {
	mu     sync.RWMutex
	reader *geoip2.Reader
	dbPath string
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

	return &GeoIPManager{
		reader: reader,
		dbPath: dbPath,
	}, nil
}

// LookupCountry returns the ISO country code for an IP address
func (g *GeoIPManager) LookupCountry(ipStr string) (string, error) {
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
		return "private", nil
	}

	record, err := g.reader.Country(ip)
	if err != nil {
		return "", fmt.Errorf("GeoIP lookup failed: %w", err)
	}

	if record.Country.IsoCode == "" {
		return "unknown", nil
	}

	return record.Country.IsoCode, nil
}

// LookupContinent returns the continent code for an IP address
func (g *GeoIPManager) LookupContinent(ipStr string) (string, error) {
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
		return "private", nil
	}

	record, err := g.reader.Country(ip)
	if err != nil {
		return "", fmt.Errorf("GeoIP lookup failed: %w", err)
	}

	if record.Continent.Code == "" {
		return "unknown", nil
	}

	return record.Continent.Code, nil
}

// GetRegion returns a normalized region identifier for an IP address
// Maps continent codes to our standard region names
func (g *GeoIPManager) GetRegion(ipStr string) (string, error) {
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
