package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// mockOracleWithTS allows controlling price responses and timestamps.
type mockOracleWithTS struct {
	priceA      math.LegacyDec
	priceB      math.LegacyDec
	errA        error
	errB        error
	timestampA  int64
	timestampB  int64
}

func (m mockOracleWithTS) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	if denom == "tokenA" {
		return m.priceA, m.errA
	}
	return m.priceB, m.errB
}

func (m mockOracleWithTS) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	price, err := m.GetPrice(ctx, denom)
	if denom == "tokenA" {
		return price, m.timestampA, err
	}
	return price, m.timestampB, err
}

// Covers oracle primary path, deviation failure, TWAP fallback, and zero-reserve guard.
func TestValidatePoolPriceWithFallback(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)

	// Fix block time to control staleness logic
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	pool, err := k.CreatePool(ctx, types.TestAddr(), "tokenA", "tokenB", math.NewInt(2_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	maxDeviation := math.LegacyNewDecWithPrec(5, 2) // 5%

	oracleFresh := mockOracleWithTS{
		priceA:     math.LegacyNewDec(2),
		priceB:     math.LegacyNewDec(1),
		timestampA: ctx.BlockTime().Unix(),
		timestampB: ctx.BlockTime().Unix(),
	}

    t.Run("oracle within deviation", func(t *testing.T) {
        // Align pool reserves exactly with oracle ratio (2:1) to remain within deviation tolerance
        p, err := k.GetPool(ctx, pool.Id)
        require.NoError(t, err)
        // For oracle price 2:1 (tokenA/tokenB), pool ratio ReserveB/ReserveA must also equal 2
        p.ReserveA = math.NewInt(1_000_000)
        p.ReserveB = math.NewInt(2_000_000)
        require.NoError(t, k.SetPool(ctx, p))

        require.NoError(t, k.ValidatePoolPriceWithFallback(ctx, pool.Id, oracleFresh, maxDeviation))
    })

	t.Run("oracle deviation too high", func(t *testing.T) {
		// Make pool 1:1 so deviation vs oracle 2:1 exceeds threshold
		p, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		p.ReserveA = math.NewInt(1_000_000)
		p.ReserveB = math.NewInt(1_000_000)
		require.NoError(t, k.SetPool(ctx, p))

		err = k.ValidatePoolPriceWithFallback(ctx, pool.Id, oracleFresh, maxDeviation)
		require.ErrorIs(t, err, types.ErrPriceDeviation)
	})

	t.Run("twap fallback when oracle unavailable", func(t *testing.T) {
		// Seed TWAP close to pool ratio 1:1 to pass
		require.NoError(t, k.SetPoolTWAP(ctx, types.PoolTWAP{
			PoolId:        pool.Id,
			TwapPrice:     math.LegacyNewDecWithPrec(1, 0),
			LastTimestamp: ctx.BlockTime().Unix(),
		}))

		err = k.ValidatePoolPriceWithFallback(ctx, pool.Id, mockOracleWithTS{
			errA: types.ErrOraclePrice,
			errB: types.ErrOraclePrice,
		}, maxDeviation)
		require.NoError(t, err)
	})

	t.Run("zero reserve guard", func(t *testing.T) {
		p, err := k.GetPool(ctx, pool.Id)
		require.NoError(t, err)
		p.ReserveA = math.ZeroInt()
		require.NoError(t, k.SetPool(ctx, p))

		err = k.ValidatePoolPriceWithFallback(ctx, pool.Id, oracleFresh, maxDeviation)
		require.ErrorIs(t, err, types.ErrInsufficientLiquidity)
	})
}
