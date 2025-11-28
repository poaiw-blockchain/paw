package types

import (
	"fmt"
	"time"
)

// ZKProof represents a zero-knowledge proof for computation verification.
type ZKProof struct {
	Proof        []byte    // Serialized Groth16 proof
	PublicInputs []byte    // Serialized public inputs
	ProofSystem  string    // "groth16"
	CircuitId    string    // Circuit identifier (e.g., "compute-verification-v1")
	GeneratedAt  time.Time // Proof generation timestamp
}

// Validate performs structural validation of the ZK proof.
func (zk *ZKProof) Validate() error {
	if len(zk.Proof) == 0 {
		return fmt.Errorf("proof data is required")
	}

	if len(zk.Proof) > 512 {
		return fmt.Errorf("proof size %d exceeds maximum 512 bytes", len(zk.Proof))
	}

	if len(zk.PublicInputs) == 0 {
		return fmt.Errorf("public inputs are required")
	}

	if zk.ProofSystem != "groth16" {
		return fmt.Errorf("unsupported proof system: %s", zk.ProofSystem)
	}

	if zk.CircuitId == "" {
		return fmt.Errorf("circuit ID is required")
	}

	if zk.GeneratedAt.IsZero() {
		return fmt.Errorf("generation timestamp is required")
	}

	return nil
}

// CircuitParams defines parameters for a ZK circuit.
type CircuitParams struct {
	CircuitId     string        // Unique circuit identifier
	Description   string        // Human-readable description
	VerifyingKey  VerifyingKey  // Verifying key for this circuit
	MaxProofSize  uint32        // Maximum allowed proof size in bytes
	GasCost       uint64        // Gas cost for verification
	Enabled       bool          // Whether this circuit is active
}

// VerifyingKey contains the verification key for a circuit.
type VerifyingKey struct {
	CircuitId        string    // Associated circuit ID
	VkData           []byte    // Serialized verifying key
	Curve            string    // Elliptic curve ("bn254")
	ProofSystem      string    // Proof system ("groth16")
	PublicInputCount uint32    // Number of public inputs
	CreatedAt        time.Time // Key creation time
}

// ZKMetrics tracks statistics about ZK proof generation and verification.
type ZKMetrics struct {
	TotalProofsGenerated       uint64    // Total proofs generated
	TotalProofsVerified        uint64    // Total proofs verified successfully
	TotalProofsFailed          uint64    // Total verification failures
	AverageVerificationTimeMs  uint64    // Average verification time in ms
	TotalGasConsumed           uint64    // Total gas consumed for verification
	LastUpdated                time.Time // Last metric update time
}

// ProofBatch represents a batch of proofs for aggregated verification.
type ProofBatch struct {
	BatchId          string      // Unique batch identifier
	Proofs           []*ZKProof  // Individual proofs in batch
	AggregatedProof  []byte      // Aggregated proof (if supported)
	CircuitId        string      // All proofs must use same circuit
	CreatedAt        time.Time   // Batch creation time
}

// Validate validates a proof batch.
func (pb *ProofBatch) Validate() error {
	if pb.BatchId == "" {
		return fmt.Errorf("batch ID is required")
	}

	if len(pb.Proofs) == 0 {
		return fmt.Errorf("batch must contain at least one proof")
	}

	if len(pb.Proofs) > 100 {
		return fmt.Errorf("batch size %d exceeds maximum 100", len(pb.Proofs))
	}

	// Verify all proofs use the same circuit
	for i, proof := range pb.Proofs {
		if err := proof.Validate(); err != nil {
			return fmt.Errorf("proof %d invalid: %w", i, err)
		}

		if proof.CircuitId != pb.CircuitId {
			return fmt.Errorf("proof %d circuit mismatch: expected %s, got %s",
				i, pb.CircuitId, proof.CircuitId)
		}
	}

	return nil
}

// ProofCache represents a cache entry for proof verification results.
type ProofCache struct {
	ProofHash        []byte    // SHA-256 hash of proof
	RequestID        uint64    // Associated request ID
	Verified         bool      // Verification result
	VerifiedAt       time.Time // When verification occurred
	CachedUntil      time.Time // Cache expiration time
}

// CircuitConstraintCount tracks constraint counts for circuits.
type CircuitConstraintCount struct {
	CircuitId        string    // Circuit identifier
	R1CSConstraints  uint64    // Number of R1CS constraints
	WitnessSize      uint64    // Size of witness data
	PublicInputSize  uint64    // Size of public inputs
	ProvingTimeMs    uint64    // Average proving time in ms
	VerifyingTimeMs  uint64    // Average verifying time in ms
	ProofSizeBytes   uint32    // Average proof size in bytes
}

// TrustedSetupParams contains parameters from MPC ceremony.
type TrustedSetupParams struct {
	CeremonyId       string    // MPC ceremony identifier
	CircuitId        string    // Associated circuit
	ParticipantCount uint32    // Number of ceremony participants
	ProvingKey       []byte    // Serialized proving key
	VerifyingKey     []byte    // Serialized verifying key
	TranscriptHash   []byte    // Hash of ceremony transcript
	CompletedAt      time.Time // Ceremony completion time
	Secure           bool      // Whether ceremony was secure (>= 1 honest participant)
}

// Validate validates trusted setup parameters.
func (tsp *TrustedSetupParams) Validate() error {
	if tsp.CeremonyId == "" {
		return fmt.Errorf("ceremony ID is required")
	}

	if tsp.CircuitId == "" {
		return fmt.Errorf("circuit ID is required")
	}

	if tsp.ParticipantCount == 0 {
		return fmt.Errorf("at least one participant required")
	}

	if len(tsp.ProvingKey) == 0 {
		return fmt.Errorf("proving key is required")
	}

	if len(tsp.VerifyingKey) == 0 {
		return fmt.Errorf("verifying key is required")
	}

	if len(tsp.TranscriptHash) != 32 {
		return fmt.Errorf("invalid transcript hash length: expected 32, got %d",
			len(tsp.TranscriptHash))
	}

	if tsp.CompletedAt.IsZero() {
		return fmt.Errorf("completion timestamp is required")
	}

	return nil
}

// RecursiveProof represents a proof that verifies other proofs (proof composition).
type RecursiveProof struct {
	Proof            []byte      // The recursive proof
	VerifiedProofs   [][]byte    // Hashes of proofs being verified
	DepthLevel       uint32      // Recursion depth
	CircuitId        string      // Recursive circuit ID
	GeneratedAt      time.Time   // Generation timestamp
}

// Validate validates a recursive proof.
func (rp *RecursiveProof) Validate() error {
	if len(rp.Proof) == 0 {
		return fmt.Errorf("recursive proof data is required")
	}

	if len(rp.VerifiedProofs) == 0 {
		return fmt.Errorf("must verify at least one proof")
	}

	if rp.DepthLevel > 10 {
		return fmt.Errorf("recursion depth %d exceeds maximum 10", rp.DepthLevel)
	}

	if rp.CircuitId == "" {
		return fmt.Errorf("circuit ID is required")
	}

	for i, hash := range rp.VerifiedProofs {
		if len(hash) != 32 {
			return fmt.Errorf("verified proof hash %d has invalid length %d", i, len(hash))
		}
	}

	return nil
}

// ProofGenerationRequest represents a request to generate a ZK proof.
type ProofGenerationRequest struct {
	RequestID        uint64              // Compute request ID
	CircuitId        string              // Circuit to use
	PublicInputs     map[string][]byte   // Public inputs
	PrivateWitness   map[string][]byte   // Private witness data
	Priority         uint32              // Generation priority (higher = more urgent)
	Deadline         time.Time           // Proof must be ready by this time
}

// Validate validates a proof generation request.
func (pgr *ProofGenerationRequest) Validate() error {
	if pgr.RequestID == 0 {
		return fmt.Errorf("request ID is required")
	}

	if pgr.CircuitId == "" {
		return fmt.Errorf("circuit ID is required")
	}

	if len(pgr.PublicInputs) == 0 {
		return fmt.Errorf("public inputs are required")
	}

	if len(pgr.PrivateWitness) == 0 {
		return fmt.Errorf("private witness is required")
	}

	if !pgr.Deadline.IsZero() && pgr.Deadline.Before(time.Now()) {
		return fmt.Errorf("deadline is in the past")
	}

	return nil
}

// ProofVerificationResult contains the result of proof verification.
type ProofVerificationResult struct {
	Proof            *ZKProof  // The verified proof
	Valid            bool      // Verification result
	Error            string    // Error message if invalid
	VerificationTime uint64    // Time taken in microseconds
	GasUsed          uint64    // Gas consumed
	VerifiedAt       time.Time // Verification timestamp
}

// ZKCircuitType represents different types of ZK circuits.
type ZKCircuitType string

const (
	CircuitTypeCompute   ZKCircuitType = "compute"   // Computation verification
	CircuitTypeEscrow    ZKCircuitType = "escrow"    // Escrow release logic
	CircuitTypeResult    ZKCircuitType = "result"    // Result correctness
	CircuitTypeRecursive ZKCircuitType = "recursive" // Recursive verification
)

// ZKSecurityLevel represents the security level of a ZK proof.
type ZKSecurityLevel string

const (
	SecurityLevelStandard ZKSecurityLevel = "standard" // 128-bit security
	SecurityLevelHigh     ZKSecurityLevel = "high"     // 256-bit security
)
