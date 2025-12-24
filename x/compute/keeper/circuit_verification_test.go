package keeper

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/circuits"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestComputeCircuitParamsHash tests hash computation for circuit parameters
func TestComputeCircuitParamsHash(t *testing.T) {
	params := types.CircuitParams{
		CircuitId:   "test-circuit-v1",
		Description: "Test circuit for verification",
		VerifyingKey: types.VerifyingKey{
			VkData:           []byte("test-verifying-key-data"),
			CircuitId:        "test-circuit-v1",
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        time.Now().UTC(),
			PublicInputCount: 3,
		},
		MaxProofSize: 2048,
		GasCost:      500000,
		Enabled:      true,
	}

	hash1, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.Len(t, hash1, sha256.Size)

	// Hash should be deterministic - same params produce same hash
	hash2, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)

	// Different VK data should produce different hash
	params.VerifyingKey.VkData = []byte("different-key-data")
	hash3, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash3)

	// Different circuit ID should produce different hash
	params.VerifyingKey.VkData = []byte("test-verifying-key-data")
	params.CircuitId = "different-circuit-v1"
	hash4, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash4)

	// Different max proof size should produce different hash
	params.CircuitId = "test-circuit-v1"
	params.MaxProofSize = 4096
	hash5, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash5)

	// Different gas cost should produce different hash
	params.MaxProofSize = 2048
	params.GasCost = 1000000
	hash6, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash6)

	// Different public input count should produce different hash
	params.GasCost = 500000
	params.VerifyingKey.PublicInputCount = 5
	hash7, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash7)

	// Different curve should produce different hash
	params.VerifyingKey.PublicInputCount = 3
	params.VerifyingKey.Curve = "bls12-381"
	hash8, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash8)

	// Different proof system should produce different hash
	params.VerifyingKey.Curve = "bn254"
	params.VerifyingKey.ProofSystem = "plonk"
	hash9, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash9)
}

// TestVerifyCircuitParamsHash tests hash verification
func TestVerifyCircuitParamsHash(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	circuitID := "test-circuit-v1"
	params := types.CircuitParams{
		CircuitId:   circuitID,
		Description: "Test circuit",
		VerifyingKey: types.VerifyingKey{
			VkData:           []byte("test-key-data"),
			CircuitId:        circuitID,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        time.Now().UTC(),
			PublicInputCount: 3,
		},
		MaxProofSize: 2048,
		GasCost:      500000,
	}

	// Test 1: No expected hash configured - should pass with warning
	valid, err := k.VerifyCircuitParamsHash(ctx, circuitID, params)
	require.NoError(t, err)
	require.True(t, valid, "should pass when no expected hash is configured")

	// Test 2: Set expected hash and verify matching params
	expectedHash, err := ComputeCircuitParamsHash(params)
	require.NoError(t, err)

	err = k.SetCircuitParamHash(ctx, circuitID, expectedHash)
	require.NoError(t, err)

	valid, err = k.VerifyCircuitParamsHash(ctx, circuitID, params)
	require.NoError(t, err)
	require.True(t, valid, "should pass when hash matches")

	// Test 3: Tampered params should fail verification
	tamperedParams := params
	tamperedParams.VerifyingKey.VkData = []byte("tampered-key-data")

	valid, err = k.VerifyCircuitParamsHash(ctx, circuitID, tamperedParams)
	require.NoError(t, err)
	require.False(t, valid, "should fail when params are tampered")

	// Test 4: Different circuit ID
	tamperedParams = params
	tamperedParams.CircuitId = "different-circuit-v1"

	valid, err = k.VerifyCircuitParamsHash(ctx, circuitID, tamperedParams)
	require.NoError(t, err)
	require.False(t, valid, "should fail when circuit ID is different")
}

// TestSetCircuitParamHash tests setting expected circuit hashes
func TestSetCircuitParamHash(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	circuitID := "test-circuit-v1"
	hash := sha256.Sum256([]byte("test-hash"))

	// Test setting hash
	err := k.SetCircuitParamHash(ctx, circuitID, hash[:])
	require.NoError(t, err)

	// Verify hash was stored
	retrievedHash, err := k.GetCircuitParamHash(ctx, circuitID)
	require.NoError(t, err)
	require.Equal(t, hash[:], retrievedHash)

	// Test invalid inputs
	err = k.SetCircuitParamHash(ctx, "", hash[:])
	require.Error(t, err, "should fail with empty circuit ID")

	err = k.SetCircuitParamHash(ctx, circuitID, []byte("short"))
	require.Error(t, err, "should fail with invalid hash length")
}

// TestGetCircuitParamHash tests retrieving circuit hashes
func TestGetCircuitParamHash(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	circuitID := "test-circuit-v1"
	hash := sha256.Sum256([]byte("test-hash"))

	// Test getting non-existent hash
	_, err := k.GetCircuitParamHash(ctx, circuitID)
	require.Error(t, err, "should fail when hash doesn't exist")

	// Set hash and retrieve
	err = k.SetCircuitParamHash(ctx, circuitID, hash[:])
	require.NoError(t, err)

	retrievedHash, err := k.GetCircuitParamHash(ctx, circuitID)
	require.NoError(t, err)
	require.Equal(t, hash[:], retrievedHash)
}

// TestListCircuitHashes tests listing all circuit hashes
func TestListCircuitHashes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	// Initially empty
	hashes, err := k.ListCircuitHashes(ctx)
	require.NoError(t, err)
	require.Empty(t, hashes)

	// Add multiple hashes
	circuit1 := "circuit-1"
	hash1 := sha256.Sum256([]byte("hash-1"))
	err = k.SetCircuitParamHash(ctx, circuit1, hash1[:])
	require.NoError(t, err)

	circuit2 := "circuit-2"
	hash2 := sha256.Sum256([]byte("hash-2"))
	err = k.SetCircuitParamHash(ctx, circuit2, hash2[:])
	require.NoError(t, err)

	// List all hashes
	hashes, err = k.ListCircuitHashes(ctx)
	require.NoError(t, err)
	require.Len(t, hashes, 2)
	require.Equal(t, hash1[:], hashes[circuit1])
	require.Equal(t, hash2[:], hashes[circuit2])

	// Verify returned map is a copy (modifications don't affect storage)
	hashes[circuit1] = []byte("modified")
	retrievedHash, err := k.GetCircuitParamHash(ctx, circuit1)
	require.NoError(t, err)
	require.Equal(t, hash1[:], retrievedHash)
}

// TestValidateCircuitParams tests circuit parameter validation
func TestValidateCircuitParams(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	validParams := types.CircuitParams{
		CircuitId:   "test-circuit-v1",
		Description: "Test circuit",
		VerifyingKey: types.VerifyingKey{
			VkData:           []byte("test-key-data"),
			CircuitId:        "test-circuit-v1",
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        time.Now().UTC(),
			PublicInputCount: 3,
		},
		MaxProofSize: 2048,
		GasCost:      500000,
	}

	// Set expected hash
	hash, err := ComputeCircuitParamsHash(validParams)
	require.NoError(t, err)
	err = k.SetCircuitParamHash(ctx, validParams.CircuitId, hash)
	require.NoError(t, err)

	// Test valid params
	err = k.ValidateCircuitParams(ctx, validParams)
	require.NoError(t, err)

	// Test empty circuit ID
	invalidParams := validParams
	invalidParams.CircuitId = ""
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "circuit_id cannot be empty")

	// Test empty description
	invalidParams = validParams
	invalidParams.Description = ""
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "description cannot be empty")

	// Test empty verifying key
	invalidParams = validParams
	invalidParams.VerifyingKey.VkData = []byte{}
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verifying key data cannot be empty")

	// Test empty curve
	invalidParams = validParams
	invalidParams.VerifyingKey.Curve = ""
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "curve cannot be empty")

	// Test empty proof system
	invalidParams = validParams
	invalidParams.VerifyingKey.ProofSystem = ""
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "proof_system cannot be empty")

	// Test zero max proof size
	invalidParams = validParams
	invalidParams.MaxProofSize = 0
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "max_proof_size must be positive")

	// Test zero gas cost
	invalidParams = validParams
	invalidParams.GasCost = 0
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "gas_cost must be positive")

	// Test zero public input count
	invalidParams = validParams
	invalidParams.VerifyingKey.PublicInputCount = 0
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public_input_count must be positive")

	// Test hash mismatch
	invalidParams = validParams
	invalidParams.VerifyingKey.VkData = []byte("tampered-data")
	err = k.ValidateCircuitParams(ctx, invalidParams)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}

// TestVerifyAllCircuitHashes tests bulk verification of all circuits
func TestVerifyAllCircuitHashes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	// Create and store circuit params
	circuit1 := "circuit-1"
	params1 := types.CircuitParams{
		CircuitId:   circuit1,
		Description: "Circuit 1",
		VerifyingKey: types.VerifyingKey{
			VkData:           []byte("key-1"),
			CircuitId:        circuit1,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        time.Now().UTC(),
			PublicInputCount: 3,
		},
		MaxProofSize: 2048,
		GasCost:      500000,
	}

	hash1, err := ComputeCircuitParamsHash(params1)
	require.NoError(t, err)
	err = k.SetCircuitParamHash(ctx, circuit1, hash1)
	require.NoError(t, err)
	err = k.SetCircuitParams(ctx, params1)
	require.NoError(t, err)

	circuit2 := "circuit-2"
	params2 := types.CircuitParams{
		CircuitId:   circuit2,
		Description: "Circuit 2",
		VerifyingKey: types.VerifyingKey{
			VkData:           []byte("key-2"),
			CircuitId:        circuit2,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        time.Now().UTC(),
			PublicInputCount: 3,
		},
		MaxProofSize: 2048,
		GasCost:      500000,
	}

	hash2, err := ComputeCircuitParamsHash(params2)
	require.NoError(t, err)
	err = k.SetCircuitParamHash(ctx, circuit2, hash2)
	require.NoError(t, err)
	err = k.SetCircuitParams(ctx, params2)
	require.NoError(t, err)

	// Verify all circuits
	results, err := k.VerifyAllCircuitHashes(ctx)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.True(t, results[circuit1], "circuit-1 should be valid")
	require.True(t, results[circuit2], "circuit-2 should be valid")

	// Tamper with one circuit's params
	tamperedParams := params1
	tamperedParams.VerifyingKey.VkData = []byte("tampered-key")
	err = k.SetCircuitParams(ctx, tamperedParams)
	require.NoError(t, err)

	// Verify all again - circuit-1 should now fail
	results, err = k.VerifyAllCircuitHashes(ctx)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.False(t, results[circuit1], "circuit-1 should be invalid after tampering")
	require.True(t, results[circuit2], "circuit-2 should still be valid")
}

// TestCircuitHashIntegrationWithCircuitManager tests integration with CircuitManager
func TestCircuitHashIntegrationWithCircuitManager(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	cm := NewCircuitManager(k)

	// Mock groth16 setup to avoid expensive crypto operations
	originalSetup := groth16Setup
	defer func() { groth16Setup = originalSetup }()

	groth16Setup = func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
		// Return mock keys
		return groth16.NewProvingKey(ecc.BN254), groth16.NewVerifyingKey(ecc.BN254), nil
	}

	// Initialize circuit - should automatically set hash
	err := cm.initializeComputeCircuit(ctx)
	require.NoError(t, err)

	// Verify hash was set
	circuitID := (&circuits.ComputeCircuit{}).GetCircuitName()
	_, err = k.GetCircuitParamHash(ctx, circuitID)
	require.NoError(t, err, "circuit param hash should be set after initialization")

	// Verify params
	params, err := k.GetCircuitParams(ctx, circuitID)
	require.NoError(t, err)

	valid, err := k.VerifyCircuitParamsHash(ctx, circuitID, *params)
	require.NoError(t, err)
	require.True(t, valid, "circuit params should pass hash verification")
}
