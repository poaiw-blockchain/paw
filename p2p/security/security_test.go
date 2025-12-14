package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite provides comprehensive P2P security testing
type SecurityTestSuite struct {
	suite.Suite
	authenticator *MessageAuthenticator
	encryptor     *MessageEncryptor
	publicKey     ed25519.PublicKey
	privateKey    ed25519.PrivateKey
}

func (s *SecurityTestSuite) SetupTest() {
	// Generate keys for testing
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	s.Require().NoError(err)

	s.publicKey = publicKey
	s.privateKey = privateKey
	s.authenticator = NewMessageAuthenticator(privateKey)

	s.encryptor, err = NewMessageEncryptor()
	s.Require().NoError(err)
}

// Test authentication failures
func (s *SecurityTestSuite) TestAuthenticationFailures() {
	t := s.T()

	// Test 1: Invalid signature
	payload := []byte("test message")
	msg, err := s.authenticator.SignMessage(payload, "peer-1")
	require.NoError(t, err)

	// Add peer key
	s.authenticator.AddPeerKey("peer-1", s.publicKey)

	// Corrupt signature
	msg.Signature[0] ^= 0xFF

	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrInvalidSignature)

	// Test 2: Unknown peer
	msg, err = s.authenticator.SignMessage(payload, "unknown-peer")
	require.NoError(t, err)

	err = s.authenticator.VerifyMessage(msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown peer")

	// Test 3: Expired message
	msg, err = s.authenticator.SignMessage(payload, "peer-1")
	require.NoError(t, err)

	msg.Timestamp = time.Now().Unix() - 400 // 400 seconds old
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrExpiredMessage)

	// Test 4: Future message (clock skew attack)
	msg, err = s.authenticator.SignMessage(payload, "peer-1")
	require.NoError(t, err)

	msg.Timestamp = time.Now().Unix() + 120 // 2 minutes in future
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrExpiredMessage)
}

// Test replay attack prevention
func (s *SecurityTestSuite) TestReplayAttackPrevention() {
	t := s.T()

	payload := []byte("test message")
	peerID := "peer-1"

	// Add peer key
	s.authenticator.AddPeerKey(peerID, s.publicKey)

	// Sign and verify first time (should succeed)
	msg, err := s.authenticator.SignMessage(payload, peerID)
	require.NoError(t, err)

	err = s.authenticator.VerifyMessage(msg)
	require.NoError(t, err)

	// Try to verify same message again (replay attack)
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrReplayAttack)

	// Sign new message with same payload (should succeed - different nonce)
	msg2, err := s.authenticator.SignMessage(payload, peerID)
	require.NoError(t, err)

	err = s.authenticator.VerifyMessage(msg2)
	require.NoError(t, err)
}

// Test connection rate limiting
func (s *SecurityTestSuite) TestConnectionRateLimiting() {
	t := s.T()

	// Create rate limiter
	limiter := NewRateLimiter(10, 20) // 10/sec, burst of 20

	// Burst should be allowed
	for i := 0; i < 20; i++ {
		allowed := limiter.Allow()
		require.True(t, allowed, "burst requests should be allowed")
	}

	// Next request should be denied
	allowed := limiter.Allow()
	require.False(t, allowed, "requests beyond burst should be denied")

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should allow a few more
	count := 0
	for i := 0; i < 5; i++ {
		if limiter.Allow() {
			count++
		}
	}
	require.Greater(t, count, 0, "should allow some requests after refill")
}

// Test malformed message handling
func (s *SecurityTestSuite) TestMalformedMessageHandling() {
	t := s.T()

	peerID := "peer-1"
	s.authenticator.AddPeerKey(peerID, s.publicKey)

	// Test empty payload
	msg, err := s.authenticator.SignMessage([]byte{}, peerID)
	require.NoError(t, err)
	err = s.authenticator.VerifyMessage(msg)
	require.NoError(t, err, "empty payload should be allowed if properly signed")

	// Test nil signature
	msg.Signature = nil
	err = s.authenticator.VerifyMessage(msg)
	require.Error(t, err)

	// Test corrupted payload
	msg, err = s.authenticator.SignMessage([]byte("original"), peerID)
	require.NoError(t, err)

	msg.Payload = []byte("modified")
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrInvalidSignature)
}

// Test message encryption and decryption
func (s *SecurityTestSuite) TestMessageEncryption() {
	t := s.T()

	// Create second encryptor for peer
	peerEncryptor, err := NewMessageEncryptor()
	require.NoError(t, err)

	// Exchange public keys
	s.encryptor.AddPeerKey("peer-1", peerEncryptor.GetPublicKey())
	peerEncryptor.AddPeerKey("self", s.encryptor.GetPublicKey())

	// Encrypt message
	plaintext := []byte("secret message")
	encrypted, err := s.encryptor.Encrypt(plaintext, "peer-1")
	require.NoError(t, err)
	require.NotEqual(t, plaintext, encrypted.Ciphertext)

	// Decrypt message
	decrypted, err := peerEncryptor.Decrypt(&EncryptedMessage{
		Ciphertext: encrypted.Ciphertext,
		Nonce:      encrypted.Nonce,
		PeerID:     "self",
		Ephemeral:  encrypted.Ephemeral,
	})
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

// Test encryption with unknown peer
func (s *SecurityTestSuite) TestEncryptionUnknownPeer() {
	t := s.T()

	plaintext := []byte("test")

	// Try to encrypt for unknown peer
	_, err := s.encryptor.Encrypt(plaintext, "unknown-peer")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown peer")
}

// Test HMAC authentication
func (s *SecurityTestSuite) TestHMACAuthentication() {
	t := s.T()

	hmacAuth := NewHMACAuthenticator()

	// Add shared key
	sharedKey := make([]byte, 32)
	_, err := rand.Read(sharedKey)
	require.NoError(t, err)

	hmacAuth.AddSharedKey("peer-1", sharedKey)

	// Authenticate message
	message := []byte("test message")
	mac, err := hmacAuth.AuthenticateHMAC(message, "peer-1")
	require.NoError(t, err)
	require.NotEmpty(t, mac)

	// Verify HMAC
	err = hmacAuth.VerifyHMAC(message, mac, "peer-1")
	require.NoError(t, err)

	// Verify fails with wrong message
	wrongMessage := []byte("wrong message")
	err = hmacAuth.VerifyHMAC(wrongMessage, mac, "peer-1")
	require.ErrorIs(t, err, ErrInvalidHMAC)

	// Verify fails with wrong MAC
	wrongMAC := make([]byte, len(mac))
	copy(wrongMAC, mac)
	wrongMAC[0] ^= 0xFF

	err = hmacAuth.VerifyHMAC(message, wrongMAC, "peer-1")
	require.ErrorIs(t, err, ErrInvalidHMAC)
}

// Test nonce cleanup
func (s *SecurityTestSuite) TestNonceCleanup() {
	t := s.T()

	// Create tracker with short max age
	tracker := NewNonceTracker(100*time.Millisecond, 50*time.Millisecond)

	// Mark nonce as used
	nonce := uint64(12345)
	allowed := tracker.CheckAndMark(nonce)
	require.True(t, allowed)

	// Immediate replay should fail
	allowed = tracker.CheckAndMark(nonce)
	require.False(t, allowed)

	// Wait for cleanup
	time.Sleep(150 * time.Millisecond)

	// Nonce should be cleaned up and allowed again
	allowed = tracker.CheckAndMark(nonce)
	require.True(t, allowed, "nonce should be cleaned up after max age")
}

// Test concurrent authentication
func (s *SecurityTestSuite) TestConcurrentAuthentication() {
	t := s.T()

	// Create multiple peer authenticators
	numPeers := 10
	authenticators := make([]*MessageAuthenticator, numPeers)
	for i := 0; i < numPeers; i++ {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		authenticators[i] = NewMessageAuthenticator(privateKey)
	}

	// Exchange keys
	for i, auth := range authenticators {
		peerID := string(rune('A' + i))
		for j, peerAuth := range authenticators {
			if i != j {
				otherID := string(rune('A' + j))
				auth.AddPeerKey(otherID, peerAuth.publicKey)
			}
		}
	}

	// Send messages concurrently
	done := make(chan bool, numPeers)

	for i, auth := range authenticators {
		go func(authIndex int, authenticator *MessageAuthenticator) {
			defer func() { done <- true }()

			peerID := string(rune('A' + authIndex))

			for j := 0; j < 100; j++ {
				payload := []byte("concurrent test message")

				// Sign
				msg, err := authenticator.SignMessage(payload, peerID)
				require.NoError(t, err)

				// Verify with random peer
				peerAuth := authenticators[(authIndex+1)%numPeers]
				err = peerAuth.VerifyMessage(msg)
				require.NoError(t, err)
			}
		}(i, auth)
	}

	// Wait for all
	for i := 0; i < numPeers; i++ {
		<-done
	}
}

// Test key removal
func (s *SecurityTestSuite) TestKeyRemoval() {
	t := s.T()

	peerID := "peer-1"
	s.authenticator.AddPeerKey(peerID, s.publicKey)

	// Should work with key
	payload := []byte("test")
	msg, err := s.authenticator.SignMessage(payload, peerID)
	require.NoError(t, err)
	err = s.authenticator.VerifyMessage(msg)
	require.NoError(t, err)

	// Remove key
	s.authenticator.RemovePeerKey(peerID)

	// Should fail without key
	msg, err = s.authenticator.SignMessage(payload, peerID)
	require.NoError(t, err)
	err = s.authenticator.VerifyMessage(msg)
	require.Error(t, err)

	// Test encryption key removal
	peerEncryptor, err := NewMessageEncryptor()
	require.NoError(t, err)

	s.encryptor.AddPeerKey(peerID, peerEncryptor.GetPublicKey())

	// Should work
	_, err = s.encryptor.Encrypt(payload, peerID)
	require.NoError(t, err)

	// Remove encryption key
	s.encryptor.RemovePeerKey(peerID)

	// Should fail
	_, err = s.encryptor.Encrypt(payload, peerID)
	require.Error(t, err)
}

// Test signature data integrity
func (s *SecurityTestSuite) TestSignatureDataIntegrity() {
	t := s.T()

	peerID := "peer-1"
	s.authenticator.AddPeerKey(peerID, s.publicKey)

	payload := []byte("test message")
	msg, err := s.authenticator.SignMessage(payload, peerID)
	require.NoError(t, err)

	// Verify original
	err = s.authenticator.VerifyMessage(msg)
	require.NoError(t, err)

	// Modify each field and verify it fails

	// Modify payload
	originalPayload := msg.Payload
	msg.Payload = []byte("modified")
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrInvalidSignature)
	msg.Payload = originalPayload

	// Modify peer ID
	originalPeerID := msg.PeerID
	msg.PeerID = "different-peer"
	err = s.authenticator.VerifyMessage(msg)
	require.Error(t, err) // Will fail due to unknown peer or invalid signature
	msg.PeerID = originalPeerID

	// Modify timestamp
	originalTimestamp := msg.Timestamp
	msg.Timestamp = originalTimestamp + 1
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrInvalidSignature)
	msg.Timestamp = originalTimestamp

	// Modify nonce
	originalNonce := msg.Nonce
	msg.Nonce = originalNonce + 1
	err = s.authenticator.VerifyMessage(msg)
	require.ErrorIs(t, err, ErrInvalidSignature)
	msg.Nonce = originalNonce
}

// Test large message encryption
func (s *SecurityTestSuite) TestLargeMessageEncryption() {
	t := s.T()

	// Create peer encryptor
	peerEncryptor, err := NewMessageEncryptor()
	require.NoError(t, err)

	// Exchange keys
	s.encryptor.AddPeerKey("peer-1", peerEncryptor.GetPublicKey())
	peerEncryptor.AddPeerKey("self", s.encryptor.GetPublicKey())

	// Test with large payload (1 MB)
	largePayload := make([]byte, 1024*1024)
	_, err = rand.Read(largePayload)
	require.NoError(t, err)

	// Encrypt
	encrypted, err := s.encryptor.Encrypt(largePayload, "peer-1")
	require.NoError(t, err)

	// Decrypt
	decrypted, err := peerEncryptor.Decrypt(&EncryptedMessage{
		Ciphertext: encrypted.Ciphertext,
		Nonce:      encrypted.Nonce,
		PeerID:     "self",
		Ephemeral:  encrypted.Ephemeral,
	})
	require.NoError(t, err)
	require.Equal(t, largePayload, decrypted)
}

func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

// Benchmarks

func BenchmarkMessageSigning(b *testing.B) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	auth := NewMessageAuthenticator(privateKey)
	payload := []byte("benchmark message payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.SignMessage(payload, "peer-1")
	}
}

func BenchmarkMessageVerification(b *testing.B) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	auth := NewMessageAuthenticator(privateKey)
	auth.AddPeerKey("peer-1", publicKey)

	payload := []byte("benchmark message payload")
	msg, _ := auth.SignMessage(payload, "peer-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.VerifyMessage(msg)
	}
}

func BenchmarkMessageEncryption(b *testing.B) {
	enc1, _ := NewMessageEncryptor()
	enc2, _ := NewMessageEncryptor()

	enc1.AddPeerKey("peer-1", enc2.GetPublicKey())

	payload := make([]byte, 1024) // 1 KB payload
	rand.Read(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc1.Encrypt(payload, "peer-1")
	}
}

func BenchmarkMessageDecryption(b *testing.B) {
	enc1, _ := NewMessageEncryptor()
	enc2, _ := NewMessageEncryptor()

	enc1.AddPeerKey("peer-1", enc2.GetPublicKey())
	enc2.AddPeerKey("self", enc1.GetPublicKey())

	payload := make([]byte, 1024)
	rand.Read(payload)

	encrypted, _ := enc1.Encrypt(payload, "peer-1")
	msg := &EncryptedMessage{
		Ciphertext: encrypted.Ciphertext,
		Nonce:      encrypted.Nonce,
		PeerID:     "self",
		Ephemeral:  encrypted.Ephemeral,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc2.Decrypt(msg)
	}
}

func BenchmarkHMACAuthentication(b *testing.B) {
	hmac := NewHMACAuthenticator()
	key := make([]byte, 32)
	rand.Read(key)
	hmac.AddSharedKey("peer-1", key)

	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hmac.AuthenticateHMAC(message, "peer-1")
	}
}

func BenchmarkHMACVerification(b *testing.B) {
	hmac := NewHMACAuthenticator()
	key := make([]byte, 32)
	rand.Read(key)
	hmac.AddSharedKey("peer-1", key)

	message := []byte("benchmark message")
	mac, _ := hmac.AuthenticateHMAC(message, "peer-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hmac.VerifyHMAC(message, mac, "peer-1")
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	limiter := NewRateLimiter(1000, 2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}
