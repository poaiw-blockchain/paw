package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Helper function to create a test provider address
func createTestProviderAddr(t *testing.T, index int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("test_provider_"))
	addr[19] = byte(index)
	return sdk.AccAddress(addr)
}

// TestGetProviderStats tests provider stats retrieval
func TestGetProviderStats(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Get stats for new provider (should return nil, nil for non-existent provider)
	stats, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Nil(t, stats) // New provider has no stats yet

	// After incrementing, stats should exist
	err = k.IncrementProviderJobCompleted(ctx, provider.String())
	require.NoError(t, err)

	stats, err = k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Equal(t, uint64(1), stats.TotalJobsCompleted)
	require.Equal(t, uint64(0), stats.TotalJobsFailed)
	require.Equal(t, uint64(0), stats.TotalDisputes)
}

// TestSetProviderStats tests setting provider stats
func TestSetProviderStats(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	stats := &keeper.ProviderStats{
		TotalJobsCompleted: 90,
		TotalJobsFailed:    5,
		TotalDisputes:      5,
		DisputesLost:       2,
		TotalEarnings:      1000000,
		AverageJobTime:     60,
	}

	err := k.SetProviderStats(ctx, provider.String(), stats)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, stats.TotalJobsCompleted, retrieved.TotalJobsCompleted)
	require.Equal(t, stats.TotalJobsFailed, retrieved.TotalJobsFailed)
	require.Equal(t, stats.TotalDisputes, retrieved.TotalDisputes)
}

// TestIncrementProviderJobCompleted tests incrementing completed job counter
func TestIncrementProviderJobCompleted(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Increment for new provider
	err := k.IncrementProviderJobCompleted(ctx, provider.String())
	require.NoError(t, err)

	stats, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, uint64(1), stats.TotalJobsCompleted)

	// Increment again
	err = k.IncrementProviderJobCompleted(ctx, provider.String())
	require.NoError(t, err)

	stats, err = k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, uint64(2), stats.TotalJobsCompleted)
}

// TestIncrementProviderJobFailed tests incrementing failed job counter
func TestIncrementProviderJobFailed(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Increment for new provider
	err := k.IncrementProviderJobFailed(ctx, provider.String())
	require.NoError(t, err)

	stats, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, uint64(1), stats.TotalJobsFailed)
}

// TestIncrementProviderDispute tests incrementing dispute counter
func TestIncrementProviderDispute(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Increment for new provider (dispute won)
	err := k.IncrementProviderDispute(ctx, provider.String(), false)
	require.NoError(t, err)

	stats, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, uint64(1), stats.TotalDisputes)
	require.Equal(t, uint64(0), stats.DisputesLost)

	// Increment again (dispute lost)
	err = k.IncrementProviderDispute(ctx, provider.String(), true)
	require.NoError(t, err)

	stats, err = k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	require.Equal(t, uint64(2), stats.TotalDisputes)
	require.Equal(t, uint64(1), stats.DisputesLost)
}

// TestBeginBlocker tests the BeginBlocker function
func TestBeginBlocker(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set initial params
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Run BeginBlocker
	err := k.BeginBlocker(ctx)
	require.NoError(t, err)
}

// TestEndBlocker tests the EndBlocker function
func TestEndBlocker(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set initial params
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Run EndBlocker
	err := k.EndBlocker(ctx)
	require.NoError(t, err)
}

// TestCleanupExpiredNonces tests nonce cleanup
func TestCleanupExpiredNonces(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Get params with default nonce expiry
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Run cleanup at low block height (should not error)
	err := k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)
}

// TestProcessPendingDisputes tests dispute processing
func TestProcessPendingDisputes(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Process disputes (should not error even with no disputes)
	err := k.ProcessPendingDisputes(ctx)
	require.NoError(t, err)
}

// TestUpdateProviderReputations tests reputation updates
func TestUpdateProviderReputations(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Register a provider first
	providerObj := types.Provider{
		Address:    provider.String(),
		Stake:      math.NewInt(1000000),
		Active:     true,
		Reputation: 100, // uint32, score 0-100
	}
	err := k.SetProvider(ctx, providerObj)
	require.NoError(t, err)

	// Set some stats
	stats := &keeper.ProviderStats{
		TotalJobsCompleted: 9,
		TotalJobsFailed:    1,
		TotalDisputes:      1,
	}
	err = k.SetProviderStats(ctx, provider.String(), stats)
	require.NoError(t, err)

	// Update reputations
	err = k.UpdateProviderReputations(ctx)
	require.NoError(t, err)
}

// TestProviderStatsWithHistory tests provider stats over multiple operations
func TestProviderStatsWithHistory(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	provider := createTestProviderAddr(t, 1)

	// Simulate multiple jobs
	for i := 0; i < 10; i++ {
		if i%3 == 0 {
			err := k.IncrementProviderJobFailed(ctx, provider.String())
			require.NoError(t, err)
		} else {
			err := k.IncrementProviderJobCompleted(ctx, provider.String())
			require.NoError(t, err)
		}
	}

	// Verify final stats
	stats, err := k.GetProviderStats(ctx, provider.String())
	require.NoError(t, err)
	totalJobs := stats.TotalJobsCompleted + stats.TotalJobsFailed
	require.Equal(t, uint64(10), totalJobs)
	require.Equal(t, uint64(6), stats.TotalJobsCompleted) // indices 1, 2, 4, 5, 7, 8
	require.Equal(t, uint64(4), stats.TotalJobsFailed)    // indices 0, 3, 6, 9
}

// TestBlockerWithContext tests ABCI handlers with different block contexts
func TestBlockerWithContext(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Test with specific block height
	ctx = ctx.WithBlockHeight(100)
	err := k.BeginBlocker(ctx)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(101)
	err = k.EndBlocker(ctx)
	require.NoError(t, err)
}

// TestBlockerWithBlockTime tests ABCI handlers with different block times
func TestBlockerWithBlockTime(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set params
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Test with specific block time
	blockTime := time.Now().Add(time.Hour)
	header := cmtproto.Header{
		Time:   blockTime,
		Height: 200,
	}
	ctx = ctx.WithBlockHeader(header)

	err := k.BeginBlocker(ctx)
	require.NoError(t, err)

	err = k.EndBlocker(ctx)
	require.NoError(t, err)
}
