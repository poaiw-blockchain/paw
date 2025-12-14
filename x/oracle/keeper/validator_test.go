package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

func TestIsAuthorizedFeeder(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	valA := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	valB := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, valA))
	require.NoError(t, keepertest.EnsureBondedValidator(ctx, valB))

	delegate := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	// Without any delegation the delegate is free to be assigned
	require.True(t, k.IsAuthorizedFeeder(ctx, delegate, valA))

	// Once assigned to validator B, validator A should be blocked from reusing it
	require.NoError(t, k.SetFeederDelegation(ctx, valB, delegate))
	require.True(t, k.IsAuthorizedFeeder(ctx, delegate, valB))
	require.False(t, k.IsAuthorizedFeeder(ctx, delegate, valA))
}
