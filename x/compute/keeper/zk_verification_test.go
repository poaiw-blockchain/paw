package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/circuits"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestZKProofRejectsInvalidProof(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	zkVerifier := computekeeper.NewZKVerifier(k)

	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	require.NoError(t, zkVerifier.InitializeCircuit(ctx, circuitID))

	providerKey := secp256k1.GenPrivKey()
	provider := sdk.AccAddress(providerKey.PubKey().Address())

	requestID := uint64(42)
	computationData := []byte("test computation")
	timestamp := time.Now().Unix()

	resultHash, err := computekeeper.ComputeResultHash(
		requestID,
		provider,
		computationData,
		timestamp,
		0,
		1000,
		2048,
	)
	require.NoError(t, err)

	proof, err := zkVerifier.GenerateProof(
		ctx,
		requestID,
		resultHash,
		provider,
		computationData,
		timestamp,
		0,
		1000,
		2048,
	)
	require.NoError(t, err)
	require.NotNil(t, proof)

	require.NotEmpty(t, proof.Proof, "proof bytes should be present")
	proof.Proof[0] ^= 0xFF

	valid, err := zkVerifier.VerifyProof(ctx, proof, requestID, resultHash, provider)
	require.Error(t, err)
	require.False(t, valid)
}

func TestZKProofIBCPacketRejectsInvalidProof(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	require.NoError(t, k.InitializeCircuits(ctx))

	err := k.VerifyIBCZKProofForTest(ctx, []byte("invalid-proof-data"), []byte("bad-input"))
	require.Error(t, err)
}

// TestZKProofRejectsOversizedProof verifies that oversized proofs are rejected
// BEFORE any significant gas consumption to prevent DoS attacks.
func TestZKProofRejectsOversizedProof(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	zkVerifier := computekeeper.NewZKVerifier(k)

	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	require.NoError(t, zkVerifier.InitializeCircuit(ctx, circuitID))

	providerKey := secp256k1.GenPrivKey()
	provider := sdk.AccAddress(providerKey.PubKey().Address())

	requestID := uint64(42)
	computationData := []byte("test computation")
	timestamp := time.Now().Unix()

	resultHash, err := computekeeper.ComputeResultHash(
		requestID,
		provider,
		computationData,
		timestamp,
		0,
		1000,
		2048,
	)
	require.NoError(t, err)

	// Create a proof that is larger than the maximum allowed size (1MB)
	// This simulates a DoS attack attempt
	oversizedProofData := make([]byte, 2*1024*1024) // 2MB - exceeds 1MB limit
	for i := range oversizedProofData {
		oversizedProofData[i] = byte(i % 256)
	}

	oversizedProof := &types.ZKProof{
		Proof:        oversizedProofData,
		PublicInputs: make([]byte, 72), // Valid size for public inputs
		ProofSystem:  "groth16",
		CircuitId:    circuitID,
		GeneratedAt:  ctx.BlockTime(),
	}

	// Get the initial gas consumed
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasBefore := sdkCtx.GasMeter().GasConsumed()

	// Attempt to verify the oversized proof - should fail immediately
	valid, err := zkVerifier.VerifyProof(ctx, oversizedProof, requestID, resultHash, provider)

	// Get the gas consumed after verification attempt
	gasAfter := sdkCtx.GasMeter().GasConsumed()
	gasConsumed := gasAfter - gasBefore

	// Verify the proof was rejected
	require.Error(t, err, "oversized proof should be rejected")
	require.False(t, valid, "oversized proof should not be valid")
	require.ErrorIs(t, err, types.ErrProofTooLarge, "error should be ErrProofTooLarge")

	// CRITICAL: Verify that minimal gas was consumed (< 5000 gas)
	// This proves we're checking size BEFORE expensive operations like deserialization.
	// Some gas is consumed by GetCircuitParams storage read (unavoidable), but
	// the important thing is we reject before deserializing the massive proof.
	require.Less(t, gasConsumed, uint64(5000),
		"oversized proof should be rejected with minimal gas consumption (got %d gas)", gasConsumed)

	// Additional security check: verify the error message doesn't leak sensitive info
	require.NotContains(t, err.Error(), "panic", "error should not contain panic information")
	require.NotContains(t, err.Error(), "internal", "error should not leak internal details")
}

// TestZKProofWithMaxAllowedSize verifies that proofs at exactly the maximum
// size are still processed (boundary condition test).
func TestZKProofWithMaxAllowedSize(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	zkVerifier := computekeeper.NewZKVerifier(k)

	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	require.NoError(t, zkVerifier.InitializeCircuit(ctx, circuitID))

	// Get the circuit params to check the max size
	circuitParams, err := k.GetCircuitParams(ctx, circuitID)
	require.NoError(t, err)
	require.NotNil(t, circuitParams)

	maxSize := circuitParams.MaxProofSize
	require.Greater(t, maxSize, uint32(0), "max proof size should be set")

	// Create a proof that is exactly at the max size
	// Note: This will fail verification because it's not a valid Groth16 proof,
	// but it should NOT be rejected for size reasons
	maxSizedProofData := make([]byte, maxSize)
	for i := range maxSizedProofData {
		maxSizedProofData[i] = byte(i % 256)
	}

	providerKey := secp256k1.GenPrivKey()
	provider := sdk.AccAddress(providerKey.PubKey().Address())
	requestID := uint64(100)
	resultHash := make([]byte, 32)

	maxSizedProof := &types.ZKProof{
		Proof:        maxSizedProofData,
		PublicInputs: make([]byte, 72),
		ProofSystem:  "groth16",
		CircuitId:    circuitID,
		GeneratedAt:  ctx.BlockTime(),
	}

	// This should NOT fail with ErrProofTooLarge
	_, err = zkVerifier.VerifyProof(ctx, maxSizedProof, requestID, resultHash, provider)

	// It will fail (because it's not a valid proof), but NOT because of size
	if err != nil {
		require.NotErrorIs(t, err, types.ErrProofTooLarge,
			"proof at max size should not be rejected for size reasons")
	}
}

// TestZKProofSizeJustOverLimit verifies that proofs even 1 byte over the limit are rejected.
func TestZKProofSizeJustOverLimit(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	zkVerifier := computekeeper.NewZKVerifier(k)

	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	require.NoError(t, zkVerifier.InitializeCircuit(ctx, circuitID))

	circuitParams, err := k.GetCircuitParams(ctx, circuitID)
	require.NoError(t, err)

	maxSize := circuitParams.MaxProofSize

	// Create a proof that is exactly 1 byte over the limit
	overLimitProofData := make([]byte, maxSize+1)

	providerKey := secp256k1.GenPrivKey()
	provider := sdk.AccAddress(providerKey.PubKey().Address())
	requestID := uint64(101)
	resultHash := make([]byte, 32)

	overLimitProof := &types.ZKProof{
		Proof:        overLimitProofData,
		PublicInputs: make([]byte, 72),
		ProofSystem:  "groth16",
		CircuitId:    circuitID,
		GeneratedAt:  ctx.BlockTime(),
	}

	// Should be rejected with ErrProofTooLarge
	valid, err := zkVerifier.VerifyProof(ctx, overLimitProof, requestID, resultHash, provider)
	require.Error(t, err)
	require.False(t, valid)
	require.ErrorIs(t, err, types.ErrProofTooLarge)
}
