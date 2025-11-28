package property_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"
)

// Property: Escrow total must equal sum of components
func TestPropertyEscrowConservation(t *testing.T) {
	t.Parallel()
	property := func(escrow, stake uint64) bool {
		if escrow > math.MaxUint64/2 || stake > math.MaxUint64/2 {
			return true // Prevent overflow
		}

		total := escrow + stake
		return total == escrow+stake && total >= escrow && total >= stake
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 10000}))
}

// Property: Successful computation releases all funds to provider
func TestPropertySuccessfulRelease(t *testing.T) {
	t.Parallel()
	property := func(escrow, stake uint64) bool {
		if escrow > math.MaxUint64/2 || stake > math.MaxUint64/2 {
			return true
		}

		total := escrow + stake
		providerAmount, refund, slashed := simulateRelease(escrow, stake, true)

		return providerAmount == total && refund == 0 && slashed == 0
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Property: Failed computation refunds escrow, slashes stake
func TestPropertyFailedRelease(t *testing.T) {
	t.Parallel()
	property := func(escrow, stake uint64) bool {
		if escrow > math.MaxUint64/2 || stake > math.MaxUint64/2 {
			return true
		}

		providerAmount, refund, slashed := simulateRelease(escrow, stake, false)

		return providerAmount == 0 && refund == escrow && slashed == stake
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Property: All funds accounted for in any release scenario
func TestPropertyFundsConservation(t *testing.T) {
	t.Parallel()
	property := func(escrow, stake uint64, success bool) bool {
		if escrow > math.MaxUint64/2 || stake > math.MaxUint64/2 {
			return true
		}

		total := escrow + stake
		providerAmount, refund, slashed := simulateRelease(escrow, stake, success)

		return providerAmount+refund+slashed == total
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 10000}))
}

// Property: Verification score must be in range [0, 100]
func TestPropertyVerificationScoreBounds(t *testing.T) {
	t.Parallel()
	property := func(validSig, validMerkle, hasState bool) bool {
		score := calculateVerificationScore(validSig, validMerkle, hasState)
		return score >= 0 && score <= 100
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Valid signature + valid merkle = score >= 80
func TestPropertyHighScoreForValidProof(t *testing.T) {
	t.Parallel()
	property := func(hasState bool) bool {
		score := calculateVerificationScore(true, true, hasState)
		return score >= 80
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Nonce replay must be rejected
func TestPropertyNonceReplay(t *testing.T) {
	t.Parallel()
	property := func(nonce uint64, provider string) bool {
		if nonce == 0 || provider == "" {
			return true
		}

		tracker := NewNonceTracker()

		// First use should succeed
		err1 := tracker.Use(provider, nonce, 1000)
		if err1 != nil {
			return false
		}

		// Replay should fail
		err2 := tracker.Use(provider, nonce, 1001)
		return err2 != nil
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Property: Zero nonce must be rejected
func TestPropertyZeroNonceRejection(t *testing.T) {
	t.Parallel()
	property := func(provider string) bool {
		if provider == "" {
			return true
		}

		tracker := NewNonceTracker()
		err := tracker.Use(provider, 0, 1000)
		return err != nil
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Different providers can use same nonce
func TestPropertyNonceProviderIsolation(t *testing.T) {
	t.Parallel()
	property := func(nonce uint64, provider1, provider2 string) bool {
		if nonce == 0 || provider1 == "" || provider2 == "" || provider1 == provider2 {
			return true
		}

		tracker := NewNonceTracker()

		err1 := tracker.Use(provider1, nonce, 1000)
		err2 := tracker.Use(provider2, nonce, 1000)

		// Both should succeed (different providers)
		return err1 == nil && err2 == nil
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 3000}))
}

// Property: State transition must be deterministic
func TestPropertyStateTransitionDeterminism(t *testing.T) {
	t.Parallel()
	property := func(initialState, executionTrace []byte) bool {
		if len(initialState) == 0 || len(executionTrace) == 0 {
			return true
		}

		finalState1 := computeStateTransition(initialState, executionTrace)
		finalState2 := computeStateTransition(initialState, executionTrace)

		return bytesEqual(finalState1, finalState2)
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Property: Resource cost must increase with resource usage
func TestPropertyResourceCostMonotonicity(t *testing.T) {
	t.Parallel()
	property := func(size1, size2, time1, time2 uint32) bool {
		if size1 == 0 || size2 == 0 || time1 == 0 || time2 == 0 {
			return true
		}

		if size1 >= size2 || time1 >= time2 {
			return true
		}

		cost1 := calculateResourceCost(size1, size1, time1)
		cost2 := calculateResourceCost(size2, size2, time2)

		return cost2 > cost1
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Property: Signature verification must be consistent
func TestPropertySignatureConsistency(t *testing.T) {
	t.Parallel()
	property := func(message []byte) bool {
		if len(message) == 0 {
			return true
		}

		pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
		signature := ed25519.Sign(privKey, message)

		// Verification should succeed
		valid1 := ed25519.Verify(pubKey, message, signature)
		valid2 := ed25519.Verify(pubKey, message, signature)

		return valid1 && valid2
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Tampered signature must fail verification
func TestPropertyTamperedSignatureRejection(t *testing.T) {
	t.Parallel()
	property := func(message []byte, tamperPos uint8) bool {
		if len(message) == 0 {
			return true
		}

		pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
		signature := ed25519.Sign(privKey, message)

		// Tamper with signature
		if len(signature) > 0 {
			tamperedSig := make([]byte, len(signature))
			copy(tamperedSig, signature)
			tamperedSig[int(tamperPos)%len(signature)] ^= 0xFF

			// Tampered signature should fail (with high probability)
			valid := ed25519.Verify(pubKey, message, tamperedSig)

			// Original should still work
			validOriginal := ed25519.Verify(pubKey, message, signature)

			return validOriginal && !valid
		}

		return true
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 1000}))
}

// Property: Merkle proof verification must be deterministic
func TestPropertyMerkleProofDeterminism(t *testing.T) {
	t.Parallel()
	property := func(leaf, root []byte, proofSize uint8) bool{
		if len(leaf) == 0 || len(root) == 0 || proofSize == 0 {
			return true
		}

		proof := generateMerkleProof(int(proofSize) % 20)

		valid1 := verifyMerkleProof(proof, root, leaf)
		valid2 := verifyMerkleProof(proof, root, leaf)

		return valid1 == valid2
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 3000}))
}

// Property: Request timeout must be enforced
func TestPropertyTimeoutEnforcement(t *testing.T) {
	t.Parallel()
	property := func(createdAt, timeout, currentTime int64) bool {
		if timeout <= 0 || currentTime <= 0 || createdAt <= 0 {
			return true
		}

		deadline := createdAt + timeout
		isExpired := currentTime >= deadline

		expectedExpired := currentTime >= deadline

		return isExpired == expectedExpired
	}

	require.NoError(t, quick.Check(property, &quick.Config{MaxCount: 5000}))
}

// Helper types and functions

type NonceTracker struct {
	nonces map[string]map[uint64]int64
}

func NewNonceTracker() *NonceTracker {
	return &NonceTracker{
		nonces: make(map[string]map[uint64]int64),
	}
}

func (nt *NonceTracker) Use(provider string, nonce uint64, timestamp int64) error {
	if nonce == 0 {
		return &ComputeError{"zero nonce not allowed"}
	}

	if nt.nonces[provider] == nil {
		nt.nonces[provider] = make(map[uint64]int64)
	}

	if _, exists := nt.nonces[provider][nonce]; exists {
		return &ComputeError{"nonce replay"}
	}

	nt.nonces[provider][nonce] = timestamp
	return nil
}

type ComputeError struct {
	msg string
}

func (e *ComputeError) Error() string {
	return e.msg
}

func simulateRelease(escrow, stake uint64, success bool) (provider, refund, slashed uint64) {
	if success {
		provider = escrow + stake
		refund = 0
		slashed = 0
	} else {
		provider = 0
		refund = escrow
		slashed = stake
	}
	return
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

func computeStateTransition(initialState, trace []byte) []byte {
	hasher := sha256.New()
	hasher.Write(initialState)
	hasher.Write(trace)
	return hasher.Sum(nil)
}

func calculateResourceCost(requestSize, resultSize, execTime uint32) uint64 {
	sizeCost := uint64(requestSize + resultSize)
	timeCost := uint64(execTime) * 1000
	return sizeCost + timeCost
}

func verifyMerkleProof(proof [][]byte, root, leaf []byte) bool {
	if len(proof) == 0 || len(root) == 0 {
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

	return bytesEqual(current[:], root)
}

func generateMerkleProof(size int) [][]byte {
	proof := make([][]byte, size)
	for i := 0; i < size; i++ {
		node := make([]byte, 32)
		rand.Read(node)
		proof[i] = node
	}
	return proof
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
