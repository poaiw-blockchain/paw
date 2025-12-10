package keeper

import (
	"crypto/sha256"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestComputeMiMCHash(t *testing.T) {
	provider := sdk.AccAddress([]byte("test_provider_addr_"))

	t.Run("valid inputs", func(t *testing.T) {
		hash, err := ComputeMiMCHash(
			1,                  // requestID
			[]byte("provider"), // providerAddressHash
			[]byte("data"),     // computationDataHash
			1000,               // executionTimestamp
			0,                  // exitCode
			100,                // cpuCyclesUsed
			1024,               // memoryBytesUsed
		)
		require.NoError(t, err)
		require.NotNil(t, hash)
		require.Len(t, hash, 32)
	})

	t.Run("deterministic", func(t *testing.T) {
		hash1, err := ComputeMiMCHash(1, []byte("provider"), []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		hash2, err := ComputeMiMCHash(1, []byte("provider"), []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		require.Equal(t, hash1, hash2)
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		hash1, err := ComputeMiMCHash(1, []byte("provider1"), []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		hash2, err := ComputeMiMCHash(2, []byte("provider1"), []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		require.NotEqual(t, hash1, hash2)
	})

	_ = provider // silence unused variable
}

func TestNewZKVerifier(t *testing.T) {
	k, _ := setupKeeperForTest(t)
	verifier := NewZKVerifier(k)

	require.NotNil(t, verifier)
	require.NotNil(t, verifier.keeper)
	require.NotNil(t, verifier.circuitCCS)
	require.NotNil(t, verifier.provingKeys)
	require.NotNil(t, verifier.verifyingKeys)
}

func TestBytesToFieldElement(t *testing.T) {
	t.Run("nil bytes", func(t *testing.T) {
		elem := bytesToFieldElement(nil)
		require.NotNil(t, elem)
	})

	t.Run("empty bytes", func(t *testing.T) {
		elem := bytesToFieldElement([]byte{})
		require.NotNil(t, elem)
	})

	t.Run("32 bytes", func(t *testing.T) {
		data := make([]byte, 32)
		for i := range data {
			data[i] = byte(i)
		}
		elem := bytesToFieldElement(data)
		require.NotNil(t, elem)
	})

	t.Run("64 bytes truncates to 32", func(t *testing.T) {
		data := make([]byte, 64)
		for i := range data {
			data[i] = byte(i % 256)
		}
		elem := bytesToFieldElement(data)
		require.NotNil(t, elem)
	})

	t.Run("deterministic", func(t *testing.T) {
		data := []byte("test data")
		elem1 := bytesToFieldElement(data)
		elem2 := bytesToFieldElement(data)
		require.Equal(t, elem1, elem2)
	})
}

func TestSerializePublicInputs(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		requestID := uint64(123)
		resultHash := make([]byte, 32)
		providerAddressHash := make([]byte, 32)

		result := serializePublicInputs(requestID, resultHash, providerAddressHash)
		require.NotNil(t, result)
		require.Len(t, result, 8+32+32)
	})

	t.Run("deterministic", func(t *testing.T) {
		requestID := uint64(123)
		resultHash := make([]byte, 32)
		providerAddressHash := make([]byte, 32)

		result1 := serializePublicInputs(requestID, resultHash, providerAddressHash)
		result2 := serializePublicInputs(requestID, resultHash, providerAddressHash)
		require.Equal(t, result1, result2)
	})

	t.Run("different request IDs produce different results", func(t *testing.T) {
		resultHash := make([]byte, 32)
		providerAddressHash := make([]byte, 32)

		result1 := serializePublicInputs(1, resultHash, providerAddressHash)
		result2 := serializePublicInputs(2, resultHash, providerAddressHash)
		require.NotEqual(t, result1, result2)
	})
}

func TestComputeVerificationCircuit_Structure(t *testing.T) {
	circuit := &ComputeVerificationCircuit{}
	require.NotNil(t, circuit)
}

func TestComputeResultHashMiMC(t *testing.T) {
	provider := sdk.AccAddress([]byte("test_provider_addr_"))

	t.Run("valid inputs", func(t *testing.T) {
		hash, err := computeResultHashMiMC(
			1,              // requestID
			provider,       // providerAddress
			[]byte("data"), // computationData
			1000,           // executionTimestamp
			0,              // exitCode
			100,            // cpuCyclesUsed
			1024,           // memoryBytesUsed
		)
		require.NoError(t, err)
		require.NotNil(t, hash)
		require.Len(t, hash, 32)
	})

	t.Run("negative timestamp returns error", func(t *testing.T) {
		_, err := computeResultHashMiMC(1, provider, []byte("data"), -1, 0, 100, 1024)
		require.Error(t, err)
		require.Contains(t, err.Error(), "timestamp")
	})

	t.Run("negative exit code returns error", func(t *testing.T) {
		_, err := computeResultHashMiMC(1, provider, []byte("data"), 1000, -1, 100, 1024)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exit code")
	})

	t.Run("deterministic", func(t *testing.T) {
		hash1, err := computeResultHashMiMC(1, provider, []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		hash2, err := computeResultHashMiMC(1, provider, []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		require.Equal(t, hash1, hash2)
	})

	t.Run("different providers produce different hashes", func(t *testing.T) {
		provider2 := sdk.AccAddress([]byte("other_provider_addr"))

		hash1, err := computeResultHashMiMC(1, provider, []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		hash2, err := computeResultHashMiMC(1, provider2, []byte("data"), 1000, 0, 100, 1024)
		require.NoError(t, err)

		require.NotEqual(t, hash1, hash2)
	})
}

func TestVerifyProofRejectsOversizedProof(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	zkVerifier := NewZKVerifier(k)

	proof := &types.ZKProof{
		Proof:        make([]byte, 10*1024*1024+1), // exceeds absolute limit
		ProofSystem:  "groth16",
		CircuitId:    "compute-verification-v2",
		PublicInputs: nil,
	}

	ok, err := zkVerifier.VerifyProof(
		sdkCtx,
		proof,
		1,
		make([]byte, sha256.Size),
		sdk.AccAddress([]byte("test_provider_addr_")),
	)
	require.False(t, ok)
	require.ErrorIs(t, err, types.ErrProofTooLarge)
}

func TestVerifyProofFailsWhenVerifyingKeyMissing(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	zkVerifier := NewZKVerifier(k)
	provider := sdk.AccAddress([]byte("test_provider_addr_"))

	// Ensure provider has balance to cover deposit
	balanceBefore := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")
	require.True(t, balanceBefore.Amount.IsPositive())

	resultHash := sha256.Sum256([]byte("result"))
	providerHash := sha256.Sum256(provider.Bytes())
	publicInputs := serializePublicInputs(1, resultHash[:], providerHash[:])

	proof := &types.ZKProof{
		Proof:        []byte{0x1, 0x2, 0x3},
		ProofSystem:  "groth16",
		CircuitId:    "compute-verification-v2",
		PublicInputs: publicInputs,
	}

	ok, err := zkVerifier.VerifyProof(
		sdkCtx,
		proof,
		1,
		resultHash[:],
		provider,
	)
	require.False(t, ok)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verifying key not found")

	balanceAfter := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")
	require.Equal(t, balanceBefore, balanceAfter, "deposit should be refunded on missing verifying key")
}
