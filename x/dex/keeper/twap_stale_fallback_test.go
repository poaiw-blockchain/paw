package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TWAP exists but is stale; current logic still returns TWAP if oracle unavailable.
func TestGetPriceWithTWAPFallback_StaleTWAPUsedWhenOracleDown(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	pool, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	// Seed TWAP with old timestamp but non-zero price
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:          pool.Id,
		LastPrice:       math.LegacyNewDec(2),
		CumulativePrice: math.LegacyZeroDec(),
		TotalSeconds:    10,
		LastTimestamp:   time.Unix(1_000, 0).Unix(), // stale
		TwapPrice:       math.LegacyNewDec(2),
	}))

	price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracleFail{})
	require.NoError(t, err)
	require.Equal(t, "twap", source)
	require.True(t, price.Equal(math.LegacyNewDec(2)))
}

// Reuse the mock from twap_fallback_spot_test
