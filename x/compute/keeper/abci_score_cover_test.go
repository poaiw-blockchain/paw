package keeper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestCalculateDisputeScorePenaltyAndDefaults(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	t.Run("nil stats returns perfect score", func(t *testing.T) {
		require.Equal(t, uint32(100), k.calculateDisputeScore(ctx, "missing"))
	})

	t.Run("loss rate and volume penalties applied", func(t *testing.T) {
		provider := "prov-disputes"
		stats := &ProviderStats{
			TotalDisputes: 20,
			DisputesLost:  5, // 25% loss -> base 75
		}
		require.NoError(t, k.SetProviderStats(ctx, provider, stats))

		score := k.calculateDisputeScore(ctx, provider)
		// Base 75 minus volume penalty ((20-10)*2 = 20) = 55
		require.Equal(t, uint32(55), score)
	})
}

func TestCalculateUptimeScoreBoundaries(t *testing.T) {
	k, _ := setupKeeperForTest(t)
	now := time.Now()

	prov := types.Provider{
		Address:      sdk.AccAddress([]byte("addr")).String(),
		RegisteredAt: now.Add(-2 * time.Hour),
		LastActiveAt: now.Add(-30 * time.Minute),
	}
	require.Equal(t, uint32(90), k.calculateUptimeScore(context.TODO(), prov, now))

	prov.LastActiveAt = now.Add(-9 * time.Hour)
	require.Equal(t, uint32(30), k.calculateUptimeScore(context.TODO(), prov, now))

	prov.RegisteredAt = now.Add(time.Minute)
	require.Equal(t, uint32(100), k.calculateUptimeScore(context.TODO(), prov, now))
}
