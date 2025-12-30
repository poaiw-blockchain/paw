package types

import (
	"bytes"
	"encoding/binary"
	"testing"

	"cosmossdk.io/math"
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

// ============================================================================
// State Types Tests - ComputeProvider, ComputeRequest, Resource, ComputeResult
// ============================================================================

func TestComputeProvider(t *testing.T) {
	provider := ComputeProvider{
		Address:       "cosmos1test",
		Stake:         math.NewInt(1000000),
		Active:        true,
		Reputation:    95.5,
		TotalJobs:     100,
		CompletedJobs: 95,
		FailedJobs:    5,
		LastActive:    1234567890,
	}

	// Test field access
	if provider.Address != "cosmos1test" {
		t.Errorf("ComputeProvider.Address = %v, want 'cosmos1test'", provider.Address)
	}

	if !provider.Stake.Equal(math.NewInt(1000000)) {
		t.Errorf("ComputeProvider.Stake = %v, want 1000000", provider.Stake)
	}

	if !provider.Active {
		t.Error("ComputeProvider.Active = false, want true")
	}

	if provider.Reputation != 95.5 {
		t.Errorf("ComputeProvider.Reputation = %v, want 95.5", provider.Reputation)
	}

	if provider.TotalJobs != 100 {
		t.Errorf("ComputeProvider.TotalJobs = %v, want 100", provider.TotalJobs)
	}

	if provider.CompletedJobs != 95 {
		t.Errorf("ComputeProvider.CompletedJobs = %v, want 95", provider.CompletedJobs)
	}

	if provider.FailedJobs != 5 {
		t.Errorf("ComputeProvider.FailedJobs = %v, want 5", provider.FailedJobs)
	}

	if provider.LastActive != 1234567890 {
		t.Errorf("ComputeProvider.LastActive = %v, want 1234567890", provider.LastActive)
	}
}

func TestComputeRequest(t *testing.T) {
	request := ComputeRequest{
		ID:             1,
		Requester:      "cosmos1requester",
		Provider:       "cosmos1provider",
		ContainerImage: "docker.io/library/ubuntu:latest",
		InputData:      []byte("input data"),
		ResourceSpec: Resource{
			CPUCores:  4,
			MemoryMB:  8192,
			StorageGB: 100,
			GPUs:      1,
		},
		EscrowAmount:    math.NewInt(1000000),
		Status:          "pending",
		SubmittedHeight: 100,
		Timeout:         3600,
	}

	// Test basic fields
	if request.ID != 1 {
		t.Errorf("ComputeRequest.ID = %v, want 1", request.ID)
	}

	if request.Requester != "cosmos1requester" {
		t.Errorf("ComputeRequest.Requester = %v, want 'cosmos1requester'", request.Requester)
	}

	if request.Provider != "cosmos1provider" {
		t.Errorf("ComputeRequest.Provider = %v, want 'cosmos1provider'", request.Provider)
	}

	if request.ContainerImage != "docker.io/library/ubuntu:latest" {
		t.Errorf("ComputeRequest.ContainerImage = %v, want 'docker.io/library/ubuntu:latest'", request.ContainerImage)
	}

	if !bytes.Equal(request.InputData, []byte("input data")) {
		t.Errorf("ComputeRequest.InputData = %v, want 'input data'", request.InputData)
	}

	if request.Status != "pending" {
		t.Errorf("ComputeRequest.Status = %v, want 'pending'", request.Status)
	}

	if request.SubmittedHeight != 100 {
		t.Errorf("ComputeRequest.SubmittedHeight = %v, want 100", request.SubmittedHeight)
	}

	if request.Timeout != 3600 {
		t.Errorf("ComputeRequest.Timeout = %v, want 3600", request.Timeout)
	}
}

func TestResource(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
	}{
		{
			name: "minimal resource",
			resource: Resource{
				CPUCores:  1,
				MemoryMB:  512,
				StorageGB: 10,
				GPUs:      0,
			},
		},
		{
			name: "full resource with GPU",
			resource: Resource{
				CPUCores:  32,
				MemoryMB:  65536,
				StorageGB: 1000,
				GPUs:      4,
			},
		},
		{
			name: "zero resource",
			resource: Resource{
				CPUCores:  0,
				MemoryMB:  0,
				StorageGB: 0,
				GPUs:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify fields are accessible
			_ = tt.resource.CPUCores
			_ = tt.resource.MemoryMB
			_ = tt.resource.StorageGB
			_ = tt.resource.GPUs
		})
	}
}

func TestComputeResult(t *testing.T) {
	result := ComputeResult{
		RequestID:   1,
		Provider:    "cosmos1provider",
		ResultData:  []byte("result data"),
		ResultHash:  "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
		Verified:    true,
		SubmittedAt: 1234567890,
		VerifiedAt:  1234567900,
	}

	if result.RequestID != 1 {
		t.Errorf("ComputeResult.RequestID = %v, want 1", result.RequestID)
	}

	if result.Provider != "cosmos1provider" {
		t.Errorf("ComputeResult.Provider = %v, want 'cosmos1provider'", result.Provider)
	}

	if !bytes.Equal(result.ResultData, []byte("result data")) {
		t.Errorf("ComputeResult.ResultData = %v, want 'result data'", result.ResultData)
	}

	if result.ResultHash != "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3" {
		t.Errorf("ComputeResult.ResultHash = %v, want expected hash", result.ResultHash)
	}

	if !result.Verified {
		t.Error("ComputeResult.Verified = false, want true")
	}

	if result.SubmittedAt != 1234567890 {
		t.Errorf("ComputeResult.SubmittedAt = %v, want 1234567890", result.SubmittedAt)
	}

	if result.VerifiedAt != 1234567900 {
		t.Errorf("ComputeResult.VerifiedAt = %v, want 1234567900", result.VerifiedAt)
	}
}

func TestComputeResultUnverified(t *testing.T) {
	result := ComputeResult{
		RequestID:   2,
		Provider:    "cosmos1provider2",
		ResultData:  []byte("pending result"),
		ResultHash:  "b665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae4",
		Verified:    false,
		SubmittedAt: 1234567890,
		VerifiedAt:  0, // Not verified yet
	}

	if result.Verified {
		t.Error("ComputeResult.Verified = true, want false for unverified result")
	}

	if result.VerifiedAt != 0 {
		t.Errorf("ComputeResult.VerifiedAt = %v, want 0 for unverified result", result.VerifiedAt)
	}
}

// ============================================================================
// NonceTracker Tests
// ============================================================================

func TestNonceTracker(t *testing.T) {
	tracker := NonceTracker{
		Provider: "cosmos1provider",
		Nonce:    12345,
		UsedAt:   1234567890,
	}

	if tracker.Provider != "cosmos1provider" {
		t.Errorf("NonceTracker.Provider = %v, want 'cosmos1provider'", tracker.Provider)
	}

	if tracker.Nonce != 12345 {
		t.Errorf("NonceTracker.Nonce = %v, want 12345", tracker.Nonce)
	}

	if tracker.UsedAt != 1234567890 {
		t.Errorf("NonceTracker.UsedAt = %v, want 1234567890", tracker.UsedAt)
	}
}

// ============================================================================
// VerificationMetrics Tests
// ============================================================================

func TestVerificationMetrics(t *testing.T) {
	metrics := VerificationMetrics{
		TotalVerifications:      100,
		SuccessfulVerifications: 90,
		FailedVerifications:     10,
		AverageScore:            85.5,
		SignatureFailures:       3,
		MerkleFailures:          2,
		StateTransitionFailures: 4,
		ReplayAttempts:          1,
	}

	if metrics.TotalVerifications != 100 {
		t.Errorf("VerificationMetrics.TotalVerifications = %v, want 100", metrics.TotalVerifications)
	}

	if metrics.SuccessfulVerifications != 90 {
		t.Errorf("VerificationMetrics.SuccessfulVerifications = %v, want 90", metrics.SuccessfulVerifications)
	}

	if metrics.FailedVerifications != 10 {
		t.Errorf("VerificationMetrics.FailedVerifications = %v, want 10", metrics.FailedVerifications)
	}

	if metrics.AverageScore != 85.5 {
		t.Errorf("VerificationMetrics.AverageScore = %v, want 85.5", metrics.AverageScore)
	}

	if metrics.SignatureFailures != 3 {
		t.Errorf("VerificationMetrics.SignatureFailures = %v, want 3", metrics.SignatureFailures)
	}

	if metrics.MerkleFailures != 2 {
		t.Errorf("VerificationMetrics.MerkleFailures = %v, want 2", metrics.MerkleFailures)
	}

	if metrics.StateTransitionFailures != 4 {
		t.Errorf("VerificationMetrics.StateTransitionFailures = %v, want 4", metrics.StateTransitionFailures)
	}

	if metrics.ReplayAttempts != 1 {
		t.Errorf("VerificationMetrics.ReplayAttempts = %v, want 1", metrics.ReplayAttempts)
	}
}

func TestVerificationMetricsZero(t *testing.T) {
	metrics := VerificationMetrics{}

	if metrics.TotalVerifications != 0 {
		t.Errorf("VerificationMetrics.TotalVerifications = %v, want 0", metrics.TotalVerifications)
	}

	if metrics.AverageScore != 0 {
		t.Errorf("VerificationMetrics.AverageScore = %v, want 0", metrics.AverageScore)
	}
}

// ============================================================================
// Request Status Transitions Tests
// ============================================================================

func TestComputeRequestStatusTransitions(t *testing.T) {
	validStatuses := []string{
		"pending",
		"assigned",
		"running",
		"completed",
		"failed",
		"cancelled",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			request := ComputeRequest{
				ID:     1,
				Status: status,
			}

			if request.Status != status {
				t.Errorf("ComputeRequest.Status = %v, want %v", request.Status, status)
			}
		})
	}
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

func TestComputeProviderEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		provider ComputeProvider
	}{
		{
			name: "zero stake",
			provider: ComputeProvider{
				Address:    "cosmos1test",
				Stake:      math.NewInt(0),
				Active:     false,
				Reputation: 0,
			},
		},
		{
			name: "max stake",
			provider: ComputeProvider{
				Address:    "cosmos1test",
				Stake:      math.NewInt(1000000000000000),
				Active:     true,
				Reputation: 100,
			},
		},
		{
			name: "inactive provider with jobs",
			provider: ComputeProvider{
				Address:       "cosmos1test",
				Stake:         math.NewInt(1000000),
				Active:        false,
				Reputation:    50,
				TotalJobs:     1000,
				CompletedJobs: 500,
				FailedJobs:    500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify we can access all fields without panic
			_ = tt.provider.Address
			_ = tt.provider.Stake
			_ = tt.provider.Active
			_ = tt.provider.Reputation
			_ = tt.provider.TotalJobs
			_ = tt.provider.CompletedJobs
			_ = tt.provider.FailedJobs
			_ = tt.provider.LastActive
		})
	}
}

func TestComputeRequestWithEmptyProvider(t *testing.T) {
	// Test request without assigned provider
	request := ComputeRequest{
		ID:             1,
		Requester:      "cosmos1requester",
		Provider:       "", // No provider assigned yet
		ContainerImage: "docker.io/library/ubuntu:latest",
		Status:         "pending",
	}

	if request.Provider != "" {
		t.Errorf("ComputeRequest.Provider = %v, want empty string", request.Provider)
	}
}

func TestComputeRequestWithNilInputData(t *testing.T) {
	request := ComputeRequest{
		ID:             1,
		Requester:      "cosmos1requester",
		ContainerImage: "docker.io/library/ubuntu:latest",
		InputData:      nil, // No input data
		Status:         "pending",
	}

	if request.InputData != nil {
		t.Error("ComputeRequest.InputData should be nil")
	}
}

func TestVerificationProofNilFields(t *testing.T) {
	// Test that nil fields cause validation errors
	vp := &VerificationProof{
		Signature:       nil,
		PublicKey:       nil,
		MerkleRoot:      nil,
		MerkleProof:     nil,
		StateCommitment: nil,
		ExecutionTrace:  nil,
		Nonce:           0,
		Timestamp:       0,
	}

	err := vp.Validate()
	if err == nil {
		t.Error("VerificationProof.Validate() should return error for nil fields")
	}
}
