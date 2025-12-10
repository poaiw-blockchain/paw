package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
