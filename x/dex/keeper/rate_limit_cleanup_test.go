package keeper_test

import (
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
)

// Seeds rate-limit entries within the cleanup scan window to verify pruning.
func TestCleanupOldRateLimitData(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Choose height so cutoffHeight = 20; scan window covers 10-19
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(86420)
	ctx = sdkCtx
	store := sdkCtx.KVStore(k.GetStoreKey())

	user := sdk.AccAddress("ratelimit_user______")
	window := int64(60)

	expiredHeight := int64(15) // within scan window -> should be pruned
	recentHeight := int64(30)  // above cutoff -> should remain

	expiredKey := keeper.RateLimitByHeightKey(expiredHeight, user, window)
	recentKey := keeper.RateLimitByHeightKey(recentHeight, user, window)
	expiredRL := keeper.RateLimitKey(user, window)

	store.Set(expiredKey, []byte{1})
	store.Set(recentKey, []byte{1})
	store.Set(expiredRL, []byte{2})

	require.NoError(t, k.CleanupOldRateLimitData(ctx))

	require.Nil(t, store.Get(expiredKey))
	require.Nil(t, store.Get(expiredRL))
	require.NotNil(t, store.Get(recentKey))

	found := false
	for _, ev := range sdkCtx.EventManager().Events() {
		if ev.Type == "rate_limit_data_cleaned" {
			found = true
			val := ev.Attributes[2].Value // rate_limits_cleaned
			count, _ := binary.Uvarint([]byte(val))
			require.GreaterOrEqual(t, int(count), 1)
		}
	}
	require.True(t, found)
}
