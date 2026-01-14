package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestAllInvariants(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("all invariants pass on clean state", func(t *testing.T) {
		invariant := AllInvariants(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariants should not be broken on clean state: %s", msg)
	})
}

func TestEscrowBalanceInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("passes on clean state", func(t *testing.T) {
		invariant := EscrowBalanceInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "escrow balance invariant should not be broken: %s", msg)
	})
}

func TestProviderStakeInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("passes on clean state", func(t *testing.T) {
		invariant := ProviderStakeInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "provider stake invariant should not be broken: %s", msg)
	})
}

func TestRequestStatusInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("passes on clean state", func(t *testing.T) {
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "request status invariant should not be broken: %s", msg)
	})

	t.Run("detects completed request without finalization flag", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Create a completed request without setting finalization flag
		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))
		now := sdkCtx.BlockTime()

		request := types.Request{
			Id:             1,
			Requester:      requester.String(),
			Provider:       provider.String(),
			Status:         types.REQUEST_STATUS_COMPLETED,
			EscrowedAmount: sdkmath.NewInt(100000),
			MaxPayment:     sdkmath.NewInt(100000),
			CreatedAt:      now.Add(-3600 * 1e9),
			CompletedAt:    &now,
			Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
			ContainerImage: "test/image",
		}

		err := k.SetRequest(ctx, request)
		require.NoError(t, err)

		// Create result for the request
		result := &types.Result{
			RequestId:   1,
			Provider:    provider.String(),
			OutputHash:  "test_hash",
			SubmittedAt: now,
			Verified:    true,
		}
		err = k.SetResult(ctx, result)
		require.NoError(t, err)

		// Verify invariant is broken (no finalization flag set)
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.True(t, broken, "invariant should be broken when completed request is not finalized")
		require.Contains(t, msg, "not finalized")
		require.Contains(t, msg, "missing settlement flag")
	})

	t.Run("passes with properly finalized completed request", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Create a completed request WITH finalization flag
		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))
		now := sdkCtx.BlockTime()

		request := types.Request{
			Id:             1,
			Requester:      requester.String(),
			Provider:       provider.String(),
			Status:         types.REQUEST_STATUS_COMPLETED,
			EscrowedAmount: sdkmath.NewInt(100000),
			MaxPayment:     sdkmath.NewInt(100000),
			CreatedAt:      now.Add(-3600 * 1e9),
			CompletedAt:    &now,
			Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
			ContainerImage: "test/image",
		}

		err := k.SetRequest(ctx, request)
		require.NoError(t, err)

		// Create result
		result := &types.Result{
			RequestId:   1,
			Provider:    provider.String(),
			OutputHash:  "test_hash",
			SubmittedAt: now,
			Verified:    true,
		}
		err = k.SetResult(ctx, result)
		require.NoError(t, err)

		// Mark request as finalized
		k.markRequestFinalized(ctx, 1)

		// Verify invariant passes
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken when request is properly finalized: %s", msg)
	})

	t.Run("passes with multiple completed requests all finalized", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		now := sdkCtx.BlockTime()

		// Create multiple completed requests, all finalized
		for i := uint64(1); i <= 5; i++ {
			provider := sdk.AccAddress([]byte("test_provider_" + string(rune(i))))

			request := types.Request{
				Id:             i,
				Requester:      requester.String(),
				Provider:       provider.String(),
				Status:         types.REQUEST_STATUS_COMPLETED,
				EscrowedAmount: sdkmath.NewInt(100000),
				MaxPayment:     sdkmath.NewInt(100000),
				CreatedAt:      now.Add(-3600 * 1e9),
				CompletedAt:    &now,
				Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
				ContainerImage: "test/image",
			}

			err := k.SetRequest(ctx, request)
			require.NoError(t, err)

			result := &types.Result{
				RequestId:   i,
				Provider:    provider.String(),
				OutputHash:  "test_hash",
				SubmittedAt: now,
				Verified:    true,
			}
			err = k.SetResult(ctx, result)
			require.NoError(t, err)

			// Mark as finalized
			k.markRequestFinalized(ctx, i)
		}

		// Verify invariant passes
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken when all requests are finalized: %s", msg)
	})

	t.Run("detects one unfinalized among multiple completed requests", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		now := sdkCtx.BlockTime()

		// Create 5 completed requests, finalize only 4
		for i := uint64(1); i <= 5; i++ {
			provider := sdk.AccAddress([]byte("test_provider_" + string(rune(i))))

			request := types.Request{
				Id:             i,
				Requester:      requester.String(),
				Provider:       provider.String(),
				Status:         types.REQUEST_STATUS_COMPLETED,
				EscrowedAmount: sdkmath.NewInt(100000),
				MaxPayment:     sdkmath.NewInt(100000),
				CreatedAt:      now.Add(-3600 * 1e9),
				CompletedAt:    &now,
				Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
				ContainerImage: "test/image",
			}

			err := k.SetRequest(ctx, request)
			require.NoError(t, err)

			result := &types.Result{
				RequestId:   i,
				Provider:    provider.String(),
				OutputHash:  "test_hash",
				SubmittedAt: now,
				Verified:    true,
			}
			err = k.SetResult(ctx, result)
			require.NoError(t, err)

			// Finalize all except request 3
			if i != 3 {
				k.markRequestFinalized(ctx, i)
			}
		}

		// Verify invariant is broken
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.True(t, broken, "invariant should be broken when one request is not finalized")
		require.Contains(t, msg, "request 3")
		require.Contains(t, msg, "not finalized")
	})

	t.Run("passes with pending and processing requests (no finalization needed)", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))
		now := sdkCtx.BlockTime()

		// Create pending request
		pendingRequest := types.Request{
			Id:             1,
			Requester:      requester.String(),
			Status:         types.REQUEST_STATUS_PENDING,
			EscrowedAmount: sdkmath.NewInt(100000),
			MaxPayment:     sdkmath.NewInt(100000),
			CreatedAt:      now,
			Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
			ContainerImage: "test/image",
		}
		err := k.SetRequest(ctx, pendingRequest)
		require.NoError(t, err)

		// Create processing request
		processingRequest := types.Request{
			Id:             2,
			Requester:      requester.String(),
			Provider:       provider.String(),
			Status:         types.REQUEST_STATUS_PROCESSING,
			EscrowedAmount: sdkmath.NewInt(100000),
			MaxPayment:     sdkmath.NewInt(100000),
			CreatedAt:      now,
			Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024},
			ContainerImage: "test/image",
		}
		err = k.SetRequest(ctx, processingRequest)
		require.NoError(t, err)

		// Verify invariant passes (finalization not needed for non-completed requests)
		invariant := RequestStatusInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken for non-completed requests: %s", msg)
	})
}

func TestNonceUniquenessInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("passes on clean state", func(t *testing.T) {
		invariant := NonceUniquenessInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "nonce uniqueness invariant should not be broken: %s", msg)
	})
}

func TestEscrowTimeoutIndexInvariant(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("passes on clean state", func(t *testing.T) {
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "escrow timeout index invariant should not be broken: %s", msg)
	})

	t.Run("passes with valid locked escrow", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Create test addresses (these are pre-funded by setupKeeperForTest)
		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))

		// Lock an escrow
		err := k.LockEscrow(ctx, requester, provider, sdkmath.NewInt(10000000), 1, 3600)
		require.NoError(t, err)

		// Verify invariant passes
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken: %s", msg)
	})

	t.Run("passes with multiple valid escrows", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))

		// Create multiple escrows
		for i := uint64(1); i <= 5; i++ {
			provider := sdk.AccAddress([]byte("test_provider_" + string(rune(i))))
			err := k.LockEscrow(ctx, requester, provider, sdkmath.NewInt(1000000), i, 3600)
			require.NoError(t, err)
		}

		// Verify invariant passes
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken: %s", msg)
	})

	t.Run("detects missing timeout index", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// Create escrow state directly without timeout index
		escrowState := types.EscrowState{
			RequestId:       1,
			Requester:       sdk.AccAddress([]byte("test_requester_addr")).String(),
			Provider:        sdk.AccAddress([]byte("test_provider_addr_")).String(),
			Amount:          sdkmath.NewInt(10000000),
			Status:          types.ESCROW_STATUS_LOCKED,
			LockedAt:        sdkCtx.BlockTime(),
			ExpiresAt:       sdkCtx.BlockTime().Add(3600 * 1e9),
			ReleaseAttempts: 0,
			Nonce:           1,
		}

		err := k.SetEscrowState(ctx, escrowState)
		require.NoError(t, err)

		// Verify invariant is broken
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.True(t, broken, "invariant should be broken when timeout index is missing")
		require.Contains(t, msg, "no timeout index entry")
	})

	t.Run("detects orphaned timeout index", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		store := k.getStore(ctx)

		// Create timeout index without escrow state
		requestID := uint64(99)
		expiresAt := sdkCtx.BlockTime().Add(3600 * 1e9)
		timeoutKey := EscrowTimeoutKey(expiresAt, requestID)
		store.Set(timeoutKey, []byte{})

		// Verify invariant is broken
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.True(t, broken, "invariant should be broken when escrow state is missing")
		require.Contains(t, msg, "has no escrow state")
	})

	t.Run("detects released escrow with timeout index", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		store := k.getStore(ctx)

		// Create released escrow state
		now := sdkCtx.BlockTime()
		escrowState := types.EscrowState{
			RequestId:       1,
			Requester:       sdk.AccAddress([]byte("test_requester_addr")).String(),
			Provider:        sdk.AccAddress([]byte("test_provider_addr_")).String(),
			Amount:          sdkmath.NewInt(10000000),
			Status:          types.ESCROW_STATUS_RELEASED,
			LockedAt:        now.Add(-7200 * 1e9),
			ExpiresAt:       now.Add(3600 * 1e9),
			ReleasedAt:      &now,
			ReleaseAttempts: 1,
			Nonce:           1,
		}

		err := k.SetEscrowState(ctx, escrowState)
		require.NoError(t, err)

		// Create timeout index (shouldn't exist for released escrow)
		timeoutKey := EscrowTimeoutKey(escrowState.ExpiresAt, 1)
		store.Set(timeoutKey, []byte{})

		// Verify invariant is broken
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.True(t, broken, "invariant should be broken when released escrow has timeout index")
		require.Contains(t, msg, "still has timeout index entry")
	})

	t.Run("passes after escrow release", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))

		// Lock escrow
		err := k.LockEscrow(ctx, requester, provider, sdkmath.NewInt(10000000), 1, 3600)
		require.NoError(t, err)

		// Release escrow
		err = k.ReleaseEscrow(ctx, 1, true)
		require.NoError(t, err)

		// Verify invariant still passes (timeout index should be cleaned up)
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken after release: %s", msg)
	})

	t.Run("passes after escrow refund", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))

		// Lock escrow
		err := k.LockEscrow(ctx, requester, provider, sdkmath.NewInt(10000000), 1, 3600)
		require.NoError(t, err)

		// Refund escrow
		err = k.RefundEscrow(ctx, 1, "test_refund")
		require.NoError(t, err)

		// Verify invariant still passes (timeout index should be cleaned up)
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken after refund: %s", msg)
	})

	t.Run("passes with challenged escrow", func(t *testing.T) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		requester := sdk.AccAddress([]byte("test_requester_addr"))
		provider := sdk.AccAddress([]byte("test_provider_addr_"))

		// Lock escrow
		err := k.LockEscrow(ctx, requester, provider, sdkmath.NewInt(10000000), 1, 3600)
		require.NoError(t, err)

		// Update to CHALLENGED status
		escrowState, err := k.GetEscrowState(ctx, 1)
		require.NoError(t, err)

		challengeEnds := sdkCtx.BlockTime().Add(1800 * 1e9)
		escrowState.Status = types.ESCROW_STATUS_CHALLENGED
		escrowState.ChallengeEndsAt = &challengeEnds

		err = k.SetEscrowState(ctx, *escrowState)
		require.NoError(t, err)

		// Verify invariant passes (challenged escrows should have timeout index)
		invariant := EscrowTimeoutIndexInvariant(*k)
		msg, broken := invariant(sdkCtx)
		require.False(t, broken, "invariant should not be broken for challenged escrow: %s", msg)
	})
}
