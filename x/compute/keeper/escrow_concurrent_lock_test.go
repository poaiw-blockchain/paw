package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestConcurrentEscrowLock_SerializedAccess tests that sequential escrow operations
// work correctly (baseline for concurrency testing)
func TestConcurrentEscrowLock_SerializedAccess(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(1000000)
	timeoutSeconds := uint64(3600)

	// Lock 10 escrows sequentially
	for i := uint64(1); i <= 10; i++ {
		err := k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		require.NoError(t, err, "sequential lock %d should succeed", i)
	}

	// Verify all escrows were created correctly
	for i := uint64(1); i <= 10; i++ {
		escrowState, err := k.GetEscrowState(ctx, i)
		require.NoError(t, err)
		require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
		require.Equal(t, amount, escrowState.Amount)
	}
}

// TestConcurrentEscrowLock_DuplicatePrevention tests that attempting to lock
// the same escrow twice (even in the same context) fails
func TestConcurrentEscrowLock_DuplicatePrevention(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// First lock should succeed
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err, "first lock should succeed")

	// Second lock of same request should fail
	err = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.Error(t, err, "second lock should fail")
	require.Contains(t, err.Error(), "already exists", "error should indicate escrow already exists")
}

// TestConcurrentEscrowLock_StateConsistencyCheck tests that the invariants
// can detect state inconsistencies
func TestConcurrentEscrowLock_StateConsistencyCheck(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(500000)
	timeoutSeconds := uint64(3600)

	// Create several escrows
	for i := uint64(100); i < 110; i++ {
		err := k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		require.NoError(t, err)
	}

	// Verify invariants pass for properly created escrows
	timeoutInvariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := timeoutInvariant(sdkCtx)
	require.False(t, broken, "timeout index invariant should pass: %s", msg)

	consistencyInvariant := EscrowStateConsistencyInvariant(*k)
	msg, broken = consistencyInvariant(sdkCtx)
	require.False(t, broken, "state consistency invariant should pass: %s", msg)
}

// TestConcurrentEscrowLock_NonceUniqueness tests that nonces are unique
// when escrows are created sequentially
func TestConcurrentEscrowLock_NonceUniqueness(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(150000)
	timeoutSeconds := uint64(3600)

	nonces := make(map[uint64]bool)

	// Create 30 escrows sequentially
	for i := uint64(400); i < 430; i++ {
		err := k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		require.NoError(t, err)

		escrowState, err := k.GetEscrowState(ctx, i)
		require.NoError(t, err)

		// Verify nonce is unique
		require.False(t, nonces[escrowState.Nonce], "nonce %d should be unique", escrowState.Nonce)
		nonces[escrowState.Nonce] = true
	}

	require.Len(t, nonces, 30, "should have 30 unique nonces")
}

// TestConcurrentEscrowLock_TimeoutIndexCreation tests that timeout indexes
// are created atomically with escrow state
func TestConcurrentEscrowLock_TimeoutIndexCreation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(400000)
	timeoutSeconds := uint64(3600)

	// Create 15 escrows
	for i := uint64(300); i < 315; i++ {
		err := k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		require.NoError(t, err)
	}

	// Verify all timeout indexes exist
	now := sdkCtx.BlockTime()
	futureTime := now.Add(7200 * time.Second)

	timeoutCount := 0
	err := k.IterateEscrowTimeouts(ctx, futureTime, func(requestID uint64, expiresAt time.Time) (stop bool, err error) {
		if requestID >= 300 && requestID < 315 {
			// Verify corresponding escrow exists
			escrowState, err := k.GetEscrowState(ctx, requestID)
			require.NoError(t, err)
			require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
			timeoutCount++
		}
		return false, nil
	})

	require.NoError(t, err)
	require.Equal(t, 15, timeoutCount, "all escrow locks should have timeout indexes")
}

// TestConcurrentEscrowLock_MixedOperationsSequential tests lock, release, and refund
// operations in sequence
func TestConcurrentEscrowLock_MixedOperationsSequential(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(600000)
	timeoutSeconds := uint64(3600)

	// Create 10 escrows
	for i := uint64(500); i < 510; i++ {
		err := k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		require.NoError(t, err)
	}

	// Release first 3
	for i := uint64(500); i < 503; i++ {
		err := k.ReleaseEscrow(ctx, i, true)
		require.NoError(t, err)
	}

	// Refund next 3
	for i := uint64(503); i < 506; i++ {
		err := k.RefundEscrow(ctx, i, "test_refund")
		require.NoError(t, err)
	}

	// Verify state consistency after all operations
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "state should be consistent after sequential operations: %s", msg)

	timeoutInvariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken = timeoutInvariant(sdkCtx)
	require.False(t, broken, "timeout indexes should be consistent: %s", msg)
}

// TestConcurrentEscrowLock_ReverseIndexCleanup tests that reverse timeout indexes
// are properly created and cleaned up
func TestConcurrentEscrowLock_ReverseIndexCleanup(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(800000)
	requestID := uint64(900)
	timeoutSeconds := uint64(3600)

	// Lock escrow
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	// Verify both forward and reverse timeout indexes exist
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)

	store := k.getStore(ctx)
	timeoutKey := EscrowTimeoutKey(escrowState.ExpiresAt, requestID)
	require.True(t, store.Has(timeoutKey), "forward timeout index must exist")

	reverseKey := EscrowTimeoutReverseKey(requestID)
	require.True(t, store.Has(reverseKey), "reverse timeout index must exist")

	// Release escrow
	err = k.ReleaseEscrow(ctx, requestID, true)
	require.NoError(t, err)

	// Verify both indexes are cleaned up
	require.False(t, store.Has(timeoutKey), "forward timeout index should be deleted")
	require.False(t, store.Has(reverseKey), "reverse timeout index should be deleted")
}

// TestConcurrentEscrowLock_NoDeadlock tests that escrow operations complete
// without hanging (basic sanity check)
func TestConcurrentEscrowLock_NoDeadlock(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(1000000)
	timeoutSeconds := uint64(3600)

	done := make(chan struct{})

	go func() {
		// Create 10 escrows
		for i := uint64(10); i < 20; i++ {
			_ = k.LockEscrow(ctx, requester, provider, amount, i, timeoutSeconds)
		}
		close(done)
	}()

	select {
	case <-done:
		// Success - operations completed
	case <-time.After(5 * time.Second):
		t.Fatal("operations did not complete - possible deadlock")
	}
}

// TestConcurrentEscrowLock_CacheContextIsolation tests that CacheContext
// provides proper transaction boundaries
func TestConcurrentEscrowLock_CacheContextIsolation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))
	amount := math.NewInt(5000000)
	requestID := uint64(1000)
	timeoutSeconds := uint64(3600)

	// Create cache context
	cacheCtx, writeFn := sdkCtx.CacheContext()

	// Lock escrow in cache context
	err := k.LockEscrow(cacheCtx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err, "lock should succeed in cache context")

	// Verify escrow exists in cache context
	_, err = k.GetEscrowState(cacheCtx, requestID)
	require.NoError(t, err, "escrow should exist in cache context")

	// Verify escrow does NOT exist in parent context (not yet written)
	_, err = k.GetEscrowState(ctx, requestID)
	require.Error(t, err, "escrow should not exist in parent context before write")

	// Write cache to parent
	writeFn()

	// Verify escrow NOW exists in parent context
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err, "escrow should exist in parent context after write")
	require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
}
