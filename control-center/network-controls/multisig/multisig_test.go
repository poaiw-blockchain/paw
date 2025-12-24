package multisig

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"
)

func TestNewVerifier(t *testing.T) {
	// Generate test keys
	pub1, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	pub2, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	tests := []struct {
		name        string
		config      *MultiSigConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &MultiSigConfig{
				Threshold: 2,
				Signers: []SignerInfo{
					{ID: "admin-1", PublicKey: pub1, Role: "admin"},
					{ID: "admin-2", PublicKey: pub2, Role: "admin"},
					{ID: "admin-3", PublicKey: pub1, Role: "admin"},
				},
			},
			expectError: false,
		},
		{
			name: "threshold zero",
			config: &MultiSigConfig{
				Threshold: 0,
				Signers: []SignerInfo{
					{ID: "admin-1", PublicKey: pub1, Role: "admin"},
				},
			},
			expectError: true,
		},
		{
			name: "threshold exceeds signers",
			config: &MultiSigConfig{
				Threshold: 3,
				Signers: []SignerInfo{
					{ID: "admin-1", PublicKey: pub1, Role: "admin"},
					{ID: "admin-2", PublicKey: pub2, Role: "admin"},
				},
			},
			expectError: true,
		},
		{
			name: "invalid public key",
			config: &MultiSigConfig{
				Threshold: 1,
				Signers: []SignerInfo{
					{ID: "admin-1", PublicKey: "invalid-hex", Role: "admin"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVerifier(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	// Generate key pairs
	pub1, priv1, _ := GenerateKeyPair()
	pub2, priv2, _ := GenerateKeyPair()
	pub3, priv3, _ := GenerateKeyPair()

	config := &MultiSigConfig{
		Threshold:               2,
		SignatureTimeoutMinutes: 60,
		Signers: []SignerInfo{
			{ID: "admin-1", PublicKey: pub1, Role: "admin"},
			{ID: "admin-2", PublicKey: pub2, Role: "admin"},
			{ID: "admin-3", PublicKey: pub3, Role: "admin"},
		},
	}

	verifier, err := NewVerifier(config)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	message := CreateSigningMessage("emergency_halt", map[string]interface{}{
		"modules": "dex,oracle",
		"reason":  "security incident",
	})
	nonce := "test-nonce-12345"

	t.Run("valid 2-of-3 signatures", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)
		sig2, _ := Sign(priv2, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
				{SignerID: "admin-2", Signature: sig2, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid but got invalid: %+v", result)
		}
		if result.ValidSignatures != 2 {
			t.Errorf("expected 2 valid signatures, got %d", result.ValidSignatures)
		}
	})

	t.Run("all 3 signatures valid", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)
		sig2, _ := Sign(priv2, message, nonce)
		sig3, _ := Sign(priv3, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
				{SignerID: "admin-2", Signature: sig2, Timestamp: time.Now()},
				{SignerID: "admin-3", Signature: sig3, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid but got invalid")
		}
		if result.ValidSignatures != 3 {
			t.Errorf("expected 3 valid signatures, got %d", result.ValidSignatures)
		}
	})

	t.Run("insufficient signatures", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid but got valid")
		}
		if result.ValidSignatures != 1 {
			t.Errorf("expected 1 valid signature, got %d", result.ValidSignatures)
		}
	})

	t.Run("wrong message", func(t *testing.T) {
		sig1, _ := Sign(priv1, "wrong message", nonce)
		sig2, _ := Sign(priv2, "wrong message", nonce)

		multiSig := &MultiSignature{
			Message: message, // original message
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
				{SignerID: "admin-2", Signature: sig2, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid but got valid")
		}
		if result.ValidSignatures != 0 {
			t.Errorf("expected 0 valid signatures, got %d", result.ValidSignatures)
		}
	})

	t.Run("unknown signer", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)

		_, unknownPriv, _ := GenerateKeyPair()
		unknownSig, _ := Sign(unknownPriv, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
				{SignerID: "unknown", Signature: unknownSig, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid but got valid")
		}
		if result.ValidSignatures != 1 {
			t.Errorf("expected 1 valid signature, got %d", result.ValidSignatures)
		}
		if len(result.InvalidSigners) != 1 || result.InvalidSigners[0] != "unknown" {
			t.Errorf("expected unknown in invalid signers")
		}
	})

	t.Run("duplicate signer", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid (duplicate should not count twice)")
		}
		if result.ValidSignatures != 1 {
			t.Errorf("expected 1 valid signature (duplicate ignored), got %d", result.ValidSignatures)
		}
	})

	t.Run("expired signature", func(t *testing.T) {
		sig1, _ := Sign(priv1, message, nonce)
		sig2, _ := Sign(priv2, message, nonce)

		multiSig := &MultiSignature{
			Message: message,
			Nonce:   nonce,
			Signatures: []Signature{
				{SignerID: "admin-1", Signature: sig1, Timestamp: time.Now().Add(-2 * time.Hour)}, // expired
				{SignerID: "admin-2", Signature: sig2, Timestamp: time.Now()},
			},
		}

		result, err := verifier.Verify(multiSig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid (one signature expired)")
		}
		if result.ValidSignatures != 1 {
			t.Errorf("expected 1 valid signature, got %d", result.ValidSignatures)
		}
	})
}

func TestCreateSigningMessage(t *testing.T) {
	message := CreateSigningMessage("emergency_halt", map[string]interface{}{
		"modules": "dex,oracle",
		"reason":  "security incident",
	})

	// Message should be deterministic (sorted keys)
	expected := "operation=emergency_halt;modules=dex,oracle;reason=security incident"
	if message != expected {
		t.Errorf("expected %q, got %q", expected, message)
	}

	// Same params should produce same message
	message2 := CreateSigningMessage("emergency_halt", map[string]interface{}{
		"reason":  "security incident",
		"modules": "dex,oracle",
	})
	if message != message2 {
		t.Error("messages should be identical regardless of param order")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Public key should be 64 hex chars (32 bytes)
	if len(pub) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(pub))
	}

	// Should be valid hex
	pubBytes, err := hex.DecodeString(pub)
	if err != nil {
		t.Errorf("invalid hex: %v", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		t.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(pubBytes))
	}

	// Private key should be valid
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("expected %d bytes for private key, got %d", ed25519.PrivateKeySize, len(priv))
	}
}

func TestSign(t *testing.T) {
	_, priv, _ := GenerateKeyPair()
	message := "test message"
	nonce := "test-nonce"

	sig, err := Sign(priv, message, nonce)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Signature should be base64 encoded
	if sig == "" {
		t.Error("signature should not be empty")
	}

	// Same inputs should produce same signature
	sig2, _ := Sign(priv, message, nonce)
	if sig != sig2 {
		t.Error("signatures should be deterministic")
	}

	// Different nonce should produce different signature
	sig3, _ := Sign(priv, message, "different-nonce")
	if sig == sig3 {
		t.Error("different nonce should produce different signature")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Threshold != 2 {
		t.Errorf("expected threshold 2, got %d", config.Threshold)
	}
	if len(config.Signers) != 3 {
		t.Errorf("expected 3 signers, got %d", len(config.Signers))
	}
	if config.SignatureTimeoutMinutes != 60 {
		t.Errorf("expected 60 minute timeout, got %d", config.SignatureTimeoutMinutes)
	}
}
