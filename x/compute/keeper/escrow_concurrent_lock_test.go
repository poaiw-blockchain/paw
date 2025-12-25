package keeper

import (
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// fundAccountForTest funds a test account with tokens
func fundAccountForTest(t *testing.T, k *Keeper, ctx sdk.Context, addr sdk.AccAddress, amount math.Int) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", amount.Int64()))
	require.NoError(t, k.bankKeeper.MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins))
}

// TestConcurrentEscrowLock_OnlyOneSucceeds tests that when two goroutines
// attempt to lock the same escrow simultaneously, only one succeeds
func TestConcurrentEscrowLock_OnlyOneSucceeds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("concurrent_req___"))
	provider := sdk.AccAddress([]byte("concurrent_prov__"))
	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Fund the requester account
	fundAccountForTest(t, k, sdkCtx, requester, math.NewInt(100000000))

	var wg sync.WaitGroup
	results := make(chan error, 2)

	// Launch two concurrent goroutines attempting to lock the same escrow
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	// Collect results
	var errors []error
	for err := range results {
		errors = append(errors, err)
	}

	require.Len(t, errors, 2, "should have 2 results")

	// Exactly one should succeed (nil error), one should fail (already exists error)
	successCount := 0
	failureCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			failureCount++
			require.Contains(t, err.Error(), "already exists", "failure should be due to escrow already existing")
		}
	}

	require.Equal(t, 1, successCount, "exactly one lock should succeed")
	require.Equal(t, 1, failureCount, "exactly one lock should fail")

	// Verify only one escrow state exists
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.NotNil(t, escrowState)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
	require.Equal(t, amount, escrowState.Amount)
}

// TestConcurrentEscrowLock_DifferentRequests tests that concurrent locks
// for different requests both succeed
func TestConcurrentEscrowLock_DifferentRequests(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("multi_requester__"))
	provider := sdk.AccAddress([]byte("multi_provider___"))
	amount := math.NewInt(5000000)
	timeoutSeconds := uint64(3600)

	var wg sync.WaitGroup
	results := make(chan struct {
		requestID uint64
		err       error
	}, 2)

	// Launch two concurrent goroutines locking different escrows
	for i := uint64(1); i <= 2; i++ {
		wg.Add(1)
		requestID := i
		go func(rid uint64) {
			defer wg.Done()
			err := k.LockEscrow(ctx, requester, provider, amount, rid, timeoutSeconds)
			results <- struct {
				requestID uint64
				err       error
			}{rid, err}
		}(requestID)
	}

	wg.Wait()
	close(results)

	// Both should succeed
	for result := range results {
		require.NoError(t, result.err, "lock for request %d should succeed", result.requestID)
	}

	// Verify both escrow states exist
	for i := uint64(1); i <= 2; i++ {
		escrowState, err := k.GetEscrowState(ctx, i)
		require.NoError(t, err)
		require.NotNil(t, escrowState)
		require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
	}
}

// TestConcurrentEscrowLock_NoDeadlock tests that concurrent escrow operations
// don't cause deadlocks
func TestConcurrentEscrowLock_NoDeadlock(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	amount := math.NewInt(1000000)
	timeoutSeconds := uint64(3600)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Launch many concurrent goroutines attempting to lock escrows
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requester := sdk.AccAddress([]byte("requester_" + string(rune(index))))
			provider := sdk.AccAddress([]byte("provider_" + string(rune(index))))
			requestID := uint64(index + 1)
			_ = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		}(i)
	}

	// Wait with timeout to detect deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock detected - operations did not complete within timeout")
	}
}

// TestConcurrentEscrowLock_StateConsistency tests that concurrent operations
// maintain state consistency
func TestConcurrentEscrowLock_StateConsistency(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	amount := math.NewInt(2000000)
	timeoutSeconds := uint64(3600)
	numLocks := 50

	var wg sync.WaitGroup

	// Create many escrows concurrently
	for i := 0; i < numLocks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requester := sdk.AccAddress([]byte("state_req_" + string(rune(index))))
			provider := sdk.AccAddress([]byte("state_prov_" + string(rune(index))))
			requestID := uint64(100 + index)
			_ = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		}(i)
	}

	wg.Wait()

	// Verify state consistency using invariants
	invariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "timeout index invariant should not be broken after concurrent operations: %s", msg)

	consistencyInvariant := EscrowStateConsistencyInvariant(*k)
	msg, broken = consistencyInvariant(sdkCtx)
	require.False(t, broken, "state consistency invariant should not be broken after concurrent operations: %s", msg)
}

// TestConcurrentEscrowLock_RaceConditionPrevention tests that the atomic
// SetEscrowStateIfNotExists prevents race conditions
func TestConcurrentEscrowLock_RaceConditionPrevention(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("race_requester___"))
	provider := sdk.AccAddress([]byte("race_provider____"))
	requestID := uint64(200)
	amount := math.NewInt(3000000)
	timeoutSeconds := uint64(3600)

	// Try to create the same escrow many times concurrently
	numAttempts := 20
	var wg sync.WaitGroup
	successCount := int32(0)
	failureCount := int32(0)

	var mu sync.Mutex // Protect counters

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)

			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successCount++
			} else {
				failureCount++
			}
		}()
	}

	wg.Wait()

	// Verify exactly one success
	require.Equal(t, int32(1), successCount, "exactly one lock attempt should succeed")
	require.Equal(t, int32(numAttempts-1), failureCount, "all other attempts should fail")

	// Verify escrow state is consistent
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.NotNil(t, escrowState)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
	require.Equal(t, amount, escrowState.Amount)
}

// TestConcurrentEscrowLock_TimeoutIndexAtomic tests that timeout index
// creation is atomic with escrow creation
func TestConcurrentEscrowLock_TimeoutIndexAtomic(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("atomic_requester_"))
	provider := sdk.AccAddress([]byte("atomic_provider__"))
	amount := math.NewInt(4000000)
	timeoutSeconds := uint64(3600)

	numConcurrentLocks := 30
	var wg sync.WaitGroup

	// Create many escrows concurrently
	for i := 0; i < numConcurrentLocks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := uint64(300 + index)
			_ = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		}(i)
	}

	wg.Wait()

	// For every successfully created escrow, verify its timeout index exists
	now := sdkCtx.BlockTime()
	futureTime := now.Add(7200 * time.Second) // Well after all timeouts

	timeoutCount := 0
	err := k.IterateEscrowTimeouts(ctx, futureTime, func(requestID uint64, expiresAt time.Time) (stop bool, err error) {
		if requestID >= 300 && requestID < 300+uint64(numConcurrentLocks) {
			// Verify corresponding escrow exists
			escrowState, err := k.GetEscrowState(ctx, requestID)
			require.NoError(t, err, "escrow state should exist for request %d", requestID)
			require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)
			timeoutCount++
		}
		return false, nil
	})

	require.NoError(t, err)
	require.Equal(t, numConcurrentLocks, timeoutCount,
		"all successful escrow locks should have timeout indexes")
}

// TestConcurrentEscrowLock_NoDuplicateNonces tests that concurrent operations
// don't generate duplicate nonces
func TestConcurrentEscrowLock_NoDuplicateNonces(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	amount := math.NewInt(1500000)
	timeoutSeconds := uint64(3600)
	numLocks := 100

	var wg sync.WaitGroup
	noncesChan := make(chan uint64, numLocks)

	// Create many escrows concurrently and collect nonces
	for i := 0; i < numLocks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requester := sdk.AccAddress([]byte("nonce_req_" + string(rune(index))))
			provider := sdk.AccAddress([]byte("nonce_prov_" + string(rune(index))))
			requestID := uint64(400 + index)

			err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
			if err == nil {
				// Get the nonce
				escrowState, err := k.GetEscrowState(ctx, requestID)
				if err == nil {
					noncesChan <- escrowState.Nonce
				}
			}
		}(i)
	}

	wg.Wait()
	close(noncesChan)

	// Collect all nonces and verify uniqueness
	nonces := make(map[uint64]bool)
	for nonce := range noncesChan {
		require.False(t, nonces[nonce], "duplicate nonce %d detected", nonce)
		nonces[nonce] = true
	}

	require.Len(t, nonces, numLocks, "should have unique nonces for all successful locks")
}

// TestConcurrentEscrowLock_MixedOperations tests concurrent lock, release, and refund
func TestConcurrentEscrowLock_MixedOperations(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("mixed_requester__"))
	provider := sdk.AccAddress([]byte("mixed_provider___"))
	amount := math.NewInt(6000000)
	timeoutSeconds := uint64(3600)

	// Create initial escrows
	numInitialEscrows := 10
	for i := 0; i < numInitialEscrows; i++ {
		requestID := uint64(500 + i)
		err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup

	// Concurrently:
	// - Lock new escrows
	// - Release existing escrows
	// - Refund existing escrows

	// Lock operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := uint64(600 + index)
			_ = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		}(i)
	}

	// Release operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := uint64(500 + index)
			_ = k.ReleaseEscrow(ctx, requestID, true) // Immediate release
		}(i)
	}

	// Refund operations
	for i := 3; i < 6; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := uint64(500 + index)
			_ = k.RefundEscrow(ctx, requestID, "concurrent_test")
		}(i)
	}

	wg.Wait()

	// Verify state consistency
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "state consistency should be maintained after mixed concurrent operations: %s", msg)

	timeoutInvariant := EscrowTimeoutIndexInvariant(*k)
	msg, broken = timeoutInvariant(sdkCtx)
	require.False(t, broken, "timeout indexes should be consistent after mixed concurrent operations: %s", msg)
}

// TestConcurrentEscrowLock_LockAndProcessExpired tests concurrent lock operations
// while processing expired escrows
func TestConcurrentEscrowLock_LockAndProcessExpired(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("expire_requester_"))
	provider := sdk.AccAddress([]byte("expire_provider__"))
	amount := math.NewInt(7000000)

	// Create some escrows that will expire
	numExpiring := 5
	for i := 0; i < numExpiring; i++ {
		requestID := uint64(700 + i)
		// Very short timeout (1 second)
		err := k.LockEscrow(ctx, requester, provider, amount, requestID, 1)
		require.NoError(t, err)
	}

	// Advance time to expire them
	newBlockTime := sdkCtx.BlockTime().Add(2 * time.Second)
	ctx = ctx.WithBlockTime(newBlockTime)

	var wg sync.WaitGroup

	// Concurrently:
	// - Process expired escrows
	// - Lock new escrows

	// Process expired
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = k.ProcessExpiredEscrows(ctx)
	}()

	// Lock new escrows
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := uint64(800 + index)
			_ = k.LockEscrow(ctx, requester, provider, amount, requestID, 3600)
		}(i)
	}

	wg.Wait()

	// Verify state consistency
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdk.UnwrapSDKContext(ctx))
	require.False(t, broken, "state should be consistent after concurrent lock and expire processing: %s", msg)
}

// TestConcurrentEscrowLock_AtomicCacheContext tests that CacheContext properly
// isolates concurrent operations
func TestConcurrentEscrowLock_AtomicCacheContext(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	requester := sdk.AccAddress([]byte("cache_requester__"))
	provider := sdk.AccAddress([]byte("cache_provider___"))
	requestID := uint64(900)
	amount := math.NewInt(8000000)
	timeoutSeconds := uint64(3600)

	// Attempt to lock the same escrow 50 times concurrently
	numAttempts := 50
	var wg sync.WaitGroup
	errors := make([]error, numAttempts)

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
		}(i)
	}

	wg.Wait()

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	// CacheContext should ensure exactly one succeeds
	require.Equal(t, 1, successCount, "CacheContext should ensure exactly one lock succeeds")
	require.Equal(t, numAttempts-1, failureCount, "all other attempts should fail atomically")

	// Verify final state
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.NotNil(t, escrowState)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)

	// Verify timeout index exists
	store := k.getStore(ctx)
	timeoutKey := EscrowTimeoutKey(escrowState.ExpiresAt, requestID)
	require.True(t, store.Has(timeoutKey), "timeout index must exist")

	// Verify reverse timeout index exists
	reverseKey := EscrowTimeoutReverseKey(requestID)
	require.True(t, store.Has(reverseKey), "reverse timeout index must exist")
}
