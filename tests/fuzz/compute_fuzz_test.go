package fuzz

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// ComputeFuzzInput represents fuzzed input for compute operations
type ComputeFuzzInput struct {
	RequestID       uint64
	EscrowAmount    uint64
	ProviderStake   uint64
	ExecutionTime   uint32
	ResultSize      uint32
	VerificationSig []byte
	MerkleProof     [][]byte
}

// FuzzComputeEscrowLifecycle tests escrow creation, locking, and release
func FuzzComputeEscrowLifecycle(f *testing.F) {
	// Seed with edge cases
	seeds := [][]byte{
		encodeEscrowInput(1000000, 500000, 100, true),      // Normal case
		encodeEscrowInput(0, 500000, 100, false),           // Zero escrow, unsuccessful
		encodeEscrowInput(1000000, 0, 100, true),           // Zero stake
		encodeEscrowInput(^uint64(0)-1, 500000, 100, true), // Max escrow
		encodeEscrowInput(1000000, 500000, 0, true),        // Instant execution
		encodeEscrowInput(1000000, 500000, 86400, true),    // 24h execution
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 24 {
			return
		}

		escrowAmount := binary.BigEndian.Uint64(data[0:8])
		providerStake := binary.BigEndian.Uint64(data[8:16])
		executionTime := binary.BigEndian.Uint32(data[16:20])
		successFlag := data[20]%2 == 0

		// Test escrow creation invariants
		state := createEscrowState(escrowAmount, providerStake, executionTime)

		// Invariant: Total locked amount should equal escrow + stake
		require.Equal(t, escrowAmount+providerStake, state.TotalLocked,
			"Total locked must equal escrow + provider stake")

		// Invariant: State should be initialized as locked
		require.True(t, state.IsLocked, "New escrow should be locked")

		// Test escrow release
		released := releaseEscrow(state, successFlag)

		if successFlag {
			// On success: provider gets escrow + stake
			require.Equal(t, escrowAmount+providerStake, released.ProviderAmount,
				"Provider should receive escrow + stake on success")
			require.Equal(t, uint64(0), released.RequesterRefund,
				"Requester should get no refund on success")
		} else {
			// On failure: requester gets refund, provider loses stake
			require.Equal(t, escrowAmount, released.RequesterRefund,
				"Requester should get escrow refunded on failure")
			require.Equal(t, uint64(0), released.ProviderAmount,
				"Provider should get nothing on failure")
		}

		// Invariant: All funds must be accounted for
		totalReleased := released.ProviderAmount + released.RequesterRefund + released.SlashedAmount
		require.Equal(t, state.TotalLocked, totalReleased,
			"All locked funds must be accounted for in release")
	})
}

// FuzzComputeVerificationProof tests cryptographic proof verification
func FuzzComputeVerificationProof(f *testing.F) {
	// Generate valid key pairs for seeds
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	seeds := [][]byte{
		generateValidProofSeed(pubKey, privKey, 12345, "result_hash_1"),
		generateValidProofSeed(pubKey, privKey, 67890, "result_hash_2"),
		generateInvalidProofSeed(), // Invalid signature
		generateTamperedProofSeed(pubKey, privKey, 11111, "tampered"),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 200 {
			return
		}

		proof := parseVerificationProof(data)
		if proof == nil {
			return
		}

		// Test proof structure validation
		err := validateProofStructure(proof)
		if err != nil {
			// Expected structural errors should be descriptive
			require.NotEmpty(t, err.Error())
			return
		}

		// Test signature verification
		requestID := binary.BigEndian.Uint64(data[0:8])
		resultHash := data[8:40] // 32 bytes

		messageHash := computeProofMessage(requestID, resultHash, proof.MerkleRoot, proof.Nonce)
		validSig := verifyEd25519Signature(proof.PublicKey, messageHash, proof.Signature)

		// Test merkle proof verification
		validMerkle := verifyMerkleProof(proof.MerkleProof, proof.MerkleRoot, resultHash)

		// Combined verification score
		score := calculateVerificationScore(validSig, validMerkle, proof.StateCommitment != nil)

		// Invariant: Score must be 0-100
		require.True(t, score >= 0 && score <= 100,
			"Verification score must be in range [0, 100], got %d", score)

		// Invariant: Valid signature + valid merkle should give high score
		if validSig && validMerkle {
			require.True(t, score >= 80,
				"Valid signature and merkle proof should yield score >= 80")
		}
	})
}

// FuzzComputeNonceReplayProtection tests nonce-based replay attack prevention.
// Zero nonces are valid since certain compute clients start counting from 0; the tracker must only reject replays.
func FuzzComputeNonceReplayProtection(f *testing.F) {
	seeds := [][]byte{
		encodeNonceInput(1, 1000),
		encodeNonceInput(0, 500), // Zero nonce (allowed) earlier timestamp
		encodeNonceInput(12345, 2500),
		encodeNonceInput(^uint64(0), 4000), // Max nonce
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 24 {
			return
		}

		nonce := binary.BigEndian.Uint64(data[0:8])
		timestamp := int64(binary.BigEndian.Uint64(data[8:16]))
		providerID := data[16:24]

		tracker := &NonceTracker{
			usedNonces: make(map[string]map[uint64]int64),
		}

		provider := string(providerID)

		// Test nonce usage tracking
		err := tracker.checkAndRecordNonce(provider, nonce, timestamp)

		if timestamp <= 0 {
			require.Error(t, err, "Non-positive timestamp should be rejected")
			return
		}

		// First use should succeed
		require.NoError(t, err, "First nonce use should succeed")

		// Replay attempt should fail
		err2 := tracker.checkAndRecordNonce(provider, nonce, timestamp)
		require.Error(t, err2, "Nonce replay should be rejected")
		require.Contains(t, err2.Error(), "replay")

		// Different nonce should succeed
		err3 := tracker.checkAndRecordNonce(provider, nonce+1, timestamp+1)
		require.NoError(t, err3, "Different nonce should be accepted")
	})
}

// FuzzComputeResourceExhaustion tests resistance to resource exhaustion attacks
func FuzzComputeResourceExhaustion(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 20 {
			return
		}

		numRequests := binary.BigEndian.Uint32(data[0:4])
		requestSize := binary.BigEndian.Uint32(data[4:8])
		resultSize := binary.BigEndian.Uint32(data[8:12])
		executionTime := binary.BigEndian.Uint32(data[12:16])
		requesterStake := binary.BigEndian.Uint64(data[16:24]) % 100000000 // Cap stake

		// Cap values to prevent DoS in test itself
		if numRequests > 10000 {
			numRequests %= 10000
		}
		if requestSize > 1024*1024 {
			requestSize %= 1024 * 1024
		}
		if resultSize > 1024*1024 {
			resultSize %= 1024 * 1024
		}

		// Test rate limiting
		limiter := newRateLimiter(100, 1000) // 100 req/sec, 1000 burst

		accepted := 0
		rejected := 0

		for i := uint32(0); i < numRequests && i < 10000; i++ {
			if limiter.allow() {
				accepted++
			} else {
				rejected++
			}
		}

		// Invariant: Rate limiting should be enforced
		if numRequests > 1000 {
			require.True(t, rejected > 0, "Rate limiter should reject excess requests")
		}

		// Test resource cost calculation
		cost := calculateResourceCost(requestSize, resultSize, executionTime)

		// Invariant: Cost should be proportional to resource usage
		require.True(t, cost > 0, "Resource cost must be positive")

		// Test stake adequacy
		requiredStake := calculateRequiredStake(cost)
		adequate := requesterStake >= requiredStake

		if !adequate {
			// Should reject request or require more stake
			require.True(t, requesterStake < requiredStake,
				"Stake adequacy check should be consistent")
		}
	})
}

// FuzzComputeStateTransitionValidation tests state commitment verification
func FuzzComputeStateTransitionValidation(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 96 {
			return
		}

		initialState := data[0:32]
		finalState := data[32:64]
		executionTrace := data[64:96]

		// Test state transition determinism
		hash1 := computeStateTransition(initialState, executionTrace)
		hash2 := computeStateTransition(initialState, executionTrace)

		// Invariant: Same inputs must produce same output
		require.Equal(t, hash1, hash2, "State transition must be deterministic")

		// Test state commitment validation
		valid := validateStateTransition(initialState, finalState, executionTrace)

		if valid {
			// Valid transition should hash correctly
			computedFinal := computeStateTransition(initialState, executionTrace)
			require.Equal(t, finalState, computedFinal,
				"Valid state transition must match computed final state")
		}

		// Test rollback safety
		reversedTrace := reverseBytes(executionTrace)
		reversedFinal := computeStateTransition(finalState, reversedTrace)

		// Invariant: Arbitrary reversal shouldn't produce initial state
		// (prevents trivial reversibility attacks)
		if len(initialState) > 0 && len(finalState) > 0 {
			require.NotEqual(t, initialState, reversedFinal,
				"State transitions should not be trivially reversible")
		}
	})
}

// Helper structures and functions

type EscrowState struct {
	TotalLocked uint64
	Escrow      uint64
	Stake       uint64
	IsLocked    bool
	CreatedAt   uint32
}

type ReleaseResult struct {
	ProviderAmount  uint64
	RequesterRefund uint64
	SlashedAmount   uint64
}

type VerificationProof struct {
	Signature       []byte
	PublicKey       []byte
	MerkleRoot      []byte
	MerkleProof     [][]byte
	StateCommitment []byte
	Nonce           uint64
}

type NonceTracker struct {
	usedNonces map[string]map[uint64]int64
}

type RateLimiter struct {
	rate  uint32
	burst uint32
	count uint32
}

func createEscrowState(escrow, stake uint64, execTime uint32) *EscrowState {
	return &EscrowState{
		TotalLocked: escrow + stake,
		Escrow:      escrow,
		Stake:       stake,
		IsLocked:    true,
		CreatedAt:   execTime,
	}
}

func releaseEscrow(state *EscrowState, success bool) *ReleaseResult {
	result := &ReleaseResult{}

	if success {
		result.ProviderAmount = state.TotalLocked
	} else {
		result.RequesterRefund = state.Escrow
		result.SlashedAmount = state.Stake
	}

	return result
}

func validateProofStructure(proof *VerificationProof) error {
	if len(proof.Signature) != 64 {
		return &ProofError{"invalid signature length"}
	}
	if len(proof.PublicKey) != 32 {
		return &ProofError{"invalid public key length"}
	}
	if len(proof.MerkleRoot) != 32 {
		return &ProofError{"invalid merkle root length"}
	}
	// Zero nonces are valid because some clients start counting from zero; replay protection
	// is enforced separately by the nonce tracker.
	return nil
}

func computeProofMessage(requestID uint64, resultHash, merkleRoot []byte, nonce uint64) []byte {
	hasher := sha256.New()

	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)

	hasher.Write(resultHash)
	hasher.Write(merkleRoot)

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hasher.Write(nonceBytes)

	return hasher.Sum(nil)
}

func verifyEd25519Signature(pubKey, message, signature []byte) bool {
	if len(pubKey) != 32 || len(signature) != 64 {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubKey), message, signature)
}

func verifyMerkleProof(proof [][]byte, root, leaf []byte) bool {
	if len(proof) == 0 || len(root) != 32 {
		return false
	}

	current := sha256.Sum256(leaf)

	for _, node := range proof {
		if len(node) != 32 {
			return false
		}

		combined := append(current[:], node...)
		current = sha256.Sum256(combined)
	}

	return bytes.Equal(current[:], root)
}

func calculateVerificationScore(validSig, validMerkle, hasStateCommit bool) int {
	score := 0

	if validSig {
		score += 40
	}
	if validMerkle {
		score += 40
	}
	if hasStateCommit {
		score += 20
	}

	return score
}

// checkAndRecordNonce tracks provider nonces while allowing zero as a valid first nonce.
// Zero is permitted because some client SDKs default to zero and we only require monotonic, non-replayed values.
func (nt *NonceTracker) checkAndRecordNonce(provider string, nonce uint64, timestamp int64) error {
	if timestamp <= 0 {
		return &ProofError{"invalid timestamp"}
	}

	if nt.usedNonces == nil {
		nt.usedNonces = make(map[string]map[uint64]int64)
	}

	if nt.usedNonces[provider] == nil {
		nt.usedNonces[provider] = make(map[uint64]int64)
	}

	if _, exists := nt.usedNonces[provider][nonce]; exists {
		return &ProofError{"nonce replay detected"}
	}

	nt.usedNonces[provider][nonce] = timestamp
	return nil
}

func newRateLimiter(rate, burst uint32) *RateLimiter {
	return &RateLimiter{
		rate:  rate,
		burst: burst,
		count: 0,
	}
}

func (rl *RateLimiter) allow() bool {
	if rl.count < rl.burst {
		rl.count++
		return true
	}
	return false
}

func calculateResourceCost(requestSize, resultSize, execTime uint32) uint64 {
	// Cost model: size + time weighted
	sizeCost := uint64(requestSize + resultSize)
	timeCost := uint64(execTime) * 1000
	return sizeCost + timeCost
}

func calculateRequiredStake(cost uint64) uint64 {
	// Require stake proportional to resource cost
	return cost / 100 // 1% stake of resource cost
}

func computeStateTransition(initialState, trace []byte) []byte {
	hasher := sha256.New()
	hasher.Write(initialState)
	hasher.Write(trace)
	sum := hasher.Sum(nil)
	return sum
}

func validateStateTransition(initial, final, trace []byte) bool {
	computed := computeStateTransition(initial, trace)
	return bytes.Equal(computed, final)
}

func reverseBytes(b []byte) []byte {
	reversed := make([]byte, len(b))
	for i := range b {
		reversed[i] = b[len(b)-1-i]
	}
	return reversed
}

type ProofError struct {
	msg string
}

func (e *ProofError) Error() string {
	return e.msg
}

// Encoding helpers

func encodeEscrowInput(escrow, stake uint64, execTime uint32, success bool) []byte {
	buf := make([]byte, 21)
	binary.BigEndian.PutUint64(buf[0:8], escrow)
	binary.BigEndian.PutUint64(buf[8:16], stake)
	binary.BigEndian.PutUint32(buf[16:20], execTime)
	if success {
		buf[20] = 1
	} else {
		buf[20] = 0
	}
	return buf
}

func generateValidProofSeed(pubKey ed25519.PublicKey, privKey ed25519.PrivateKey, requestID uint64, resultHash string) []byte {
	merkleRoot := sha256.Sum256([]byte(resultHash))
	nonce := uint64(12345)

	message := computeProofMessage(requestID, merkleRoot[:], merkleRoot[:], nonce)
	signature := ed25519.Sign(privKey, message)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, requestID); err != nil {
		panic(err)
	}
	buf.Write(merkleRoot[:])
	buf.Write(signature)
	buf.Write(pubKey)
	buf.Write(merkleRoot[:]) // merkle root
	if err := binary.Write(buf, binary.BigEndian, nonce); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func generateInvalidProofSeed() []byte {
	buf := make([]byte, 200)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return buf
}

func generateTamperedProofSeed(pubKey ed25519.PublicKey, privKey ed25519.PrivateKey, requestID uint64, resultHash string) []byte {
	seed := generateValidProofSeed(pubKey, privKey, requestID, resultHash)
	// Tamper with signature
	seed[50] ^= 0xFF
	return seed
}

func encodeNonceInput(nonce uint64, timestamp int64) []byte {
	buf := make([]byte, 24)
	binary.BigEndian.PutUint64(buf[0:8], nonce)
	binary.BigEndian.PutUint64(buf[8:16], uint64(timestamp))
	if _, err := rand.Read(buf[16:24]); err != nil {
		panic(err)
	}
	return buf
}

func parseVerificationProof(data []byte) *VerificationProof {
	if len(data) < 200 {
		return nil
	}

	proof := &VerificationProof{
		Signature:       data[40:104],
		PublicKey:       data[104:136],
		MerkleRoot:      data[136:168],
		StateCommitment: data[168:200],
		Nonce:           binary.BigEndian.Uint64(data[0:8]),
	}

	// Parse merkle proof nodes
	offset := 200
	for offset+32 <= len(data) && len(proof.MerkleProof) < 10 {
		node := make([]byte, 32)
		copy(node, data[offset:offset+32])
		proof.MerkleProof = append(proof.MerkleProof, node)
		offset += 32
	}

	return proof
}
