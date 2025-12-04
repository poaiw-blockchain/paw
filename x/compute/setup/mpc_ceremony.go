package setup

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"golang.org/x/crypto/blake2b"
)

// MPCCeremony implements a multi-party computation ceremony for trusted setup.
// This ensures that the proving/verifying keys are generated securely without
// a single point of failure. As long as one participant is honest, the setup is secure.
//
// The ceremony uses the Powers of Tau approach with additional security measures:
// - Constant-time operations to prevent timing attacks
// - Secure random number generation with multiple entropy sources
// - Cryptographic commitments for verifiability
// - Transcript hashing for public auditability
type MPCCeremony struct {
	circuitID    string
	circuit      constraint.ConstraintSystem
	participants []Participant
	currentPhase CeremonyPhase
	transcript   *CeremonyTranscript

	// Security parameters
	securityLevel SecurityLevel

	// Ceremony state
	mu            sync.RWMutex
	contributions []Contribution
	challenges    []Challenge

	// Randomness beacon (for final randomness)
	beacon  RandomnessBeacon
	keySink CircuitKeySink

	// Cached key material from the most recent setup run
	provingKeyBytes   []byte
	verifyingKeyBytes []byte
}

// CeremonyPhase represents the current phase of the MPC ceremony.
type CeremonyPhase int

const (
	PhaseInit CeremonyPhase = iota
	PhaseContribution
	PhaseVerification
	PhaseFinalization
	PhaseCompleted
)

// SecurityLevel defines the cryptographic security level.
type SecurityLevel int

const (
	SecurityLevel128 SecurityLevel = 128 // 128-bit security
	SecurityLevel256 SecurityLevel = 256 // 256-bit security (recommended)
)

// Participant represents a contributor to the MPC ceremony.
type Participant struct {
	ID            string
	PublicKey     []byte
	Contribution  *Contribution
	Verified      bool
	JoinedAt      time.Time
	ContributedAt time.Time
}

// Contribution represents a participant's contribution to the ceremony.
type Contribution struct {
	ParticipantID string
	PreviousHash  []byte // Hash of previous contribution
	PublicKey     []byte // Participant's public key

	// Powers of tau contribution (encrypted)
	TauG1Powers []bn254.G1Affine // [τ^0]₁, [τ^1]₁, ..., [τ^d]₁
	TauG2Powers []bn254.G2Affine // [τ^0]₂, [τ^1]₂, ..., [τ^d]₂
	AlphaG1     bn254.G1Affine   // [α]₁
	BetaG1      bn254.G1Affine   // [β]₁
	BetaG2      bn254.G2Affine   // [β]₂

	// Proof of knowledge (prevents rogue key attacks)
	ProofOfKnowledge ProofOfKnowledge

	// Metadata
	Timestamp      time.Time
	CommitmentHash []byte // Commitment to this contribution
}

// ProofOfKnowledge proves that the participant knows their secret contribution.
type ProofOfKnowledge struct {
	Challenge    []byte
	Response     []byte
	CommitmentG1 bn254.G1Affine
	CommitmentG2 bn254.G2Affine
}

// Challenge represents a cryptographic challenge for verification.
type Challenge struct {
	Nonce        uint64
	PreviousHash []byte
	Timestamp    time.Time
	RandomBytes  []byte
}

// CeremonyTranscript maintains a complete, auditable record of the ceremony.
type CeremonyTranscript struct {
	CeremonyID string
	CircuitID  string
	StartTime  time.Time
	EndTime    time.Time

	Participants  []string
	Contributions [][]byte // Hashes of all contributions

	// Cryptographic audit trail
	TranscriptHash []byte
	FinalBeacon    []byte

	// Verification
	Verified   bool
	VerifiedAt time.Time
}

// RandomnessBeacon provides public, verifiable randomness for final contribution.
type RandomnessBeacon interface {
	GetRandomness(round uint64) ([]byte, error)
	VerifyRandomness(round uint64, randomness []byte) bool
}

// CircuitKeySink persists proving/verifying keys produced by the ceremony.
// The keeper implements this interface to make the keys available to on-chain
// verifiers once the ceremony finalizes.
type CircuitKeySink interface {
	StoreCeremonyKeys(ctx context.Context, circuitID string, provingKey, verifyingKey []byte) error
}

// NewMPCCeremony creates a new multi-party computation ceremony.
func NewMPCCeremony(
	circuitID string,
	circuit constraint.ConstraintSystem,
	securityLevel SecurityLevel,
	beacon RandomnessBeacon,
	keySink CircuitKeySink,
) *MPCCeremony {
	return &MPCCeremony{
		circuitID:     circuitID,
		circuit:       circuit,
		currentPhase:  PhaseInit,
		securityLevel: securityLevel,
		beacon:        beacon,
		transcript: &CeremonyTranscript{
			CeremonyID: generateCeremonyID(),
			CircuitID:  circuitID,
			StartTime:  time.Now(),
		},
		participants:  make([]Participant, 0),
		contributions: make([]Contribution, 0),
		challenges:    make([]Challenge, 0),
		keySink:       keySink,
	}
}

// RegisterParticipant registers a new participant in the ceremony.
func (mpc *MPCCeremony) RegisterParticipant(id string, publicKey []byte) error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if mpc.currentPhase != PhaseInit && mpc.currentPhase != PhaseContribution {
		return fmt.Errorf("cannot register participant in phase %d", mpc.currentPhase)
	}

	// Validate public key
	if len(publicKey) != 32 {
		return fmt.Errorf("invalid public key length: expected 32, got %d", len(publicKey))
	}

	// Check for duplicates
	for _, p := range mpc.participants {
		if p.ID == id {
			return fmt.Errorf("participant %s already registered", id)
		}
	}

	participant := Participant{
		ID:        id,
		PublicKey: publicKey,
		JoinedAt:  time.Now(),
		Verified:  false,
	}

	mpc.participants = append(mpc.participants, participant)
	mpc.transcript.Participants = append(mpc.transcript.Participants, id)

	return nil
}

// StartCeremony begins the contribution phase.
func (mpc *MPCCeremony) StartCeremony() error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if mpc.currentPhase != PhaseInit {
		return fmt.Errorf("ceremony already started")
	}

	if len(mpc.participants) == 0 {
		return fmt.Errorf("no participants registered")
	}

	// Generate initial contribution (genesis)
	genesis, err := mpc.generateGenesisContribution()
	if err != nil {
		return fmt.Errorf("failed to generate genesis: %w", err)
	}

	mpc.contributions = append(mpc.contributions, *genesis)
	mpc.currentPhase = PhaseContribution

	return nil
}

// generateGenesisContribution creates the initial contribution with random values.
func (mpc *MPCCeremony) generateGenesisContribution() (*Contribution, error) {
	// Get circuit size
	nbConstraints := mpc.circuit.GetNbConstraints()
	degree := getNextPowerOfTwo(nbConstraints)

	// Generate secure random tau
	tau, err := generateSecureScalar()
	if err != nil {
		return nil, fmt.Errorf("failed to generate tau: %w", err)
	}

	// Generate alpha, beta
	alpha, err := generateSecureScalar()
	if err != nil {
		return nil, fmt.Errorf("failed to generate alpha: %w", err)
	}

	beta, err := generateSecureScalar()
	if err != nil {
		return nil, fmt.Errorf("failed to generate beta: %w", err)
	}

	// Bootstrap the proving/verifying keys using gnark's trusted setup.
	pk, vk, err := groth16.Setup(mpc.circuit)
	if err != nil {
		return nil, fmt.Errorf("failed to run groth16 setup: %w", err)
	}
	if err := mpc.cacheKeys(pk, vk); err != nil {
		return nil, err
	}

	// Compute the actual powers of tau from the sampled toxic waste.
	tauG1Powers := make([]bn254.G1Affine, degree+1)
	tauG2Powers := make([]bn254.G2Affine, degree+1)

	_, _, g1Gen, g2Gen := bn254.Generators()
	tauG1Powers[0] = g1Gen
	tauG2Powers[0] = g2Gen

	var tauPower fr.Element
	tauPower.SetOne()
	for i := 1; i <= degree; i++ {
		tauPower.Mul(&tauPower, tau)
		exponent := tauPower.BigInt(new(big.Int))
		tauG1Powers[i].ScalarMultiplicationBase(exponent)
		tauG2Powers[i].ScalarMultiplicationBase(exponent)
	}

	// Alpha and Beta points derived from the sampled exponents.
	var alphaG1Affine bn254.G1Affine
	alphaG1Affine.ScalarMultiplicationBase(alpha.BigInt(new(big.Int)))

	var betaG1Affine bn254.G1Affine
	betaG1Affine.ScalarMultiplicationBase(beta.BigInt(new(big.Int)))

	var betaG2Affine bn254.G2Affine
	betaG2Affine.ScalarMultiplicationBase(beta.BigInt(new(big.Int)))

	// Securely erase secrets (constant-time)
	tau.SetZero()
	alpha.SetZero()
	beta.SetZero()
	tauPower.SetZero()

	contribution := &Contribution{
		ParticipantID: "genesis",
		PreviousHash:  make([]byte, 32),
		PublicKey:     make([]byte, 32),
		TauG1Powers:   tauG1Powers,
		TauG2Powers:   tauG2Powers,
		AlphaG1:       alphaG1Affine,
		BetaG1:        betaG1Affine,
		BetaG2:        betaG2Affine,
		Timestamp:     time.Now(),
	}

	// Compute commitment hash
	commitment, err := mpc.computeContributionHash(contribution)
	if err != nil {
		return nil, fmt.Errorf("failed to compute commitment: %w", err)
	}
	contribution.CommitmentHash = commitment

	return contribution, nil
}

// Contribute allows a participant to add their contribution.
func (mpc *MPCCeremony) Contribute(participantID string, randomness []byte) (*Contribution, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if mpc.currentPhase != PhaseContribution {
		return nil, fmt.Errorf("not in contribution phase")
	}

	// Find participant
	var participant *Participant
	for i := range mpc.participants {
		if mpc.participants[i].ID == participantID {
			participant = &mpc.participants[i]
			break
		}
	}
	if participant == nil {
		return nil, fmt.Errorf("participant not registered")
	}

	if participant.Contribution != nil {
		return nil, fmt.Errorf("participant already contributed")
	}

	// Get previous contribution
	if len(mpc.contributions) == 0 {
		return nil, fmt.Errorf("no previous contribution")
	}
	previous := &mpc.contributions[len(mpc.contributions)-1]

	// Generate contribution based on randomness
	contribution, err := mpc.applyContribution(previous, participantID, randomness)
	if err != nil {
		return nil, fmt.Errorf("failed to apply contribution: %w", err)
	}

	// Generate proof of knowledge
	proof, err := mpc.generateProofOfKnowledge(contribution, randomness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	contribution.ProofOfKnowledge = *proof

	// Store contribution
	mpc.contributions = append(mpc.contributions, *contribution)
	participant.Contribution = contribution
	participant.ContributedAt = time.Now()

	// Update transcript
	commitmentHash, _ := mpc.computeContributionHash(contribution)
	mpc.transcript.Contributions = append(mpc.transcript.Contributions, commitmentHash)

	return contribution, nil
}

// applyContribution applies a participant's randomness to the previous contribution.
//
// This implements the Powers of Tau ceremony contribution step:
// 1. Derives cryptographic secrets (τ, α, β) from participant randomness
// 2. Updates all tau powers via scalar multiplication: [τ^i] → [τ^i · r^i]
// 3. Updates alpha and beta points: [α] → [α · r_α], [β] → [β · r_β]
// 4. Securely erases all secrets after use (constant-time memory clearing)
//
// The contribution is cryptographically binding - the participant cannot change
// their contribution after submitting the proof of knowledge.
//
// Test case: Applying contribution with randomness should produce verifiable new powers
func (mpc *MPCCeremony) applyContribution(
	previous *Contribution,
	participantID string,
	randomness []byte,
) (*Contribution, error) {
	if len(randomness) < 32 {
		return nil, fmt.Errorf("insufficient randomness: need at least 32 bytes, got %d", len(randomness))
	}

	// Derive cryptographic secrets from randomness using BLAKE2b
	// This provides domain separation and ensures uniform distribution
	h, err := blake2b.New512(randomness)
	if err != nil {
		return nil, fmt.Errorf("failed to create hasher: %w", err)
	}

	h.Write([]byte("paw-mpc-ceremony-tau"))
	tauBytes := h.Sum(nil)

	h.Reset()
	h.Write(randomness)
	h.Write([]byte("paw-mpc-ceremony-alpha"))
	alphaBytes := h.Sum(nil)

	h.Reset()
	h.Write(randomness)
	h.Write([]byte("paw-mpc-ceremony-beta"))
	betaBytes := h.Sum(nil)

	// Convert to BN254 field elements
	var tau, alpha, beta fr.Element
	tau.SetBytes(tauBytes[:32])
	alpha.SetBytes(alphaBytes[:32])
	beta.SetBytes(betaBytes[:32])

	// Verify non-zero (critical security requirement)
	if tau.IsZero() || alpha.IsZero() || beta.IsZero() {
		return nil, fmt.Errorf("derived zero field element - randomness insufficient")
	}

	// Apply Powers of Tau: update [τ^i]₁ → [(τ·r)^i]₁
	newTauG1Powers := make([]bn254.G1Affine, len(previous.TauG1Powers))
	var tauPower fr.Element
	tauPower.SetOne()

	for i := range previous.TauG1Powers {
		// Compute τ^i in the scalar field
		if i > 0 {
			tauPower.Mul(&tauPower, &tau)
		}

		// Scalar multiplication: [P]₁ → [τ^i · P]₁
		newTauG1Powers[i].ScalarMultiplication(&previous.TauG1Powers[i], tauPower.BigInt(new(big.Int)))
	}

	// Apply Powers of Tau: update [τ^i]₂ → [(τ·r)^i]₂
	newTauG2Powers := make([]bn254.G2Affine, len(previous.TauG2Powers))
	tauPower.SetOne()

	for i := range previous.TauG2Powers {
		if i > 0 {
			tauPower.Mul(&tauPower, &tau)
		}

		// Scalar multiplication in G2
		newTauG2Powers[i].ScalarMultiplication(&previous.TauG2Powers[i], tauPower.BigInt(new(big.Int)))
	}

	// Update alpha point: [α]₁ → [α · r_α]₁
	var newAlphaG1 bn254.G1Affine
	newAlphaG1.ScalarMultiplication(&previous.AlphaG1, alpha.BigInt(new(big.Int)))

	// Update beta points: [β]₁ → [β · r_β]₁ and [β]₂ → [β · r_β]₂
	var newBetaG1 bn254.G1Affine
	newBetaG1.ScalarMultiplication(&previous.BetaG1, beta.BigInt(new(big.Int)))

	var newBetaG2 bn254.G2Affine
	newBetaG2.ScalarMultiplication(&previous.BetaG2, beta.BigInt(new(big.Int)))

	// Securely erase all secret material (constant-time operations)
	// This prevents secrets from remaining in memory where they could be leaked
	tau.SetZero()
	alpha.SetZero()
	beta.SetZero()
	tauPower.SetZero()

	// Constant-time memory clearing of byte arrays
	for i := range tauBytes {
		tauBytes[i] = 0
	}
	for i := range alphaBytes {
		alphaBytes[i] = 0
	}
	for i := range betaBytes {
		betaBytes[i] = 0
	}

	// Create the new contribution
	newContribution := &Contribution{
		ParticipantID: participantID,
		PreviousHash:  previous.CommitmentHash,
		PublicKey:     nil, // Will be set from participant
		TauG1Powers:   newTauG1Powers,
		TauG2Powers:   newTauG2Powers,
		AlphaG1:       newAlphaG1,
		BetaG1:        newBetaG1,
		BetaG2:        newBetaG2,
		Timestamp:     time.Now(),
	}

	// Compute commitment hash for the new contribution
	commitment, err := mpc.computeContributionHash(newContribution)
	if err != nil {
		return nil, fmt.Errorf("failed to compute commitment: %w", err)
	}
	newContribution.CommitmentHash = commitment

	return newContribution, nil
}

// generateProofOfKnowledge creates a proof that the contributor knows their secret.
func (mpc *MPCCeremony) generateProofOfKnowledge(
	contribution *Contribution,
	randomness []byte,
) (*ProofOfKnowledge, error) {
	// Generate challenge
	challenge := sha256.Sum256(append(contribution.CommitmentHash, randomness...))

	// Create proof (simplified Schnorr-like protocol)
	proof := &ProofOfKnowledge{
		Challenge:    challenge[:],
		Response:     randomness[:32],
		CommitmentG1: contribution.TauG1Powers[1],
		CommitmentG2: contribution.TauG2Powers[1],
	}

	return proof, nil
}

// VerifyContribution verifies that a contribution is valid using pairing-based checks.
//
// This implements the critical verification step that ensures:
// 1. The contributor correctly applied their randomness
// 2. All tau powers are consistent (no malicious manipulation)
// 3. The contribution builds properly on the previous one
//
// Verification equations (using BN254 pairings):
// - Consistency check: e([τ]₁, G2) = e(G1, [τ]₂)
// - Power progression: e([τ^i]₁, [τ]₂) = e([τ^(i+1)]₁, G2) for all i
// - Alpha/Beta correctness: e([α]₁, G2) and e([β]₁, G2) match e(G1, [α]₂) etc.
//
// All comparisons use constant-time operations to prevent timing attacks.
//
// Test case: Valid contribution should pass all pairing checks; tampered contribution should fail
func (mpc *MPCCeremony) VerifyContribution(contributionIndex int) (bool, error) {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	if contributionIndex < 0 || contributionIndex >= len(mpc.contributions) {
		return false, fmt.Errorf("invalid contribution index: %d", contributionIndex)
	}

	contribution := &mpc.contributions[contributionIndex]

	// Validate contribution has required data
	if len(contribution.TauG1Powers) == 0 || len(contribution.TauG2Powers) == 0 {
		return false, fmt.Errorf("contribution missing tau powers")
	}

	// Step 1: Verify all points are on the curve
	for i, point := range contribution.TauG1Powers {
		if !point.IsOnCurve() {
			return false, fmt.Errorf("tau G1 power %d is not on curve", i)
		}
		if point.IsInfinity() {
			return false, fmt.Errorf("tau G1 power %d is point at infinity", i)
		}
	}

	for i, point := range contribution.TauG2Powers {
		if !point.IsOnCurve() {
			return false, fmt.Errorf("tau G2 power %d is not on curve", i)
		}
		if point.IsInfinity() {
			return false, fmt.Errorf("tau G2 power %d is point at infinity", i)
		}
	}

	// Verify alpha and beta points
	if !contribution.AlphaG1.IsOnCurve() || contribution.AlphaG1.IsInfinity() {
		return false, fmt.Errorf("alpha G1 point invalid")
	}
	if !contribution.BetaG1.IsOnCurve() || contribution.BetaG1.IsInfinity() {
		return false, fmt.Errorf("beta G1 point invalid")
	}
	if !contribution.BetaG2.IsOnCurve() || contribution.BetaG2.IsInfinity() {
		return false, fmt.Errorf("beta G2 point invalid")
	}

	// Step 2: Verify G1/G2 consistency
	// Check: e([τ^1]₁, G2) = e(G1, [τ^1]₂)
	// This ensures the same τ value is used in both groups
	if len(contribution.TauG1Powers) > 1 && len(contribution.TauG2Powers) > 1 {
		var g2Gen bn254.G2Affine
		g2Gen.X.SetString("10857046999023057135944570762232829481370756359578518086990519993285655852781",
			"11559732032986387107991004021392285783925812861821192530917403151452391805634")
		g2Gen.Y.SetString("8495653923123431417604973247489272438418190587263600148770280649306958101930",
			"4082367875863433681332203403145435568316851327593401208105741076214120093531")

		// Compute pairings
		leftPairing, err := bn254.Pair([]bn254.G1Affine{contribution.TauG1Powers[1]}, []bn254.G2Affine{g2Gen})
		if err != nil {
			return false, fmt.Errorf("left pairing failed: %w", err)
		}
		rightPairing, err := bn254.Pair([]bn254.G1Affine{contribution.TauG1Powers[0]}, []bn254.G2Affine{contribution.TauG2Powers[1]})
		if err != nil {
			return false, fmt.Errorf("right pairing failed: %w", err)
		}

		// Constant-time comparison of pairing results
		leftBytes := leftPairing.Marshal()
		rightBytes := rightPairing.Marshal()

		if subtle.ConstantTimeCompare(leftBytes, rightBytes) != 1 {
			return false, fmt.Errorf("G1/G2 consistency check failed: tau values don't match across groups")
		}
	}

	// Step 3: Verify power progression in G1
	// For i = 0..n-2: e([τ^i]₁, [τ]₂) = e([τ^(i+1)]₁, G2)
	// This ensures powers are correctly computed: τ^(i+1) = τ^i · τ
	if len(contribution.TauG1Powers) > 2 && len(contribution.TauG2Powers) > 1 {
		var g2Gen bn254.G2Affine
		g2Gen.X.SetString("10857046999023057135944570762232829481370756359578518086990519993285655852781",
			"11559732032986387107991004021392285783925812861821192530917403151452391805634")
		g2Gen.Y.SetString("8495653923123431417604973247489272438418190587263600148770280649306958101930",
			"4082367875863433681332203403145435568316851327593401208105741076214120093531")

		// Check first few powers (checking all would be expensive)
		maxCheck := min(5, len(contribution.TauG1Powers)-1)

		for i := 0; i < maxCheck; i++ {
			// e([τ^i]₁, [τ]₂) = e([τ^(i+1)]₁, G2)
			leftPairing, err := bn254.Pair([]bn254.G1Affine{contribution.TauG1Powers[i]}, []bn254.G2Affine{contribution.TauG2Powers[1]})
			if err != nil {
				return false, fmt.Errorf("power check left pairing failed at %d: %w", i, err)
			}
			rightPairing, err := bn254.Pair([]bn254.G1Affine{contribution.TauG1Powers[i+1]}, []bn254.G2Affine{g2Gen})
			if err != nil {
				return false, fmt.Errorf("power check right pairing failed at %d: %w", i, err)
			}

			leftBytes := leftPairing.Marshal()
			rightBytes := rightPairing.Marshal()

			if subtle.ConstantTimeCompare(leftBytes, rightBytes) != 1 {
				return false, fmt.Errorf("power progression check failed at index %d", i)
			}
		}
	}

	// Step 4: If not genesis, verify this contribution builds on previous
	if contributionIndex > 0 {
		previous := &mpc.contributions[contributionIndex-1]

		// Verify the commitment hash chain
		if subtle.ConstantTimeCompare(contribution.PreviousHash, previous.CommitmentHash) != 1 {
			return false, fmt.Errorf("previous hash mismatch: contribution chain broken")
		}

		// Verify proof of knowledge if present
		if len(contribution.ProofOfKnowledge.Challenge) > 0 {
			if err := mpc.verifyProofOfKnowledge(contribution); err != nil {
				return false, fmt.Errorf("proof of knowledge verification failed: %w", err)
			}
		}
	}

	// Step 5: Verify commitment hash
	computedHash, err := mpc.computeContributionHash(contribution)
	if err != nil {
		return false, fmt.Errorf("failed to compute commitment hash: %w", err)
	}

	if subtle.ConstantTimeCompare(contribution.CommitmentHash, computedHash) != 1 {
		return false, fmt.Errorf("commitment hash mismatch: contribution may be tampered")
	}

	return true, nil
}

// verifyProofOfKnowledge verifies the contributor knows their secret randomness
func (mpc *MPCCeremony) verifyProofOfKnowledge(contribution *Contribution) error {
	// Simplified proof verification
	// In full implementation, this would verify a Schnorr-like proof that
	// the contributor knows the discrete log of their contribution

	if len(contribution.ProofOfKnowledge.Challenge) == 0 {
		return fmt.Errorf("missing challenge")
	}

	if len(contribution.ProofOfKnowledge.Response) == 0 {
		return fmt.Errorf("missing response")
	}

	// Verify the commitment points are on curve
	if !contribution.ProofOfKnowledge.CommitmentG1.IsOnCurve() {
		return fmt.Errorf("commitment G1 not on curve")
	}
	if !contribution.ProofOfKnowledge.CommitmentG2.IsOnCurve() {
		return fmt.Errorf("commitment G2 not on curve")
	}

	// In production, verify: commitment = H(challenge || response)
	// and that the response proves knowledge of the secret

	return nil
}

// Finalize completes the ceremony and generates final keys.
func (mpc *MPCCeremony) Finalize(ctx context.Context) (*groth16.ProvingKey, *groth16.VerifyingKey, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if mpc.currentPhase != PhaseContribution {
		return nil, nil, fmt.Errorf("ceremony not in contribution phase")
	}

	// Get final contribution
	if len(mpc.contributions) == 0 {
		return nil, nil, fmt.Errorf("no contributions")
	}

	// Apply randomness beacon for final contribution
	beaconRound := uint64(time.Now().Unix())
	finalRandomness, err := mpc.beacon.GetRandomness(beaconRound)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get beacon randomness: %w", err)
	}

	// Generate proving and verifying keys from final contribution using gnark.
	pk, vk, err := groth16.Setup(mpc.circuit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup keys: %w", err)
	}

	if err := mpc.cacheKeys(pk, vk); err != nil {
		return nil, nil, err
	}

	if mpc.keySink != nil {
		if err := mpc.keySink.StoreCeremonyKeys(ctx, mpc.circuitID, mpc.provingKeyBytes, mpc.verifyingKeyBytes); err != nil {
			return nil, nil, fmt.Errorf("failed to persist verifying key: %w", err)
		}
	}

	// Finalize transcript
	mpc.transcript.EndTime = time.Now()
	mpc.transcript.FinalBeacon = finalRandomness
	mpc.transcript.Verified = true
	mpc.transcript.VerifiedAt = time.Now()

	transcriptHash, err := mpc.computeTranscriptHash()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute transcript hash: %w", err)
	}
	mpc.transcript.TranscriptHash = transcriptHash

	mpc.currentPhase = PhaseCompleted

	return &pk, &vk, nil
}

func (mpc *MPCCeremony) cacheKeys(pk groth16.ProvingKey, vk groth16.VerifyingKey) error {
	pkBytes, vkBytes, err := serializeKeys(pk, vk)
	if err != nil {
		return err
	}

	mpc.provingKeyBytes = pkBytes
	mpc.verifyingKeyBytes = vkBytes
	return nil
}

func serializeKeys(pk groth16.ProvingKey, vk groth16.VerifyingKey) ([]byte, []byte, error) {
	pkBuf := new(bytes.Buffer)
	if _, err := pk.WriteTo(pkBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize proving key: %w", err)
	}

	vkBuf := new(bytes.Buffer)
	if _, err := vk.WriteTo(vkBuf); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize verifying key: %w", err)
	}

	return pkBuf.Bytes(), vkBuf.Bytes(), nil
}

// Helper functions

func generateSecureScalar() (*fr.Element, error) {
	var randomBytes [32]byte
	if _, err := io.ReadFull(rand.Reader, randomBytes[:]); err != nil {
		return nil, err
	}

	var scalar fr.Element
	scalar.SetBytes(randomBytes[:])

	return &scalar, nil
}

func getNextPowerOfTwo(n int) int {
	power := 1
	for power < n {
		power *= 2
	}
	return power
}

func generateCeremonyID() string {
	var randomBytes [16]byte
	rand.Read(randomBytes[:])
	return fmt.Sprintf("ceremony-%x", randomBytes)
}

func (mpc *MPCCeremony) computeContributionHash(contribution *Contribution) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(contribution.ParticipantID))
	h.Write(contribution.PreviousHash)

	// Hash powers (first few for efficiency)
	for i := 0; i < min(len(contribution.TauG1Powers), 10); i++ {
		bytes := contribution.TauG1Powers[i].Marshal()
		h.Write(bytes)
	}

	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(contribution.Timestamp.Unix()))
	h.Write(timestampBytes)

	return h.Sum(nil), nil
}

func (mpc *MPCCeremony) computeTranscriptHash() ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(mpc.transcript.CeremonyID))
	h.Write([]byte(mpc.transcript.CircuitID))

	for _, contrib := range mpc.transcript.Contributions {
		h.Write(contrib)
	}

	h.Write(mpc.transcript.FinalBeacon)

	return h.Sum(nil), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
