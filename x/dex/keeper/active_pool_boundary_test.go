package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Tests ActivePoolTTL cutoff boundary (exactly at TTL should stay, before TTL should clean)
func TestCleanupInactivePoolsBoundary(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	now := time.Unix(1_700_000_000, 0)
	ctx = ctx.WithBlockTime(now)

	cutoff := now.Add(-keeper.ActivePoolTTL)

	// Pool A: exact cutoff timestamp -> should stay
	require.NoError(t, k.MarkPoolActive(ctx, 10))
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:        10,
		LastTimestamp: cutoff.Unix(),
		TwapPrice:     math.LegacyNewDec(1),
	}))

	// Pool B: older than cutoff -> should be removed
	require.NoError(t, k.MarkPoolActive(ctx, 11))
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:        11,
		LastTimestamp: cutoff.Add(-time.Second).Unix(),
		TwapPrice:     math.LegacyNewDec(1),
	}))

	require.NoError(t, k.CleanupInactivePools(ctx))

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	require.NotNil(t, store.Get(keeper.ActivePoolKey(10)))
	require.Nil(t, store.Get(keeper.ActivePoolKey(11)))
}
