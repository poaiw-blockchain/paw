package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestZKMetricsLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	metrics, err := k.GetZKMetrics(ctx)
	require.NoError(t, err)
	require.False(t, metrics.LastUpdated.IsZero())

	updated := types.ZKMetrics{
		TotalProofsGenerated:       12,
		TotalProofsVerified:        10,
		TotalProofsFailed:          2,
		AverageVerificationTimeMs:  123,
		TotalGasConsumed:           5555,
		LastUpdated:                ctx.BlockTime().Add(time.Minute),
	}

	require.NoError(t, k.SetZKMetrics(ctx, updated))

	stored, err := k.GetZKMetrics(ctx)
	require.NoError(t, err)
	require.Equal(t, updated.TotalProofsGenerated, stored.TotalProofsGenerated)
	require.Equal(t, updated.TotalProofsVerified, stored.TotalProofsVerified)
	require.Equal(t, updated.TotalProofsFailed, stored.TotalProofsFailed)
	require.Equal(t, updated.AverageVerificationTimeMs, stored.AverageVerificationTimeMs)
	require.Equal(t, updated.TotalGasConsumed, stored.TotalGasConsumed)
	require.True(t, stored.LastUpdated.Equal(updated.LastUpdated))
}
