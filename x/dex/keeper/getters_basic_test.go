package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Ensures GetAllSwapCommits returns all active commitments in storage.
func TestGetAllSwapCommits(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	trader := types.TestAddr()

	pool, err := k.CreatePool(ctx, trader, "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	commitHash1 := keeper.ComputeSwapCommitmentHash(pool.Id, "upaw", "uusdc", math.NewInt(10_000), math.NewInt(1_000), []byte("salt1"), trader)
	commitHash2 := keeper.ComputeSwapCommitmentHash(pool.Id, "uusdc", "upaw", math.NewInt(5_000), math.NewInt(500), []byte("salt2"), trader)
	commitHash1Str := string(commitHash1)
	commitHash2Str := string(commitHash2)

	require.NoError(t, k.SetSwapCommit(ctx, types.SwapCommit{
		Trader:       trader.String(),
		SwapHash:     commitHash1Str,
		CommitHeight: 1,
		ExpiryHeight: 100,
	}))
	require.NoError(t, k.SetSwapCommit(ctx, types.SwapCommit{
		Trader:       trader.String(),
		SwapHash:     commitHash2Str,
		CommitHeight: 1,
		ExpiryHeight: 100,
	}))

	commits := k.GetAllSwapCommits(ctx)
	require.Len(t, commits, 2)

	var hashes []string
	for _, c := range commits {
		hashes = append(hashes, c.SwapHash)
	}

	require.Contains(t, hashes, commitHash1Str)
	require.Contains(t, hashes, commitHash2Str)
}

// Verifies LP fee retrieval for existing and missing entries.
func TestGetPoolLPFees(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.GetStoreKey())

	// No fees stored -> zero
	fee, err := k.GetPoolLPFees(ctx, 1, "upaw")
	require.NoError(t, err)
	require.True(t, fee.IsZero())

	// Seed fee value and verify retrieval
	expected := math.NewInt(123_456)
	bz, err := expected.Marshal()
	require.NoError(t, err)
	store.Set(types.GetPoolLPFeeKey(1, "upaw"), bz)

	fee, err = k.GetPoolLPFees(ctx, 1, "upaw")
	require.NoError(t, err)
	require.True(t, fee.Equal(expected))
}

// Validates liquidity share getter handling present and missing shares.
func TestGetLiquidityShares(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	lp := types.TestAddr()

	// Missing shares -> zero
	shares, err := k.GetLiquidityShares(ctx, 1, lp)
	require.NoError(t, err)
	require.True(t, shares.IsZero())

	// Set shares and verify round-trip
	expected := math.NewInt(777_000)
	require.NoError(t, k.SetLiquidityShares(ctx, 1, lp, expected))

	shares, err = k.GetLiquidityShares(ctx, 1, lp)
	require.NoError(t, err)
	require.True(t, shares.Equal(expected))
}

// Exercises last-liquidity-action tracking, including not-found and stored cases.
func TestGetLastLiquidityActionBlock(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	lp := types.TestAddr()

	height, found, err := k.GetLastLiquidityActionBlock(ctx, 1, lp)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, int64(0), height)

	// Record an action at current block height
	require.NoError(t, k.SetLastLiquidityActionBlock(ctx, 1, lp))

	height, found, err = k.GetLastLiquidityActionBlock(ctx, 1, lp)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, sdk.UnwrapSDKContext(ctx).BlockHeight(), height)
}

// Confirms limit order ID counter increments monotonically from 1.
func TestGetNextOrderID(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	id1, err := k.GetNextOrderID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	id2, err := k.GetNextOrderID(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)
}

// Validates paginated liquidity iteration including next-key cursor.
func TestIterateLiquidityByPoolPaginated(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	// Create a pool and three providers with deterministic lexicographic ordering.
	poolID, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	// Remove creator's initial liquidity to isolate pagination dataset.
	require.NoError(t, k.SetLiquidity(ctx, poolID.Id, types.TestAddr(), math.ZeroInt()))

	providers := []sdk.AccAddress{
		sdk.AccAddress([]byte("aaaa_provider_______")),
		sdk.AccAddress([]byte("aaab_provider_______")),
		sdk.AccAddress([]byte("aaac_provider_______")),
	}
	shares := []math.Int{math.NewInt(100), math.NewInt(200), math.NewInt(300)}
	for i, p := range providers {
		require.NoError(t, k.SetLiquidity(ctx, poolID.Id, p, shares[i]))
		got, err := k.GetLiquidity(ctx, poolID.Id, p)
		require.NoError(t, err)
		require.True(t, got.Equal(shares[i]))
	}

	// First page (limit 2)
	res, err := k.IterateLiquidityByPoolPaginated(ctx, poolID.Id, nil, 2)
	require.NoError(t, err)
	require.Len(t, res.Positions, 2)
	require.Equal(t, providers[0].String(), res.Positions[0].Provider.String())
	require.Equal(t, providers[1].String(), res.Positions[1].Provider.String())
	require.NotNil(t, res.NextKey)

	// Second page starting after previous last provider
	next := sdk.AccAddress(res.NextKey)
	res2, err := k.IterateLiquidityByPoolPaginated(ctx, poolID.Id, next, 2)
	require.NoError(t, err)
	require.Len(t, res2.Positions, 1)
	require.Equal(t, providers[2].String(), res2.Positions[0].Provider.String())
	require.Nil(t, res2.NextKey)
}

// Corrupt shares entry should raise an error during pagination.
func TestIterateLiquidityByPoolPaginated_CorruptEntry(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	poolID, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	provider := sdk.AccAddress(make([]byte, 20)) // lexicographically first
	key := append(keeper.LiquidityKeyByPoolPrefix(poolID.Id), provider.Bytes()...)
	store.Set(key, []byte("corrupt"))

	_, err = k.IterateLiquidityByPoolPaginated(ctx, poolID.Id, nil, 1)
	require.Error(t, err)
}

// Ensures pool count helpers handle set/increment/decrement and never go negative.
func TestTotalPoolsCount_SetIncrementDecrement(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	require.Equal(t, uint64(0), k.GetTotalPoolsCount(ctx))

	k.IncrementTotalPoolsCount(ctx)
	k.IncrementTotalPoolsCount(ctx)
	require.Equal(t, uint64(2), k.GetTotalPoolsCount(ctx))

	k.DecrementTotalPoolsCount(ctx)
	require.Equal(t, uint64(1), k.GetTotalPoolsCount(ctx))

	k.SetTotalPoolsCount(ctx, 10)
	require.Equal(t, uint64(10), k.GetTotalPoolsCount(ctx))

	// Decrement below zero should clamp at zero
	for i := 0; i < 15; i++ {
		k.DecrementTotalPoolsCount(ctx)
	}
	require.Equal(t, uint64(0), k.GetTotalPoolsCount(ctx))
}

// Verifies pool version monotonic increment for graph cache invalidation.
func TestIncrementPoolVersion(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	require.Equal(t, uint64(0), k.GetPoolVersion(ctx))
	k.IncrementPoolVersion(ctx)
	require.Equal(t, uint64(1), k.GetPoolVersion(ctx))
	k.IncrementPoolVersion(ctx)
	require.Equal(t, uint64(2), k.GetPoolVersion(ctx))
}

// Counts liquidity providers for a pool.
func TestGetLiquidityProviderCount(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	poolID, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	// Remove creator liquidity so only test providers counted.
	require.NoError(t, k.SetLiquidity(ctx, poolID.Id, types.TestAddr(), math.ZeroInt()))

	providers := []sdk.AccAddress{
		sdk.AccAddress([]byte("count_provider_1___")),
		sdk.AccAddress([]byte("count_provider_2___")),
	}
	for _, p := range providers {
		require.NoError(t, k.SetLiquidity(ctx, poolID.Id, p, math.NewInt(123)))
	}

	count, err := k.GetLiquidityProviderCount(ctx, poolID.Id)
	require.NoError(t, err)
	require.Equal(t, uint64(len(providers)), count)
}

// OnAcknowledgementSwapPacket emits failure event when ack contains error.
func TestOnAcknowledgementSwapPacket_ErrorEvent(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	packet := channeltypes.Packet{
		Data: []byte(`{"swap_id":"swap123"}`),
	}
	ack := channeltypes.NewResultAcknowledgement([]byte(`{"error":"failed"}`))

	err := k.OnAcknowledgementSwapPacket(ctx, packet, ack)
	require.NoError(t, err)

	found := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == types.EventTypeDexCrossChainSwapFailed {
			found = true
			requireEventAttr(t, ev, types.AttributeKeySwapID, "swap123")
			requireEventAttr(t, ev, types.AttributeKeyError, "failed")
		}
	}
	require.True(t, found, "expected cross-chain swap failure event")
}

// Dex hooks setter should set once and panic on duplicate registration.
func TestSetHooks(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	h := &dummyHooks{}
	k.SetHooks(h)

	// Returned hooks should be identical
	require.Equal(t, h, k.GetHooks())

	// Second call should panic to prevent double registration
	require.Panics(t, func() { k.SetHooks(h) })

	// Verify hooks are callable via MultiDexHooks path
	err := k.GetHooks().AfterSwap(ctx, 1, "sender", "upaw", "uusdc", math.NewInt(1), math.NewInt(2))
	require.NoError(t, err)
	require.True(t, h.afterSwapCalled)
}

// Channel capability getter should return not-found when untouched.
func TestGetChannelCapability_NotFound(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	cap, found := k.GetChannelCapability(ctx, "dex", "channel-0")
	require.False(t, found)
	require.Nil(t, cap)
}

// dummyHooks implements types.DexHooks for testing.
type dummyHooks struct {
	afterSwapCalled bool
}

func (d *dummyHooks) AfterSwap(ctx context.Context, poolID uint64, sender string, tokenIn, tokenOut string, amountIn, amountOut math.Int) error {
	d.afterSwapCalled = true
	return nil
}

func (d *dummyHooks) AfterPoolCreated(ctx context.Context, poolID uint64, tokenA, tokenB string, creator string) error {
	return nil
}

func (d *dummyHooks) AfterLiquidityChanged(ctx context.Context, poolID uint64, provider string, deltaA, deltaB math.Int, isAdd bool) error {
	return nil
}

func (d *dummyHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	return nil
}

// Helper to assert event attributes.
func requireEventAttr(t *testing.T, ev sdk.Event, key, expected string) {
	t.Helper()
	for _, attr := range ev.Attributes {
		if string(attr.Key) == key && string(attr.Value) == expected {
			return
		}
	}
	require.Failf(t, "attribute not found", "wanted key=%s value=%s", key, expected)
}
