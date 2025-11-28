package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// Task 141: P2P Message Authentication

var (
	ErrInvalidSignature = errors.New("invalid message signature")
	ErrInvalidHMAC      = errors.New("invalid HMAC")
	ErrReplayAttack     = errors.New("replay attack detected")
	ErrExpiredMessage   = errors.New("message expired")
)

// MessageAuthenticator handles message authentication
type MessageAuthenticator struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	peerKeys   map[string]ed25519.PublicKey
	nonces     *NonceTracker
}

// NonceTracker prevents replay attacks
type NonceTracker struct {
	used      map[uint64]time.Time
	maxAge    time.Duration
	cleanupInterval time.Duration
}

// NewMessageAuthenticator creates a new message authenticator
func NewMessageAuthenticator(privateKey ed25519.PrivateKey) *MessageAuthenticator {
	return &MessageAuthenticator{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
		peerKeys:   make(map[string]ed25519.PublicKey),
		nonces:     NewNonceTracker(5 * time.Minute, 1 * time.Minute),
	}
}

// NewNonceTracker creates a new nonce tracker
func NewNonceTracker(maxAge, cleanupInterval time.Duration) *NonceTracker {
	nt := &NonceTracker{
		used:            make(map[uint64]time.Time),
		maxAge:          maxAge,
		cleanupInterval: cleanupInterval,
	}

	// Start cleanup goroutine
	go nt.cleanup()

	return nt
}

// CheckAndMark checks if nonce is used and marks it
func (nt *NonceTracker) CheckAndMark(nonce uint64) bool {
	now := time.Now()

	// Check if nonce was used
	if usedAt, exists := nt.used[nonce]; exists {
		// Check if still within max age
		if now.Sub(usedAt) < nt.maxAge {
			return false // Replay attack
		}
	}

	// Mark nonce as used
	nt.used[nonce] = now
	return true
}

// cleanup removes old nonces
func (nt *NonceTracker) cleanup() {
	ticker := time.NewTicker(nt.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for nonce, usedAt := range nt.used {
			if now.Sub(usedAt) > nt.maxAge {
				delete(nt.used, nonce)
			}
		}
	}
}

// AuthenticatedMessage wraps a message with authentication
type AuthenticatedMessage struct {
	Payload   []byte
	Signature []byte
	PeerID    string
	Timestamp int64
	Nonce     uint64
}

// SignMessage signs a message
func (ma *MessageAuthenticator) SignMessage(payload []byte, peerID string) (*AuthenticatedMessage, error) {
	// Generate nonce
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create message
	msg := &AuthenticatedMessage{
		Payload:   payload,
		PeerID:    peerID,
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
	}

	// Create signature data
	sigData := ma.getSignatureData(msg)

	// Sign
	signature := ed25519.Sign(ma.privateKey, sigData)
	msg.Signature = signature

	return msg, nil
}

// VerifyMessage verifies a message signature
func (ma *MessageAuthenticator) VerifyMessage(msg *AuthenticatedMessage) error {
	// Check timestamp (max 5 minutes old)
	age := time.Now().Unix() - msg.Timestamp
	if age > 300 || age < -60 {
		return ErrExpiredMessage
	}

	// Check nonce for replay attacks
	if !ma.nonces.CheckAndMark(msg.Nonce) {
		return ErrReplayAttack
	}

	// Get peer public key
	peerKey, exists := ma.peerKeys[msg.PeerID]
	if !exists {
		return fmt.Errorf("unknown peer: %s", msg.PeerID)
	}

	// Verify signature
	sigData := ma.getSignatureData(msg)
	if !ed25519.Verify(peerKey, sigData, msg.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// getSignatureData creates the data to be signed
func (ma *MessageAuthenticator) getSignatureData(msg *AuthenticatedMessage) []byte {
	data := make([]byte, 0, len(msg.Payload)+len(msg.PeerID)+16)
	data = append(data, msg.Payload...)
	data = append(data, []byte(msg.PeerID)...)

	timestampBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBuf, uint64(msg.Timestamp))
	data = append(data, timestampBuf...)

	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, msg.Nonce)
	data = append(data, nonceBuf...)

	return data
}

// AddPeerKey adds a peer's public key
func (ma *MessageAuthenticator) AddPeerKey(peerID string, publicKey ed25519.PublicKey) {
	ma.peerKeys[peerID] = publicKey
}

// RemovePeerKey removes a peer's public key
func (ma *MessageAuthenticator) RemovePeerKey(peerID string) {
	delete(ma.peerKeys, peerID)
}

// generateNonce generates a random nonce
func generateNonce() (uint64, error) {
	var nonce uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &nonce)
	return nonce, err
}

// Task 142: P2P Message Encryption

// MessageEncryptor handles message encryption
type MessageEncryptor struct {
	privateKey [32]byte
	publicKey  [32]byte
	peerKeys   map[string][32]byte
}

// NewMessageEncryptor creates a new message encryptor
func NewMessageEncryptor() (*MessageEncryptor, error) {
	// Generate Curve25519 key pair
	publicKey, privateKey, err := generateCurve25519KeyPair()
	if err != nil {
		return nil, err
	}

	return &MessageEncryptor{
		privateKey: privateKey,
		publicKey:  publicKey,
		peerKeys:   make(map[string][32]byte),
	}, nil
}

// generateCurve25519KeyPair generates a Curve25519 key pair
func generateCurve25519KeyPair() ([32]byte, [32]byte, error) {
	var privateKey, publicKey [32]byte

	if _, err := rand.Read(privateKey[:]); err != nil {
		return publicKey, privateKey, err
	}

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey, privateKey, nil
}

// EncryptedMessage represents an encrypted message
type EncryptedMessage struct {
	Ciphertext []byte
	Nonce      []byte
	PeerID     string
	Ephemeral  []byte // Ephemeral public key
}

// Encrypt encrypts a message for a specific peer
func (me *MessageEncryptor) Encrypt(plaintext []byte, peerID string) (*EncryptedMessage, error) {
	// Get peer public key
	peerKey, exists := me.peerKeys[peerID]
	if !exists {
		return nil, fmt.Errorf("unknown peer: %s", peerID)
	}

	// Derive shared secret using ECDH
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &me.privateKey, &peerKey)

	// Derive encryption key using HKDF
	encKey := make([]byte, 32)
	kdf := hkdf.New(sha256.New, sharedSecret[:], []byte("p2p-encryption"), []byte(peerID))
	if _, err := io.ReadFull(kdf, encKey); err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedMessage{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		PeerID:     peerID,
		Ephemeral:  me.publicKey[:],
	}, nil
}

// Decrypt decrypts a message from a peer
func (me *MessageEncryptor) Decrypt(msg *EncryptedMessage) ([]byte, error) {
	// Get peer public key
	peerKey, exists := me.peerKeys[msg.PeerID]
	if !exists {
		return nil, fmt.Errorf("unknown peer: %s", msg.PeerID)
	}

	// Derive shared secret
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &me.privateKey, &peerKey)

	// Derive encryption key
	encKey := make([]byte, 32)
	kdf := hkdf.New(sha256.New, sharedSecret[:], []byte("p2p-encryption"), []byte(msg.PeerID))
	if _, err := io.ReadFull(kdf, encKey); err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, msg.Nonce, msg.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// AddPeerKey adds a peer's public key for encryption
func (me *MessageEncryptor) AddPeerKey(peerID string, publicKey [32]byte) {
	me.peerKeys[peerID] = publicKey
}

// RemovePeerKey removes a peer's public key
func (me *MessageEncryptor) RemovePeerKey(peerID string) {
	delete(me.peerKeys, peerID)
}

// GetPublicKey returns the encryptor's public key
func (me *MessageEncryptor) GetPublicKey() [32]byte {
	return me.publicKey
}

// HMAC-based message authentication (lightweight alternative)

// HMACAuthenticator uses HMAC for message authentication
type HMACAuthenticator struct {
	sharedKeys map[string][]byte
}

// NewHMACAuthenticator creates a new HMAC authenticator
func NewHMACAuthenticator() *HMACAuthenticator {
	return &HMACAuthenticator{
		sharedKeys: make(map[string][]byte),
	}
}

// AuthenticateHMAC creates HMAC for a message
func (ha *HMACAuthenticator) AuthenticateHMAC(message []byte, peerID string) ([]byte, error) {
	key, exists := ha.sharedKeys[peerID]
	if !exists {
		return nil, fmt.Errorf("no shared key for peer: %s", peerID)
	}

	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil), nil
}

// VerifyHMAC verifies HMAC of a message
func (ha *HMACAuthenticator) VerifyHMAC(message, mac []byte, peerID string) error {
	expectedMAC, err := ha.AuthenticateHMAC(message, peerID)
	if err != nil {
		return err
	}

	if !hmac.Equal(mac, expectedMAC) {
		return ErrInvalidHMAC
	}

	return nil
}

// AddSharedKey adds a shared key for a peer
func (ha *HMACAuthenticator) AddSharedKey(peerID string, key []byte) {
	ha.sharedKeys[peerID] = key
}

// RemoveSharedKey removes a shared key
func (ha *HMACAuthenticator) RemoveSharedKey(peerID string) {
	delete(ha.sharedKeys, peerID)
}
