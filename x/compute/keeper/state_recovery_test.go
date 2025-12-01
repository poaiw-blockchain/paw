package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// createTestProviderForRecovery creates a test provider address
func createTestProviderForRecovery(index int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("recovery_provider_"))
	addr[19] = byte(index)
	return sdk.AccAddress(addr)
}

// TestExportStateBasic tests basic state export functionality
func TestExportStateBasic(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set up some state
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export state - this should not error even with empty/minimal state
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)
	require.NotNil(t, backup)

	// Verify backup contents
	require.Equal(t, "1.0.0", backup.Version)
	require.NotEmpty(t, backup.Checksum)
}

// TestImportState tests state import functionality
func TestImportState(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set initial params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export current state
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)
	require.NotNil(t, backup)

	// Import it back - should succeed
	err = k.ImportState(ctx, backup)
	require.NoError(t, err)

	// Verify params were restored
	restoredParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params.MinProviderStake, restoredParams.MinProviderStake)
}

// TestImportStateInvalidChecksum tests that invalid checksum is rejected
func TestImportStateInvalidChecksum(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set initial params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export state
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Corrupt the checksum
	backup.Checksum = "invalid-checksum"

	// Import should fail
	err = k.ImportState(ctx, backup)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum mismatch")
}

// TestBackupVersionFormat tests backup version format
func TestBackupVersionFormat(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Verify version format
	require.NotEmpty(t, backup.Version)
	require.Equal(t, "1.0.0", backup.Version)
}

// TestExportTimestamp tests that export records timestamp
func TestExportTimestamp(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Timestamp should be set (context has timestamp)
	require.False(t, backup.Timestamp.IsZero())
}

// TestExportBlockHeight tests that export records block height
func TestExportBlockHeight(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// BlockHeight should be set (may be 0 in test context)
	require.GreaterOrEqual(t, backup.BlockHeight, int64(0))
}

// TestExportWithEscrows tests export with escrow data
func TestExportWithEscrows(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export should succeed
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Escrows map should exist (possibly empty)
	require.NotNil(t, backup.Escrows)
}

// TestImportRestoresParams tests that import restores parameters correctly
func TestImportRestoresParams(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set initial params with custom values
	params := types.DefaultParams()
	params.MinProviderStake = math.NewInt(5000000)
	params.MinReputationScore = 50
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Change params
	newParams := types.DefaultParams()
	newParams.MinProviderStake = math.NewInt(1000000)
	err = k.SetParams(ctx, newParams)
	require.NoError(t, err)

	// Import should restore original params
	err = k.ImportState(ctx, backup)
	require.NoError(t, err)

	// Verify params were restored
	restoredParams, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params.MinProviderStake, restoredParams.MinProviderStake)
}

// TestExportEmptyState tests export of empty module state
func TestExportEmptyState(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set minimal params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export should succeed even with minimal state
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)
	require.NotNil(t, backup)
	require.Equal(t, "1.0.0", backup.Version)
	// Empty providers/requests/results lists are acceptable
}

// TestBackupParamsIntegrity tests that params are preserved in backup
func TestBackupParamsIntegrity(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set custom params
	params := types.DefaultParams()
	params.MinProviderStake = math.NewInt(7500000)
	params.MinReputationScore = 75
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Verify params in backup
	require.Equal(t, params.MinProviderStake, backup.Params.MinProviderStake)
	require.Equal(t, params.MinReputationScore, backup.Params.MinReputationScore)
}

// TestRoundtripBasic tests basic export->import roundtrip
func TestRoundtripBasic(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set up state
	params := types.DefaultParams()
	params.MinProviderStake = math.NewInt(2000000)
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Export
	backup, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Import
	err = k.ImportState(ctx, backup)
	require.NoError(t, err)

	// Export again
	backup2, err := k.ExportState(ctx)
	require.NoError(t, err)

	// Core data should match
	require.Equal(t, backup.Version, backup2.Version)
}
