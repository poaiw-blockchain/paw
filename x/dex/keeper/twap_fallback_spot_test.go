package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TWAP exists but zero/invalid -> should fallback to spot when reserves are positive.
func TestGetPriceWithTWAPFallback_ZeroTWAPFallsToSpot(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	pool, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(2_000_000))
	require.NoError(t, err)

	// Seed TWAP with zero price to force invalid TWAP path
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:          pool.Id,
		LastPrice:       math.LegacyZeroDec(),
		CumulativePrice: math.LegacyZeroDec(),
		TotalSeconds:    10,
		LastTimestamp:   1,
		TwapPrice:       math.LegacyZeroDec(),
	}))

	price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracleFail{})
	require.NoError(t, err)
	require.Equal(t, "spot", source)
	// Spot price = reserveB / reserveA = 2
	require.True(t, price.Equal(math.LegacyNewDec(2)))
}

type mockOracleFail struct{}

func (mockOracleFail) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	return math.LegacyZeroDec(), types.ErrOraclePrice
}

func (mockOracleFail) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	return math.LegacyZeroDec(), 0, types.ErrOraclePrice
}
