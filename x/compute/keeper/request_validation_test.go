package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

func TestValidateRequesterBalance(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	unfunded := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	err := k.ValidateRequesterBalance(ctx, unfunded, sdkmath.NewInt(1_000_000))
	require.Error(t, err)

	funded := sdk.AccAddress([]byte("test_requester_addr"))
	err = k.ValidateRequesterBalance(ctx, funded, sdkmath.NewInt(500_000_000))
	require.NoError(t, err)
}
