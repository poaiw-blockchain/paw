package keeper_test

import (
	"bytes"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
)

func TestBeginBlocker_UpdatePoolTWAPs(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithBlockTime(time.Unix(1_000, 0))

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	runBegin := func() {
		require.NoError(t, k.BeginBlocker(sdk.WrapSDKContext(ctx)))
	}

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	runBegin()

	record, found, err := k.GetPoolTWAP(sdk.WrapSDKContext(ctx), poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, int64(1_000), record.LastTimestamp)

	priceOne := sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(2_000_000)).Quo(sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1_000_000)))
	require.True(t, record.TwapPrice.Equal(priceOne))

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())
	runBegin()

	record, found, err = k.GetPoolTWAP(sdk.WrapSDKContext(ctx), poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(10), record.TotalSeconds)
	require.True(t, record.TwapPrice.Equal(priceOne))

	// Modify pool price to create a second TWAP interval.
	pool, err := k.GetPool(sdk.WrapSDKContext(ctx), poolID)
	require.NoError(t, err)
	pool.ReserveA = sdkmath.NewInt(1_000_000)
	pool.ReserveB = sdkmath.NewInt(4_000_000)
	require.NoError(t, k.SetPool(sdk.WrapSDKContext(ctx), pool))

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())
	runBegin()

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())
	runBegin()

	record, found, err = k.GetPoolTWAP(sdk.WrapSDKContext(ctx), poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(30), record.TotalSeconds)

	priceTwo := sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(4_000_000)).Quo(sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1_000_000)))
	expected := priceOne.MulInt64(20).Add(priceTwo.MulInt64(10)).QuoInt64(30)
	diff := record.TwapPrice.Sub(expected).Abs()
	require.True(t, diff.LTE(sdkmath.LegacyMustNewDecFromStr("0.0000001")), "unexpected TWAP: got %s expected %s", record.TwapPrice, expected)

	events := ctx.EventManager().Events()
	foundBeginEvent := false
	for _, evt := range events {
		if evt.Type == "dex_begin_block" {
			foundBeginEvent = true
			break
		}
	}
	require.True(t, foundBeginEvent, "expected dex_begin_block event")
}

func TestBeginBlocker_DistributeProtocolFees(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(500_000), sdkmath.NewInt(1_000_000))

	amountIn := sdkmath.NewInt(1_000_000)
	_, _, err := k.CollectSwapFees(sdk.WrapSDKContext(ctx), poolID, "upaw", amountIn)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.BeginBlocker(sdk.WrapSDKContext(ctx)))

	protocolFees, err := k.GetProtocolFees(sdk.WrapSDKContext(ctx), "upaw")
	require.NoError(t, err)
	require.True(t, protocolFees.IsZero(), "protocol fees should be cleared after distribution")

	foundEvent := false
	for _, evt := range ctx.EventManager().Events() {
		if evt.Type == "dex_protocol_fees_distributed" {
			foundEvent = true
			break
		}
	}
	require.True(t, foundEvent, "expected dex_protocol_fees_distributed event")
}

func TestEndBlocker_CircuitBreakerRecovery(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(1_000_000))

	state := keeper.CircuitBreakerState{
		Enabled:       true,
		PausedUntil:   ctx.BlockTime().Add(-1 * time.Minute),
		TriggerReason: "test-trigger",
	}
	require.NoError(t, k.SetCircuitBreakerState(sdk.WrapSDKContext(ctx), poolID, state))

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.EndBlocker(sdk.WrapSDKContext(ctx)))

	recoveredState, err := k.GetCircuitBreakerState(sdk.WrapSDKContext(ctx), poolID)
	require.NoError(t, err)
	require.False(t, recoveredState.Enabled)
	require.True(t, recoveredState.PausedUntil.IsZero())
	require.Empty(t, recoveredState.TriggerReason)

	hasRecoveryEvent := false
	hasEndEvent := false
	for _, evt := range ctx.EventManager().Events() {
		switch evt.Type {
		case "circuit_breaker_recovered":
			hasRecoveryEvent = true
		case "dex_end_block":
			hasEndEvent = true
		}
	}
	require.True(t, hasRecoveryEvent, "expected circuit_breaker_recovered event")
	require.True(t, hasEndEvent, "expected dex_end_block event")
}

func TestEndBlocker_CleanupRateLimitData(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithBlockHeight(100_000)
	oldHeight := ctx.BlockHeight() - 86_400 - 5
	newHeight := ctx.BlockHeight() - 100
	window := int64(60)
	oldUser := sdk.AccAddress(bytes.Repeat([]byte{0x01}, 20))
	newUser := sdk.AccAddress(bytes.Repeat([]byte{0x02}, 20))

	keeper.SetRateLimitEntryForTest(k, ctx, oldHeight, oldUser, window)
	keeper.SetRateLimitEntryForTest(k, ctx, newHeight, newUser, window)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.EndBlocker(sdk.WrapSDKContext(ctx)))

	require.False(t, keeper.RateLimitEntryExistsForTest(k, ctx, oldUser, window))
	require.False(t, keeper.RateLimitIndexExistsForTest(k, ctx, oldHeight, oldUser, window))

	require.True(t, keeper.RateLimitEntryExistsForTest(k, ctx, newUser, window))
	require.True(t, keeper.RateLimitIndexExistsForTest(k, ctx, newHeight, newUser, window))

	foundCleanupEvent := false
	for _, evt := range ctx.EventManager().Events() {
		if evt.Type == "rate_limit_data_cleaned" {
			foundCleanupEvent = true
			break
		}
	}
	require.True(t, foundCleanupEvent, "expected rate_limit_data_cleaned event")
}
