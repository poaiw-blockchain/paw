package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

type mockOracleTS struct {
	price map[string]math.LegacyDec
	err   error
	ts    int64
}

func (m mockOracleTS) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	if m.err != nil {
		return math.LegacyZeroDec(), m.err
	}
	if p, ok := m.price[denom]; ok {
		return p, nil
	}
	return math.LegacyZeroDec(), types.ErrOraclePrice
}

func (m mockOracleTS) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	if m.err != nil {
		return math.LegacyZeroDec(), 0, m.err
	}
	if p, ok := m.price[denom]; ok {
		return p, m.ts, nil
	}
	return math.LegacyZeroDec(), m.ts, nil
}

func TestUpdateCumulativePriceOnSwap_Edges(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// First call should initialize record
	ctx = ctx.WithBlockTime(time.Unix(1_000, 0))
	require.NoError(t, k.UpdateCumulativePriceOnSwap(ctx, 1, math.LegacyNewDec(2), math.LegacyNewDec(0)))

	record, found, err := k.GetPoolTWAP(ctx, 1)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, int64(1_000), record.LastTimestamp)
	require.True(t, record.TwapPrice.Equal(math.LegacyNewDec(2)))

	// Zero elapsed time should not change cumulative/seconds
	require.NoError(t, k.UpdateCumulativePriceOnSwap(ctx, 1, math.LegacyNewDec(3), math.LegacyNewDec(0)))
	record, found, err = k.GetPoolTWAP(ctx, 1)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(0), record.TotalSeconds)

	// Advance time and ensure accumulation happens
	ctx = ctx.WithBlockTime(time.Unix(1_010, 0))
	require.NoError(t, k.UpdateCumulativePriceOnSwap(ctx, 1, math.LegacyNewDec(4), math.LegacyNewDec(0)))
	record, found, err = k.GetPoolTWAP(ctx, 1)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(10), record.TotalSeconds)
	require.True(t, record.CumulativePrice.GT(math.LegacyZeroDec()))
}

func TestPoolValueAndValidatePrice(t *testing.T) {
	k, _, ctx := keepertest.DexKeeperWithBank(t)
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	pool, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "uusdc", math.NewInt(2_000_000), math.NewInt(1_000_000))
	require.NoError(t, err)

	oracle := mockOraclePrice{
		prices: map[string]math.LegacyDec{
			"upaw":  math.LegacyNewDec(2), // $2
			"uusdc": math.LegacyNewDec(1), // $1
		},
		ts: ctx.BlockTime().Unix(),
	}

	val, err := k.GetPoolValueUSD(ctx, pool.Id, oracle)
	require.NoError(t, err)
	// value = 2,000,000*2 + 1,000,000*1 = 5,000,000
	require.True(t, val.Equal(math.LegacyNewDec(5_000_000)))

	// ValidatePoolPrice with small deviation should pass
	require.NoError(t, k.ValidatePoolPrice(ctx, pool.Id, oracle, math.LegacyNewDecWithPrec(10, 2))) // 10%

	// Stale oracle should be rejected
	stale := mockOraclePrice{
		prices: oracle.prices,
		ts:     ctx.BlockTime().Add(-2 * time.Minute).Unix(),
	}
	err = k.ValidatePoolPrice(ctx, pool.Id, stale, math.LegacyNewDecWithPrec(10, 2))
	require.ErrorIs(t, err, types.ErrOraclePrice)

	// Deviating pool price should be rejected
	p, _ := k.GetPool(ctx, pool.Id)
	p.ReserveA = math.NewInt(10_000_000) // change ratio
	require.NoError(t, k.SetPool(ctx, p))
	err = k.ValidatePoolPrice(ctx, pool.Id, oracle, math.LegacyNewDecWithPrec(1, 2)) // 1%
	require.ErrorIs(t, err, types.ErrPriceDeviation)
}

type mockOraclePrice struct {
	prices map[string]math.LegacyDec
	ts     int64
}

func (m mockOraclePrice) GetPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	if p, ok := m.prices[denom]; ok {
		return p, nil
	}
	return math.LegacyZeroDec(), types.ErrOraclePrice
}

func (m mockOraclePrice) GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error) {
	p, err := m.GetPrice(ctx, denom)
	return p, m.ts, err
}

func TestDistributeProtocolFees(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.GetStoreKey())

	// Seed two protocol fee entries
	bz, _ := math.NewInt(123).Marshal()
	store.Set(types.GetProtocolFeeKey("denomX"), bz)
	bz, _ = math.NewInt(456).Marshal()
	store.Set(types.GetProtocolFeeKey("denomY"), bz)

	require.NoError(t, k.DistributeProtocolFees(ctx))

	require.Nil(t, store.Get(types.GetProtocolFeeKey("denomX")))
	require.Nil(t, store.Get(types.GetProtocolFeeKey("denomY")))
}

func TestMarkPoolActiveSetsKey(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	require.NoError(t, k.MarkPoolActive(ctx, 99))

	store := sdk.UnwrapSDKContext(ctx).KVStore(k.GetStoreKey())
	require.NotNil(t, store.Get(keeper.ActivePoolKey(99)))
}
