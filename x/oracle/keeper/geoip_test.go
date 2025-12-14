package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/keeper"
)

// TestIsValidIP tests IP address validation
func TestIsValidIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "8.8.8.8", true},
		{"valid IPv4 private", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv6", "2001:4860:4860::8888", true},
		{"valid IPv6 localhost", "::1", true},
		{"invalid IP empty", "", false},
		{"invalid IP text", "not.an.ip", false},
		{"invalid IP partial", "192.168.1", false},
		{"invalid IP overflow", "256.256.256.256", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := keeper.IsValidIP(tt.ip)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestIsPublicIP tests public IP detection
func TestIsPublicIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"public IPv4 Google DNS", "8.8.8.8", true},
		{"public IPv4 Cloudflare", "1.1.1.1", true},
		{"public IPv4 OpenDNS", "208.67.222.222", true},
		{"private IPv4 class A", "10.0.0.1", false},
		{"private IPv4 class B", "172.16.0.1", false},
		{"private IPv4 class C", "192.168.1.1", false},
		{"localhost IPv4", "127.0.0.1", false},
		{"localhost IPv6", "::1", false},
		{"link-local IPv4", "169.254.1.1", false},
		{"link-local IPv6", "fe80::1", false},
		{"unspecified IPv4", "0.0.0.0", false},
		{"unspecified IPv6", "::", false},
		{"invalid IP", "not.an.ip", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := keeper.IsPublicIP(tt.ip)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestGeoIPManager tests the GeoIP manager functionality
// Note: These tests will be skipped if GeoIP database is not available
func TestGeoIPManager(t *testing.T) {
	t.Parallel()

	// Try to create GeoIP manager
	manager, err := keeper.NewGeoIPManager("")
	if err != nil {
		t.Skipf("GeoIP database not available: %v", err)
		return
	}
	defer manager.Close()

	t.Run("lookup known public IP", func(t *testing.T) {
		// Google DNS (US)
		country, err := manager.LookupCountry("8.8.8.8")
		require.NoError(t, err)
		// Note: Exact country may vary based on GeoIP database version
		require.NotEmpty(t, country)
	})

	t.Run("lookup localhost returns private", func(t *testing.T) {
		country, err := manager.LookupCountry("127.0.0.1")
		require.NoError(t, err)
		require.Equal(t, "private", country)
	})

	t.Run("lookup private IP returns private", func(t *testing.T) {
		country, err := manager.LookupCountry("192.168.1.1")
		require.NoError(t, err)
		require.Equal(t, "private", country)
	})

	t.Run("lookup invalid IP returns error", func(t *testing.T) {
		_, err := manager.LookupCountry("not.an.ip")
		require.Error(t, err)
	})

	t.Run("get region for public IP", func(t *testing.T) {
		// Google DNS should resolve to a valid region
		region, err := manager.GetRegion("8.8.8.8")
		require.NoError(t, err)
		require.NotEmpty(t, region)
		// Should be one of the valid regions
		validRegions := []string{
			"north_america", "south_america", "europe",
			"asia", "africa", "oceania", "antarctica",
		}
		require.Contains(t, validRegions, region)
	})

	t.Run("verify IP matches claimed region", func(t *testing.T) {
		// Get actual region first
		actualRegion, err := manager.GetRegion("8.8.8.8")
		require.NoError(t, err)

		// Verify matches
		matches, err := manager.VerifyIPMatchesRegion("8.8.8.8", actualRegion)
		require.NoError(t, err)
		require.True(t, matches)
	})

	t.Run("verify IP does not match wrong region", func(t *testing.T) {
		// Google DNS is in North America
		// Try to claim it's in Asia (should fail)
		actualRegion, err := manager.GetRegion("8.8.8.8")
		require.NoError(t, err)

		// Pick a different region
		wrongRegion := "asia"
		if actualRegion == "asia" {
			wrongRegion = "europe"
		}

		matches, err := manager.VerifyIPMatchesRegion("8.8.8.8", wrongRegion)
		if err == nil {
			// If no error, should not match
			require.False(t, matches)
		}
		// Some GeoIP databases may return error for mismatch
	})

	t.Run("get country and region together", func(t *testing.T) {
		country, region, err := manager.GetCountryFromRegion("8.8.8.8")
		require.NoError(t, err)
		require.NotEmpty(t, country)
		require.NotEmpty(t, region)
	})
}

// TestGeoIPManagerEdgeCases tests edge cases and error handling
func TestGeoIPManagerEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("create manager with invalid path", func(t *testing.T) {
		_, err := keeper.NewGeoIPManager("/invalid/path/to/database.mmdb")
		require.Error(t, err)
	})

	t.Run("create manager with empty path uses defaults", func(t *testing.T) {
		// Should try default paths, may succeed or fail depending on system
		_, err := keeper.NewGeoIPManager("")
		// Error is acceptable if no default database found
		if err != nil {
			require.Contains(t, err.Error(), "GeoIP database not found")
		}
	})
}

// BenchmarkGeoIPLookup benchmarks GeoIP lookup performance
func BenchmarkGeoIPLookup(b *testing.B) {
	manager, err := keeper.NewGeoIPManager("")
	if err != nil {
		b.Skipf("GeoIP database not available: %v", err)
		return
	}
	defer manager.Close()

	b.Run("lookup country", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = manager.LookupCountry("8.8.8.8")
		}
	})

	b.Run("lookup continent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = manager.LookupContinent("8.8.8.8")
		}
	})

	b.Run("get region", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = manager.GetRegion("8.8.8.8")
		}
	})

	b.Run("verify region match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = manager.VerifyIPMatchesRegion("8.8.8.8", "north_america")
		}
	})
}
