package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestCatastrophicFailureRecovery_InvariantDetectsMismatch tests that
// EscrowStateConsistencyInvariant detects catastrophic failures where
// bank transfers succeeded but state updates failed
func TestCatastrophicFailureRecovery_InvariantDetectsMismatch(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	provider := sdk.AccAddress([]byte("catastrophic_prov"))

	// Simulate catastrophic failure: store a failure record
	failureID := uint64(1)
	failure := &types.CatastrophicFailure{
		Id:          failureID,
		RequestId:   100,
		Account:     provider.String(),
		Amount:      math.NewInt(5000000),
		Reason:      "bank transfer succeeded but state update failed",
		OccurredAt:  now,
		BlockHeight: sdkCtx.BlockHeight(),
		Resolved:    false,
		ResolvedAt:  nil,
	}

	err := k.StoreCatastrophicFailure(ctx, failure.RequestId, provider, failure.Amount, failure.Reason)
	require.NoError(t, err)

	// Verify failure was stored
	storedFailure, err := k.GetCatastrophicFailure(ctx, failureID)
	require.NoError(t, err)
	require.Equal(t, failure.RequestId, storedFailure.RequestId)
	require.Equal(t, failure.Account, storedFailure.Account)
	require.Equal(t, failure.Amount, storedFailure.Amount)
	require.False(t, storedFailure.Resolved)

	// Run EscrowStateConsistencyInvariant - should detect the catastrophic failure
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken when unresolved catastrophic failures exist")
	require.Contains(t, msg, "unresolved catastrophic failures", "message should mention catastrophic failures")
	require.Contains(t, msg, "Request 100", "message should mention the failed request")
	require.Contains(t, msg, provider.String(), "message should mention the provider account")
}

// TestCatastrophicFailureRecovery_MultipleFailures tests handling of multiple
// catastrophic failures
func TestCatastrophicFailureRecovery_MultipleFailures(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	// Create multiple catastrophic failures
	failures := []struct {
		requestID uint64
		account   sdk.AccAddress
		amount    math.Int
		reason    string
	}{
		{
			requestID: 1,
			account:   sdk.AccAddress([]byte("provider1________")),
			amount:    math.NewInt(1000000),
			reason:    "release transfer succeeded, state update failed",
		},
		{
			requestID: 2,
			account:   sdk.AccAddress([]byte("requester2_______")),
			amount:    math.NewInt(2000000),
			reason:    "refund transfer succeeded, state update failed",
		},
		{
			requestID: 3,
			account:   sdk.AccAddress([]byte("provider3________")),
			amount:    math.NewInt(3000000),
			reason:    "release transfer succeeded, timeout index cleanup failed",
		},
	}

	for _, f := range failures {
		err := k.StoreCatastrophicFailure(ctx, f.requestID, f.account, f.amount, f.reason)
		require.NoError(t, err)
	}

	// Verify all failures were stored
	allFailures, err := k.GetAllCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, allFailures, 3, "should have 3 catastrophic failures")

	// Verify all are unresolved
	unresolvedFailures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, unresolvedFailures, 3, "all 3 failures should be unresolved")

	// Run invariant - should be broken
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken with multiple catastrophic failures")
	require.Contains(t, msg, "3 unresolved catastrophic failures", "should report count")

	// Verify each failure is mentioned
	for _, f := range failures {
		require.Contains(t, msg, f.account.String(), "should mention account %s", f.account.String())
	}
}

// TestCatastrophicFailureRecovery_ResolvedFailuresPassInvariant tests that
// resolved catastrophic failures don't break the invariant
func TestCatastrophicFailureRecovery_ResolvedFailuresPassInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	provider := sdk.AccAddress([]byte("resolved_provider"))

	// Create a catastrophic failure
	failureID := uint64(1)
	err := k.StoreCatastrophicFailure(ctx, 100, provider, math.NewInt(1000000), "test failure")
	require.NoError(t, err)

	// Verify invariant is broken before resolution
	invariant := EscrowStateConsistencyInvariant(*k)
	_, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken before resolution")

	// Mark failure as resolved
	failure, err := k.GetCatastrophicFailure(ctx, failureID)
	require.NoError(t, err)
	failure.Resolved = true
	resolvedAt := now
	failure.ResolvedAt = &resolvedAt

	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(failure)
	require.NoError(t, err)
	store.Set(CatastrophicFailureKey(failureID), bz)

	// Verify failure is now resolved
	resolvedFailure, err := k.GetCatastrophicFailure(ctx, failureID)
	require.NoError(t, err)
	require.True(t, resolvedFailure.Resolved)
	require.NotNil(t, resolvedFailure.ResolvedAt)

	// Verify no unresolved failures
	unresolvedFailures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, unresolvedFailures, 0, "should have no unresolved failures")

	// Verify invariant now passes
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "invariant should pass after resolution: %s", msg)
}

// TestCatastrophicFailureRecovery_TimeoutIndexMismatch tests detection of
// escrows with missing timeout indexes (catastrophic state)
func TestCatastrophicFailureRecovery_TimeoutIndexMismatch(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("timeout_requester"))
	provider := sdk.AccAddress([]byte("timeout_provider_"))

	// Create a LOCKED escrow without creating its timeout index (catastrophic state)
	escrow := types.EscrowState{
		RequestId:       50,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(5000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           50,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))

	// Do NOT create timeout index - this simulates catastrophic failure

	// Run EscrowStateConsistencyInvariant
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken for LOCKED escrow without timeout index")
	require.Contains(t, msg, "missing timeout index", "should indicate missing timeout index")
	require.Contains(t, msg, "escrow 50", "should mention request ID")
}

// TestCatastrophicFailureRecovery_ReleasedWithNilTimestamp tests detection of
// inconsistent escrow state (RELEASED status but nil ReleasedAt)
func TestCatastrophicFailureRecovery_ReleasedWithNilTimestamp(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("released_req_____"))
	provider := sdk.AccAddress([]byte("released_prov____"))

	// Create escrow with RELEASED status but nil ReleasedAt (catastrophic inconsistency)
	escrow := types.EscrowState{
		RequestId:       60,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(6000000),
		Status:          types.ESCROW_STATUS_RELEASED,
		LockedAt:        now.Add(-7200 * time.Second),
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleasedAt:      nil, // INCONSISTENT: should not be nil for RELEASED status
		ReleaseAttempts: 1,
		Nonce:           60,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))

	// Run EscrowStateConsistencyInvariant
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken for RELEASED escrow with nil timestamp")
	require.Contains(t, msg, "ReleasedAt is nil", "should indicate missing timestamp")
	require.Contains(t, msg, "escrow 60", "should mention request ID")
}

// TestCatastrophicFailureRecovery_RefundedWithNilTimestamp tests detection of
// inconsistent escrow state (REFUNDED status but nil RefundedAt)
func TestCatastrophicFailureRecovery_RefundedWithNilTimestamp(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("refunded_req_____"))
	provider := sdk.AccAddress([]byte("refunded_prov____"))

	// Create escrow with REFUNDED status but nil RefundedAt (catastrophic inconsistency)
	escrow := types.EscrowState{
		RequestId:       70,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(7000000),
		Status:          types.ESCROW_STATUS_REFUNDED,
		LockedAt:        now.Add(-7200 * time.Second),
		ExpiresAt:       now.Add(-100 * time.Second), // Expired
		RefundedAt:      nil,                         // INCONSISTENT: should not be nil for REFUNDED status
		ReleaseAttempts: 0,
		Nonce:           70,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))

	// Run EscrowStateConsistencyInvariant
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken for REFUNDED escrow with nil timestamp")
	require.Contains(t, msg, "RefundedAt is nil", "should indicate missing timestamp")
	require.Contains(t, msg, "escrow 70", "should mention request ID")
}

// TestCatastrophicFailureRecovery_OrphanedTimeoutIndexDetection tests that
// released/refunded escrows with orphaned timeout indexes are detected
func TestCatastrophicFailureRecovery_OrphanedTimeoutIndexDetection(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("orphan_requester_"))
	provider := sdk.AccAddress([]byte("orphan_provider__"))
	releasedAt := now

	// Create a RELEASED escrow
	escrow := types.EscrowState{
		RequestId:       80,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(8000000),
		Status:          types.ESCROW_STATUS_RELEASED,
		LockedAt:        now.Add(-7200 * time.Second),
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleasedAt:      &releasedAt,
		ReleaseAttempts: 1,
		Nonce:           80,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))

	// Manually create timeout index (should have been cleaned up when escrow was released)
	orphanedTimeoutKey := EscrowTimeoutKey(escrow.ExpiresAt, escrow.RequestId)
	store.Set(orphanedTimeoutKey, []byte{})

	// Run EscrowStateConsistencyInvariant
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.True(t, broken, "invariant should be broken for released escrow with orphaned timeout index")
	require.Contains(t, msg, "orphaned timeout index", "should indicate orphaned timeout index")
	require.Contains(t, msg, "escrow 80", "should mention request ID")
}

// TestCatastrophicFailureRecovery_CleanStatePassesInvariant tests that
// a clean state with no catastrophic failures passes the invariant
func TestCatastrophicFailureRecovery_CleanStatePassesInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("clean_requester__"))
	provider := sdk.AccAddress([]byte("clean_provider___"))

	// Create properly formed LOCKED escrow with timeout index
	escrow := types.EscrowState{
		RequestId:       90,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(9000000),
		Status:          types.ESCROW_STATUS_LOCKED,
		LockedAt:        now,
		ExpiresAt:       now.Add(3600 * time.Second),
		ReleaseAttempts: 0,
		Nonce:           90,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow.RequestId, escrow.ExpiresAt))

	// Verify no catastrophic failures
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, failures, 0, "should have no catastrophic failures")

	// Run EscrowStateConsistencyInvariant - should pass
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "invariant should pass with clean state: %s", msg)
}

// TestCatastrophicFailureRecovery_StorageFailure tests behavior when
// catastrophic failure storage itself fails (edge case)
func TestCatastrophicFailureRecovery_StorageFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	provider := sdk.AccAddress([]byte("storage_fail_prov"))

	// Test that recordCatastrophicFailure doesn't panic even if storage fails
	// (In a real scenario, this would emit an event even if storage fails)
	require.NotPanics(t, func() {
		k.recordCatastrophicFailure(ctx, 999, provider, math.NewInt(1000000), "test failure")
	}, "recordCatastrophicFailure should not panic")

	// Verify the failure was stored (in our test case, storage should succeed)
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1, "failure should be stored")
}

// TestCatastrophicFailureRecovery_ChallengedEscrowWithTimeoutIndex tests that
// CHALLENGED escrows properly maintain their timeout indexes
func TestCatastrophicFailureRecovery_ChallengedEscrowWithTimeoutIndex(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()
	requester := sdk.AccAddress([]byte("challenge_req____"))
	provider := sdk.AccAddress([]byte("challenge_prov___"))
	challengeEnds := now.Add(1800 * time.Second)

	// Create CHALLENGED escrow with timeout index
	escrow := types.EscrowState{
		RequestId:       95,
		Requester:       requester.String(),
		Provider:        provider.String(),
		Amount:          math.NewInt(9500000),
		Status:          types.ESCROW_STATUS_CHALLENGED,
		LockedAt:        now.Add(-3600 * time.Second),
		ExpiresAt:       now.Add(7200 * time.Second),
		ChallengeEndsAt: &challengeEnds,
		ReleaseAttempts: 1,
		Nonce:           95,
	}
	require.NoError(t, k.SetEscrowState(ctx, escrow))
	require.NoError(t, k.setEscrowTimeoutIndex(ctx, escrow.RequestId, escrow.ExpiresAt))

	// Run EscrowStateConsistencyInvariant - should pass
	invariant := EscrowStateConsistencyInvariant(*k)
	msg, broken := invariant(sdkCtx)
	require.False(t, broken, "invariant should pass for CHALLENGED escrow with timeout index: %s", msg)

	// Verify timeout index exists
	store := k.getStore(ctx)
	timeoutKey := EscrowTimeoutKey(escrow.ExpiresAt, escrow.RequestId)
	require.True(t, store.Has(timeoutKey), "timeout index should exist for CHALLENGED escrow")
}
