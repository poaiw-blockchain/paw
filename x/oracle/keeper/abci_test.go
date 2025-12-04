package keeper_test

import (
	"bytes"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestBeginBlocker_AggregatePrices(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(30).
		WithBlockTime(time.Unix(1_700_000_000, 0)).
		WithEventManager(sdk.NewEventManager())

	asset := "PAW/USD"
	priceSchedule := []struct {
		addr  sdk.ValAddress
		price sdkmath.LegacyDec
	}{
		{addr: makeValidatorAddress(0x01), price: sdkmath.LegacyNewDec(90)},
		{addr: makeValidatorAddress(0x02), price: sdkmath.LegacyNewDec(100)},
		{addr: makeValidatorAddress(0x03), price: sdkmath.LegacyNewDec(110)},
		{addr: makeValidatorAddress(0x04), price: sdkmath.LegacyNewDec(125)},
	}

	for _, entry := range priceSchedule {
		keepertest.RegisterTestOracle(t, k, ctx, entry.addr.String())
		submitValidatorPrice(t, k, ctx, entry.addr, asset, entry.price)
	}

	require.NoError(t, k.BeginBlocker(sdk.WrapSDKContext(ctx)))

	aggregated, err := k.GetPrice(ctx, asset)
	require.NoError(t, err)
	expected := sdkmath.LegacyNewDec(100)
	require.True(t, aggregated.Price.Equal(expected), "expected aggregated price %s got %s", expected, aggregated.Price)
	require.Equal(t, ctx.BlockHeight(), aggregated.BlockHeight)
	require.Equal(t, ctx.BlockTime().Unix(), aggregated.BlockTime)
	require.Equal(t, uint32(len(priceSchedule)), aggregated.NumValidators)

	events := ctx.EventManager().Events()
	require.True(t, eventExists(events, "oracle_begin_block", "", ""), "expected oracle_begin_block event")
	require.True(t, eventExists(events, "price_aggregated", "asset", asset), "expected price_aggregated event for asset")
}

func TestEndBlocker_ProcessSlashWindows(t *testing.T) {
	kImpl, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(1).
		WithBlockTime(time.Unix(1_700_000_100, 0)).
		WithEventManager(sdk.NewEventManager())

	params := types.DefaultParams()
	params.VotePeriod = 1
	params.MinValidPerWindow = 1
	require.NoError(t, kImpl.SetParams(ctx, params))

	asset := "PAW/USD"
	activeVal := makeValidatorAddress(0x10)
	missingVal := makeValidatorAddress(0x20)
	keepertest.RegisterTestOracle(t, kImpl, ctx, activeVal.String())
	keepertest.RegisterTestOracle(t, kImpl, ctx, missingVal.String())

	price := types.Price{
		Asset:         asset,
		Price:         sdkmath.LegacyNewDec(100),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 1,
	}
	require.NoError(t, kImpl.SetPrice(ctx, price))
	submitValidatorPrice(t, kImpl, ctx, activeVal, asset, sdkmath.LegacyNewDec(100))

	require.NoError(t, kImpl.EndBlocker(sdk.WrapSDKContext(ctx)))

	events := ctx.EventManager().Events()
	require.True(t, eventExists(events, "oracle_slash", "validator", missingVal.String()), "expected slash for missing validator")
	require.True(t, eventExists(events, "oracle_end_block", "", ""), "expected oracle_end_block event")

	missingOracle, err := kImpl.GetValidatorOracle(ctx, missingVal.String())
	require.NoError(t, err)
	require.GreaterOrEqual(t, missingOracle.MissCounter, uint64(1))
}

func makeValidatorAddress(tag byte) sdk.ValAddress {
	return sdk.ValAddress(bytes.Repeat([]byte{tag}, 20))
}

func submitValidatorPrice(t *testing.T, k *keeper.Keeper, ctx sdk.Context, val sdk.ValAddress, asset string, price sdkmath.LegacyDec) {
	t.Helper()
	vp := types.ValidatorPrice{
		ValidatorAddr: val.String(),
		Asset:         asset,
		Price:         price,
		BlockHeight:   ctx.BlockHeight(),
		VotingPower:   1,
	}
	require.NoError(t, k.SetValidatorPrice(sdk.WrapSDKContext(ctx), vp))
}

func eventExists(events sdk.Events, eventType string, attrKey string, attrValue string) bool {
	for _, evt := range events {
		if evt.Type != eventType {
			continue
		}
		if attrKey == "" {
			return true
		}
		for _, attr := range evt.Attributes {
			if attr.Key == attrKey && string(attr.Value) == attrValue {
				return true
			}
		}
	}
	return false
}
