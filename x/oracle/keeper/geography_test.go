package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestCheckByzantineToleranceGeography(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	// SEC-11: Use block height past bootstrap grace period (10000) for error testing
	ctx = ctx.WithBlockHeight(10001)
	params := types.DefaultParams()
	params.MinGeographicRegions = 2
	params.AllowedRegions = []string{"na", "eu", "apac"}
	require.NoError(t, k.SetParams(ctx, params))

	regions := []string{"na", "eu", "apac"}
	var validators []sdk.ValAddress
	for i := 0; i < 7; i++ {
		val := sdk.ValAddress([]byte(fmt.Sprintf("validator_geo_%02d", i)))
		validators = append(validators, val)
		keepertest.RegisterTestOracle(t, k, ctx, val.String())
		require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
			ValidatorAddr:    val.String(),
			GeographicRegion: regions[i%len(regions)],
			IsActive:         true,
		}))
	}

	// Should pass with sufficient diversity
	require.NoError(t, k.CheckByzantineTolerance(ctx))

	// Force insufficient diversity: set both to the same region
	for _, val := range validators {
		require.NoError(t, k.SetValidatorOracle(ctx, types.ValidatorOracle{
			ValidatorAddr:    val.String(),
			GeographicRegion: "na",
			IsActive:         true,
		}))
	}
	err := k.CheckByzantineTolerance(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient geographic diversity")
}
