package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ProofOfWork implements hashcash-style proof-of-work for spam prevention
type ProofOfWork struct {
	difficulty int // Number of leading zeros required
}

// Challenge represents a PoW challenge
type Challenge struct {
	Resource   string `json:"resource"`   // The resource being requested (e.g., address)
	Timestamp  int64  `json:"timestamp"`  // Unix timestamp
	Difficulty int    `json:"difficulty"` // Number of leading zeros required
	ExpiresAt  int64  `json:"expires_at"` // Challenge expiration time
}

// Solution represents a PoW solution
type Solution struct {
	Resource  string `json:"resource"`  // Must match challenge
	Timestamp int64  `json:"timestamp"` // Must match challenge
	Nonce     string `json:"nonce"`     // The proof-of-work nonce
}

const (
	// DefaultDifficulty is the default number of leading zeros required (4 = ~16^4 = 65536 attempts)
	DefaultDifficulty = 4

	// ChallengeTTL is how long a challenge is valid (5 minutes)
	ChallengeTTL = 5 * time.Minute

	// MaxNonceLength prevents DoS via extremely long nonces
	MaxNonceLength = 64
)

// NewProofOfWork creates a new proof-of-work validator
func NewProofOfWork(difficulty int) *ProofOfWork {
	if difficulty < 1 {
		difficulty = DefaultDifficulty
	}
	if difficulty > 8 {
		// Cap at 8 to prevent excessive client-side computation
		difficulty = 8
	}
	return &ProofOfWork{
		difficulty: difficulty,
	}
}

// GenerateChallenge creates a new PoW challenge for a resource
func (p *ProofOfWork) GenerateChallenge(resource string) *Challenge {
	now := time.Now()
	return &Challenge{
		Resource:   resource,
		Timestamp:  now.Unix(),
		Difficulty: p.difficulty,
		ExpiresAt:  now.Add(ChallengeTTL).Unix(),
	}
}

// VerifySolution verifies a proof-of-work solution
func (p *ProofOfWork) VerifySolution(challenge *Challenge, solution *Solution) error {
	// Validate solution matches challenge
	if solution.Resource != challenge.Resource {
		return fmt.Errorf("solution resource does not match challenge")
	}

	if solution.Timestamp != challenge.Timestamp {
		return fmt.Errorf("solution timestamp does not match challenge")
	}

	// Check challenge hasn't expired
	now := time.Now().Unix()
	if now > challenge.ExpiresAt {
		return fmt.Errorf("challenge has expired")
	}

	// Check challenge is not from the future (clock skew tolerance: 1 minute)
	if challenge.Timestamp > now+60 {
		return fmt.Errorf("challenge timestamp is in the future")
	}

	// Validate nonce length to prevent DoS
	if len(solution.Nonce) > MaxNonceLength {
		return fmt.Errorf("nonce too long")
	}

	// Compute hash
	data := fmt.Sprintf("%s:%d:%s", solution.Resource, solution.Timestamp, solution.Nonce)
	hash := sha256.Sum256([]byte(data))
	hashHex := hex.EncodeToString(hash[:])

	// Check if hash has required number of leading zeros
	requiredPrefix := strings.Repeat("0", challenge.Difficulty)
	if !strings.HasPrefix(hashHex, requiredPrefix) {
		return fmt.Errorf("invalid proof-of-work: hash does not meet difficulty requirement")
	}

	return nil
}

// ComputeSolution computes a valid proof-of-work solution (for testing/client implementation)
// This is intentionally inefficient brute-force - clients should implement this
func (p *ProofOfWork) ComputeSolution(challenge *Challenge) (*Solution, error) {
	requiredPrefix := strings.Repeat("0", challenge.Difficulty)

	// Try different nonces until we find a valid one
	for nonce := 0; nonce < 10000000; nonce++ {
		nonceStr := strconv.Itoa(nonce)
		data := fmt.Sprintf("%s:%d:%s", challenge.Resource, challenge.Timestamp, nonceStr)
		hash := sha256.Sum256([]byte(data))
		hashHex := hex.EncodeToString(hash[:])

		if strings.HasPrefix(hashHex, requiredPrefix) {
			return &Solution{
				Resource:  challenge.Resource,
				Timestamp: challenge.Timestamp,
				Nonce:     nonceStr,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to compute solution after maximum attempts")
}

// GetDifficulty returns the current difficulty level
func (p *ProofOfWork) GetDifficulty() int {
	return p.difficulty
}

// SetDifficulty updates the difficulty level
func (p *ProofOfWork) SetDifficulty(difficulty int) {
	if difficulty < 1 {
		difficulty = 1
	}
	if difficulty > 8 {
		difficulty = 8
	}
	p.difficulty = difficulty
}

// EstimateWorkTime estimates the time to solve a challenge at current difficulty
// Assumes ~1M hashes per second (typical for client-side JS)
func EstimateWorkTime(difficulty int) time.Duration {
	// Average attempts needed: 16^difficulty / 2
	attempts := (1 << (difficulty * 4)) / 2

	// Assume 1M hashes/sec
	hashesPerSecond := 1000000
	seconds := attempts / hashesPerSecond

	return time.Duration(seconds) * time.Second
}
