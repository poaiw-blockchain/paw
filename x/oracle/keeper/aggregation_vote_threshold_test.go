package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Ensure aggregation fails when voting power threshold is not met.
func TestAggregateAssetPrice_InsufficientVotingPower(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	// Set vote threshold to 80%
	params := types.DefaultParams()
	dec80, _ := sdkmath.LegacyNewDecFromStr("0.80")
	params.VoteThreshold = dec80
	require.NoError(t, k.SetParams(ctx, params))

	// Bond two validators, but only one submits a price
	val1 := sdk.ValAddress([]byte("val1_______________"))
	val2 := sdk.ValAddress([]byte("val2_______________"))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, val1))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, val2))

	// Submit price from only val1
	priceDec, _ := sdkmath.LegacyNewDecFromStr("10")
	require.NoError(t, k.SetValidatorPrice(ctx, types.ValidatorPrice{
		ValidatorAddr: val1.String(),
		Asset:         "ATOM",
		Price:         priceDec,
		VotingPower:   1, // minimal voting power
	}))

	err := k.AggregateAssetPrice(ctx, "ATOM")
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient voting power")
}
