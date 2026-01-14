// Package multisig provides multi-signature verification for emergency operations.
// This ensures that critical operations like emergency halts require approval from
// multiple authorized signers (N-of-M threshold signatures).
package multisig

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MultiSigConfig defines the multi-signature requirements
type MultiSigConfig struct {
	// Threshold is the minimum number of signatures required (N)
	Threshold int `json:"threshold"`

	// Signers is the list of authorized signer public keys
	Signers []SignerInfo `json:"signers"`

	// SignatureTimeoutMinutes is how long signatures remain valid
	SignatureTimeoutMinutes int `json:"signature_timeout_minutes"`
}

// SignerInfo contains information about an authorized signer
type SignerInfo struct {
	// ID is a human-readable identifier (e.g., "admin-1", "security-lead")
	ID string `json:"id"`

	// PublicKey is the Ed25519 public key in hex encoding
	PublicKey string `json:"public_key"`

	// Role describes the signer's role
	Role string `json:"role"`
}

// Signature represents a single signature in a multi-sig
type Signature struct {
	// SignerID is the ID of the signer
	SignerID string `json:"signer_id"`

	// Signature is the base64-encoded Ed25519 signature
	Signature string `json:"signature"`

	// Timestamp is when the signature was created
	Timestamp time.Time `json:"timestamp"`
}

// MultiSignature represents a collection of signatures for verification
type MultiSignature struct {
	// Message is the canonical message that was signed
	Message string `json:"message"`

	// Signatures is the list of individual signatures
	Signatures []Signature `json:"signatures"`

	// Nonce prevents replay attacks
	Nonce string `json:"nonce"`
}

// Verifier handles multi-signature verification
type Verifier struct {
	config *MultiSigConfig
	// publicKeys maps signer ID to decoded public key
	publicKeys map[string]ed25519.PublicKey
}

// NewVerifier creates a new multi-signature verifier
func NewVerifier(config *MultiSigConfig) (*Verifier, error) {
	if config.Threshold <= 0 {
		return nil, fmt.Errorf("threshold must be positive")
	}
	if config.Threshold > len(config.Signers) {
		return nil, fmt.Errorf("threshold (%d) cannot exceed number of signers (%d)",
			config.Threshold, len(config.Signers))
	}

	v := &Verifier{
		config:     config,
		publicKeys: make(map[string]ed25519.PublicKey),
	}

	// Decode and store public keys
	for _, signer := range config.Signers {
		pubKeyBytes, err := hex.DecodeString(signer.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("invalid public key for signer %s: %w", signer.ID, err)
		}
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid public key size for signer %s", signer.ID)
		}
		v.publicKeys[signer.ID] = ed25519.PublicKey(pubKeyBytes)
	}

	return v, nil
}

// VerifyResult contains the result of multi-signature verification
type VerifyResult struct {
	Valid             bool     `json:"valid"`
	ValidSignatures   int      `json:"valid_signatures"`
	RequiredThreshold int      `json:"required_threshold"`
	ValidSigners      []string `json:"valid_signers"`
	InvalidSigners    []string `json:"invalid_signers"`
	Errors            []string `json:"errors,omitempty"`
}

// Verify checks if a multi-signature meets the threshold requirement
func (v *Verifier) Verify(multiSig *MultiSignature) (*VerifyResult, error) {
	result := &VerifyResult{
		RequiredThreshold: v.config.Threshold,
		ValidSigners:      make([]string, 0),
		InvalidSigners:    make([]string, 0),
		Errors:            make([]string, 0),
	}

	// Track which signers have been seen to prevent duplicate signatures
	seenSigners := make(map[string]bool)

	// Compute the canonical message hash
	messageHash := ComputeMessageHash(multiSig.Message, multiSig.Nonce)

	timeout := time.Duration(v.config.SignatureTimeoutMinutes) * time.Minute
	now := time.Now()

	for _, sig := range multiSig.Signatures {
		// Check for duplicate signer
		if seenSigners[sig.SignerID] {
			result.Errors = append(result.Errors,
				fmt.Sprintf("duplicate signature from signer %s", sig.SignerID))
			continue
		}
		seenSigners[sig.SignerID] = true

		// Get the signer's public key
		pubKey, exists := v.publicKeys[sig.SignerID]
		if !exists {
			result.InvalidSigners = append(result.InvalidSigners, sig.SignerID)
			result.Errors = append(result.Errors,
				fmt.Sprintf("unknown signer: %s", sig.SignerID))
			continue
		}

		// Check signature timestamp
		if timeout > 0 && now.Sub(sig.Timestamp) > timeout {
			result.InvalidSigners = append(result.InvalidSigners, sig.SignerID)
			result.Errors = append(result.Errors,
				fmt.Sprintf("signature from %s has expired", sig.SignerID))
			continue
		}

		// Decode and verify signature
		sigBytes, err := base64.StdEncoding.DecodeString(sig.Signature)
		if err != nil {
			result.InvalidSigners = append(result.InvalidSigners, sig.SignerID)
			result.Errors = append(result.Errors,
				fmt.Sprintf("invalid signature encoding from %s", sig.SignerID))
			continue
		}

		if !ed25519.Verify(pubKey, messageHash, sigBytes) {
			result.InvalidSigners = append(result.InvalidSigners, sig.SignerID)
			result.Errors = append(result.Errors,
				fmt.Sprintf("signature verification failed for %s", sig.SignerID))
			continue
		}

		result.ValidSigners = append(result.ValidSigners, sig.SignerID)
		result.ValidSignatures++
	}

	// Check if threshold is met
	result.Valid = result.ValidSignatures >= v.config.Threshold

	return result, nil
}

// ComputeMessageHash computes the canonical hash for signing
func ComputeMessageHash(message, nonce string) []byte {
	h := sha256.New()
	h.Write([]byte(message))
	h.Write([]byte(nonce))
	return h.Sum(nil)
}

// CreateSigningMessage creates a canonical message for signing
func CreateSigningMessage(operation string, params map[string]interface{}) string {
	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical message
	var parts []string
	parts = append(parts, fmt.Sprintf("operation=%s", operation))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, params[k]))
	}

	return strings.Join(parts, ";")
}

// Sign creates a signature for the given message using a private key
func Sign(privateKey ed25519.PrivateKey, message, nonce string) (string, error) {
	messageHash := ComputeMessageHash(message, nonce)
	signature := ed25519.Sign(privateKey, messageHash)
	return base64.StdEncoding.EncodeToString(signature), nil
}

// ParseMultiSigFromJSON parses a multi-signature from JSON
func ParseMultiSigFromJSON(data []byte) (*MultiSignature, error) {
	var multiSig MultiSignature
	if err := json.Unmarshal(data, &multiSig); err != nil {
		return nil, fmt.Errorf("failed to parse multi-signature: %w", err)
	}
	return &multiSig, nil
}

// DefaultConfig returns a default multi-sig configuration (2-of-3)
func DefaultConfig() *MultiSigConfig {
	return &MultiSigConfig{
		Threshold:               2,
		SignatureTimeoutMinutes: 60, // 1 hour
		Signers: []SignerInfo{
			{ID: "admin-1", Role: "network-admin", PublicKey: ""},
			{ID: "admin-2", Role: "security-lead", PublicKey: ""},
			{ID: "admin-3", Role: "on-call-engineer", PublicKey: ""},
		},
	}
}

// GenerateKeyPair generates a new Ed25519 key pair for testing
func GenerateKeyPair() (publicKey string, privateKey ed25519.PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(pub), priv, nil
}
