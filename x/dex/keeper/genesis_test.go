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
	providerOne := sdk.AccAddress(bytes.Repeat([]byte{0x3}, 20)).String()
	providerTwo := sdk.AccAddress(bytes.Repeat([]byte{0x4}, 20)).String()

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

	circuitBreakerStates := []types.CircuitBreakerStateExport{
		{
			PoolId:            1,
			Enabled:           false,
			PausedUntil:       0,
			LastPrice:         sdkmath.LegacyMustNewDecFromStr("2.0"),
			TriggeredBy:       "",
			TriggerReason:     "",
			NotificationsSent: 0,
			LastNotification:  0,
			PersistenceKey:    "",
		},
		{
			PoolId:            2,
			Enabled:           true,
			PausedUntil:       1_700_000_200,
			LastPrice:         sdkmath.LegacyMustNewDecFromStr("0.5"),
			TriggeredBy:       "system",
			TriggerReason:     "price deviation",
			NotificationsSent: 2,
			LastNotification:  1_700_000_150,
			PersistenceKey:    "pool_2_cb",
		},
	}

	liquidityPositions := []types.LiquidityPositionExport{
		{
			PoolId:   1,
			Provider: providerOne,
			Shares:   sdkmath.NewInt(500_000),
		},
		{
			PoolId:   1,
			Provider: providerTwo,
			Shares:   sdkmath.NewInt(400_000),
		},
		{
			PoolId:   2,
			Provider: providerOne,
			Shares:   sdkmath.NewInt(700_000),
		},
		{
			PoolId:   2,
			Provider: providerTwo,
			Shares:   sdkmath.NewInt(500_000),
		},
	}

	genesis := types.GenesisState{
		Params:               params,
		Pools:                pools,
		NextPoolId:           3,
		PoolTwapRecords:      twaps,
		CircuitBreakerStates: circuitBreakerStates,
		LiquidityPositions:   liquidityPositions,
	}

	require.NoError(t, k.InitGenesis(sdk.WrapSDKContext(ctx), genesis))

	exported, err := k.ExportGenesis(sdk.WrapSDKContext(ctx))
	require.NoError(t, err)

	require.Equal(t, genesis.Params, exported.Params)
	require.Equal(t, genesis.Pools, exported.Pools)
	require.Equal(t, genesis.PoolTwapRecords, exported.PoolTwapRecords)
	require.Equal(t, genesis.NextPoolId, exported.NextPoolId)
	require.Equal(t, len(genesis.CircuitBreakerStates), len(exported.CircuitBreakerStates))
	require.Equal(t, len(genesis.LiquidityPositions), len(exported.LiquidityPositions))

	// Verify circuit breaker states match
	for i, expected := range genesis.CircuitBreakerStates {
		actual := exported.CircuitBreakerStates[i]
		require.Equal(t, expected.PoolId, actual.PoolId)
		require.Equal(t, expected.Enabled, actual.Enabled)
		require.Equal(t, expected.LastPrice, actual.LastPrice)
		require.Equal(t, expected.TriggeredBy, actual.TriggeredBy)
		require.Equal(t, expected.TriggerReason, actual.TriggerReason)
		require.Equal(t, expected.NotificationsSent, actual.NotificationsSent)
	}

	// Verify liquidity positions match
	for i, expected := range genesis.LiquidityPositions {
		actual := exported.LiquidityPositions[i]
		require.Equal(t, expected.PoolId, actual.PoolId)
		require.Equal(t, expected.Provider, actual.Provider)
		require.Equal(t, expected.Shares, actual.Shares)
	}
}

// TestGenesisSharesValidation tests that InitGenesis validates LP shares sum equals pool.TotalShares
func TestGenesisSharesValidation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	params := types.DefaultParams()
	creatorOne := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 20)).String()
	providerOne := sdk.AccAddress(bytes.Repeat([]byte{0x3}, 20)).String()
	providerTwo := sdk.AccAddress(bytes.Repeat([]byte{0x4}, 20)).String()

	t.Run("valid shares sum", func(t *testing.T) {
		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(600_000),
			},
			{
				PoolId:   1,
				Provider: providerTwo,
				Shares:   sdkmath.NewInt(400_000),
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(sdk.WrapSDKContext(ctx), genesis)
		require.NoError(t, err)
	})

	t.Run("invalid shares sum - too many shares", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(700_000),
			},
			{
				PoolId:   1,
				Provider: providerTwo,
				Shares:   sdkmath.NewInt(400_000), // Total = 1,100,000 > pool.TotalShares
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(sdk.WrapSDKContext(ctx), genesis)
		require.Error(t, err)
		require.Contains(t, err.Error(), "shares mismatch")
	})

	t.Run("invalid shares sum - too few shares", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(1_000_000),
				ReserveB:    sdkmath.NewInt(2_000_000),
				TotalShares: sdkmath.NewInt(1_000_000),
				Creator:     creatorOne,
			},
		}

		liquidityPositions := []types.LiquidityPositionExport{
			{
				PoolId:   1,
				Provider: providerOne,
				Shares:   sdkmath.NewInt(300_000), // Total = 300,000 < pool.TotalShares
			},
		}

		genesis := types.GenesisState{
			Params:             params,
			Pools:              pools,
			NextPoolId:         2,
			LiquidityPositions: liquidityPositions,
		}

		err := k.InitGenesis(sdk.WrapSDKContext(ctx), genesis)
		require.Error(t, err)
		require.Contains(t, err.Error(), "shares mismatch")
	})

	t.Run("empty pool with no liquidity positions", func(t *testing.T) {
		k, ctx := keepertest.DexKeeper(t) // Fresh keeper for this test

		pools := []types.Pool{
			{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "paw",
				ReserveA:    sdkmath.NewInt(0),
				ReserveB:    sdkmath.NewInt(0),
				TotalShares: sdkmath.NewInt(0),
				Creator:     creatorOne,
			},
		}

		genesis := types.GenesisState{
			Params:     params,
			Pools:      pools,
			NextPoolId: 2,
		}

		err := k.InitGenesis(sdk.WrapSDKContext(ctx), genesis)
		require.NoError(t, err)
	})
}
