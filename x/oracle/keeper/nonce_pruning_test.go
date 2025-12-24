package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/types"
)

// TestPruneExpiredNonces_Integration tests the oracle keeper's nonce pruning integration
func TestPruneExpiredNonces_Integration(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set default params
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60 // 7 days
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create nonces at different ages
	oldTime := baseTime.Add(-10 * 24 * time.Hour) // 10 days old
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)

	// Create expired nonces
	for i := 0; i < 5; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := f.keeper.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Create recent nonces
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	for i := 0; i < 3; i++ {
		channel := "channel-1"
		sender := "sender" + string(rune('A'+i))
		err := f.keeper.ValidateIncomingPacketNonce(f.sdkCtx, channel, sender, uint64(i+1), baseTime.Unix())
		require.NoError(t, err)
	}

	// Prune expired nonces
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 5, prunedCount) // Only old nonces should be pruned

	// Verify recent nonces still work
	err = f.keeper.ValidateIncomingPacketNonce(f.sdkCtx, "channel-1", "senderA", 2, baseTime.Unix())
	require.NoError(t, err)
}

// TestPruneExpiredNonces_CustomTTL tests pruning with custom TTL parameter
func TestPruneExpiredNonces_CustomTTL(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set custom TTL to 2 days
	params := types.DefaultParams()
	params.NonceTtlSeconds = 2 * 24 * 60 * 60
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create nonce 3 days old (should be pruned)
	oldTime := baseTime.Add(-3 * 24 * time.Hour)
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)
	err = f.keeper.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Create nonce 1 day old (should not be pruned)
	recentTime := baseTime.Add(-1 * 24 * time.Hour)
	recentCtx := f.sdkCtx.WithBlockTime(recentTime)
	err = f.keeper.ValidateIncomingPacketNonce(recentCtx, "channel-0", "sender2", 1, recentTime.Unix())
	require.NoError(t, err)

	// Prune
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_ZeroTTLUsesDefault tests that zero TTL falls back to default
func TestPruneExpiredNonces_ZeroTTLUsesDefault(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set TTL to 0 (should use default)
	params := types.DefaultParams()
	params.NonceTtlSeconds = 0
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create nonce older than default TTL (7 days)
	oldTime := baseTime.Add(-8 * 24 * time.Hour)
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)
	err = f.keeper.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Prune (should use default 7-day TTL)
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestEndBlocker_PrunesNonces tests that EndBlocker calls nonce pruning
func TestEndBlocker_PrunesNonces(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set params
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create expired nonces
	oldTime := baseTime.Add(-10 * 24 * time.Hour)
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)
	for i := 0; i < 3; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := f.keeper.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Run EndBlocker
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	err = f.keeper.EndBlocker(f.ctx)
	require.NoError(t, err)

	// Verify nonces were pruned (can recreate with nonce 1)
	for i := 0; i < 3; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := f.keeper.ValidateIncomingPacketNonce(f.sdkCtx, channel, sender, 1, baseTime.Unix())
		require.NoError(t, err, "nonce should have been pruned for sender %c", 'A'+i)
	}
}

// TestPruneExpiredNonces_GovernanceUpdate tests that governance can update the TTL
func TestPruneExpiredNonces_GovernanceUpdate(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Start with 7-day TTL
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create nonce 5 days old
	oldTime := baseTime.Add(-5 * 24 * time.Hour)
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)
	err = f.keeper.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Update TTL to 3 days via governance
	params.NonceTtlSeconds = 3 * 24 * 60 * 60
	err = f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Now prune - the 5-day old nonce should be pruned with new 3-day TTL
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_HighVolume tests pruning with many nonces
func TestPruneExpiredNonces_HighVolume(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set params
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create 200 expired nonces
	oldTime := baseTime.Add(-10 * 24 * time.Hour)
	oldCtx := f.sdkCtx.WithBlockTime(oldTime)
	for i := 0; i < 200; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i%26)) + string(rune('A'+i/26))
		err := f.keeper.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Prune - should respect batch limit of 100 per call
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)
	totalPruned := 0

	// First call
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.LessOrEqual(t, prunedCount, 100)
	totalPruned += prunedCount

	// Second call
	prunedCount, err = f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	totalPruned += prunedCount

	// Should have pruned all 200 across two calls
	require.Equal(t, 200, totalPruned)
}

// TestPruneExpiredNonces_PreservesActiveNonces tests that active nonces are never pruned
func TestPruneExpiredNonces_PreservesActiveNonces(t *testing.T) {
	f := SetupKeeperTestFixture(t)
	baseTime := time.Now()
	f.sdkCtx = f.sdkCtx.WithBlockTime(baseTime)

	// Set short TTL for testing
	params := types.DefaultParams()
	params.NonceTtlSeconds = 1 * 24 * 60 * 60 // 1 day
	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create active nonce
	err = f.keeper.ValidateIncomingPacketNonce(f.sdkCtx, "channel-0", "active-sender", 1, baseTime.Unix())
	require.NoError(t, err)

	// Advance time but within TTL (12 hours)
	laterTime := baseTime.Add(12 * time.Hour)
	f.sdkCtx = f.sdkCtx.WithBlockTime(laterTime)

	// Prune - should not prune anything
	prunedCount, err := f.keeper.PruneExpiredNonces(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 0, prunedCount)

	// Verify nonce still active
	err = f.keeper.ValidateIncomingPacketNonce(f.sdkCtx, "channel-0", "active-sender", 2, laterTime.Unix())
	require.NoError(t, err)
}

// TestParams_NonceTTLValidation tests that the nonce TTL parameter is properly stored
func TestParams_NonceTTLValidation(t *testing.T) {
	f := SetupKeeperTestFixture(t)

	// Set params with custom nonce TTL
	params := types.Params{
		VotePeriod:                 30,
		VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
		SlashWindow:                10000,
		MinValidPerWindow:          100,
		TwapLookbackWindow:         1000,
		AuthorizedChannels:         []types.AuthorizedChannel{},
		AllowedRegions:             []string{"global"},
		MinGeographicRegions:       1,
		MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
		MaxValidatorsPerIp:         3,
		MaxValidatorsPerAsn:        5,
		RequireGeographicDiversity: false,
		NonceTtlSeconds:            3 * 24 * 60 * 60, // 3 days
	}

	err := f.keeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Retrieve and verify
	retrievedParams, err := f.keeper.GetParams(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3*24*60*60), retrievedParams.NonceTtlSeconds)
}

// TestDefaultParams_IncludesNonceTTL tests that default params include nonce TTL
func TestDefaultParams_IncludesNonceTTL(t *testing.T) {
	params := types.DefaultParams()
	require.Equal(t, uint64(7*24*60*60), params.NonceTtlSeconds) // 7 days
}
