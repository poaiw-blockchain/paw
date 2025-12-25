package types

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestVerificationProof_Validate(t *testing.T) {
	validSignature := make([]byte, 64)
	validPublicKey := make([]byte, 32)
	validMerkleRoot := make([]byte, 32)
	validStateCommitment := make([]byte, 32)
	validExecutionTrace := []byte("execution_trace_hash")
	validMerkleProof := [][]byte{make([]byte, 32), make([]byte, 32)}

	tests := []struct {
		name    string
		vp      *VerificationProof
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid proof",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: false,
		},
		{
			name: "invalid signature length - too short",
			vp: &VerificationProof{
				Signature:       make([]byte, 63),
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid signature length",
		},
		{
			name: "invalid signature length - too long",
			vp: &VerificationProof{
				Signature:       make([]byte, 65),
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid signature length",
		},
		{
			name: "invalid public key length",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       make([]byte, 31),
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid public key length",
		},
		{
			name: "invalid merkle root length",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      make([]byte, 31),
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid merkle root length",
		},
		{
			name: "invalid state commitment length",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: make([]byte, 31),
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid state commitment length",
		},
		{
			name: "empty execution trace",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  []byte{},
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "execution trace hash is required",
		},
		{
			name: "empty merkle proof",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     [][]byte{},
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "merkle proof path is required",
		},
		{
			name: "invalid merkle proof node length",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     [][]byte{make([]byte, 31)},
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "invalid merkle proof node",
		},
		{
			name: "zero nonce",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           0,
				Timestamp:       1000,
			},
			wantErr: true,
			errMsg:  "nonce must be non-zero",
		},
		{
			name: "zero timestamp",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       0,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "negative timestamp",
			vp: &VerificationProof{
				Signature:       validSignature,
				PublicKey:       validPublicKey,
				MerkleRoot:      validMerkleRoot,
				MerkleProof:     validMerkleProof,
				StateCommitment: validStateCommitment,
				ExecutionTrace:  validExecutionTrace,
				Nonce:           1,
				Timestamp:       -1,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.vp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VerificationProof.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("VerificationProof.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestVerificationProof_ComputeMessageHash(t *testing.T) {
	vp := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      bytes.Repeat([]byte{0x01}, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: bytes.Repeat([]byte{0x02}, 32),
		ExecutionTrace:  []byte("test"),
		Nonce:           12345,
		Timestamp:       67890,
	}

	requestID := uint64(100)
	resultHash := "test_result_hash"

	hash := vp.ComputeMessageHash(requestID, resultHash)

	// Verify hash length (SHA-256 produces 32 bytes)
	if len(hash) != 32 {
		t.Errorf("ComputeMessageHash() returned hash of length %d, want 32", len(hash))
	}

	// Verify determinism - same inputs should produce same hash
	hash2 := vp.ComputeMessageHash(requestID, resultHash)
	if !bytes.Equal(hash, hash2) {
		t.Error("ComputeMessageHash() is not deterministic")
	}

	// Verify different inputs produce different hashes
	hash3 := vp.ComputeMessageHash(requestID+1, resultHash)
	if bytes.Equal(hash, hash3) {
		t.Error("ComputeMessageHash() produced same hash for different request IDs")
	}

	hash4 := vp.ComputeMessageHash(requestID, "different_hash")
	if bytes.Equal(hash, hash4) {
		t.Error("ComputeMessageHash() produced same hash for different result hashes")
	}
}

func TestVerificationProof_ComputeMessageHash_WithNegativeTimestamp(t *testing.T) {
	vp := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("test"),
		Nonce:           1,
		Timestamp:       -100,
	}

	hash := vp.ComputeMessageHash(1, "test")

	// Should not panic and should return a 32-byte hash
	if len(hash) != 32 {
		t.Errorf("ComputeMessageHash() with negative timestamp returned hash of length %d, want 32", len(hash))
	}
}

func TestModuleConstants(t *testing.T) {
	if ModuleName != "compute" {
		t.Errorf("ModuleName = %v, want 'compute'", ModuleName)
	}

	if StoreKey != ModuleName {
		t.Errorf("StoreKey = %v, want %v", StoreKey, ModuleName)
	}

	if PortID != "compute" {
		t.Errorf("PortID = %v, want 'compute'", PortID)
	}

	if VerificationPassThreshold != 50 {
		t.Errorf("VerificationPassThreshold = %v, want 50", VerificationPassThreshold)
	}

	if MaxVerificationScore != 100 {
		t.Errorf("MaxVerificationScore = %v, want 100", MaxVerificationScore)
	}
}

func TestIBCEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EventTypeChannelOpen", EventTypeChannelOpen, "compute_channel_open"},
		{"EventTypeChannelOpenAck", EventTypeChannelOpenAck, "compute_channel_open_ack"},
		{"EventTypeChannelOpenConfirm", EventTypeChannelOpenConfirm, "compute_channel_open_confirm"},
		{"EventTypeChannelClose", EventTypeChannelClose, "compute_channel_close"},
		{"EventTypePacketReceive", EventTypePacketReceive, "compute_packet_receive"},
		{"EventTypePacketAck", EventTypePacketAck, "compute_packet_ack"},
		{"EventTypePacketTimeout", EventTypePacketTimeout, "compute_packet_timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestIBCAttributeKeys(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"AttributeKeyChannelID", AttributeKeyChannelID, "channel_id"},
		{"AttributeKeyPortID", AttributeKeyPortID, "port_id"},
		{"AttributeKeyCounterpartyPortID", AttributeKeyCounterpartyPortID, "counterparty_port_id"},
		{"AttributeKeyCounterpartyChannelID", AttributeKeyCounterpartyChannelID, "counterparty_channel_id"},
		{"AttributeKeyPacketType", AttributeKeyPacketType, "packet_type"},
		{"AttributeKeySequence", AttributeKeySequence, "sequence"},
		{"AttributeKeyAckSuccess", AttributeKeyAckSuccess, "ack_success"},
		{"AttributeKeyPendingOperations", AttributeKeyPendingOperations, "pending_operations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestSaturateInt64ToUint64_InComputeMessageHash(t *testing.T) {
	// Test that negative timestamps are handled correctly in ComputeMessageHash
	vp := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("test"),
		Nonce:           1,
		Timestamp:       -12345,
	}

	// Should not panic
	hash := vp.ComputeMessageHash(1, "test")

	// Verify we got a valid hash
	if len(hash) != 32 {
		t.Errorf("Expected 32-byte hash, got %d bytes", len(hash))
	}

	// Verify the timestamp is saturated to 0 in the hash calculation
	// by comparing with a proof with timestamp 0
	vp2 := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("test"),
		Nonce:           1,
		Timestamp:       0,
	}

	// Note: timestamp 0 is invalid in Validate(), but we can still compute a hash
	hash2 := vp2.ComputeMessageHash(1, "test")

	// With negative timestamp saturated to 0, but since we're converting to uint64
	// the result should be the same as timestamp 0
	if !bytes.Equal(hash, hash2) {
		t.Error("Negative timestamp should be saturated to 0 in hash computation")
	}
}

func BenchmarkVerificationProof_ComputeMessageHash(b *testing.B) {
	vp := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      bytes.Repeat([]byte{0x01}, 32),
		MerkleProof:     [][]byte{make([]byte, 32)},
		StateCommitment: bytes.Repeat([]byte{0x02}, 32),
		ExecutionTrace:  []byte("test_execution_trace"),
		Nonce:           12345,
		Timestamp:       67890,
	}

	requestID := uint64(100)
	resultHash := "test_result_hash_string"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vp.ComputeMessageHash(requestID, resultHash)
	}
}

func BenchmarkVerificationProof_Validate(b *testing.B) {
	vp := &VerificationProof{
		Signature:       make([]byte, 64),
		PublicKey:       make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		MerkleProof:     [][]byte{make([]byte, 32), make([]byte, 32), make([]byte, 32)},
		StateCommitment: make([]byte, 32),
		ExecutionTrace:  []byte("execution_trace_hash"),
		Nonce:           1,
		Timestamp:       1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vp.Validate()
	}
}

func TestBigEndianEncoding(t *testing.T) {
	// Verify that we're using big-endian encoding consistently
	testVal := uint64(0x0102030405060708)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, testVal)

	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if !bytes.Equal(buf, expected) {
		t.Errorf("BigEndian encoding mismatch: got %v, want %v", buf, expected)
	}

	// Verify decoding
	decoded := binary.BigEndian.Uint64(buf)
	if decoded != testVal {
		t.Errorf("BigEndian decoding mismatch: got %v, want %v", decoded, testVal)
	}
}
