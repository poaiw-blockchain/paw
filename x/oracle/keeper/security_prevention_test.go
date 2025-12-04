package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestOutlierSubmissionRejected ensures statistical outliers are rejected with a typed error.
func TestOutlierSubmissionRejected(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	asset := "ATOM"

	// Seed honest validator prices clustered around 100
	for i := 0; i < 3; i++ {
		validator := createTestValidatorWithIndex(t, i)
		err := k.SetValidatorPrice(ctx, types.ValidatorPrice{
			ValidatorAddr: validator.String(),
			Asset:         asset,
			Price:         sdkmath.LegacyNewDec(100 + int64(i)),
			BlockHeight:   1,
		})
		require.NoError(t, err)
	}

	attacker := createTestValidatorWithIndex(t, 10)
	outlierPrice := sdkmath.LegacyNewDec(1000)

	err := k.ImplementDataPoisoningPrevention(ctx, attacker.String(), asset, outlierPrice)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOutlierDetected)
	require.Contains(t, err.Error(), "deviates")
}
