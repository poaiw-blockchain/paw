package keeper

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestStoreCatastrophicFailure verifies that catastrophic failures are persisted to state
func TestStoreCatastrophicFailure(t *testing.T) {
	keeper, ctx := setupKeeperForTest(t)

	// Create test account
	account := sdk.AccAddress([]byte("test_account"))
	amount := math.NewInt(1000000)
	requestID := uint64(123)
	reason := "test failure: funds sent but state update failed"

	// Store catastrophic failure
	err := keeper.StoreCatastrophicFailure(ctx, requestID, account, amount, reason)
	require.NoError(t, err, "storing catastrophic failure should succeed")

	// Verify it was stored
	failures, err := keeper.GetAllCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1, "should have exactly one catastrophic failure")

	failure := failures[0]
	require.Equal(t, requestID, failure.RequestId)
	require.Equal(t, account.String(), failure.Account)
	require.True(t, failure.Amount.Equal(amount))
	require.Equal(t, reason, failure.Reason)
	require.False(t, failure.Resolved)
	require.Nil(t, failure.ResolvedAt)
}

// TestGetCatastrophicFailure verifies retrieving a single failure by ID
func TestGetCatastrophicFailure(t *testing.T) {
	keeper, ctx := setupKeeperForTest(t)

	account := sdk.AccAddress([]byte("test_account"))
	amount := math.NewInt(2000000)
	requestID := uint64(456)
	reason := "refund sent but state update failed"

	// Store failure
	err := keeper.StoreCatastrophicFailure(ctx, requestID, account, amount, reason)
	require.NoError(t, err)

	// Get by ID (should be ID 1 since it's the first)
	failure, err := keeper.GetCatastrophicFailure(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, failure)
	require.Equal(t, uint64(1), failure.Id)
	require.Equal(t, requestID, failure.RequestId)
	require.Equal(t, account.String(), failure.Account)
	require.True(t, failure.Amount.Equal(amount))
	require.Equal(t, reason, failure.Reason)
}

// TestGetCatastrophicFailureNotFound verifies error handling for missing failures
func TestGetCatastrophicFailureNotFound(t *testing.T) {
	keeper, ctx := setupKeeperForTest(t)

	// Try to get non-existent failure
	failure, err := keeper.GetCatastrophicFailure(ctx, 999)
	require.Error(t, err)
	require.Nil(t, failure)
	require.Contains(t, err.Error(), "not found")
}

// TestRecordCatastrophicFailureStoresAndEmits verifies the recordCatastrophicFailure helper
func TestRecordCatastrophicFailureStoresAndEmits(t *testing.T) {
	keeper, ctx := setupKeeperForTest(t)

	account := sdk.AccAddress([]byte("test_account"))
	amount := math.NewInt(5000000)
	requestID := uint64(789)
	reason := "critical: payment sent but state update failed"

	// Record catastrophic failure (should both emit event and store)
	keeper.recordCatastrophicFailure(ctx, requestID, account, amount, reason)

	// Verify it was stored
	failures, err := keeper.GetAllCatastrophicFailures(ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)

	// Verify event was emitted
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()

	foundEvent := false
	for _, event := range events {
		if event.Type == "escrow_catastrophic_failure" {
			foundEvent = true
			break
		}
	}

	require.True(t, foundEvent, "should have emitted escrow_catastrophic_failure event")
}
