package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// mockOracle implements the minimal OracleKeeper interface for testing fallbacks.
type mockOracle struct {
	priceA math.LegacyDec
	priceB math.LegacyDec
	errA   error
	errB   error
}

func (m mockOracle) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	if denom == "tokenA" {
		return m.priceA, m.errA
	}
	return m.priceB, m.errB
}

func (m mockOracle) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	price, err := m.GetPrice(ctx, denom)
	return price, 0, err
}

func TestGetPriceWithTWAPFallback_OrderOfSources(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	pool, err := k.CreatePool(ctx, types.TestAddr(), "tokenA", "tokenB", math.NewInt(2_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	t.Run("oracle primary", func(t *testing.T) {
		price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracle{
			priceA: math.LegacyNewDec(2), // tokenA = $2
			priceB: math.LegacyNewDec(1), // tokenB = $1
		})
		require.NoError(t, err)
		require.Equal(t, "oracle", source)
		require.True(t, price.Equal(math.LegacyNewDec(2))) // ratio 2/1
	})

	t.Run("twap fallback when oracle unavailable", func(t *testing.T) {
		// Seed TWAP record
		require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
			PoolId:          pool.Id,
			LastPrice:       math.LegacyNewDecWithPrec(15, 1), // 1.5
			CumulativePrice: math.LegacyZeroDec(),
			TotalSeconds:    10,
			LastTimestamp:   1,
			TwapPrice:       math.LegacyNewDecWithPrec(15, 1),
		}))

		price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracle{
			errA: types.ErrOraclePrice,
			errB: types.ErrOraclePrice,
		})
		require.NoError(t, err)
		require.Equal(t, "twap", source)
		require.True(t, price.Equal(math.LegacyNewDecWithPrec(15, 1)))
	})

	t.Run("spot fallback when oracle and twap missing", func(t *testing.T) {
		// Clear TWAP for this run
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.KVStore(k.GetStoreKey()).Delete(keeper.PoolTWAPKey(pool.Id))

		price, source, err := k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracle{
			errA: types.ErrOraclePrice,
			errB: types.ErrOraclePrice,
		})
		require.NoError(t, err)
		require.Equal(t, "spot", source)
		// pool reserves are 2:1 so price should be ~0.5 (tokenB/tokenA)
		require.True(t, price.Equal(math.LegacyNewDecWithPrec(5, 1)))
	})

	t.Run("error when no sources and zero reserves", func(t *testing.T) {
		// Zero-out reserves to force final error path
		p, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		p.ReserveA = math.ZeroInt()
		p.ReserveB = math.ZeroInt()
		require.NoError(t, k.SetPool(ctx, p))

		_, _, err = k.GetPriceWithTWAPFallback(ctx, pool.Id, mockOracle{
			errA: types.ErrOraclePrice,
			errB: types.ErrOraclePrice,
		})
		require.ErrorIs(t, err, types.ErrOraclePrice)
	})
}

func TestGetAllPoolTWAPs(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{PoolId: 1, TwapPrice: math.LegacyNewDec(1)}))
	require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{PoolId: 2, TwapPrice: math.LegacyNewDec(2)}))

	records, err := k.GetAllPoolTWAPs(ctx)
	require.NoError(t, err)
	require.Len(t, records, 2)
}

func TestPoolCountAccounting(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	require.Equal(t, uint64(0), k.GetTotalPoolsCount(ctx))

	k.IncrementTotalPoolsCount(ctx)
	k.IncrementTotalPoolsCount(ctx)
	require.Equal(t, uint64(2), k.GetTotalPoolsCount(ctx))

	k.DecrementTotalPoolsCount(ctx)
	require.Equal(t, uint64(1), k.GetTotalPoolsCount(ctx))

	k.SetTotalPoolsCount(ctx, 42)
	require.Equal(t, uint64(42), k.GetTotalPoolsCount(ctx))

	currentVersion := k.GetPoolVersion(ctx)
	k.IncrementPoolVersion(ctx)
	require.Equal(t, currentVersion+1, k.GetPoolVersion(ctx))
}
