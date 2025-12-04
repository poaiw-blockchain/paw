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

	// Event types
	EventTypeChannelOpen        = "channel_open"
	EventTypeChannelOpenAck     = "channel_open_ack"
	EventTypeChannelOpenConfirm = "channel_open_confirm"
	EventTypeChannelClose       = "channel_close"
	EventTypePacketReceive      = "packet_receive"
	EventTypePacketAck          = "packet_ack"
	EventTypePacketTimeout      = "packet_timeout"

	// Event attribute keys
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyPortID                = "port_id"
	AttributeKeyCounterpartyPortID    = "counterparty_port_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeyPacketType            = "packet_type"
	AttributeKeySequence              = "sequence"
	AttributeKeyAckSuccess            = "ack_success"
	AttributeKeyPendingOperations     = "pending_operations"
)

// VerificationProof represents a cryptographic proof of correct computation execution.
// This structure enables multi-layer verification including signatures, merkle proofs,
// and state transition validation.
type VerificationProof struct {
	Signature       []byte   // Ed25519 signature over the result hash
	PublicKey       []byte   // Provider's public key for signature verification
	MerkleRoot      []byte   // Root hash of execution trace merkle tree
	MerkleProof     [][]byte // Merkle inclusion proof path
	StateCommitment []byte   // Commitment to final computation state
	ExecutionTrace  []byte   // Deterministic execution log hash
	Nonce           uint64   // Replay attack prevention nonce
	Timestamp       int64    // Proof generation timestamp
}

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
	binary.BigEndian.PutUint64(timestampBytes, uint64(vp.Timestamp))
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
