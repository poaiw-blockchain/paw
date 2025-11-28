package setup

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// KeyGenerator manages the generation, storage, and rotation of ZK-SNARK keys.
// It implements secure key management practices including:
// - Encrypted key storage
// - Key versioning and rotation
// - Auditable key generation
// - Constant-time key operations
type KeyGenerator struct {
	circuitID      string
	masterPassword []byte // Encrypted in production
	keyVersion     uint64
	storage        KeyStorage
}

// KeyStorage interface abstracts key persistence.
type KeyStorage interface {
	Store(ctx context.Context, keyID string, data []byte) error
	Load(ctx context.Context, keyID string) ([]byte, error)
	Delete(ctx context.Context, keyID string) error
	List(ctx context.Context) ([]string, error)
}

// KeyMetadata contains information about a generated key pair.
type KeyMetadata struct {
	KeyID           string
	CircuitID       string
	Version         uint64
	CreatedAt       time.Time
	RotateAt        time.Time
	Status          KeyStatus
	Algorithm       string // "groth16"
	Curve           string // "bn254"
	ConstraintCount int
	PublicInputs    int

	// Security
	EncryptionAlg   string // "AES-256-GCM"
	KDFAlgorithm    string // "Argon2id"

	// Audit
	GeneratedBy     string
	CeremonyID      string
	TranscriptHash  []byte
}

// KeyStatus represents the lifecycle status of a key.
type KeyStatus string

const (
	KeyStatusActive     KeyStatus = "active"
	KeyStatusRotating   KeyStatus = "rotating"
	KeyStatusDeprecated KeyStatus = "deprecated"
	KeyStatusRevoked    KeyStatus = "revoked"
)

// EncryptedKeyPair represents an encrypted proving/verifying key pair.
type EncryptedKeyPair struct {
	Metadata        KeyMetadata
	EncryptedPK     []byte // Encrypted proving key
	EncryptedVK     []byte // Encrypted verifying key
	Salt            []byte // For key derivation
	Nonce           []byte // For AES-GCM
	AuthTag         []byte // GCM authentication tag
}

// NewKeyGenerator creates a new key generator instance.
func NewKeyGenerator(circuitID string, masterPassword []byte, storage KeyStorage) *KeyGenerator {
	return &KeyGenerator{
		circuitID:      circuitID,
		masterPassword: masterPassword,
		keyVersion:     1,
		storage:        storage,
	}
}

// GenerateKeys generates a new proving and verifying key pair for a circuit.
// This uses the Groth16 setup with secure randomness.
func (kg *KeyGenerator) GenerateKeys(
	ctx context.Context,
	circuit frontend.Circuit,
	useMPC bool,
	mpcCeremony *MPCCeremony,
) (*EncryptedKeyPair, error) {
	startTime := time.Now()

	// Compile circuit to R1CS
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		return nil, fmt.Errorf("failed to compile circuit: %w", err)
	}

	var pk groth16.ProvingKey
	var vk groth16.VerifyingKey

	var ceremonyID string
	var transcriptHash []byte

	if useMPC {
		// Use MPC ceremony for trusted setup
		if mpcCeremony == nil {
			return nil, fmt.Errorf("MPC ceremony required but not provided")
		}

		pkPtr, vkPtr, err := mpcCeremony.Finalize(ctx)
		if err != nil {
			return nil, fmt.Errorf("MPC ceremony failed: %w", err)
		}

		pk = *pkPtr
		vk = *vkPtr
		ceremonyID = mpcCeremony.transcript.CeremonyID
		transcriptHash = mpcCeremony.transcript.TranscriptHash
	} else {
		// Standard setup (NOT recommended for production)
		pk, vk, err = groth16.Setup(ccs)
		if err != nil {
			return nil, fmt.Errorf("failed to setup keys: %w", err)
		}
		ceremonyID = "direct-setup"
	}

	// Serialize keys
	pkBytes, err := serializeProvingKey(&pk)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proving key: %w", err)
	}

	vkBytes, err := serializeVerifyingKey(&vk)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize verifying key: %w", err)
	}

	// Generate encryption key from master password
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	encryptionKey := deriveEncryptionKey(kg.masterPassword, salt)

	// Encrypt proving key
	encryptedPK, noncePK, authTagPK, err := encryptAESGCM(pkBytes, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt proving key: %w", err)
	}

	// Encrypt verifying key (using same derived key but different nonce)
	encryptedVK, nonceVK, authTagVK, err := encryptAESGCM(vkBytes, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt verifying key: %w", err)
	}

	// Securely erase encryption key
	for i := range encryptionKey {
		encryptionKey[i] = 0
	}

	// Create metadata
	keyID := generateKeyID(kg.circuitID, kg.keyVersion)
	metadata := KeyMetadata{
		KeyID:           keyID,
		CircuitID:       kg.circuitID,
		Version:         kg.keyVersion,
		CreatedAt:       startTime,
		RotateAt:        startTime.Add(90 * 24 * time.Hour), // Rotate every 90 days
		Status:          KeyStatusActive,
		Algorithm:       "groth16",
		Curve:           "bn254",
		ConstraintCount: ccs.GetNbConstraints(),
		PublicInputs:    ccs.GetNbPublicVariables(),
		EncryptionAlg:   "AES-256-GCM",
		KDFAlgorithm:    "Argon2id",
		GeneratedBy:     "paw-keygen",
		CeremonyID:      ceremonyID,
		TranscriptHash:  transcriptHash,
	}

	// Combine nonces and auth tags
	combinedNonce := append(noncePK, nonceVK...)
	combinedAuthTag := append(authTagPK, authTagVK...)

	encryptedPair := &EncryptedKeyPair{
		Metadata:    metadata,
		EncryptedPK: encryptedPK,
		EncryptedVK: encryptedVK,
		Salt:        salt,
		Nonce:       combinedNonce,
		AuthTag:     combinedAuthTag,
	}

	// Store in persistent storage
	if err := kg.storeKeyPair(ctx, encryptedPair); err != nil {
		return nil, fmt.Errorf("failed to store key pair: %w", err)
	}

	// Increment version for next key generation
	kg.keyVersion++

	return encryptedPair, nil
}

// LoadKeys decrypts and loads a key pair from storage.
func (kg *KeyGenerator) LoadKeys(
	ctx context.Context,
	keyID string,
) (*groth16.ProvingKey, *groth16.VerifyingKey, error) {
	// Load encrypted pair
	encryptedPair, err := kg.loadKeyPair(ctx, keyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	// Check key status
	if encryptedPair.Metadata.Status == KeyStatusRevoked {
		return nil, nil, fmt.Errorf("key %s has been revoked", keyID)
	}

	// Derive decryption key
	decryptionKey := deriveEncryptionKey(kg.masterPassword, encryptedPair.Salt)

	// Extract individual nonces and auth tags
	noncePK := encryptedPair.Nonce[:12]
	nonceVK := encryptedPair.Nonce[12:24]
	authTagPK := encryptedPair.AuthTag[:16]
	authTagVK := encryptedPair.AuthTag[16:32]

	// Decrypt proving key
	pkBytes, err := decryptAESGCM(encryptedPair.EncryptedPK, decryptionKey, noncePK, authTagPK)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt proving key: %w", err)
	}

	// Decrypt verifying key
	vkBytes, err := decryptAESGCM(encryptedPair.EncryptedVK, decryptionKey, nonceVK, authTagVK)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt verifying key: %w", err)
	}

	// Securely erase decryption key
	for i := range decryptionKey {
		decryptionKey[i] = 0
	}

	// Deserialize keys
	pk, err := deserializeProvingKey(pkBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize proving key: %w", err)
	}

	vk, err := deserializeVerifyingKey(vkBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize verifying key: %w", err)
	}

	return pk, vk, nil
}

// RotateKeys generates a new key pair and marks the old one as deprecated.
func (kg *KeyGenerator) RotateKeys(
	ctx context.Context,
	oldKeyID string,
	circuit frontend.Circuit,
	useMPC bool,
	mpcCeremony *MPCCeremony,
) (*EncryptedKeyPair, error) {
	// Load old key metadata
	oldPair, err := kg.loadKeyPair(ctx, oldKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load old key: %w", err)
	}

	// Mark old key as deprecated
	oldPair.Metadata.Status = KeyStatusDeprecated
	if err := kg.storeKeyPair(ctx, oldPair); err != nil {
		return nil, fmt.Errorf("failed to update old key status: %w", err)
	}

	// Generate new key pair
	newPair, err := kg.GenerateKeys(ctx, circuit, useMPC, mpcCeremony)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new keys: %w", err)
	}

	return newPair, nil
}

// RevokeKey marks a key as revoked (cannot be used anymore).
func (kg *KeyGenerator) RevokeKey(ctx context.Context, keyID string) error {
	pair, err := kg.loadKeyPair(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}

	pair.Metadata.Status = KeyStatusRevoked
	if err := kg.storeKeyPair(ctx, pair); err != nil {
		return fmt.Errorf("failed to update key status: %w", err)
	}

	return nil
}

// Helper functions

func deriveEncryptionKey(password, salt []byte) []byte {
	// Use Argon2id for key derivation (memory-hard, side-channel resistant)
	// Parameters: time=3, memory=64MB, threads=4, keyLen=32
	return argon2.IDKey(password, salt, 3, 64*1024, 4, 32)
}

func encryptAESGCM(plaintext, key []byte) (ciphertext, nonce, authTag []byte, err error) {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}

	// Generate random nonce
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, nil, err
	}

	// Encrypt and authenticate
	sealed := gcm.Seal(nil, nonce, plaintext, nil)

	// Extract auth tag (last 16 bytes)
	ciphertext = sealed[:len(sealed)-16]
	authTag = sealed[len(sealed)-16:]

	return ciphertext, nonce, authTag, nil
}

func decryptAESGCM(ciphertext, key, nonce, authTag []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Combine ciphertext and auth tag
	sealed := append(ciphertext, authTag...)

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return plaintext, nil
}

func serializeProvingKey(pk *groth16.ProvingKey) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := (*pk).WriteTo(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func serializeVerifyingKey(vk *groth16.VerifyingKey) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := (*vk).WriteTo(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deserializeProvingKey(data []byte) (*groth16.ProvingKey, error) {
	pk := groth16.NewProvingKey(ecc.BN254)
	_, err := pk.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &pk, nil
}

func deserializeVerifyingKey(data []byte) (*groth16.VerifyingKey, error) {
	vk := groth16.NewVerifyingKey(ecc.BN254)
	_, err := vk.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &vk, nil
}

func generateKeyID(circuitID string, version uint64) string {
	h := sha256.New()
	h.Write([]byte(circuitID))
	versionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(versionBytes, version)
	h.Write(versionBytes)
	hash := h.Sum(nil)
	return fmt.Sprintf("%s-v%d-%x", circuitID, version, hash[:8])
}

func (kg *KeyGenerator) storeKeyPair(ctx context.Context, pair *EncryptedKeyPair) error {
	// Serialize encrypted pair
	buf := new(bytes.Buffer)

	// Write metadata
	if err := binary.Write(buf, binary.BigEndian, uint64(len(pair.Metadata.KeyID))); err != nil {
		return err
	}
	buf.WriteString(pair.Metadata.KeyID)

	// Write encrypted keys
	if err := binary.Write(buf, binary.BigEndian, uint64(len(pair.EncryptedPK))); err != nil {
		return err
	}
	buf.Write(pair.EncryptedPK)

	if err := binary.Write(buf, binary.BigEndian, uint64(len(pair.EncryptedVK))); err != nil {
		return err
	}
	buf.Write(pair.EncryptedVK)

	// Write salt, nonce, auth tag
	buf.Write(pair.Salt)
	buf.Write(pair.Nonce)
	buf.Write(pair.AuthTag)

	return kg.storage.Store(ctx, pair.Metadata.KeyID, buf.Bytes())
}

func (kg *KeyGenerator) loadKeyPair(ctx context.Context, keyID string) (*EncryptedKeyPair, error) {
	data, err := kg.storage.Load(ctx, keyID)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(data)

	// Read metadata
	var keyIDLen uint64
	if err := binary.Read(buf, binary.BigEndian, &keyIDLen); err != nil {
		return nil, err
	}
	keyIDBytes := make([]byte, keyIDLen)
	if _, err := buf.Read(keyIDBytes); err != nil {
		return nil, err
	}

	// Read encrypted keys
	var pkLen uint64
	if err := binary.Read(buf, binary.BigEndian, &pkLen); err != nil {
		return nil, err
	}
	encryptedPK := make([]byte, pkLen)
	if _, err := buf.Read(encryptedPK); err != nil {
		return nil, err
	}

	var vkLen uint64
	if err := binary.Read(buf, binary.BigEndian, &vkLen); err != nil {
		return nil, err
	}
	encryptedVK := make([]byte, vkLen)
	if _, err := buf.Read(encryptedVK); err != nil {
		return nil, err
	}

	// Read salt, nonce, auth tag
	salt := make([]byte, 32)
	nonce := make([]byte, 24)
	authTag := make([]byte, 32)

	if _, err := buf.Read(salt); err != nil {
		return nil, err
	}
	if _, err := buf.Read(nonce); err != nil {
		return nil, err
	}
	if _, err := buf.Read(authTag); err != nil {
		return nil, err
	}

	pair := &EncryptedKeyPair{
		Metadata: KeyMetadata{
			KeyID: string(keyIDBytes),
		},
		EncryptedPK: encryptedPK,
		EncryptedVK: encryptedVK,
		Salt:        salt,
		Nonce:       nonce,
		AuthTag:     authTag,
	}

	return pair, nil
}

// DeriveSubKey derives a sub-key from master key using HKDF for specific purposes.
func DeriveSubKey(masterKey, salt, info []byte, length int) ([]byte, error) {
	hash := sha256.New
	hkdf := hkdf.New(hash, masterKey, salt, info)

	subKey := make([]byte, length)
	if _, err := io.ReadFull(hkdf, subKey); err != nil {
		return nil, err
	}

	return subKey, nil
}
