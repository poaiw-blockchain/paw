package keeper

import (
	"bytes"
	"crypto/sha256"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestCommitRandomness(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Generate a valid commitment hash
	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i)
	}
	commitmentHash := GenerateCommitmentHash(revealValue, validator)

	// Test successful commitment
	err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
	require.NoError(t, err)

	// Verify commitment was stored
	commitment, err := k.GetRandomnessCommitment(sdkCtx, validator)
	require.NoError(t, err)
	require.Equal(t, validator.String(), commitment.Validator)
	require.Equal(t, commitmentHash, commitment.CommitmentHash)
	require.Equal(t, types.RANDOMNESS_COMMITMENT_STATUS_COMMITTED, commitment.Status)
	require.Nil(t, commitment.RevealValue)
}

func TestCommitRandomnessInvalidSize(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Test with invalid size commitment hash
	invalidHash := make([]byte, 16) // Wrong size
	err := k.CommitRandomness(sdkCtx, validator, invalidHash)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid commitment hash size")
}

func TestCommitRandomnessDuplicateBlock(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// First commitment
	revealValue := make([]byte, RandomnessRevealSize)
	commitmentHash := GenerateCommitmentHash(revealValue, validator)
	err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
	require.NoError(t, err)

	// Second commitment for same block should fail
	revealValue2 := make([]byte, RandomnessRevealSize)
	revealValue2[0] = 1
	commitmentHash2 := GenerateCommitmentHash(revealValue2, validator)
	err = k.CommitRandomness(sdkCtx, validator, commitmentHash2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already has a commitment")
}

func TestRevealRandomness(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Generate and commit
	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i * 2)
	}
	commitmentHash := GenerateCommitmentHash(revealValue, validator)
	err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
	require.NoError(t, err)

	// Reveal
	err = k.RevealRandomness(sdkCtx, validator, revealValue)
	require.NoError(t, err)

	// Verify reveal was stored
	commitment, err := k.GetRandomnessCommitment(sdkCtx, validator)
	require.NoError(t, err)
	require.Equal(t, types.RANDOMNESS_COMMITMENT_STATUS_REVEALED, commitment.Status)
	require.Equal(t, revealValue, commitment.RevealValue)
	require.NotNil(t, commitment.RevealedAt)
}

func TestRevealRandomnessInvalidValue(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Generate and commit
	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i)
	}
	commitmentHash := GenerateCommitmentHash(revealValue, validator)
	err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
	require.NoError(t, err)

	// Try to reveal with wrong value
	wrongValue := make([]byte, RandomnessRevealSize)
	wrongValue[0] = 255 // Different value
	err = k.RevealRandomness(sdkCtx, validator, wrongValue)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

func TestRevealRandomnessNoCommitment(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Try to reveal without commitment
	revealValue := make([]byte, RandomnessRevealSize)
	err := k.RevealRandomness(sdkCtx, validator, revealValue)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no commitment found")
}

func TestRevealRandomnessAlreadyRevealed(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	// Generate, commit, and reveal
	revealValue := make([]byte, RandomnessRevealSize)
	commitmentHash := GenerateCommitmentHash(revealValue, validator)
	err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
	require.NoError(t, err)
	err = k.RevealRandomness(sdkCtx, validator, revealValue)
	require.NoError(t, err)

	// Try to reveal again
	err = k.RevealRandomness(sdkCtx, validator, revealValue)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already revealed")
}

func TestGetAggregatedRandomness(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get randomness without any commits (should still work with block entropy)
	seed := []byte("test_seed")
	result := k.GetAggregatedRandomness(sdkCtx, seed)
	require.NotNil(t, result)
	require.True(t, result.Sign() > 0)

	// Get again with same seed - should be deterministic
	result2 := k.GetAggregatedRandomness(sdkCtx, seed)
	require.Equal(t, result.Bytes(), result2.Bytes())

	// Different seed should give different result
	differentSeed := []byte("different_seed")
	result3 := k.GetAggregatedRandomness(sdkCtx, differentSeed)
	require.NotEqual(t, result.Bytes(), result3.Bytes())
}

func TestSimulateCommitReveal(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validator := sdk.AccAddress([]byte("validator_address___"))

	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i + 10)
	}

	// Simulate commit-reveal in one call
	err := k.SimulateCommitReveal(sdkCtx, validator, revealValue)
	require.NoError(t, err)

	// Verify commitment is revealed
	commitment, err := k.GetRandomnessCommitment(sdkCtx, validator)
	require.NoError(t, err)
	require.Equal(t, types.RANDOMNESS_COMMITMENT_STATUS_REVEALED, commitment.Status)
	require.Equal(t, revealValue, commitment.RevealValue)
}

func TestProcessRandomnessRevealPhase(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create multiple validators and their reveals
	validators := []sdk.AccAddress{
		sdk.AccAddress([]byte("validator_addr_1____")),
		sdk.AccAddress([]byte("validator_addr_2____")),
		sdk.AccAddress([]byte("validator_addr_3____")),
	}

	for i, validator := range validators {
		revealValue := make([]byte, RandomnessRevealSize)
		for j := range revealValue {
			revealValue[j] = byte(i + j)
		}
		err := k.SimulateCommitReveal(sdkCtx, validator, revealValue)
		require.NoError(t, err)
	}

	// Process reveal phase
	err := k.ProcessRandomnessRevealPhase(sdkCtx)
	require.NoError(t, err)

	// Verify aggregated randomness was stored
	require.True(t, k.HasAggregatedRandomness(sdkCtx))
	aggregated := k.GetStoredAggregatedRandomness(sdkCtx)
	require.NotNil(t, aggregated)
	require.Len(t, aggregated, 32) // SHA256 output
}

func TestCountCommitmentsForBlock(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Initially zero
	count := k.CountCommitmentsForBlock(sdkCtx, sdkCtx.BlockHeight())
	require.Equal(t, 0, count)

	// Add commitments
	for i := 0; i < 5; i++ {
		validator := sdk.AccAddress([]byte("validator_addr_" + string(rune('A'+i)) + "____"))
		revealValue := make([]byte, RandomnessRevealSize)
		revealValue[0] = byte(i)
		commitmentHash := GenerateCommitmentHash(revealValue, validator)
		err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
		require.NoError(t, err)
	}

	// Count should be 5
	count = k.CountCommitmentsForBlock(sdkCtx, sdkCtx.BlockHeight())
	require.Equal(t, 5, count)
}

func TestMaxRandomnessParticipants(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Add max participants
	for i := 0; i < MaxRandomnessParticipants; i++ {
		validator := sdk.AccAddress([]byte("validator_addr_" + string(rune('A'+i)) + "_pad"))
		revealValue := make([]byte, RandomnessRevealSize)
		revealValue[0] = byte(i)
		commitmentHash := GenerateCommitmentHash(revealValue, validator)
		err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
		require.NoError(t, err)
	}

	// One more should fail
	extraValidator := sdk.AccAddress([]byte("extra_validator_addr"))
	revealValue := make([]byte, RandomnessRevealSize)
	commitmentHash := GenerateCommitmentHash(revealValue, extraValidator)
	err := k.CommitRandomness(sdkCtx, extraValidator, commitmentHash)
	require.Error(t, err)
	require.Contains(t, err.Error(), "maximum randomness participants")
}

func TestGenerateCommitmentHash(t *testing.T) {
	validator := sdk.AccAddress([]byte("test_validator_addr_"))
	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i)
	}

	// Generate hash
	hash := GenerateCommitmentHash(revealValue, validator)
	require.Len(t, hash, 32) // SHA256 output

	// Verify hash is deterministic
	hash2 := GenerateCommitmentHash(revealValue, validator)
	require.True(t, bytes.Equal(hash, hash2))

	// Verify different inputs produce different hashes
	revealValue2 := make([]byte, RandomnessRevealSize)
	revealValue2[0] = 255
	hash3 := GenerateCommitmentHash(revealValue2, validator)
	require.False(t, bytes.Equal(hash, hash3))

	// Manually verify the hash computation
	hasher := sha256.New()
	hasher.Write(revealValue)
	hasher.Write(validator.Bytes())
	expectedHash := hasher.Sum(nil)
	require.True(t, bytes.Equal(hash, expectedHash))
}

func TestGetAllRandomnessCommitments(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Initially empty
	commitments := k.GetAllRandomnessCommitments(sdkCtx)
	require.Empty(t, commitments)

	// Add some commitments
	validators := []sdk.AccAddress{
		sdk.AccAddress([]byte("validator_addr_1____")),
		sdk.AccAddress([]byte("validator_addr_2____")),
	}

	for i, validator := range validators {
		revealValue := make([]byte, RandomnessRevealSize)
		revealValue[0] = byte(i)
		commitmentHash := GenerateCommitmentHash(revealValue, validator)
		err := k.CommitRandomness(sdkCtx, validator, commitmentHash)
		require.NoError(t, err)
	}

	// Should have 2 commitments
	commitments = k.GetAllRandomnessCommitments(sdkCtx)
	require.Len(t, commitments, 2)
}

func TestGenerateSecureRandomnessIntegration(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup some randomness commits
	validator := sdk.AccAddress([]byte("validator_address___"))
	revealValue := make([]byte, RandomnessRevealSize)
	for i := range revealValue {
		revealValue[i] = byte(i * 3)
	}
	err := k.SimulateCommitReveal(sdkCtx, validator, revealValue)
	require.NoError(t, err)

	// Process reveal phase to aggregate
	err = k.ProcessRandomnessRevealPhase(sdkCtx)
	require.NoError(t, err)

	// Now GenerateSecureRandomness should use the aggregated randomness
	seed := []byte("provider_selection_seed")
	randomness := k.GenerateSecureRandomness(sdkCtx, seed)
	require.NotNil(t, randomness)
	require.True(t, randomness.Sign() > 0)

	// Verify it's different from legacy (without commits)
	// Create fresh context without commits
	k2, ctx2 := setupKeeperForTest(t)
	sdkCtx2 := sdk.UnwrapSDKContext(ctx2)
	legacyRandomness := k2.GenerateSecureRandomnessLegacy(sdkCtx2, seed)
	require.NotNil(t, legacyRandomness)

	// They should be different (different entropy sources)
	require.NotEqual(t, randomness.Bytes(), legacyRandomness.Bytes())
}
