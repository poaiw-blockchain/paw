package keeper_test

import (
	"bytes"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestDexGenesisRoundTrip(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	params.MaxSlippagePercent = sdkmath.LegacyNewDecWithPrec(25, 2)
	params.AuthorizedChannels = []types.AuthorizedChannel{
		{PortId: types.PortID, ChannelId: "channel-0"},
	}

	creatorOne := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	creatorTwo := sdk.AccAddress(bytes.Repeat([]byte{0x2}, 20)).String()

	pools := []types.Pool{
		{
			Id:          1,
			TokenA:      "atom",
			TokenB:      "paw",
			ReserveA:    sdkmath.NewInt(1_000_000),
			ReserveB:    sdkmath.NewInt(2_000_000),
			TotalShares: sdkmath.NewInt(900_000),
			Creator:     creatorOne,
		},
		{
			Id:          2,
			TokenA:      "paw",
			TokenB:      "usdc",
			ReserveA:    sdkmath.NewInt(3_000_000),
			ReserveB:    sdkmath.NewInt(1_500_000),
			TotalShares: sdkmath.NewInt(1_200_000),
			Creator:     creatorTwo,
		},
	}

	twaps := []types.PoolTWAP{
		{
			PoolId:          1,
			LastPrice:       sdkmath.LegacyMustNewDecFromStr("2.0"),
			CumulativePrice: sdkmath.LegacyMustNewDecFromStr("20.0"),
			TotalSeconds:    120,
			LastTimestamp:   1_700_000_000,
			TwapPrice:       sdkmath.LegacyZeroDec(),
		},
		{
			PoolId:          2,
			LastPrice:       sdkmath.LegacyMustNewDecFromStr("0.5"),
			CumulativePrice: sdkmath.LegacyMustNewDecFromStr("5.0"),
			TotalSeconds:    60,
			LastTimestamp:   1_700_000_100,
			TwapPrice:       sdkmath.LegacyZeroDec(),
		},
	}

	genesis := types.GenesisState{
		Params:          params,
		Pools:           pools,
		NextPoolId:      3,
		PoolTwapRecords: twaps,
	}

	require.NoError(t, k.InitGenesis(sdk.WrapSDKContext(ctx), genesis))

	exported, err := k.ExportGenesis(sdk.WrapSDKContext(ctx))
	require.NoError(t, err)

	require.Equal(t, genesis.Params, exported.Params)
	require.Equal(t, genesis.Pools, exported.Pools)
	require.Equal(t, genesis.PoolTwapRecords, exported.PoolTwapRecords)
	require.Equal(t, genesis.NextPoolId, exported.NextPoolId)
}
