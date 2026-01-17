package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Verifies GetTotalProtocolFeesValue aggregates across denoms and ignores zero entries.
func TestGetTotalProtocolFeesValue(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())

	// Manually seed protocol fee buckets
	set := func(denom string, amt int64) {
		bz, err := math.NewInt(amt).Marshal()
		require.NoError(t, err)
		store.Set(append(types.ProtocolFeeKeyPrefix, []byte(denom)...), bz)
	}

	set("upaw", 1_000_000)
	set("uusdc", 2_500_000)
	set("zero", 0)

	coins, err := k.GetTotalProtocolFeesValue(ctx)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 1_000_000),
		sdk.NewInt64Coin("uusdc", 2_500_000),
	), coins)
}

// Malformed entries should be skipped while valid non-zero entries are aggregated.
func TestGetTotalProtocolFeesValue_SkipsCorruptEntries(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())

	// valid
	bz, err := math.NewInt(500).Marshal()
	require.NoError(t, err)
	store.Set(append(types.ProtocolFeeKeyPrefix, []byte("upaw")...), bz)

	// corrupt (non-int bytes)
	store.Set(append(types.ProtocolFeeKeyPrefix, []byte("bad")...), []byte("notanint"))

	coins, err := k.GetTotalProtocolFeesValue(ctx)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("upaw", 500)), coins)
}
