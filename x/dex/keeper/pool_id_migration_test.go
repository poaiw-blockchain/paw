package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestPoolIDCollisionAfterMigration tests that pool ID sequences don't collide
// after a genesis import/export migration scenario. This ensures that the NextPoolID
// counter is properly preserved and incremented across chain upgrades.
//
// SCENARIO:
// 1. Create multiple pools on original chain (IDs: 1, 2, 3)
// 2. Export genesis state (NextPoolID should be 4)
// 3. Import genesis state into new chain
// 4. Create new pools on upgraded chain
// 5. Verify no ID collisions occur (new pools get IDs: 4, 5, 6...)
//
// This test addresses P1-DATA-2 from ROADMAP_PRODUCTION.md
func TestPoolIDCollisionAfterMigration(t *testing.T) {
	// Phase 1: Original chain - create pools
	k1, ctx1 := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create 3 pools on original chain
	pool1, err := k1.CreatePool(ctx1, creator, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(1), pool1.Id)

	pool2, err := k1.CreatePool(ctx1, creator, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(2), pool2.Id)

	pool3, err := k1.CreatePool(ctx1, creator, "uatom", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(3), pool3.Id)

	// Phase 2: Export genesis from original chain
	exportedGenesis, err := k1.ExportGenesis(ctx1)
	require.NoError(t, err)
	require.Len(t, exportedGenesis.Pools, 3, "Should have exported 3 pools")
	// Verify NextPoolID is greater than the highest pool ID created
	require.Greater(t, exportedGenesis.NextPoolId, pool3.Id,
		"NextPoolID should be greater than last created pool ID")

	// Verify exported pools have correct IDs
	poolIDs := make(map[uint64]bool)
	for _, pool := range exportedGenesis.Pools {
		poolIDs[pool.Id] = true
	}
	require.True(t, poolIDs[1] && poolIDs[2] && poolIDs[3],
		"Exported pools should have IDs 1, 2, 3")

	// Phase 3: Import genesis into new chain (simulating upgrade)
	k2, ctx2 := keepertest.DexKeeper(t)
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Verify imported pools exist with correct IDs
	importedPool1, err := k2.GetPool(ctx2, 1)
	require.NoError(t, err)
	require.Equal(t, pool1.TokenA, importedPool1.TokenA)
	require.Equal(t, pool1.TokenB, importedPool1.TokenB)

	importedPool2, err := k2.GetPool(ctx2, 2)
	require.NoError(t, err)
	require.Equal(t, pool2.TokenA, importedPool2.TokenA)

	importedPool3, err := k2.GetPool(ctx2, 3)
	require.NoError(t, err)
	require.Equal(t, pool3.TokenA, importedPool3.TokenA)

	// Phase 4: Create new pools on upgraded chain
	// These should get sequential IDs starting from NextPoolID (no collisions with existing pools)
	nextID := exportedGenesis.NextPoolId
	pool4, err := k2.CreatePool(ctx2, creator, "upaw", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool4.Id, pool3.Id, "New pool ID should be greater than existing pool IDs")
	require.NotEqual(t, pool1.Id, pool4.Id, "No collision with pool1")
	require.NotEqual(t, pool2.Id, pool4.Id, "No collision with pool2")
	require.NotEqual(t, pool3.Id, pool4.Id, "No collision with pool3")
	firstNewPoolID := pool4.Id

	pool5, err := k2.CreatePool(ctx2, creator, "uatom", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool5.Id, pool4.Id, "Pool IDs should increment")

	pool6, err := k2.CreatePool(ctx2, creator, "uusdc", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool6.Id, pool5.Id, "Pool IDs should increment")

	_ = nextID // Use this to avoid unused variable

	// Phase 5: Verify all pools are accessible and unique
	allPools, err := k2.GetAllPools(ctx2)
	require.NoError(t, err)
	require.Len(t, allPools, 6, "Should have 6 total pools")

	// Verify no duplicate IDs
	seenIDs := make(map[uint64]bool)
	maxID := uint64(0)
	for _, pool := range allPools {
		require.False(t, seenIDs[pool.Id], "Duplicate pool ID %d detected", pool.Id)
		seenIDs[pool.Id] = true
		require.Greater(t, pool.Id, uint64(0), "Pool ID must be positive")
		if pool.Id > maxID {
			maxID = pool.Id
		}
	}

	// Verify all expected pools exist (IDs 1, 2, 3 from original, plus 3 new ones)
	require.True(t, seenIDs[1], "Pool 1 should exist")
	require.True(t, seenIDs[2], "Pool 2 should exist")
	require.True(t, seenIDs[3], "Pool 3 should exist")
	require.True(t, seenIDs[pool4.Id], "Pool 4 should exist")
	require.True(t, seenIDs[pool5.Id], "Pool 5 should exist")
	require.True(t, seenIDs[pool6.Id], "Pool 6 should exist")

	// Final verification: Export again and check NextPoolID is greater than max pool ID
	finalGenesis, err := k2.ExportGenesis(ctx2)
	require.NoError(t, err)
	require.Greater(t, finalGenesis.NextPoolId, maxID,
		"NextPoolID should be greater than highest pool ID")

	_ = firstNewPoolID // Use to avoid unused variable
}

// TestPoolIDCollisionWithGaps tests that pool ID counter works correctly
// even when there are "gaps" in the pool ID sequence (e.g., if pools were deleted)
func TestPoolIDCollisionWithGaps(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create pools 1, 2, 3
	_, err := k.CreatePool(ctx, creator, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)

	_, err = k.CreatePool(ctx, creator, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)

	_, err = k.CreatePool(ctx, creator, "uatom", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)

	// Manually set NextPoolID to 10 (simulating some pools were deleted or never created)
	k.SetNextPoolId(ctx, 10)

	// Create new pool - should get ID 10, not 4
	pool10, err := k.CreatePool(ctx, creator, "upaw", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(10), pool10.Id,
		"Should use NextPoolID counter even with gaps in sequence")

	// Next pool should be 11
	pool11, err := k.CreatePool(ctx, creator, "uatom", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(11), pool11.Id)
}

// TestPoolIDMigrationWithLargePoolCount tests migration behavior with many pools
// to ensure counter doesn't overflow or reset unexpectedly
func TestPoolIDMigrationWithLargePoolCount(t *testing.T) {
	k1, ctx1 := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Set a high starting pool ID (simulating mature chain with many pools)
	k1.SetNextPoolId(ctx1, 1000)

	// Create pools starting from ID 1000
	pool1000, err := k1.CreatePool(ctx1, creator, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(1000), pool1000.Id)

	pool1001, err := k1.CreatePool(ctx1, creator, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(1001), pool1001.Id)

	// Export genesis
	exportedGenesis, err := k1.ExportGenesis(ctx1)
	require.NoError(t, err)
	require.Greater(t, exportedGenesis.NextPoolId, pool1001.Id,
		"NextPoolID should be greater than last pool ID")

	// Import into new chain
	k2, ctx2 := keepertest.DexKeeper(t)
	err = k2.InitGenesis(ctx2, *exportedGenesis)
	require.NoError(t, err)

	// Create new pool - should get ID greater than 1001 and not collide
	pool1002, err := k2.CreatePool(ctx2, creator, "uatom", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool1002.Id, pool1001.Id,
		"New pool ID should be greater than existing pool IDs")
	require.NotEqual(t, pool1000.Id, pool1002.Id, "Should not collide with pool 1000")
	require.NotEqual(t, pool1001.Id, pool1002.Id, "Should not collide with pool 1001")

	// Verify old pools still accessible
	_, err = k2.GetPool(ctx2, 1000)
	require.NoError(t, err)
	_, err = k2.GetPool(ctx2, 1001)
	require.NoError(t, err)
}

// TestPoolIDCollisionPreventionMultipleMigrations tests that sequential migrations
// (multiple export/import cycles) maintain ID uniqueness
func TestPoolIDCollisionPreventionMultipleMigrations(t *testing.T) {
	creator := types.TestAddr()

	// First chain: Create pools 1, 2
	k1, ctx1 := keepertest.DexKeeper(t)
	_, err := k1.CreatePool(ctx1, creator, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	_, err = k1.CreatePool(ctx1, creator, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)

	// First migration
	genesis1, err := k1.ExportGenesis(ctx1)
	require.NoError(t, err)
	require.Greater(t, genesis1.NextPoolId, uint64(2),
		"NextPoolID should be greater than 2 after creating 2 pools")

	k2, ctx2 := keepertest.DexKeeper(t)
	err = k2.InitGenesis(ctx2, *genesis1)
	require.NoError(t, err)

	// Create pool 3 on second chain
	pool3, err := k2.CreatePool(ctx2, creator, "uatom", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool3.Id, uint64(2),
		"Third pool should have ID greater than 2")

	// Second migration
	genesis2, err := k2.ExportGenesis(ctx2)
	require.NoError(t, err)
	require.Greater(t, genesis2.NextPoolId, pool3.Id,
		"NextPoolID should be greater than last pool ID")

	k3, ctx3 := keepertest.DexKeeper(t)
	err = k3.InitGenesis(ctx3, *genesis2)
	require.NoError(t, err)

	// Create pools 4, 5 on third chain
	pool4, err := k3.CreatePool(ctx3, creator, "upaw", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool4.Id, pool3.Id, "Pool 4 ID should be greater than pool 3")
	require.NotEqual(t, pool3.Id, pool4.Id, "No collision with pool 3")

	pool5, err := k3.CreatePool(ctx3, creator, "uatom", "uosmo",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Greater(t, pool5.Id, pool4.Id, "Pool 5 ID should be greater than pool 4")

	// Verify all 5 pools exist and are unique
	allPools, err := k3.GetAllPools(ctx3)
	require.NoError(t, err)
	require.Len(t, allPools, 5)

	seenIDs := make(map[uint64]bool)
	for _, pool := range allPools {
		require.False(t, seenIDs[pool.Id], "Duplicate pool ID after multiple migrations")
		seenIDs[pool.Id] = true
	}
}

// TestPoolIDExportImportWithZeroNextPoolID tests behavior when NextPoolID is 0 in genesis.
// This can happen with manually created or very early genesis files.
func TestPoolIDExportImportWithZeroNextPoolID(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Create a pool normally
	pool1, err := k.CreatePool(ctx, creator, "upaw", "uatom",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(1), pool1.Id)

	// Export genesis
	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)

	// NOTE: When NextPoolID is 0 in genesis (edge case, but possible with manual genesis creation),
	// the GetNextPoolID function will start from 1 on import. This means if there are existing
	// pools, we could get ID collisions. This is expected behavior for corrupted genesis data.
	// In production, genesis validation should reject NextPoolID=0 if pools exist.

	// Test the normal case: NextPoolID should be > 0 after pools exist
	require.Greater(t, exported.NextPoolId, uint64(0),
		"NextPoolID should never be 0 when pools exist")

	// Test with fresh genesis (no pools, NextPoolID = 0)
	k2, ctx2 := keepertest.DexKeeper(t)
	freshGenesis := exported
	freshGenesis.Pools = nil
	freshGenesis.LiquidityPositions = nil
	freshGenesis.NextPoolId = 0 // Explicitly set to 0 (simulating empty genesis)

	err = k2.InitGenesis(ctx2, *freshGenesis)
	require.NoError(t, err)

	// Creating first pool should work with NextPoolID=0 (will use ID 1)
	firstPool, err := k2.CreatePool(ctx2, creator, "upaw", "uusdc",
		sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))
	require.NoError(t, err)
	require.Equal(t, uint64(1), firstPool.Id,
		"First pool should get ID 1 when starting from NextPoolID=0")
}
