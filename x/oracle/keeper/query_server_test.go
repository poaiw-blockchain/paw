package keeper_test

import (
	"bytes"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func setupOracleQueryServer(t *testing.T) (types.QueryServer, *keeper.Keeper, sdk.Context) {
	t.Helper()

	k, ctx := keepertest.OracleKeeper(t)
	return keeper.NewQueryServerImpl(*k), k, ctx
}

func TestOracleQueryServer_Prices(t *testing.T) {
	server, k, ctx := setupOracleQueryServer(t)
	ctx = ctx.WithBlockHeight(100)

	priceOne := types.Price{
		Asset:         "PAW/USD",
		Price:         sdkmath.LegacyNewDec(100),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 3,
	}
	priceTwo := types.Price{
		Asset:         "ATOM/USD",
		Price:         sdkmath.LegacyNewDec(12),
		BlockHeight:   ctx.BlockHeight(),
		BlockTime:     ctx.BlockTime().Unix(),
		NumValidators: 2,
	}

	require.NoError(t, k.SetPrice(sdk.WrapSDKContext(ctx), priceOne))
	require.NoError(t, k.SetPrice(sdk.WrapSDKContext(ctx), priceTwo))

	resp, err := server.Price(sdk.WrapSDKContext(ctx), &types.QueryPriceRequest{Asset: "PAW/USD"})
	require.NoError(t, err)
	require.Equal(t, priceOne, *resp.Price)

	_, err = server.Price(sdk.WrapSDKContext(ctx), &types.QueryPriceRequest{})
	require.Error(t, err)

	_, err = server.Price(sdk.WrapSDKContext(ctx), &types.QueryPriceRequest{Asset: "UNKNOWN"})
	require.Error(t, err)

	pricesResp, err := server.Prices(sdk.WrapSDKContext(ctx), &types.QueryPricesRequest{
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, pricesResp.Prices, 1)
	require.NotNil(t, pricesResp.Pagination)
	require.NotNil(t, pricesResp.Pagination.NextKey)
}

func TestOracleQueryServer_Validators(t *testing.T) {
	server, k, ctx := setupOracleQueryServer(t)

	valOne := makeValAddr(0x01)
	valTwo := makeValAddr(0x02)

	require.NoError(t, k.SetValidatorOracle(sdk.WrapSDKContext(ctx), types.ValidatorOracle{
		ValidatorAddr:    valOne.String(),
		MissCounter:      2,
		TotalSubmissions: 10,
	}))
	require.NoError(t, k.SetValidatorOracle(sdk.WrapSDKContext(ctx), types.ValidatorOracle{
		ValidatorAddr:    valTwo.String(),
		MissCounter:      0,
		TotalSubmissions: 5,
	}))

	resp, err := server.Validator(sdk.WrapSDKContext(ctx), &types.QueryValidatorRequest{
		ValidatorAddr: valOne.String(),
	})
	require.NoError(t, err)
	require.Equal(t, valOne.String(), resp.Validator.ValidatorAddr)
	require.Equal(t, uint64(2), resp.Validator.MissCounter)

	_, err = server.Validator(sdk.WrapSDKContext(ctx), &types.QueryValidatorRequest{ValidatorAddr: "bad"})
	require.Error(t, err)

	validatorsResp, err := server.Validators(sdk.WrapSDKContext(ctx), &types.QueryValidatorsRequest{
		Pagination: &query.PageRequest{Limit: 1},
	})
	require.NoError(t, err)
	require.Len(t, validatorsResp.Validators, 1)
	require.NotNil(t, validatorsResp.Pagination)
	require.NotNil(t, validatorsResp.Pagination.NextKey)
}

func makeValAddr(tag byte) sdk.ValAddress {
	return sdk.ValAddress(bytes.Repeat([]byte{tag}, 20))
}
