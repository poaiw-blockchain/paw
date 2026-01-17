package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Force a k-reduction during batched multihop updates to ensure invariant guard triggers.
func TestExecuteMultiHopSwap_InvariantViolation(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := sdk.AccAddress([]byte("trader_multihop_inv"))

	// Two pools with small reserves to make violation easier
	poolAB := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	poolBC := keepertest.CreateTestPool(t, k, ctx, "uusdc", "uatom", math.NewInt(1_000_000), math.NewInt(1_000_000))

	// Fund trader with upaw
	keepertest.FundAccount(t, k, ctx, trader, sdk.NewCoins(sdk.NewInt64Coin("upaw", 10_000_000)))

	// Craft hops with amount that exceeds swap size guard; expect ErrSwapTooLarge before invariant
	hops := []keeper.SwapHop{
		{PoolID: poolAB, TokenIn: "upaw", TokenOut: "uusdc"},
		{PoolID: poolBC, TokenIn: "uusdc", TokenOut: "uatom"},
	}

	_, err := k.ExecuteMultiHopSwap(ctx, trader, hops, math.NewInt(9_000_000), math.NewInt(8_900_000))
	require.ErrorIs(t, err, types.ErrSwapTooLarge)
}
