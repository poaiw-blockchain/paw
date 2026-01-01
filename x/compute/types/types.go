package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const (
	// ModuleName defines the module name
	ModuleName = "compute"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// PortID is the default port ID for compute IBC module
	PortID = "compute"

	// Verification scoring thresholds
	VerificationPassThreshold = 50
	MaxVerificationScore      = 100

	// IBC event types (kept here for IBC-specific events)
	EventTypeChannelOpen        = "compute_channel_open"
	EventTypeChannelOpenAck     = "compute_channel_open_ack"
	EventTypeChannelOpenConfirm = "compute_channel_open_confirm"
	EventTypeChannelClose       = "compute_channel_close"
	EventTypePacketReceive      = "compute_packet_receive"
	EventTypePacketAck          = "compute_packet_ack"
	EventTypePacketTimeout      = "compute_packet_timeout"

	// IBC event attribute keys
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyPortID                = "port_id"
	AttributeKeyCounterpartyPortID    = "counterparty_port_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeyPacketType            = "packet_type"
	AttributeKeySequence              = "sequence"
	AttributeKeyAckSuccess            = "ack_success"
	AttributeKeyPendingOperations     = "pending_operations"
)

// Hash algorithm version constants for VerificationProof.
// SEC-3.4: These match the HashAlgorithm enum in state.proto.
const (
	// HashAlgorithmSHA256 is the default hash algorithm (version 1).
	HashAlgorithmSHA256 uint32 = 1
	// HashAlgorithmSHA3_256 is reserved for future use (version 2).
	HashAlgorithmSHA3_256 uint32 = 2
	// HashAlgorithmBLAKE3 is reserved for future use (version 3).
	HashAlgorithmBLAKE3 uint32 = 3
)

// NOTE: VerificationProof struct is now generated from proto in state.pb.go
// The struct definition was moved to proto/paw/compute/v1/state.proto
// for proper serialization support. The methods below extend the generated type.

// Validate performs structural validation of the verification proof.
func (vp *VerificationProof) Validate() error {
	if len(vp.Signature) != 64 {
		return fmt.Errorf("invalid signature length: expected 64, got %d", len(vp.Signature))
	}

	if len(vp.PublicKey) != 32 {
		return fmt.Errorf("invalid public key length: expected 32, got %d", len(vp.PublicKey))
	}

	if len(vp.MerkleRoot) != 32 {
		return fmt.Errorf("invalid merkle root length: expected 32, got %d", len(vp.MerkleRoot))
	}

	if len(vp.StateCommitment) != 32 {
		return fmt.Errorf("invalid state commitment length: expected 32, got %d", len(vp.StateCommitment))
	}

	if len(vp.ExecutionTrace) == 0 {
		return fmt.Errorf("execution trace hash is required")
	}

	if len(vp.MerkleProof) == 0 {
		return fmt.Errorf("merkle proof path is required")
	}

	for i, node := range vp.MerkleProof {
		if len(node) != 32 {
			return fmt.Errorf("invalid merkle proof node %d: expected 32 bytes, got %d", i, len(node))
		}
	}

	if vp.Nonce == 0 {
		return fmt.Errorf("nonce must be non-zero for replay protection")
	}

	if vp.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	// SEC-3.4: Validate hash algorithm version.
	// Version 0 is treated as version 1 (SHA256) for backwards compatibility.
	// Currently only SHA256 (version 0 or 1) is supported.
	if vp.HashAlgorithmVersion > HashAlgorithmSHA256 {
		return fmt.Errorf("unsupported hash algorithm version: %d (only version 0 or 1 supported)", vp.HashAlgorithmVersion)
	}

	return nil
}

// ComputeMessageHash creates the canonical hash of the computation result for signature verification.
func (vp *VerificationProof) ComputeMessageHash(requestID uint64, resultHash string) []byte {
	hasher := sha256.New()

	// Include request ID
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)

	// Include result hash
	hasher.Write([]byte(resultHash))

	// Include merkle root
	hasher.Write(vp.MerkleRoot)

	// Include state commitment
	hasher.Write(vp.StateCommitment)

	// Include nonce for replay protection
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, vp.Nonce)
	hasher.Write(nonceBytes)

	// Include timestamp
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, SaturateInt64ToUint64(vp.Timestamp))
	hasher.Write(timestampBytes)

	return hasher.Sum(nil)
}

// NonceTracker tracks used nonces for replay attack prevention.
type NonceTracker struct {
	Provider string
	Nonce    uint64
	UsedAt   int64
}

// VerificationMetrics tracks verification statistics for monitoring and analysis.
type VerificationMetrics struct {
	TotalVerifications      uint64
	SuccessfulVerifications uint64
	FailedVerifications     uint64
	AverageScore            float64
	SignatureFailures       uint64
	MerkleFailures          uint64
	StateTransitionFailures uint64
	ReplayAttempts          uint64
}
