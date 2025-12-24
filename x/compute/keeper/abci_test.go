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

// TestCleanupExpiredNonces tests nonce cleanup with configurable retention
func TestCleanupExpiredNonces(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	// Set retention period to 1000 blocks for testing
	params := types.DefaultParams()
	params.NonceRetentionBlocks = 1000
	require.NoError(t, k.SetParams(ctx, params))

	// Create test provider addresses
	provider1 := createTestProviderAddr(t, 1)
	provider2 := createTestProviderAddr(t, 2)

	// Record nonces at different heights
	// Height 100: Record nonces that will be old
	ctx = ctx.WithBlockHeight(100)
	k.RecordNonceUsageForTesting(ctx, provider1, 1)
	k.RecordNonceUsageForTesting(ctx, provider1, 2)
	k.RecordNonceUsageForTesting(ctx, provider2, 1)

	// Height 500: Record nonces that will be in the middle
	ctx = ctx.WithBlockHeight(500)
	k.RecordNonceUsageForTesting(ctx, provider1, 3)
	k.RecordNonceUsageForTesting(ctx, provider2, 2)

	// Height 1050: Record recent nonces that should NOT be cleaned
	ctx = ctx.WithBlockHeight(1050)
	k.RecordNonceUsageForTesting(ctx, provider1, 4)
	k.RecordNonceUsageForTesting(ctx, provider2, 3)

	// Verify all nonces exist before cleanup
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 1))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 2))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 3))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 4))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider2, 1))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider2, 2))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider2, 3))

	// Advance to height 1105 and run cleanup
	// Cutoff = 1105 - 1000 = 105, so nonces at height 100 should be cleaned
	ctx = ctx.WithBlockHeight(1105)
	err := k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)

	// Verify old nonces (height 100) were cleaned
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider1, 1))
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider1, 2))
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider2, 1))

	// Verify middle nonces (height 500) were NOT cleaned (within 1000 block window)
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 3))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider2, 2))

	// Verify recent nonces (height 1050) were NOT cleaned
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider1, 4))
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider2, 3))

	// Test cleanup at low block height (should not error or clean anything)
	ctx = ctx.WithBlockHeight(500)
	err = k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)
}

// TestCleanupExpiredNoncesCustomRetention tests cleanup with custom retention periods
func TestCleanupExpiredNoncesCustomRetention(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	provider := createTestProviderAddr(t, 1)

	// Test with shorter retention for testing (300 blocks)
	params := types.DefaultParams()
	params.NonceRetentionBlocks = 300
	require.NoError(t, k.SetParams(ctx, params))

	// Record nonce at height 1
	ctx = ctx.WithBlockHeight(1)
	k.RecordNonceUsageForTesting(ctx, provider, 1)

	// Record nonce at height 50 (should be cleaned)
	ctx = ctx.WithBlockHeight(50)
	k.RecordNonceUsageForTesting(ctx, provider, 2)

	// Record nonce at height 250 (should be kept)
	ctx = ctx.WithBlockHeight(250)
	k.RecordNonceUsageForTesting(ctx, provider, 3)

	// Cleanup at height 400 (cutoff = 400 - 300 = 100)
	// Cleanup processes heights 0-99 (startHeight=0, cutoffHeight=100, loop is height < cutoffHeight)
	// So nonces at heights 1 and 50 should be cleaned, but not height 100 or above
	ctx = ctx.WithBlockHeight(400)
	err := k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)

	// Nonces at heights 1 and 50 should be deleted (< cutoff height)
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider, 1), "nonce at height 1 should be deleted")
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider, 2), "nonce at height 50 should be deleted")

	// Nonce at height 250 should be kept (within retention window)
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider, 3), "nonce at height 250 should be kept")
}

// TestCleanupExpiredNoncesZeroRetention tests behavior with zero retention
func TestCleanupExpiredNoncesZeroRetention(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)

	provider := createTestProviderAddr(t, 1)

	// Set zero retention (should use default of 17280)
	params := types.DefaultParams()
	params.NonceRetentionBlocks = 0
	require.NoError(t, k.SetParams(ctx, params))

	// Record nonce at recent height
	ctx = ctx.WithBlockHeight(100)
	k.RecordNonceUsageForTesting(ctx, provider, 1)

	// Cleanup at height 200 should NOT clean (default retention is 17280)
	ctx = ctx.WithBlockHeight(200)
	err := k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)
	require.True(t, k.CheckReplayAttackForTesting(ctx, provider, 1), "nonce should be kept with default retention")

	// Cleanup at a much later height (> 17280 blocks later)
	// At height 18000, cutoff = 18000 - 17280 = 720
	// Cleanup will process heights 620-719 (startHeight = 720-100 = 620)
	// Height 100 is not in this range, so we need to go even later
	ctx = ctx.WithBlockHeight(18000)
	err = k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)
	// Height 100 is well below cutoff (720) but cleanup only processes last 100 blocks
	// Need to advance block height so that height 100 falls within the cleanup range
	// For height 100 to be in range, we need cutoffHeight > 100 and startHeight <= 100
	// startHeight = cutoffHeight - 100, so cutoffHeight - 100 <= 100, cutoffHeight <= 200
	// currentHeight - retentionBlocks <= 200, currentHeight <= 200 + 17280 = 17480
	ctx = ctx.WithBlockHeight(17480)
	err = k.CleanupExpiredNonces(ctx)
	require.NoError(t, err)
	// Cutoff = 17480 - 17280 = 200, startHeight = 100, processes heights 100-199
	require.False(t, k.CheckReplayAttackForTesting(ctx, provider, 1), "nonce should be deleted with default retention")
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
package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestProcessPendingDisputesTransitionsStatuses(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()

	// Evidence submission -> voting transition
	disputeEvidence := types.Dispute{
		Id:             1,
		RequestId:      10,
		Status:         types.DISPUTE_STATUS_EVIDENCE_SUBMISSION,
		EvidenceEndsAt: now.Add(-time.Hour),
	}
	require.NoError(t, k.setDispute(sdkCtx, disputeEvidence))

	// Voting expired path (ResolveDispute may fail but should not panic)
	disputeVoting := types.Dispute{
		Id:           2,
		RequestId:    20,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: now.Add(-time.Hour),
	}
	require.NoError(t, k.setDispute(sdkCtx, disputeVoting))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	d1, err := k.getDispute(sdkCtx, 1)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_VOTING, d1.Status)
}

func TestProcessPendingDisputesAutoResolve(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Lower consensus threshold to zero to allow resolution with one vote
	gov, err := k.GetGovernanceParams(sdkCtx)
	require.NoError(t, err)
	gov.ConsensusThreshold = math.LegacyZeroDec()
	require.NoError(t, k.SetGovernanceParams(sdkCtx, gov))

	voter := sdk.AccAddress([]byte("voter_address"))
	dispute := types.Dispute{
		Id:           3,
		RequestId:    30,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: sdkCtx.BlockTime().Add(-time.Minute),
		Votes: []types.DisputeVote{
			{
				Validator:   voter.String(),
				Option:      types.DISPUTE_VOTE_PROVIDER_FAULT,
				VotingPower: math.NewInt(1),
			},
		},
	}
	require.NoError(t, k.setDispute(sdkCtx, dispute))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	resolved, err := k.getDispute(sdkCtx, dispute.Id)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_RESOLVED, resolved.Status)
}

func TestProcessPendingDisputesSkipsInvalidAuthority(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Corrupt authority so resolution is skipped
	k.authority = "invalid"

	dispute := types.Dispute{
		Id:           4,
		RequestId:    40,
		Status:       types.DISPUTE_STATUS_VOTING,
		VotingEndsAt: sdkCtx.BlockTime().Add(-time.Minute),
	}
	require.NoError(t, k.setDispute(sdkCtx, dispute))

	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	stored, err := k.getDispute(sdkCtx, dispute.Id)
	require.NoError(t, err)
	require.Equal(t, types.DISPUTE_STATUS_VOTING, stored.Status)
}
package keeper

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestProcessPendingDisputesMultiQueueEmitsEvents(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	now := sdkCtx.BlockTime()

	d1 := types.Dispute{Id: 11, RequestId: 101, Status: types.DISPUTE_STATUS_EVIDENCE_SUBMISSION, EvidenceEndsAt: now.Add(-time.Minute)}
	d2 := types.Dispute{Id: 12, RequestId: 102, Status: types.DISPUTE_STATUS_VOTING, VotingEndsAt: now.Add(-time.Minute)}
	require.NoError(t, k.setDispute(sdkCtx, d1))
	require.NoError(t, k.setDispute(sdkCtx, d2))

	// ensure authority is valid to attempt auto-resolution
	require.NoError(t, k.ProcessPendingDisputes(sdkCtx))

	events := sdkCtx.EventManager().Events()
	require.NotEmpty(t, events)
}
package keeper

import (
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
	k, ctx := setupKeeperForTest(t)
	now := time.Now()

	prov := types.Provider{
		Address:      sdk.AccAddress([]byte("addr")).String(),
		RegisteredAt: now.Add(-2 * time.Hour),
		LastActiveAt: now.Add(-30 * time.Minute),
	}
	require.Equal(t, uint32(90), k.calculateUptimeScore(ctx, prov, now))

	prov.LastActiveAt = now.Add(-9 * time.Hour)
	require.Equal(t, uint32(30), k.calculateUptimeScore(ctx, prov, now))

	prov.RegisteredAt = now.Add(time.Minute)
	require.Equal(t, uint32(100), k.calculateUptimeScore(ctx, prov, now))
}
