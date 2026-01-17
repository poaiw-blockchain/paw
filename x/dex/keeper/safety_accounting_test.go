package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app/ibcutil"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestSafeUpdateReserves(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	// Happy path adds and subtracts keep reserves positive
	newA, newB, err := k.SafeUpdateReserves(math.NewInt(1_000), math.NewInt(2_000), math.NewInt(500), math.NewInt(-300))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1_500), newA)
	require.Equal(t, math.NewInt(1_700), newB)

	// Underflow should be rejected
	_, _, err = k.SafeUpdateReserves(math.NewInt(100), math.NewInt(100), math.ZeroInt().Sub(math.NewInt(200)), math.ZeroInt())
	require.ErrorIs(t, err, types.ErrInsufficientLiquidity)

	// Zero reserve should be rejected
	_, _, err = k.SafeUpdateReserves(math.NewInt(10), math.NewInt(10), math.NewInt(-10), math.ZeroInt())
	require.ErrorIs(t, err, types.ErrInsufficientLiquidity)
}

func TestSafeValidateConstantProduct(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	// Valid: k increases
	require.NoError(t, k.SafeValidateConstantProduct(math.NewInt(100), math.NewInt(100), math.NewInt(110), math.NewInt(100)))

	// Invalid: k decreases
	err := k.SafeValidateConstantProduct(math.NewInt(100), math.NewInt(100), math.NewInt(110), math.NewInt(90))
	require.ErrorIs(t, err, types.ErrInvariantViolation)
}

func TestUpdateCommitRevealMetrics(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Seed one commitment to exercise iterator path
	store := sdkCtx.KVStore(k.GetStoreKey())
	store.Set(keeper.SwapCommitmentKey([]byte("hash123")), []byte{0x01})

	k.UpdateCommitRevealMetrics(ctx) // should not panic or error
}

func TestAuthorizedChannels(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	channels := []ibcutil.AuthorizedChannel{
		{PortId: "dex", ChannelId: "channel-1"},
		{PortId: "dex", ChannelId: "channel-2"},
	}

	require.NoError(t, k.SetAuthorizedChannels(ctx, channels))

	got, err := k.GetAuthorizedChannels(ctx)
	require.NoError(t, err)
	require.Equal(t, channels, got)
}

func TestCleanupInactivePools(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	// Set block time "now"
	now := time.Unix(1_700_000_000, 0)
	ctx = ctx.WithBlockTime(now)

	// Active pool with old timestamp (should be removed)
	require.NoError(t, k.MarkPoolActive(ctx, 1))
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:        1,
		LastTimestamp: now.Add(-48 * time.Hour).Unix(),
		TwapPrice:     math.LegacyNewDec(1),
	}))

	// Active pool with recent timestamp (should stay)
	require.NoError(t, k.MarkPoolActive(ctx, 2))
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:        2,
		LastTimestamp: now.Add(-6 * time.Hour).Unix(),
		TwapPrice:     math.LegacyNewDec(2),
	}))

	// Pool without TWAP record should be kept
	require.NoError(t, k.MarkPoolActive(ctx, 3))

	require.NoError(t, k.CleanupInactivePools(ctx))

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	require.Nil(t, store.Get(keeper.ActivePoolKey(1)), "stale pool should be removed")
	require.NotNil(t, store.Get(keeper.ActivePoolKey(2)), "recent pool should remain")
	require.NotNil(t, store.Get(keeper.ActivePoolKey(3)), "pool without TWAP should remain")
}
