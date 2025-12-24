package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestPruneExpiredNonces_Integration tests the oracle keeper's nonce pruning integration
func TestPruneExpiredNonces_Integration(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now()
	sdkCtx = sdkCtx.WithBlockTime(baseTime)

	// Set default params
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60 // 7 days
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Create nonces at different ages
	oldTime := baseTime.Add(-10 * 24 * time.Hour) // 10 days old
	oldCtx := sdkCtx.WithBlockTime(oldTime)

	// Create expired nonces
	for i := 0; i < 5; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i))
		err := k.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Create recent nonces
	sdkCtx = sdkCtx.WithBlockTime(baseTime)
	for i := 0; i < 3; i++ {
		channel := "channel-1"
		sender := "sender" + string(rune('A'+i))
		err := k.ValidateIncomingPacketNonce(sdkCtx, channel, sender, uint64(i+1), baseTime.Unix())
		require.NoError(t, err)
	}

	// Prune expired nonces
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 5, prunedCount) // Only old nonces should be pruned

	// Verify recent nonces still work
	err = k.ValidateIncomingPacketNonce(sdkCtx, "channel-1", "senderA", 2, baseTime.Unix())
	require.NoError(t, err)
}

// TestPruneExpiredNonces_CustomTTL tests pruning with custom TTL parameter
func TestPruneExpiredNonces_CustomTTL(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now()
	sdkCtx = sdkCtx.WithBlockTime(baseTime)

	// Set custom TTL to 2 days
	params := types.DefaultParams()
	params.NonceTtlSeconds = 2 * 24 * 60 * 60
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Create nonce 3 days old (should be pruned)
	oldTime := baseTime.Add(-3 * 24 * time.Hour)
	oldCtx := sdkCtx.WithBlockTime(oldTime)
	err = k.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Create nonce 1 day old (should not be pruned)
	recentTime := baseTime.Add(-1 * 24 * time.Hour)
	recentCtx := sdkCtx.WithBlockTime(recentTime)
	err = k.ValidateIncomingPacketNonce(recentCtx, "channel-0", "sender2", 1, recentTime.Unix())
	require.NoError(t, err)

	// Prune
	sdkCtx = sdkCtx.WithBlockTime(baseTime)
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_ZeroTTLUsesDefault tests that zero TTL falls back to default
func TestPruneExpiredNonces_ZeroTTLUsesDefault(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now()
	sdkCtx = sdkCtx.WithBlockTime(baseTime)

	// Set TTL to 0 (should use default)
	params := types.DefaultParams()
	params.NonceTtlSeconds = 0
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Create nonce older than default TTL (7 days)
	oldTime := baseTime.Add(-8 * 24 * time.Hour)
	oldCtx := sdkCtx.WithBlockTime(oldTime)
	err = k.ValidateIncomingPacketNonce(oldCtx, "channel-0", "sender1", 1, oldTime.Unix())
	require.NoError(t, err)

	// Prune (should use default 7-day TTL)
	sdkCtx = sdkCtx.WithBlockTime(baseTime)
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, prunedCount)
}

// TestPruneExpiredNonces_HighVolume tests pruning with many nonces
func TestPruneExpiredNonces_HighVolume(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now()
	sdkCtx = sdkCtx.WithBlockTime(baseTime)

	// Set params
	params := types.DefaultParams()
	params.NonceTtlSeconds = 7 * 24 * 60 * 60
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Create 200 expired nonces
	oldTime := baseTime.Add(-10 * 24 * time.Hour)
	oldCtx := sdkCtx.WithBlockTime(oldTime)
	for i := 0; i < 200; i++ {
		channel := "channel-0"
		sender := "sender" + string(rune('A'+i%26)) + string(rune('A'+i/26))
		err := k.ValidateIncomingPacketNonce(oldCtx, channel, sender, uint64(i+1), oldTime.Unix())
		require.NoError(t, err)
	}

	// Prune - should respect batch limit of 100 per call
	sdkCtx = sdkCtx.WithBlockTime(baseTime)
	totalPruned := 0

	// First call
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	require.LessOrEqual(t, prunedCount, 100)
	totalPruned += prunedCount

	// Second call
	prunedCount, err = k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	totalPruned += prunedCount

	// Should have pruned all 200 across two calls
	require.Equal(t, 200, totalPruned)
}

// TestPruneExpiredNonces_PreservesActiveNonces tests that active nonces are never pruned
func TestPruneExpiredNonces_PreservesActiveNonces(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	baseTime := time.Now()
	sdkCtx = sdkCtx.WithBlockTime(baseTime)

	// Set short TTL for testing
	params := types.DefaultParams()
	params.NonceTtlSeconds = 1 * 24 * 60 * 60 // 1 day
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Create active nonce
	err = k.ValidateIncomingPacketNonce(sdkCtx, "channel-0", "active-sender", 1, baseTime.Unix())
	require.NoError(t, err)

	// Advance time but within TTL (12 hours)
	laterTime := baseTime.Add(12 * time.Hour)
	sdkCtx = sdkCtx.WithBlockTime(laterTime)

	// Prune - should not prune anything
	prunedCount, err := k.PruneExpiredNonces(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 0, prunedCount)

	// Verify nonce still active
	err = k.ValidateIncomingPacketNonce(sdkCtx, "channel-0", "active-sender", 2, laterTime.Unix())
	require.NoError(t, err)
}

// TestParams_NonceTTLValidation tests that the nonce TTL parameter is properly stored
func TestParams_NonceTTLValidation(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

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

	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Retrieve and verify
	retrievedParams, err := k.GetParams(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(3*24*60*60), retrievedParams.NonceTtlSeconds)
}

// TestDefaultParams_IncludesNonceTTL tests that default params include nonce TTL
func TestDefaultParams_IncludesNonceTTL(t *testing.T) {
	params := types.DefaultParams()
	require.Equal(t, uint64(7*24*60*60), params.NonceTtlSeconds) // 7 days
}
