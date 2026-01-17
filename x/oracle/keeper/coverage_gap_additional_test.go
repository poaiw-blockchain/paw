package keeper_test

import (
	"math/big"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Price override lifecycle: set → get → clear, including expiry handling and GetPriceWithOverride fallback.
func TestPriceOverrideLifecycle(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	pair := "ATOM/USD"
	price := big.NewInt(12345)

	require.NoError(t, k.SetPriceOverride(ctx, pair, price, 3600, "actor", "reason"))

	got, ok := k.GetPriceOverride(ctx, pair)
	require.True(t, ok)
	expectedScaled := sdkmath.LegacyNewDecFromBigInt(price).BigInt()
	require.Equal(t, 0, expectedScaled.Cmp(got))

	// Expire by advancing block time
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Hour))
	_, ok = k.GetPriceOverride(ctx, pair)
	require.False(t, ok, "expired override should be removed")

	// Clear should be a no-op after expiry but still not panic
	k.ClearPriceOverride(ctx, pair)
	_, ok = k.GetPriceOverride(ctx, pair)
	require.False(t, ok)
}

// GetPriceWithOverride should return override when present, otherwise fall back to stored price.
func TestGetPriceWithOverrideFallback(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	pair := "ATOM/USD"

	// Seed normal price
	require.NoError(t, k.SetPrice(ctx, types.Price{
		Asset:       pair,
		Price:       sdkmath.LegacyNewDec(100),
		BlockHeight: ctx.BlockHeight(),
	}))

	// Without override -> returns stored price
	price, ok := k.GetPriceWithOverride(ctx, pair)
	require.True(t, ok)
	require.Equal(t, 0, sdkmath.LegacyNewDec(100).BigInt().Cmp(price))

	// With override -> returns override instead
	override := big.NewInt(200)
	require.NoError(t, k.SetPriceOverride(ctx, pair, override, 600, "actor", "reason"))

	price, ok = k.GetPriceWithOverride(ctx, pair)
	require.True(t, ok)
	expectedOverride := sdkmath.LegacyNewDecFromBigInt(override).BigInt()
	require.Equal(t, 0, expectedOverride.Cmp(price))
}

// Slashing disable/enable toggles state.
func TestSlashingDisableEnable(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	require.False(t, k.IsSlashingDisabled(ctx))
	require.NoError(t, k.DisableSlashing(ctx, "actor", "maintenance"))
	require.True(t, k.IsSlashingDisabled(ctx))
	require.NoError(t, k.EnableSlashing(ctx, "actor", "back-online"))
	require.False(t, k.IsSlashingDisabled(ctx))
}

// Whitelisted oracle source governance actions.
func TestWhitelistedOracleSources(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	chain := "chain-whitelist-1"
	require.False(t, k.IsWhitelistedOracleSource(chain))

	k.AddWhitelistedOracleSource(ctx, chain)
	require.True(t, k.IsWhitelistedOracleSource(chain))

	k.RemoveWhitelistedOracleSource(ctx, chain)
	require.False(t, k.IsWhitelistedOracleSource(chain))
}

// Feed-level circuit breaker toggling.
func TestFeedCircuitBreaker(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	feed := "prices"
	require.False(t, k.IsFeedCircuitBreakerOpen(ctx, feed))

	require.NoError(t, k.OpenFeedCircuitBreaker(ctx, feed, "actor", "maintenance"))
	require.True(t, k.IsFeedCircuitBreakerOpen(ctx, feed))

	require.NoError(t, k.CloseFeedCircuitBreaker(ctx, feed, "actor", "restored"))
	require.False(t, k.IsFeedCircuitBreakerOpen(ctx, feed))
}
