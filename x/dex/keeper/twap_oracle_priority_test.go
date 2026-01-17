package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// When oracle is available, it should be preferred even if TWAP exists.
func TestGetPriceWithTWAPFallback_OraclePreferredOverTWAP(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	pool, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(1_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	// Seed TWAP price
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
		PoolId:          pool.Id,
		LastPrice:       math.LegacyNewDec(2),
		CumulativePrice: math.LegacyZeroDec(),
		TotalSeconds:    10,
		LastTimestamp:   1,
		TwapPrice:       math.LegacyNewDec(2),
	}))

	oracle := mockOracleSuccess{
		prices: map[string]math.LegacyDec{
			"upaw":  math.LegacyNewDec(5),
			"uusdc": math.LegacyNewDec(1),
		},
	}

	price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, oracle)
	require.NoError(t, err)
	require.Equal(t, "oracle", source)
	require.True(t, price.Equal(math.LegacyNewDec(5))) // upaw/ uUSDC ratio
}

type mockOracleSuccess struct {
	prices map[string]math.LegacyDec
}

func (m mockOracleSuccess) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	if p, ok := m.prices[denom]; ok {
		return p, nil
	}
	return math.LegacyZeroDec(), types.ErrOraclePrice
}

func (m mockOracleSuccess) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	p, err := m.GetPrice(ctx, denom)
	return p, 1_700_000_000, err
}
