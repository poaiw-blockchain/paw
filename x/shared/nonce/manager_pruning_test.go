package nonce_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/shared/nonce"
)

// TestPruneExpiredNonces_BasicPruning tests basic nonce pruning functionality
func TestPruneExpiredNonces_BasicPruning(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	ctx = ctx.WithBlockTime(baseTime)

	// Create nonces at different times
	channels := []string{"channel-0", "channel-1", "channel-2"}
	senders := []string{"sender1", "sender2"}

	// Set up nonces with different ages
	for i, channel := range channels {
		for j, sender := range senders {
			// Set context time to progressively older timestamps
			oldTime := baseTime.Add(-time.Duration(i*24+j*12) * time.Hour)
			oldCtx := ctx.WithBlockTime(oldTime)

			// Create a nonce (this will store timestamp)
			nonce := uint64((i+1)*10 + j)
			err := manager.ValidateIncomingPacketNonce(oldCtx, channel, sender, nonce, oldTime.Unix())
			require.NoError(t, err)
		}
	}

	// Move to current time and prune nonces older than 2 days
	ctx = ctx.WithBlockTime(baseTime)
	ttlSeconds := int64(2 * 24 * 60 * 60) // 2 days
	maxPrune := 100

	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, maxPrune)
	require.NoError(t, err)

	// Should prune channel-2 entries (3+ days old)
	// channel-0 (0-12 hours) and channel-1 (24-36 hours) are within 2 days
	expectedPruned := 2 // channel-2 with 2 senders
	require.Equal(t, expectedPruned, prunedCount)
}

// TestPruneExpiredNonces_BatchLimiting tests that pruning respects batch size limits
func TestPruneExpiredNonces_BatchLimiting(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()

	// Create many expired nonces (more than batch limit)
	oldTime := baseTime.Add(-10 * 24 * time.Hour) // 10 days old
	oldCtx := ctx.WithBlockTime(oldTime)

	// Create 50 expired nonces
	for i := 0; i < 50; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := manager.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Prune with batch limit of 20
	ctx = ctx.WithBlockTime(baseTime)
	ttlSeconds := int64(7 * 24 * 60 * 60) // 7 days
	maxPrune := 20

	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, maxPrune)
	require.NoError(t, err)

	// Should respect batch limit
	require.Equal(t, maxPrune, prunedCount)

	// Second call should prune more
	prunedCount2, err := manager.PruneExpiredNonces(ctx, ttlSeconds, maxPrune)
	require.NoError(t, err)
	require.Equal(t, maxPrune, prunedCount2)

	// Third call should prune remaining
	prunedCount3, err := manager.PruneExpiredNonces(ctx, ttlSeconds, maxPrune)
	require.NoError(t, err)
	require.Equal(t, 10, prunedCount3) // 50 - 20 - 20 = 10
}

// TestPruneExpiredNonces_NoExpiredNonces tests behavior when no nonces are expired
func TestPruneExpiredNonces_NoExpiredNonces(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	ctx = ctx.WithBlockTime(baseTime)

	// Create fresh nonces
	for i := 0; i < 10; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, uint64(i+1), baseTime.Unix())
		require.NoError(t, err)
	}

	// Try to prune with 7-day TTL (nothing should be pruned)
	ttlSeconds := int64(7 * 24 * 60 * 60)
	maxPrune := 100

	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, maxPrune)
	require.NoError(t, err)
	require.Equal(t, 0, prunedCount)
}

// TestPruneExpiredNonces_PreservesRecentNonces tests that recent nonces are not pruned
func TestPruneExpiredNonces_PreservesRecentNonces(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()

	// Create old nonces
	oldTime := baseTime.Add(-10 * 24 * time.Hour)
	oldCtx := ctx.WithBlockTime(oldTime)
	err := manager.ValidateIncomingPacketNonce(oldCtx, "channel-0", "old-sender", 1, oldTime.Unix())
	require.NoError(t, err)

	// Create recent nonces
	ctx = ctx.WithBlockTime(baseTime)
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "new-sender", 1, baseTime.Unix())
	require.NoError(t, err)

	// Prune with 7-day TTL
	ttlSeconds := int64(7 * 24 * 60 * 60)
	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, 100)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount) // Only old nonce should be pruned

	// Verify recent nonce still works (would reject if pruned and recreated)
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "new-sender", 2, baseTime.Unix())
	require.NoError(t, err)
}

// TestPruneExpiredNonces_DefaultTTL tests that default TTL is used when ttlSeconds <= 0
func TestPruneExpiredNonces_DefaultTTL(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()

	// Create nonce older than default TTL (7 days)
	oldTime := baseTime.Add(-8 * 24 * time.Hour)
	oldCtx := ctx.WithBlockTime(oldTime)
	err := manager.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Prune with zero TTL (should use default)
	ctx = ctx.WithBlockTime(baseTime)
	prunedCount, err := manager.PruneExpiredNonces(ctx, 0, 100)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_AtomicCleanup tests that all related keys are deleted atomically
func TestPruneExpiredNonces_AtomicCleanup(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	oldTime := baseTime.Add(-10 * 24 * time.Hour)

	// Create both incoming and outgoing nonces
	oldCtx := ctx.WithBlockTime(oldTime)
	channel := "channel-0"
	sender := "sender1"

	// Incoming nonce
	err := manager.ValidateIncomingPacketNonce(oldCtx, channel, sender, 1, oldTime.Unix())
	require.NoError(t, err)

	// Outgoing nonce
	_ = manager.NextOutboundNonce(oldCtx, channel, sender)

	// Prune
	ctx = ctx.WithBlockTime(baseTime)
	ttlSeconds := int64(7 * 24 * 60 * 60)
	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, 100)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount) // Count is per channel/sender pair

	// Verify both nonces are gone - should be able to start from 1 again
	err = manager.ValidateIncomingPacketNonce(ctx, channel, sender, 1, baseTime.Unix())
	require.NoError(t, err) // Should succeed since old nonce was pruned

	outNonce := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(1), outNonce) // Should restart from 1
}

// TestPruneExpiredNonces_MultipleChannels tests pruning across multiple channels
func TestPruneExpiredNonces_MultipleChannels(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	oldTime := baseTime.Add(-10 * 24 * time.Hour)

	// Create nonces on multiple channels
	channels := []string{"channel-0", "channel-1", "channel-2", "channel-3"}
	oldCtx := ctx.WithBlockTime(oldTime)

	for _, channel := range channels {
		err := manager.ValidateIncomingPacketNonce(oldCtx, channel, "sender1", 1, oldTime.Unix())
		require.NoError(t, err)
	}

	// Prune all expired nonces
	ctx = ctx.WithBlockTime(baseTime)
	ttlSeconds := int64(7 * 24 * 60 * 60)
	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, 100)
	require.NoError(t, err)
	require.Equal(t, 4, prunedCount) // All 4 channels should be pruned
}

// TestPruneExpiredNonces_EdgeCaseTTL tests pruning exactly at TTL boundary
func TestPruneExpiredNonces_EdgeCaseTTL(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	ttlSeconds := int64(7 * 24 * 60 * 60) // 7 days

	// Create nonce exactly at TTL boundary
	boundaryTime := baseTime.Add(-time.Duration(ttlSeconds) * time.Second)
	boundaryCtx := ctx.WithBlockTime(boundaryTime)
	err := manager.ValidateIncomingPacketNonce(boundaryCtx, "channel-0", "sender1", 1, boundaryTime.Unix())
	require.NoError(t, err)

	// Create nonce just before TTL boundary (should not be pruned)
	justBeforeTime := baseTime.Add(-time.Duration(ttlSeconds-1) * time.Second)
	justBeforeCtx := ctx.WithBlockTime(justBeforeTime)
	err = manager.ValidateIncomingPacketNonce(justBeforeCtx, "channel-0", "sender2", 1, justBeforeTime.Unix())
	require.NoError(t, err)

	// Prune at current time
	ctx = ctx.WithBlockTime(baseTime)
	prunedCount, err := manager.PruneExpiredNonces(ctx, ttlSeconds, 100)
	require.NoError(t, err)

	// Only the nonce exactly at or beyond TTL should be pruned
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_EmptyStore tests pruning with no nonces in store
func TestPruneExpiredNonces_EmptyStore(t *testing.T) {
	manager, ctx := setupManager(t)

	// Prune empty store
	prunedCount, err := manager.PruneExpiredNonces(ctx, 7*24*60*60, 100)
	require.NoError(t, err)
	require.Equal(t, 0, prunedCount)
}

// TestPruneExpiredNonces_ReplayProtectionMaintained tests that replay protection still works after pruning
func TestPruneExpiredNonces_ReplayProtectionMaintained(t *testing.T) {
	manager, ctx := setupManager(t)
	baseTime := time.Now()
	ctx = ctx.WithBlockTime(baseTime)

	// Create recent nonce
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 5, baseTime.Unix())
	require.NoError(t, err)

	// Prune (nothing should be pruned)
	prunedCount, err := manager.PruneExpiredNonces(ctx, 7*24*60*60, 100)
	require.NoError(t, err)
	require.Equal(t, 0, prunedCount)

	// Try to replay - should still be rejected
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 5, baseTime.Unix())
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay attack")

	// Next nonce should work
	err = manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 6, baseTime.Unix())
	require.NoError(t, err)
}

// TestExtractChannelSenderFromKey tests the key parsing helper function
func TestExtractChannelSenderFromKey(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx = ctx.WithBlockTime(time.Now())

	errorProvider := &MockErrorProvider{}
	manager := nonce.NewManager(storeKey, errorProvider, "testmodule")

	// Create a nonce to generate a real key
	err := manager.ValidateIncomingPacketNonce(ctx, "test-channel", "test-sender", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// The key should be parseable (we can verify by pruning)
	prunedCount, err := manager.PruneExpiredNonces(
		ctx.WithBlockTime(ctx.BlockTime().Add(10*24*time.Hour)),
		7*24*60*60,
		100,
	)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}
