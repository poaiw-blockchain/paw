package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestUpdateReputationAdvanced(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	provider := sdk.AccAddress([]byte("rep_provider_addr"))

	specs := types.ComputeSpec{
		CpuCores:  4,
		MemoryMb:  4096,
		StorageGb: 50,
	}
	err := k.SetProvider(ctx, types.Provider{
		Address:        provider.String(),
		AvailableSpecs: specs,
		Stake:          math.NewInt(1_000_000),
		Reputation:     80,
		Active:         true,
	})
	require.NoError(t, err)

	err = k.UpdateReputationAdvanced(ctx, provider, true, 95, 1500, math.NewInt(1000))
	require.NoError(t, err)

	rep, err := k.GetProviderReputation(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, uint64(1), rep.TotalRequests)
	require.Greater(t, rep.ReliabilityScore, 0.5)

	providerRecord, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, rep.OverallScore, providerRecord.Reputation)
}

func TestApplyReputationDecay(t *testing.T) {
	now := time.Now().UTC()
	rep := &types.ProviderReputation{
		Provider:          "rep_decay",
		OverallScore:      90,
		ReliabilityScore:  0.9,
		AccuracyScore:     0.8,
		SpeedScore:        0.85,
		AvailabilityScore: 0.88,
		LastDecayTimestamp: now.Add(-48 * time.Hour),
	}

	k, _ := setupKeeperForTest(t)
	err := k.applyReputationDecay(rep, now)
	require.NoError(t, err)
	require.Less(t, rep.ReliabilityScore, 0.9)
	require.Equal(t, now, rep.LastDecayTimestamp)
	require.LessOrEqual(t, rep.OverallScore, uint32(100))
	require.GreaterOrEqual(t, rep.OverallScore, uint32(0))
}

func TestDecayTowardsNeutral(t *testing.T) {
	require.Equal(t, 0.5, decayTowardsNeutral(0.6, 0.2))
	require.Equal(t, 0.5, decayTowardsNeutral(0.4, 0.2))
	require.InEpsilon(t, 0.55, decayTowardsNeutral(0.6, 0.05), 1e-9)
}

func TestSelectProviderAdvanced(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	provider := sdk.AccAddress([]byte("adv_provider_addr"))

	specs := types.ComputeSpec{
		CpuCores:  4,
		MemoryMb:  4096,
		GpuCount:  0,
		StorageGb: 20,
	}
	err := k.SetProvider(ctx, types.Provider{
		Address:        provider.String(),
		AvailableSpecs: specs,
		Stake:          math.NewInt(500_000),
		Reputation:     90,
		Active:         true,
	})
	require.NoError(t, err)
	require.NoError(t, k.setActiveProviderIndex(ctx, provider, true))

	selected, err := k.SelectProviderAdvanced(ctx, types.ComputeSpec{
		CpuCores:  2,
		MemoryMb:  1024,
		StorageGb: 5,
	}, 1, "")
	require.NoError(t, err)
	require.Equal(t, provider.String(), selected.String())
}

func TestIsProviderEligible(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	provider := sdk.AccAddress([]byte("eligible_provider"))
	specs := types.ComputeSpec{
		CpuCores:  2,
		MemoryMb:  2048,
		StorageGb: 10,
	}
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:        provider.String(),
		AvailableSpecs: specs,
		Stake:          math.NewInt(100_000),
		Reputation:     80,
		Active:         true,
	}))

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.True(t, k.isProviderEligible(ctx, provider, specs, params))

	// Inactive provider should fail eligibility
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:        provider.String(),
		AvailableSpecs: specs,
		Stake:          math.NewInt(100_000),
		Reputation:     80,
		Active:         false,
	}))
	require.False(t, k.isProviderEligible(ctx, provider, specs, params))
}

func TestCalculateSelectionScore(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	provider := sdk.AccAddress([]byte("score_provider"))
	specs := types.ComputeSpec{
		CpuCores:  4,
		MemoryMb:  4096,
		StorageGb: 10,
	}
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:        provider.String(),
		AvailableSpecs: specs,
		Stake:          math.NewInt(1_000_000),
		Reputation:     85,
		Active:         true,
	}))
	require.NoError(t, k.SetProviderLoadTracker(ctx, types.ProviderLoadTracker{
		Provider:              provider.String(),
		MaxConcurrentRequests: 10,
		CurrentRequests:       5,
		TotalCpuCores:         10,
		UsedCpuCores:          5,
		TotalMemoryMb:         8192,
		UsedMemoryMb:          4096,
		TotalGpus:             2,
		UsedGpus:              1,
		LastUpdated:           sdk.UnwrapSDKContext(ctx).BlockTime(),
	}))

	rep := &types.ProviderReputation{
		Provider:     provider.String(),
		OverallScore: 85,
	}
	score := k.calculateSelectionScore(ctx, types.Provider{
		Address:    provider.String(),
		Stake:      math.NewInt(1_000_000),
		Reputation: 85,
	}, rep, specs, 42)

	require.True(t, score.LoadFactor <= 1.0 && score.LoadFactor >= 0)
	require.Greater(t, score.TotalScore, 0.0)
}

func TestWeightedRandomSelection(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	candidates := []ProviderSelectionScore{
		{Provider: "p1", TotalScore: 1.0},
		{Provider: "p2", TotalScore: 0.5},
	}

	selected := k.weightedRandomSelection(ctx, candidates, 99)
	require.NotNil(t, selected)
	require.True(t, selected.Provider == "p1" || selected.Provider == "p2")
}

func TestIterateProviderReputations(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	providerA := sdk.AccAddress([]byte("rep_iter_a"))
	providerB := sdk.AccAddress([]byte("rep_iter_b"))

	require.NoError(t, k.SetProviderReputation(ctx, types.ProviderReputation{
		Provider:     providerA.String(),
		OverallScore: 70,
	}))
	require.NoError(t, k.SetProviderReputation(ctx, types.ProviderReputation{
		Provider:     providerB.String(),
		OverallScore: 80,
	}))

	count := 0
	err := k.IterateProviderReputations(ctx, func(rep types.ProviderReputation) (bool, error) {
		count++
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
