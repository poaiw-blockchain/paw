package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

func TestActivityTracking(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Initially, no pools should be active
	activePoolIDs := k.GetActivePoolIDs(ctx)
	require.Empty(t, activePoolIDs, "no pools should be active initially")

	// Mark pool 1 as active
	err := k.MarkPoolActive(ctx, 1)
	require.NoError(t, err)

	// Mark pool 5 as active
	err = k.MarkPoolActive(ctx, 5)
	require.NoError(t, err)

	// Mark pool 1 again (should be idempotent)
	err = k.MarkPoolActive(ctx, 1)
	require.NoError(t, err)

	// Check active pools
	activePoolIDs = k.GetActivePoolIDs(ctx)
	require.Len(t, activePoolIDs, 2, "should have 2 active pools")
	require.Contains(t, activePoolIDs, uint64(1))
	require.Contains(t, activePoolIDs, uint64(5))

	// Clear active pools
	k.ClearActivePoolIDs(ctx)

	// Verify all active pools are cleared
	activePoolIDs = k.GetActivePoolIDs(ctx)
	require.Empty(t, activePoolIDs, "all active pools should be cleared")
}

func TestActivityTrackingWithManyPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Mark many pools as active (simulating high activity)
	for i := uint64(1); i <= 100; i++ {
		err := k.MarkPoolActive(ctx, i)
		require.NoError(t, err)
	}

	// Verify all pools are tracked
	activePoolIDs := k.GetActivePoolIDs(ctx)
	require.Len(t, activePoolIDs, 100, "should have 100 active pools")

	// Clear and verify
	k.ClearActivePoolIDs(ctx)
	activePoolIDs = k.GetActivePoolIDs(ctx)
	require.Empty(t, activePoolIDs, "all active pools should be cleared")
}
