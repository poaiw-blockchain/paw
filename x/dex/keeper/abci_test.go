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
	// UPDATED TEST: BeginBlocker no longer iterates pools to update TWAPs
	// TWAP updates now happen lazily on swaps via UpdateCumulativePriceOnSwap
	// This test now verifies that BeginBlocker succeeds without updating TWAPs

	k, ctx := keepertest.DexKeeper(t)
	ctx = ctx.WithBlockTime(time.Unix(1_000, 0))

	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt", sdkmath.NewInt(1_000_000), sdkmath.NewInt(2_000_000))

	runBegin := func() {
		require.NoError(t, k.BeginBlocker(ctx))
	}

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	runBegin()

	// After BeginBlocker, TWAP should NOT be automatically updated anymore
	// It should only update on swaps
	_, found, err := k.GetPoolTWAP(ctx, poolID)
	require.NoError(t, err)
	// TWAP record should not exist yet since no swaps have occurred
	require.False(t, found, "TWAP should not be created by BeginBlocker")

	// Manually update TWAP to simulate a swap
	priceOne := sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(2_000_000)).Quo(sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1_000_000)))
	err = k.UpdateCumulativePriceOnSwap(ctx, poolID, priceOne, sdkmath.LegacyOneDec().Quo(priceOne))
	require.NoError(t, err)

	record, found, err := k.GetPoolTWAP(ctx, poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, int64(1_000), record.LastTimestamp)
	require.True(t, record.TwapPrice.Equal(priceOne))

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())
	runBegin()

	// BeginBlocker should not have updated the TWAP
	record, found, err = k.GetPoolTWAP(ctx, poolID)
	require.NoError(t, err)
	require.True(t, found)
	// Timestamp should still be old since BeginBlocker doesn't update TWAP
	require.Equal(t, int64(1_000), record.LastTimestamp, "BeginBlocker should not update TWAP")

	// Simulate another swap to update TWAP
	err = k.UpdateCumulativePriceOnSwap(ctx, poolID, priceOne, sdkmath.LegacyOneDec().Quo(priceOne))
	require.NoError(t, err)

	record, found, err = k.GetPoolTWAP(ctx, poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(10), record.TotalSeconds)
	require.True(t, record.TwapPrice.Equal(priceOne))

	// Advance time and simulate another swap with changed price
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())

	// Modify pool price to create a second TWAP interval.
	pool, err := k.GetPool(ctx, poolID)
	require.NoError(t, err)
	pool.ReserveA = sdkmath.NewInt(1_000_000)
	pool.ReserveB = sdkmath.NewInt(4_000_000)
	require.NoError(t, k.SetPool(ctx, pool))

	// Simulate a swap with new price (this captures the 10s since last swap)
	priceTwo := sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(4_000_000)).Quo(sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1_000_000)))
	err = k.UpdateCumulativePriceOnSwap(ctx, poolID, priceTwo, sdkmath.LegacyOneDec().Quo(priceTwo))
	require.NoError(t, err)

	// Advance another 10 seconds
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(10 * time.Second)).WithEventManager(sdk.NewEventManager())
	runBegin()

	// Simulate another swap
	err = k.UpdateCumulativePriceOnSwap(ctx, poolID, priceTwo, sdkmath.LegacyOneDec().Quo(priceTwo))
	require.NoError(t, err)

	record, found, err = k.GetPoolTWAP(ctx, poolID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(30), record.TotalSeconds)

	// TWAP calculation: (priceOne * 10s) + (priceOne * 10s) + (priceTwo * 10s) / 30s
	// First interval: priceOne for 10s (from first to second swap)
	// Second interval: priceOne for 10s (from second to third swap, price hasn't changed yet in TWAP)
	// Third interval: priceTwo for 10s (from third to fourth swap)
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
	_, _, err := k.CollectSwapFees(ctx, poolID, "upaw", amountIn)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.BeginBlocker(ctx))

	protocolFees, err := k.GetProtocolFees(ctx, "upaw")
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
	require.NoError(t, k.SetCircuitBreakerState(ctx, poolID, state))

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.EndBlocker(ctx))

	recoveredState, err := k.GetPoolCircuitBreakerState(ctx, poolID)
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
	require.NoError(t, k.EndBlocker(ctx))

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
