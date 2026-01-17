package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// When all submissions are filtered, fallback to last price if available; otherwise error.
func TestAggregateAssetPrice_Fallbacks(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)

	// Set loose threshold to avoid voting power failure
	params := types.DefaultParams()
	params.VoteThreshold = sdkmath.LegacyNewDecWithPrec(5, 1) // 0.5
	require.NoError(t, k.SetParams(ctx, params))

	val := sdk.ValAddress([]byte("val_outlier_____"))
	require.NoError(t, keepertest.EnsureBondedValidatorWithKeeper(ctx, sk, val))

	// Submit a single price (will be kept, not filtered)
	require.NoError(t, k.SetValidatorPrice(ctx, types.ValidatorPrice{
		ValidatorAddr: val.String(),
		Asset:         "ATOM",
		Price:         sdkmath.LegacyNewDec(100),
		VotingPower:   10,
	}))

	// Store a last known price to exercise stale fallback later
	require.NoError(t, k.SetPrice(ctx, types.Price{
		Asset:       "ATOM",
		Price:       sdkmath.LegacyNewDec(90),
		BlockHeight: ctx.BlockHeight() - 10,
	}))

	// Case 1: normal path should succeed
	require.NoError(t, k.AggregateAssetPrice(ctx, "ATOM"))

	// Case 2: remove all current submissions to force fallback to stored price
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	store := ctx.KVStore(k.GetStoreKey())
	store.Delete(keeper.GetValidatorPriceKey(val, "ATOM"))
	store.Delete(keeper.GetValidatorPriceByAssetKey("ATOM", val))

	err := k.AggregateAssetPrice(ctx, "ATOM")
	require.NoError(t, err, "should fall back to stale price, not error")

	// Case 3: no fallback available -> expect error
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	// Remove stored price to eliminate fallback
	store = ctx.KVStore(k.GetStoreKey())
	store.Delete(keeper.GetPriceKey("ATOM"))

	err = k.AggregateAssetPrice(ctx, "ATOM")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no fallback available")
}
