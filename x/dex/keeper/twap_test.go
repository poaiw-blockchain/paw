package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestTWAPCumulativePriceUpdate(t *testing.T) {
	keeper, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create a test pool
	pool := &types.Pool{
		Id:       1,
		TokenA:   "atom",
		TokenB:   "osmo",
		ReserveA: math.NewInt(1000000),
		ReserveB: math.NewInt(2000000),
	}
	err := keeper.SetPool(ctx, pool)
	require.NoError(t, err)

	// Initial price: 2.0 (reserveB / reserveA)
	initialPrice := math.LegacyNewDec(2)

	// First update - should initialize TWAP
	err = keeper.UpdateCumulativePriceOnSwap(ctx, pool.Id, initialPrice, math.LegacyNewDec(0).Quo(math.LegacyNewDec(2)))
	require.NoError(t, err)

	// Verify initial state
	record, found, err := keeper.GetPoolTWAP(ctx, pool.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, initialPrice, record.LastPrice)
	require.Equal(t, math.LegacyZeroDec(), record.CumulativePrice)
	require.Equal(t, uint64(0), record.TotalSeconds)

	// Simulate time passing (10 seconds)
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(10 * time.Second))
	ctx = sdkCtx

	// Second update with different price (3.0)
	newPrice := math.LegacyNewDec(3)
	err = keeper.UpdateCumulativePriceOnSwap(ctx, pool.Id, newPrice, math.LegacyNewDec(1).Quo(math.LegacyNewDec(3)))
	require.NoError(t, err)

	// Verify cumulative price accumulated
	record, found, err = keeper.GetPoolTWAP(ctx, pool.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, newPrice, record.LastPrice)

	// CumulativePrice should be initialPrice * 10 seconds = 2 * 10 = 20
	expectedCumulative := initialPrice.MulInt64(10)
	require.Equal(t, expectedCumulative, record.CumulativePrice)
	require.Equal(t, uint64(10), record.TotalSeconds)

	// TWAP = CumulativePrice / TotalSeconds = 20 / 10 = 2.0
	expectedTWAP := expectedCumulative.QuoInt64(10)
	require.Equal(t, expectedTWAP, record.TwapPrice)
}

func TestTWAPGetCurrentTWAP(t *testing.T) {
	keeper, ctx := keepertest.DexKeeper(t)

	// Create a test pool
	pool := &types.Pool{
		Id:       1,
		TokenA:   "atom",
		TokenB:   "osmo",
		ReserveA: math.NewInt(1000000),
		ReserveB: math.NewInt(2000000),
	}
	err := keeper.SetPool(ctx, pool)
	require.NoError(t, err)

	// Should return error when no TWAP data exists
	_, err = keeper.GetCurrentTWAP(ctx, pool.Id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no TWAP data")

	// Initialize TWAP
	price := math.LegacyNewDec(2)
	err = keeper.UpdateCumulativePriceOnSwap(ctx, pool.Id, price, math.LegacyNewDec(1).Quo(math.LegacyNewDec(2)))
	require.NoError(t, err)

	// Should return initial price
	twap, err := keeper.GetCurrentTWAP(ctx, pool.Id)
	require.NoError(t, err)
	require.Equal(t, price, twap)
}

func TestTWAPNonexistentPool(t *testing.T) {
	keeper, ctx := keepertest.DexKeeper(t)

	// Query TWAP for nonexistent pool
	_, err := keeper.GetCurrentTWAP(ctx, 999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no TWAP data")
}

func TestTWAPMultipleUpdates(t *testing.T) {
	keeper, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create a test pool
	pool := &types.Pool{
		Id:       1,
		TokenA:   "atom",
		TokenB:   "osmo",
		ReserveA: math.NewInt(1000000),
		ReserveB: math.NewInt(2000000),
	}
	err := keeper.SetPool(ctx, pool)
	require.NoError(t, err)

	// Simulate multiple swaps over time
	prices := []math.LegacyDec{
		math.LegacyNewDec(2), // t=0
		math.LegacyNewDec(3), // t=10
		math.LegacyNewDec(4), // t=20
		math.LegacyNewDec(3), // t=30
	}

	timestamps := []int64{0, 10, 20, 30}

	for i, price := range prices {
		// Update context time
		if i > 0 {
			sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Duration(timestamps[i]-timestamps[i-1]) * time.Second))
			ctx = sdkCtx
		}

		// Update cumulative price
		err = keeper.UpdateCumulativePriceOnSwap(ctx, pool.Id, price, math.LegacyNewDec(1).Quo(price))
		require.NoError(t, err)
	}

	// Verify final TWAP
	record, found, err := keeper.GetPoolTWAP(ctx, pool.Id)
	require.NoError(t, err)
	require.True(t, found)

	// CumulativePrice = 2*10 + 3*10 + 4*10 = 20 + 30 + 40 = 90
	// TWAP = 90 / 30 = 3.0
	expectedCumulative := math.LegacyNewDec(2).MulInt64(10).
		Add(math.LegacyNewDec(3).MulInt64(10)).
		Add(math.LegacyNewDec(4).MulInt64(10))
	expectedTWAP := expectedCumulative.QuoInt64(30)

	require.Equal(t, expectedCumulative, record.CumulativePrice)
	require.Equal(t, uint64(30), record.TotalSeconds)
	require.Equal(t, expectedTWAP, record.TwapPrice)
}

func TestTWAPPerformance(t *testing.T) {
	// This test verifies that TWAP operations are O(1), not O(n)
	keeper, ctx := keepertest.DexKeeper(t)

	// Create multiple pools
	numPools := 100
	for i := uint64(1); i <= uint64(numPools); i++ {
		pool := &types.Pool{
			Id:       i,
			TokenA:   "atom",
			TokenB:   "osmo",
			ReserveA: math.NewInt(1000000),
			ReserveB: math.NewInt(2000000),
		}
		err := keeper.SetPool(ctx, pool)
		require.NoError(t, err)

		// Initialize TWAP
		price := math.LegacyNewDec(2)
		err = keeper.UpdateCumulativePriceOnSwap(ctx, i, price, math.LegacyNewDec(1).Quo(math.LegacyNewDec(2)))
		require.NoError(t, err)
	}

	// Update TWAP for a single pool should be O(1)
	// Not O(n) where n is the number of pools
	start := time.Now()
	price := math.LegacyNewDec(3)
	err := keeper.UpdateCumulativePriceOnSwap(ctx, 50, price, math.LegacyNewDec(1).Quo(math.LegacyNewDec(3)))
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Should complete in microseconds, not scale with number of pools
	// This is a loose upper bound - actual execution should be much faster
	require.Less(t, elapsed, 10*time.Millisecond, "TWAP update should be O(1), not O(n)")

	// Verify the update
	record, found, err := keeper.GetPoolTWAP(ctx, 50)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, price, record.LastPrice)
}

func TestBeginBlockerNoLongerIteratesAllPools(t *testing.T) {
	// This test verifies that BeginBlocker no longer iterates all pools
	keeper, ctx := keepertest.DexKeeper(t)

	// Create many pools
	numPools := 50
	for i := uint64(1); i <= uint64(numPools); i++ {
		pool := &types.Pool{
			Id:       i,
			TokenA:   "atom",
			TokenB:   "osmo",
			ReserveA: math.NewInt(1000000),
			ReserveB: math.NewInt(2000000),
		}
		err := keeper.SetPool(ctx, pool)
		require.NoError(t, err)
	}

	// Run BeginBlocker (which should now be a no-op for TWAP updates)
	start := time.Now()
	err := keeper.BeginBlocker(ctx)
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Should be extremely fast since UpdatePoolTWAPs is now a no-op
	require.Less(t, elapsed, 5*time.Millisecond, "BeginBlocker should not iterate pools")
}
