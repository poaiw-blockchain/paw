package keeper

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app/ibcutil"
)

func TestKeeper_GetStore(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("returns valid store", func(t *testing.T) {
		store := k.getStore(ctx)
		require.NotNil(t, store)
	})
}

func TestKeeper_IsAuthorizedChannel(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("unknown channel is not authorized", func(t *testing.T) {
		authorized := ibcutil.IsAuthorizedChannel(sdkCtx, k, "compute", "unknown-channel")
		require.False(t, authorized)
	})
}

func TestKeeper_AuthorizeChannel(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("authorize channel", func(t *testing.T) {
		err := ibcutil.AuthorizeChannel(sdkCtx, k, "compute", "channel-0")
		require.NoError(t, err)
		authorized := ibcutil.IsAuthorizedChannel(sdkCtx, k, "compute", "channel-0")
		require.True(t, authorized)
	})
}

func TestKeeper_SetAuthorizedChannelsWithValidation(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	t.Run("set authorized channels", func(t *testing.T) {
		channels := []ibcutil.AuthorizedChannel{
			{PortId: "compute", ChannelId: "channel-0"},
			{PortId: "compute", ChannelId: "channel-1"},
		}
		err := ibcutil.SetAuthorizedChannelsWithValidation(sdkCtx, k, channels)
		require.NoError(t, err)

		require.True(t, ibcutil.IsAuthorizedChannel(sdkCtx, k, "compute", "channel-0"))
		require.True(t, ibcutil.IsAuthorizedChannel(sdkCtx, k, "compute", "channel-1"))
	})
}

func TestKeeper_GetCircuitManager(t *testing.T) {
	k, _ := setupKeeperForTest(t)

	t.Run("returns circuit manager", func(t *testing.T) {
		cm := k.GetCircuitManager()
		require.NotNil(t, cm)
	})
}

func TestKeeper_InitializeCircuits(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("initializes circuits", func(t *testing.T) {
		// Stub groth16 setup to avoid expensive key generation in unit tests.
		original := groth16Setup
		groth16Setup = func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
			return groth16.NewProvingKey(ecc.BN254), groth16.NewVerifyingKey(ecc.BN254), nil
		}
		t.Cleanup(func() {
			groth16Setup = original
		})

		err := k.InitializeCircuits(ctx)
		require.NoError(t, err)
	})
}
