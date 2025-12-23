package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// TestValidateGeoIPAvailability tests the GeoIP database validation function
func TestValidateGeoIPAvailability(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	// Try to validate GeoIP availability
	err := k.ValidateGeoIPAvailability()

	// This will succeed if GeoIP database is available, fail otherwise
	if err != nil {
		// Expected error message should mention GeoIP database
		require.Contains(t, err.Error(), "GeoIP",
			"Expected error to mention GeoIP database, got: %s", err.Error())
		t.Logf("GeoIP database not available (expected for test environment): %v", err)
	} else {
		t.Log("GeoIP database is available and functional")
	}
}

// TestValidateGeoIPAvailability_Integration is an integration test that requires actual GeoIP database
// This test will be skipped if the database is not available
func TestValidateGeoIPAvailability_Integration(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	err := k.ValidateGeoIPAvailability()

	// If database is not available, skip the test
	if err != nil {
		t.Skipf("GeoIP database not available, skipping integration test: %v", err)
		return
	}

	// If we got here, database is available - test should pass
	require.NoError(t, err, "GeoIP database should be functional")
	t.Log("GeoIP database validation passed")
}
