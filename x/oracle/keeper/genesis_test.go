package keeper_test

import (
	"bytes"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestOracleGenesisRoundTrip(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	params := types.DefaultParams()
	params.VotePeriod = 15
	params.MinValidPerWindow = 5
	params.AuthorizedChannels = []types.AuthorizedChannel{
		{PortId: types.PortID, ChannelId: "channel-42"},
	}

	valOne := sdk.ValAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	valTwo := sdk.ValAddress(bytes.Repeat([]byte{0x2}, 20)).String()

	now := ctx.BlockTime()

	genesis := types.GenesisState{
		Params: params,
		Prices: []types.Price{
			{
				Asset:         "PAW/USD",
				Price:         sdkmath.LegacyMustNewDecFromStr("1.25"),
				BlockHeight:   10,
				BlockTime:     now.Unix(),
				NumValidators: 2,
			},
			{
				Asset:         "ATOM/USD",
				Price:         sdkmath.LegacyMustNewDecFromStr("12.50"),
				BlockHeight:   10,
				BlockTime:     now.Unix(),
				NumValidators: 1,
			},
		},
		ValidatorPrices: []types.ValidatorPrice{
			{
				ValidatorAddr: valOne,
				Asset:         "PAW/USD",
				Price:         sdkmath.LegacyMustNewDecFromStr("1.30"),
				BlockHeight:   10,
				VotingPower:   10,
			},
			{
				ValidatorAddr: valTwo,
				Asset:         "ATOM/USD",
				Price:         sdkmath.LegacyMustNewDecFromStr("12.45"),
				BlockHeight:   10,
				VotingPower:   8,
			},
		},
		ValidatorOracles: []types.ValidatorOracle{
			{
				ValidatorAddr:    valOne,
				MissCounter:      1,
				TotalSubmissions: 5,
				IsActive:         true,
				GeographicRegion: "global",
			},
			{
				ValidatorAddr:    valTwo,
				MissCounter:      0,
				TotalSubmissions: 3,
				IsActive:         true,
				GeographicRegion: "global",
			},
		},
		PriceSnapshots: []types.PriceSnapshot{
			{
				Asset:       "PAW/USD",
				Price:       sdkmath.LegacyMustNewDecFromStr("1.20"),
				BlockHeight: 8,
				BlockTime:   now.Add(-5 * time.Minute).Unix(),
			},
		},
	}

	require.NoError(t, k.InitGenesis(ctx, genesis))

	exported, err := k.ExportGenesis(ctx)
	require.NoError(t, err)
	require.Equal(t, genesis.Params, exported.Params)
	require.ElementsMatch(t, genesis.Prices, exported.Prices)
	require.ElementsMatch(t, genesis.ValidatorPrices, exported.ValidatorPrices)
	require.ElementsMatch(t, genesis.ValidatorOracles, exported.ValidatorOracles)
	require.ElementsMatch(t, genesis.PriceSnapshots, exported.PriceSnapshots)
}
