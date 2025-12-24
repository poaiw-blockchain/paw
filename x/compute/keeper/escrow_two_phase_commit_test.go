package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestLockEscrow_TwoPhaseCommit_AllOrNothing verifies that LockEscrow uses
// two-phase commit so all operations succeed or all fail
func TestLockEscrow_TwoPhaseCommit_AllOrNothing(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := createTestProvider(t)

	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Get initial balances
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress(types.ModuleName)
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	initialRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Lock escrow
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	// Verify all state was committed atomically
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	finalRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Funds transferred
	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount.Add(amount)))
	require.True(t, finalRequesterBalance.Amount.Equal(initialRequesterBalance.Amount.Sub(amount)))

	// State saved
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_LOCKED, escrowState.Status)

	// Timeout index created
	// Note: We can't directly access the timeout index from tests, but we verify
	// by checking that escrow expires properly in ProcessExpiredEscrows tests
}

// TestLockEscrow_DuplicateRequestID_NoFundsLost verifies that attempting to
// lock escrow with duplicate request ID doesn't lose funds due to two-phase commit
func TestLockEscrow_DuplicateRequestID_NoFundsLost(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := createTestProvider(t)

	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Get initial balances
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress(types.ModuleName)
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	initialRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Lock escrow first time
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	balanceAfterFirstLock := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Try to lock again with same request ID
	err = k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	// Verify NO funds were lost - balance should remain unchanged after failed second lock
	finalRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")
	require.True(t, finalRequesterBalance.Amount.Equal(balanceAfterFirstLock.Amount),
		"funds were lost on failed duplicate lock")

	// Module balance should only increase by the first lock amount
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount.Add(amount)))

	// No catastrophic failure should be recorded
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures, "catastrophic failure recorded when none should exist")
}

// TestReleaseEscrow_TwoPhaseCommit_BankAndStateAtomic verifies that ReleaseEscrow
// uses two-phase commit so bank transfer and state update happen atomically
func TestReleaseEscrow_TwoPhaseCommit_BankAndStateAtomic(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := createTestProvider(t)

	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Lock escrow
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	// Get balances before release
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress(types.ModuleName)
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	initialProviderBalance := k.GetBankKeeper().GetBalance(sdkCtx, provider, "upaw")

	// Release escrow
	err = k.ReleaseEscrow(ctx, requestID, true)
	require.NoError(t, err)

	// Verify funds were transferred
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	finalProviderBalance := k.GetBankKeeper().GetBalance(sdkCtx, provider, "upaw")

	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount.Sub(amount)))
	require.True(t, finalProviderBalance.Amount.Equal(initialProviderBalance.Amount.Add(amount)))

	// Verify state was updated
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_RELEASED, escrowState.Status)
	require.NotNil(t, escrowState.ReleasedAt)

	// No catastrophic failure should be recorded
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures)
}

// TestReleaseEscrow_NonexistentEscrow_NoStateChange verifies that releasing
// a non-existent escrow doesn't cause any state changes
func TestReleaseEscrow_NonexistentEscrow_NoStateChange(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Try to release non-existent escrow
	err := k.ReleaseEscrow(ctx, 99999, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// No catastrophic failure should be recorded
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures)
}

// TestRefundEscrow_TwoPhaseCommit_BankAndStateAtomic verifies that RefundEscrow
// uses two-phase commit so bank transfer and state update happen atomically
func TestRefundEscrow_TwoPhaseCommit_BankAndStateAtomic(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := createTestProvider(t)

	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Lock escrow
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	// Get balances before refund
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress(types.ModuleName)
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	initialRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Refund escrow
	err = k.RefundEscrow(ctx, requestID, "test_refund")
	require.NoError(t, err)

	// Verify funds were refunded
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	finalRequesterBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount.Sub(amount)))
	require.True(t, finalRequesterBalance.Amount.Equal(initialRequesterBalance.Amount.Add(amount)))

	// Verify state was updated
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_REFUNDED, escrowState.Status)
	require.NotNil(t, escrowState.RefundedAt)

	// No catastrophic failure should be recorded
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures)
}

// TestRefundEscrow_AlreadyRefunded_Idempotent verifies that refunding
// an already-refunded escrow is idempotent and doesn't transfer funds twice
func TestRefundEscrow_AlreadyRefunded_Idempotent(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	requester := createTestRequester(t)
	provider := createTestProvider(t)

	amount := math.NewInt(10000000)
	requestID := uint64(1)
	timeoutSeconds := uint64(3600)

	// Lock and refund escrow
	err := k.LockEscrow(ctx, requester, provider, amount, requestID, timeoutSeconds)
	require.NoError(t, err)

	err = k.RefundEscrow(ctx, requestID, "test_refund")
	require.NoError(t, err)

	// Get balance after first refund
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	balanceAfterFirstRefund := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")

	// Try to refund again
	err = k.RefundEscrow(ctx, requestID, "second_refund")
	require.NoError(t, err) // Should be idempotent

	// Balance should be unchanged
	finalBalance := k.GetBankKeeper().GetBalance(sdkCtx, requester, "upaw")
	require.True(t, finalBalance.Amount.Equal(balanceAfterFirstRefund.Amount),
		"funds were transferred twice on duplicate refund")

	// State should still show single refund
	escrowState, err := k.GetEscrowState(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, types.ESCROW_STATUS_REFUNDED, escrowState.Status)
}

// TestEscrowOperations_NoLeakedFunds verifies that no funds are leaked
// across multiple escrow operations
func TestEscrowOperations_NoLeakedFunds(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	moduleAddr := k.GetModuleAddress(types.ModuleName)

	// Track initial module balance
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")

	// Create multiple escrows
	for i := 1; i <= 5; i++ {
		requester := createTestRequester(t)
		provider := createTestProvider(t)
		amount := math.NewInt(1000000 * int64(i))
		requestID := uint64(i)

		err := k.LockEscrow(ctx, requester, provider, amount, requestID, 3600)
		require.NoError(t, err)
	}

	// Release half, refund half
	for i := 1; i <= 5; i++ {
		requestID := uint64(i)
		if i%2 == 0 {
			err := k.ReleaseEscrow(ctx, requestID, true)
			require.NoError(t, err)
		} else {
			err := k.RefundEscrow(ctx, requestID, "test")
			require.NoError(t, err)
		}
	}

	// All funds should have been released/refunded
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount),
		"funds leaked: initial=%s final=%s", initialModuleBalance.Amount, finalModuleBalance.Amount)

	// No catastrophic failures
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures)
}

// TestProcessExpiredEscrows_AtomicRefunds verifies that processing expired
// escrows maintains atomicity even when processing multiple escrows
func TestProcessExpiredEscrows_AtomicRefunds(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Now().UTC()
	ctx = ctx.WithBlockTime(blockTime)

	moduleAddr := k.GetModuleAddress(types.ModuleName)
	initialModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")

	// Create escrows with short timeout
	totalExpiring := math.ZeroInt()
	for i := 1; i <= 3; i++ {
		requester := createTestRequester(t)
		provider := createTestProvider(t)
		amount := math.NewInt(1000000)
		requestID := uint64(i)

		err := k.LockEscrow(ctx, requester, provider, amount, requestID, 100)
		require.NoError(t, err)
		totalExpiring = totalExpiring.Add(amount)
	}

	// Advance time past expiration
	newBlockTime := blockTime.Add(200 * time.Second)
	ctx = ctx.WithBlockTime(newBlockTime)
	sdkCtx = sdk.UnwrapSDKContext(ctx)

	// Process expired escrows
	err := k.ProcessExpiredEscrows(ctx)
	require.NoError(t, err)

	// All expired escrows should be refunded
	for i := 1; i <= 3; i++ {
		escrowState, err := k.GetEscrowState(ctx, uint64(i))
		require.NoError(t, err)
		require.Equal(t, types.ESCROW_STATUS_REFUNDED, escrowState.Status)
	}

	// Module balance should be back to initial (all funds refunded)
	finalModuleBalance := k.GetBankKeeper().GetBalance(sdkCtx, moduleAddr, "upaw")
	require.True(t, finalModuleBalance.Amount.Equal(initialModuleBalance.Amount))

	// No catastrophic failures
	failures, err := k.GetUnresolvedCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Empty(t, failures)
}
