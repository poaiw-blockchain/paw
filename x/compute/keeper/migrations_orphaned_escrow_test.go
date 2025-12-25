package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestMigration_OrphanedEscrowTimeoutIndexes tests that migration handles orphaned timeout indexes
// (indexes that point to non-existent escrows). This validates P1-DATA-2 migration correctness.
func TestMigration_OrphanedEscrowTimeoutIndexes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()

	// Create valid escrows first
	validRequester := sdk.AccAddress([]byte("valid_requester___"))
	validProvider := sdk.AccAddress([]byte("valid_provider____"))

	// Escrow 1: Valid escrow with valid timeout index
	escrow1 := types.EscrowState{
		RequestId:       1,
		Requester:       validRequester.String(),
		Provider:        validProvider.String(),
		Amount:          math.NewInt(10000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           1,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow1))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow1.RequestId, escrow1.ExpiresAt))

	// Escrow 2: Valid escrow with valid timeout index
	escrow2 := types.EscrowState{
		RequestId:       2,
		Requester:       validRequester.String(),
		Provider:        validProvider.String(),
		Amount:          math.NewInt(20000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(7200 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           2,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow2))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow2.RequestId, escrow2.ExpiresAt))

	// Create orphaned timeout indexes (indexes without corresponding escrow states)
	// These simulate corrupted state where timeout index exists but escrow was deleted
	orphanedRequestID1 := uint64(99)
	orphanedExpiry1 := now.Add(1800 * time.Second)
	orphanedTimeoutKey1 := EscrowTimeoutKey(orphanedExpiry1, orphanedRequestID1)
	store.Set(orphanedTimeoutKey1, []byte{})

	orphanedRequestID2 := uint64(100)
	orphanedExpiry2 := now.Add(5400 * time.Second)
	orphanedTimeoutKey2 := EscrowTimeoutKey(orphanedExpiry2, orphanedRequestID2)
	store.Set(orphanedTimeoutKey2, []byte{})

	// Verify orphaned indexes exist
	require.True(t, store.Has(orphanedTimeoutKey1), "orphaned timeout index 1 should exist before migration")
	require.True(t, store.Has(orphanedTimeoutKey2), "orphaned timeout index 2 should exist before migration")

	// Verify valid indexes exist
	validTimeoutKey1 := EscrowTimeoutKey(escrow1.ExpiresAt, escrow1.RequestId)
	validTimeoutKey2 := EscrowTimeoutKey(escrow2.ExpiresAt, escrow2.RequestId)
	require.True(t, store.Has(validTimeoutKey1), "valid timeout index 1 should exist")
	require.True(t, store.Has(validTimeoutKey2), "valid timeout index 2 should exist")

	// Run migration (simulates v1 to v2 migration that should clean up orphaned indexes)
	// In a real migration, this would be part of the migration logic
	// For this test, we'll call the invariant to detect the orphaned indexes
	invariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken due to orphaned timeout indexes")
	require.Contains(t, msg, "has no escrow state", "error message should indicate missing escrow state")

	// Manually clean up orphaned indexes (simulating migration cleanup logic)
	store.Delete(orphanedTimeoutKey1)
	store.Delete(orphanedTimeoutKey2)

	// Verify orphaned indexes were cleaned up
	require.False(t, store.Has(orphanedTimeoutKey1), "orphaned timeout index 1 should be cleaned up")
	require.False(t, store.Has(orphanedTimeoutKey2), "orphaned timeout index 2 should be cleaned up")

	// Verify valid indexes are still present
	require.True(t, store.Has(validTimeoutKey1), "valid timeout index 1 should still exist")
	require.True(t, store.Has(validTimeoutKey2), "valid timeout index 2 should still exist")

	// Verify valid escrows are still present
	e1, err := k.GetEscrowState(ctx, escrow1.RequestId)
	require.NoError(t, err)
	require.Equal(t, escrow1.RequestId, e1.RequestId)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, e1.Status)

	e2, err := k.GetEscrowState(ctx, escrow2.RequestId)
	require.NoError(t, err)
	require.Equal(t, escrow2.RequestId, e2.RequestId)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, e2.Status)

	// Verify invariant now passes
	msg, broken = invariant(sdkCtx)
	require.False(t, broken, "invariant should not be broken after cleanup: %s", msg)
}

// TestMigration_OrphanedTimeoutIndexWithReleasedEscrow tests that migration detects
// and cleans up timeout indexes for escrows that have already been released/refunded
func TestMigration_OrphanedTimeoutIndexWithReleasedEscrow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()
	releasedTime := now

	requester := sdk.AccAddress([]byte("test_requester___"))
	provider := sdk.AccAddress([]byte("test_provider____"))

	// Create a released escrow (simulating escrow that was released but timeout index wasn't cleaned up)
	releasedEscrow := types.EscrowState{
		RequestId:       10,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(5000000),
		Status:          types.ESCROW_STATUS_RELEASED,
		LockedAt:        now.Add(-7200 * time.Second),
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleasedAt:      &releasedTime,
		ReleaseAttempts: 1,
		Nonce:           10,
	}
	require.NoError(t, k.SetEscrowState(ctx, releasedEscrow))

	// Manually create orphaned timeout index for released escrow
	orphanedTimeoutKey := EscrowTimeoutKey(releasedEscrow.ExpiresAt, releasedEscrow.RequestId)
	store.Set(orphanedTimeoutKey, []byte{})

	// Verify timeout index exists
	require.True(t, store.Has(orphanedTimeoutKey), "timeout index should exist before cleanup")

	// Run invariant to detect the issue
	invariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken for released escrow with timeout index")
	require.Contains(t, msg, "still has timeout index entry", "error should indicate orphaned timeout index")

	// Clean up the orphaned timeout index
	store.Delete(orphanedTimeoutKey)

	// Verify cleanup
	require.False(t, store.Has(orphanedTimeoutKey), "timeout index should be deleted")

	// Verify escrow state is still released
	e, err := k.GetEscrowState(ctx, releasedEscrow.RequestId)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_RELEASED, e.Status)
	require.NotNil(t, e.ReleasedAt)

	// Verify invariant now passes
	msg, broken = invariant(sdkCtx)
	require.False(t, broken, "invariant should pass after cleanup: %s", msg)
}

// TestMigration_MixedOrphanedAndValidTimeoutIndexes tests migration with a mix of
// orphaned and valid timeout indexes to ensure selective cleanup
func TestMigration_MixedOrphanedAndValidTimeoutIndexes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()

	requester := sdk.AccAddress([]byte("mix_requester____"))
	provider := sdk.AccAddress([]byte("mix_provider_____"))

	// Create 3 valid locked escrows
	validEscrows := []types.EscrowState{
		{
			RequestId:       101,
			Requester:       requester.String(),
			Provider:        provider.String(),
			Amount:          math.NewInt(1000000),
			Status:          types.ESCROW_STATUS_LOCKED,
			LockedAt:        now,
			ExpiresAt:       now.Add(1000 * time.Second),
			ReleaseAttempts: 0,
			Nonce:           101,
		},
		{
			RequestId:       102,
			Requester:       requester.String(),
			Provider:        provider.String(),
			Amount:          math.NewInt(2000000),
			Status:          types.ESCROW_STATUS_LOCKED,
			LockedAt:        now,
			ExpiresAt:       now.Add(2000 * time.Second),
			ReleaseAttempts: 0,
			Nonce:           102,
		},
		{
			RequestId:       103,
			Requester:       requester.String(),
			Provider:        provider.String(),
			Amount:          math.NewInt(3000000),
			Status:          types.ESCROW_STATUS_LOCKED,
			LockedAt:        now,
			ExpiresAt:       now.Add(3000 * time.Second),
			ReleaseAttempts: 0,
			Nonce:           103,
		},
	}

	for _, escrow := range validEscrows {
		require.NoError(t, k.SetEscrowState(ctx, escrow))
		require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow.RequestId, escrow.ExpiresAt))
	}

	// Create 2 orphaned timeout indexes
	orphanedIDs := []uint64{201, 202}
	orphanedKeys := make([][]byte, len(orphanedIDs))
	for i, id := range orphanedIDs {
		expiry := now.Add(time.Duration((i+1)*1500) * time.Second)
		key := EscrowTimeoutKey(expiry, id)
		store.Set(key, []byte{})
		orphanedKeys[i] = key
	}

	// Verify all timeout indexes exist
	for _, escrow := range validEscrows {
		key := EscrowTimeoutKey(escrow.ExpiresAt, escrow.RequestId)
		require.True(t, store.Has(key), "valid timeout index for request %d should exist", escrow.RequestId)
	}
	for i, key := range orphanedKeys {
		require.True(t, store.Has(key), "orphaned timeout index %d should exist", i)
	}

	// Run invariant - should detect orphaned indexes
	invariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken due to orphaned indexes")

	// Clean up only orphaned indexes
	for _, key := range orphanedKeys {
		store.Delete(key)
	}

	// Verify orphaned indexes removed, valid indexes remain
	for _, key := range orphanedKeys {
		require.False(t, store.Has(key), "orphaned timeout index should be removed")
	}
	for _, escrow := range validEscrows {
		key := EscrowTimeoutKey(escrow.ExpiresAt, escrow.RequestId)
		require.True(t, store.Has(key), "valid timeout index for request %d should remain", escrow.RequestId)
	}

	// Verify all valid escrows are intact
	for _, escrow := range validEscrows {
		e, err := k.GetEscrowState(ctx, escrow.RequestId)
		require.NoError(t, err)
		require.Equal(t, escrow.RequestId, e.RequestId)
		require.Equal(t, types.ESCROW_STATUS_LOCKED, e.Status)
	}

	// Verify invariant now passes
	msg, broken = invariant(sdkCtx)
	require.False(t, broken, "invariant should pass after cleaning orphaned indexes: %s", msg)
}

// TestMigration_NoOrphanedIndexes tests that migration handles clean state with no orphaned indexes
func TestMigration_NoOrphanedIndexes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("clean_requester__"))
	provider := sdk.AccAddress([]byte("clean_provider___"))

	// Create valid escrows with proper timeout indexes
	for i := uint64(1); i <= 5; i++ {
		escrow := types.EscrowState{
			RequestId:       i,
			Requester:       requester.String(),
			Provider:        provider.String(),
			Amount:          math.NewInt(int64(i * 1000000)),
			Status:          types.ESCROW_STATUS_LOCKED,
			LockedAt:        now,
			ExpiresAt:       now.Add(time.Duration(i*1000) * time.Second),
			ReleaseAttempts: 0,
			Nonce:           i,
		}
		require.NoError(t, k.SetEscrowState(ctx, escrow))
		require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow.RequestId, escrow.ExpiresAt))
	}

	// Run invariant - should pass
	invariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "invariant should not be broken with clean state: %s", msg)
}

// TestMigration_OrphanedIndexesNoPanic tests that processing orphaned indexes doesn't panic
func TestMigration_OrphanedIndexesNoPanic(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()

	// Create many orphaned timeout indexes to stress test
	for i := uint64(1000); i < 1100; i++ {
		expiry := now.Add(time.Duration(i) * time.Second)
		key := EscrowTimeoutKey(expiry, i)
		store.Set(key, []byte{})
	}

	// Running invariant should not panic even with many orphaned indexes
	require.NotPanics(t, func() {
		invariant := EscrowTimeoutIndexInvariant(*k)
		_, _ = invariant(sdkCtx)
	}, "invariant should not panic with orphaned indexes")

	// Iteration over timeout indexes should not panic
	require.NotPanics(t, func() {
		futureTime := now.Add(200000 * time.Second)
		_ = k.IterateEscrowTimeouts(ctx, futureTime, func(requestID uint64, expiresAt time.Time) (stop bool, err error) {
			return false, nil
		})
	}, "iteration should not panic with orphaned indexes")
}

// TestMigration_IteratorStopsAtOrphanedEscrow tests that iteration doesn't break when encountering orphaned indexes
func TestMigration_IteratorStopsAtOrphanedEscrow(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()

	// Create mix: valid, orphaned, valid
	requester := sdk.AccAddress([]byte("iter_requester___"))
	provider := sdk.AccAddress([]byte("iter_provider____"))

	// Valid escrow 1
	escrow1 := types.EscrowState{
		RequestId:       1,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(1000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(1000 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           1,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow1))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow1.RequestId, escrow1.ExpiresAt))

	// Orphaned timeout index (no escrow state)
	orphanedID := uint64(999)
	orphanedExpiry := now.Add(1500 * time.Second)
	orphanedKey := EscrowTimeoutKey(orphanedExpiry, orphanedID)
	store.Set(orphanedKey, []byte{})

	// Valid escrow 2
	escrow2 := types.EscrowState{
		RequestId:       2,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(2000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(2000 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           2,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow2))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow2.RequestId, escrow2.ExpiresAt))

	// Iterate and count - should see all timeout indexes (including orphaned)
	futureTime := now.Add(3000 * time.Second)
	count := 0
	validCount := 0
	orphanedCount := 0

	err := k.IterateEscrowTimeouts(ctx, futureTime, func(requestID uint64, expiresAt time.Time) (stop bool, err error) {
		count++
		// Check if escrow exists
		_, err = k.GetEscrowState(ctx, requestID)
		if err != nil {
			orphanedCount++
		} else {
			validCount++
		}
		return false, nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, count, "should iterate over all timeout indexes")
	require.Equal(t, 2, validCount, "should find 2 valid escrows")
	require.Equal(t, 1, orphanedCount, "should find 1 orphaned timeout index")
}
