package keeper_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestVerificationProofValidation tests the structural validation of verification proofs
func TestVerificationProofValidation(t *testing.T) {
	tests := []struct {
		name      string
		proof     *types.VerificationProof
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid proof",
			proof: &types.VerificationProof{
				Signature:       make([]byte, 64),
				PublicKey:       make([]byte, 32),
				MerkleRoot:      make([]byte, 32),
				MerkleProof:     [][]byte{make([]byte, 32)},
				StateCommitment: make([]byte, 32),
				ExecutionTrace:  make([]byte, 32),
				Nonce:           1,
				Timestamp:       time.Now().Unix(),
			},
			shouldErr: false,
		},
		{
			name: "invalid signature length",
			proof: &types.VerificationProof{
				Signature:       make([]byte, 32), // Should be 64
				PublicKey:       make([]byte, 32),
				MerkleRoot:      make([]byte, 32),
				MerkleProof:     [][]byte{make([]byte, 32)},
				StateCommitment: make([]byte, 32),
				ExecutionTrace:  make([]byte, 32),
				Nonce:           1,
				Timestamp:       time.Now().Unix(),
			},
			shouldErr: true,
			errMsg:    "invalid signature length",
		},
		{
			name: "invalid public key length",
			proof: &types.VerificationProof{
				Signature:       make([]byte, 64),
				PublicKey:       make([]byte, 16), // Should be 32
				MerkleRoot:      make([]byte, 32),
				MerkleProof:     [][]byte{make([]byte, 32)},
				StateCommitment: make([]byte, 32),
				ExecutionTrace:  make([]byte, 32),
				Nonce:           1,
				Timestamp:       time.Now().Unix(),
			},
			shouldErr: true,
			errMsg:    "invalid public key length",
		},
		{
			name: "zero nonce",
			proof: &types.VerificationProof{
				Signature:       make([]byte, 64),
				PublicKey:       make([]byte, 32),
				MerkleRoot:      make([]byte, 32),
				MerkleProof:     [][]byte{make([]byte, 32)},
				StateCommitment: make([]byte, 32),
				ExecutionTrace:  make([]byte, 32),
				Nonce:           0, // Should be non-zero
				Timestamp:       time.Now().Unix(),
			},
			shouldErr: true,
			errMsg:    "nonce must be non-zero",
		},
		{
			name: "empty merkle proof",
			proof: &types.VerificationProof{
				Signature:       make([]byte, 64),
				PublicKey:       make([]byte, 32),
				MerkleRoot:      make([]byte, 32),
				MerkleProof:     [][]byte{}, // Should have at least one node
				StateCommitment: make([]byte, 32),
				ExecutionTrace:  make([]byte, 32),
				Nonce:           1,
				Timestamp:       time.Now().Unix(),
			},
			shouldErr: true,
			errMsg:    "merkle proof path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proof.Validate()
			if tt.shouldErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestComputeMessageHash tests the canonical message hash computation
func TestComputeMessageHash(t *testing.T) {
	proof := &types.VerificationProof{
		MerkleRoot:      make([]byte, 32),
		StateCommitment: make([]byte, 32),
		Nonce:           12345,
		Timestamp:       1234567890,
	}

	// Fill with deterministic data
	for i := range proof.MerkleRoot {
		proof.MerkleRoot[i] = byte(i)
		proof.StateCommitment[i] = byte(i * 2)
	}

	requestID := uint64(100)
	resultHash := "abcdef1234567890"

	hash1 := proof.ComputeMessageHash(requestID, resultHash)
	hash2 := proof.ComputeMessageHash(requestID, resultHash)

	// Same inputs should produce same hash
	require.Equal(t, hash1, hash2)
	require.Len(t, hash1, 32) // SHA-256 output

	// Different request ID should produce different hash
	hash3 := proof.ComputeMessageHash(requestID+1, resultHash)
	require.NotEqual(t, hash1, hash3)

	// Different result hash should produce different hash
	hash4 := proof.ComputeMessageHash(requestID, "different")
	require.NotEqual(t, hash1, hash4)
}

// TestEd25519SignatureVerification tests Ed25519 signature verification
func TestEd25519SignatureVerification(t *testing.T) {
	// Generate a real Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	requestID := uint64(100)
	resultHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	proof := &types.VerificationProof{
		PublicKey:       publicKey,
		MerkleRoot:      make([]byte, 32),
		StateCommitment: make([]byte, 32),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	// Compute message hash
	message := proof.ComputeMessageHash(requestID, resultHash)

	// Sign the message
	signature := ed25519.Sign(privateKey, message)
	proof.Signature = signature

	require.Len(t, signature, 64)

	// Verify signature
	valid := ed25519.Verify(publicKey, message, signature)
	require.True(t, valid)

	// Invalid signature should fail
	invalidSig := make([]byte, 64)
	copy(invalidSig, signature)
	invalidSig[0] ^= 0xFF // Flip bits
	invalidValid := ed25519.Verify(publicKey, message, invalidSig)
	require.False(t, invalidValid)
}

// TestMerkleProofConstruction tests merkle proof construction and validation
func TestMerkleProofConstruction(t *testing.T) {
	// Create a simple 4-leaf merkle tree
	leaves := [][]byte{
		[]byte("execution step 1"),
		[]byte("execution step 2"),
		[]byte("execution step 3"),
		[]byte("execution step 4"),
	}

	// Hash all leaves
	leafHashes := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		hash := sha256.Sum256(leaf)
		leafHashes[i] = hash[:]
	}

	// Build level 1 (pairs of leaves)
	level1 := make([][]byte, 2)
	for i := 0; i < 2; i++ {
		hasher := sha256.New()
		hasher.Write(leafHashes[i*2])
		hasher.Write(leafHashes[i*2+1])
		level1[i] = hasher.Sum(nil)
	}

	// Build root
	rootHasher := sha256.New()
	rootHasher.Write(level1[0])
	rootHasher.Write(level1[1])
	root := rootHasher.Sum(nil)

	// Create proof for leaf 0
	proof := [][]byte{
		leafHashes[1], // Sibling at level 0
		level1[1],     // Sibling at level 1
	}

	// Verify the proof
	currentHash := leafHashes[0]
	for _, sibling := range proof {
		hasher := sha256.New()
		hasher.Write(currentHash)
		hasher.Write(sibling)
		currentHash = hasher.Sum(nil)
	}

	require.Equal(t, root, currentHash)
}

// TestProofSerialization tests binary serialization and deserialization
func TestProofSerialization(t *testing.T) {
	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	// Create proof
	originalProof := &types.VerificationProof{
		PublicKey:       publicKey,
		MerkleRoot:      make([]byte, 32),
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32), make([]byte, 32)},
		Nonce:           12345,
		Timestamp:       time.Now().Unix(),
	}

	// Fill with test data
	for i := range originalProof.MerkleRoot {
		originalProof.MerkleRoot[i] = byte(i)
		originalProof.StateCommitment[i] = byte(i * 2)
		originalProof.ExecutionTrace[i] = byte(i * 3)
	}
	for j := range originalProof.MerkleProof {
		for i := range originalProof.MerkleProof[j] {
			originalProof.MerkleProof[j][i] = byte(i + j)
		}
	}

	// Compute signature
	message := originalProof.ComputeMessageHash(100, "test_hash")
	originalProof.Signature = ed25519.Sign(privateKey, message)

	// Serialize
	proofBytes := serializeProof(originalProof)
	require.GreaterOrEqual(t, len(proofBytes), 200)

	// Deserialize
	deserializedProof, err := deserializeProof(proofBytes)
	require.NoError(t, err)

	// Compare
	require.Equal(t, originalProof.Signature, deserializedProof.Signature)
	require.Equal(t, originalProof.PublicKey, deserializedProof.PublicKey)
	require.Equal(t, originalProof.MerkleRoot, deserializedProof.MerkleRoot)
	require.Equal(t, originalProof.StateCommitment, deserializedProof.StateCommitment)
	require.Equal(t, originalProof.ExecutionTrace, deserializedProof.ExecutionTrace)
	require.Equal(t, originalProof.Nonce, deserializedProof.Nonce)
	require.Equal(t, originalProof.Timestamp, deserializedProof.Timestamp)
	require.Equal(t, len(originalProof.MerkleProof), len(deserializedProof.MerkleProof))
}

// Helper function to serialize proof (matches keeper implementation)
func serializeProof(proof *types.VerificationProof) []byte {
	var buf []byte

	// Signature (64 bytes)
	buf = append(buf, proof.Signature...)

	// Public key (32 bytes)
	buf = append(buf, proof.PublicKey...)

	// Merkle root (32 bytes)
	buf = append(buf, proof.MerkleRoot...)

	// Merkle proof count (1 byte)
	buf = append(buf, byte(len(proof.MerkleProof)))

	// Merkle proof nodes
	for _, node := range proof.MerkleProof {
		buf = append(buf, node...)
	}

	// State commitment (32 bytes)
	buf = append(buf, proof.StateCommitment...)

	// Execution trace (32 bytes)
	buf = append(buf, proof.ExecutionTrace...)

	// Nonce (8 bytes)
	nonceBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBz, proof.Nonce)
	buf = append(buf, nonceBz...)

	// Timestamp (8 bytes)
	timestampBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBz, uint64(proof.Timestamp))
	buf = append(buf, timestampBz...)

	return buf
}

// Helper function to deserialize proof (test version of keeper implementation)
func deserializeProof(proofBytes []byte) (*types.VerificationProof, error) {
	if len(proofBytes) < 200 {
		return nil, fmt.Errorf("proof too short: minimum 200 bytes required, got %d", len(proofBytes))
	}

	proof := &types.VerificationProof{}
	offset := 0

	proof.Signature = make([]byte, 64)
	copy(proof.Signature, proofBytes[offset:offset+64])
	offset += 64

	proof.PublicKey = make([]byte, 32)
	copy(proof.PublicKey, proofBytes[offset:offset+32])
	offset += 32

	proof.MerkleRoot = make([]byte, 32)
	copy(proof.MerkleRoot, proofBytes[offset:offset+32])
	offset += 32

	merkleProofLen := int(proofBytes[offset])
	offset += 1

	proof.MerkleProof = make([][]byte, merkleProofLen)
	for i := 0; i < merkleProofLen; i++ {
		node := make([]byte, 32)
		copy(node, proofBytes[offset:offset+32])
		proof.MerkleProof[i] = node
		offset += 32
	}

	proof.StateCommitment = make([]byte, 32)
	copy(proof.StateCommitment, proofBytes[offset:offset+32])
	offset += 32

	proof.ExecutionTrace = make([]byte, 32)
	copy(proof.ExecutionTrace, proofBytes[offset:offset+32])
	offset += 32

	proof.Nonce = binary.BigEndian.Uint64(proofBytes[offset : offset+8])
	offset += 8

	proof.Timestamp = int64(binary.BigEndian.Uint64(proofBytes[offset : offset+8]))

	return proof, nil
}

// BenchmarkEd25519Verification benchmarks signature verification performance
func BenchmarkEd25519Verification(b *testing.B) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	message := make([]byte, 32)
	signature := ed25519.Sign(privateKey, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ed25519.Verify(publicKey, message, signature)
	}
}

// BenchmarkMerkleProofValidation benchmarks merkle proof validation
func BenchmarkMerkleProofValidation(b *testing.B) {
	proof := &types.VerificationProof{
		ExecutionTrace: make([]byte, 32),
		MerkleRoot:     make([]byte, 32),
		MerkleProof:    make([][]byte, 10), // 10-level tree
	}

	for i := range proof.MerkleProof {
		proof.MerkleProof[i] = make([]byte, 32)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leafHash := sha256.Sum256(proof.ExecutionTrace)
		currentHash := leafHash[:]

		for _, sibling := range proof.MerkleProof {
			hasher := sha256.New()
			hasher.Write(currentHash)
			hasher.Write(sibling)
			currentHash = hasher.Sum(nil)
		}
	}
}

// BenchmarkFullVerification benchmarks complete verification process
func BenchmarkFullVerification(b *testing.B) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	proof := &types.VerificationProof{
		PublicKey:       publicKey,
		MerkleRoot:      make([]byte, 32),
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  make([]byte, 32),
		MerkleProof:     make([][]byte, 8),
		Nonce:           1,
		Timestamp:       time.Now().Unix(),
	}

	for i := range proof.MerkleProof {
		proof.MerkleProof[i] = make([]byte, 32)
	}

	message := proof.ComputeMessageHash(100, "test_hash")
	proof.Signature = ed25519.Sign(privateKey, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Signature verification
		ed25519.Verify(publicKey, message, proof.Signature)

		// Merkle proof validation
		leafHash := sha256.Sum256(proof.ExecutionTrace)
		currentHash := leafHash[:]
		for _, sibling := range proof.MerkleProof {
			hasher := sha256.New()
			hasher.Write(currentHash)
			hasher.Write(sibling)
			currentHash = hasher.Sum(nil)
		}

		// State transition
		hasher := sha256.New()
		hasher.Write([]byte("container"))
		hasher.Write(proof.ExecutionTrace)
		_ = hasher.Sum(nil)
	}
}
