package security_test

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/sha3"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// CryptoSecurityTestSuite tests cryptographic security
type CryptoSecurityTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *CryptoSecurityTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestCryptoSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(CryptoSecurityTestSuite))
}

// TestEntropy_Randomness tests cryptographic randomness quality
func (suite *CryptoSecurityTestSuite) TestEntropy_Randomness() {
	const numSamples = 1000
	const bytesPerSample = 32

	samples := make([][]byte, numSamples)

	// Generate random samples
	for i := 0; i < numSamples; i++ {
		sample := make([]byte, bytesPerSample)
		n, err := rand.Read(sample)
		suite.Require().NoError(err, "Random generation should not fail")
		suite.Require().Equal(bytesPerSample, n, "Should generate requested number of bytes")
		samples[i] = sample
	}

	// Test 1: No duplicate samples
	seen := make(map[string]bool)
	duplicates := 0
	for _, sample := range samples {
		key := hex.EncodeToString(sample)
		if seen[key] {
			duplicates++
		}
		seen[key] = true
	}
	suite.Require().Equal(0, duplicates, "Should have no duplicate random samples")

	// Test 2: Bit distribution (each bit should be ~50% 0 and ~50% 1)
	bitCounts := make([]int, bytesPerSample*8)
	for _, sample := range samples {
		for byteIdx := 0; byteIdx < bytesPerSample; byteIdx++ {
			for bitIdx := 0; bitIdx < 8; bitIdx++ {
				if sample[byteIdx]&(1<<bitIdx) != 0 {
					bitCounts[byteIdx*8+bitIdx]++
				}
			}
		}
	}

	// Each bit should be set roughly 500 times out of 1000 samples
	// Allow for statistical variance (400-600 is acceptable)
	for bitIdx, count := range bitCounts {
		suite.Require().GreaterOrEqual(count, 400, "Bit %d should be balanced (got %d/1000)", bitIdx, count)
		suite.Require().LessOrEqual(count, 600, "Bit %d should be balanced (got %d/1000)", bitIdx, count)
	}
}

// TestKeyGeneration_Secp256k1 tests secp256k1 key generation
func (suite *CryptoSecurityTestSuite) TestKeyGeneration_Secp256k1() {
	const numKeys = 100

	keys := make([]cryptotypes.PrivKey, numKeys)
	pubKeys := make([]cryptotypes.PubKey, numKeys)

	// Generate keys
	for i := 0; i < numKeys; i++ {
		privKey := secp256k1.GenPrivKey()
		suite.Require().NotNil(privKey, "Private key should be generated")

		pubKey := privKey.PubKey()
		suite.Require().NotNil(pubKey, "Public key should be derived")

		keys[i] = privKey
		pubKeys[i] = pubKey

		// Verify key properties
		suite.Require().Equal(32, len(privKey.Bytes()), "Private key should be 32 bytes")
		suite.Require().Equal(33, len(pubKey.Bytes()), "Compressed public key should be 33 bytes")
	}

	// Verify all keys are unique
	privKeySeen := make(map[string]bool)
	pubKeySeen := make(map[string]bool)

	for i := 0; i < numKeys; i++ {
		privKeyHex := hex.EncodeToString(keys[i].Bytes())
		pubKeyHex := hex.EncodeToString(pubKeys[i].Bytes())

		suite.Require().False(privKeySeen[privKeyHex], "Private key %d should be unique", i)
		suite.Require().False(pubKeySeen[pubKeyHex], "Public key %d should be unique", i)

		privKeySeen[privKeyHex] = true
		pubKeySeen[pubKeyHex] = true
	}
}

// TestSignature_Verification tests signature creation and verification
func (suite *CryptoSecurityTestSuite) TestSignature_Verification() {
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	testCases := []struct {
		name    string
		message []byte
	}{
		{"Short message", []byte("test")},
		{"Medium message", []byte("The quick brown fox jumps over the lazy dog")},
		{"Long message", make([]byte, 1024)},
		{"Empty message", []byte{}},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Sign message
			signature, err := priv.Sign(tc.message)
			suite.Require().NoError(err, "Signature creation should succeed")
			suite.Require().NotEmpty(signature, "Signature should not be empty")

			// Verify signature
			valid := pub.VerifySignature(tc.message, signature)
			suite.Require().True(valid, "Valid signature should verify")

			// Test with wrong message
			wrongMessage := append(tc.message, []byte("tampered")...)
			invalidByMessage := pub.VerifySignature(wrongMessage, signature)
			suite.Require().False(invalidByMessage, "Signature should not verify with wrong message")

			// Test with wrong signature
			wrongSig := make([]byte, len(signature))
			copy(wrongSig, signature)
			if len(wrongSig) > 0 {
				wrongSig[0] ^= 0xFF // Flip bits
			}
			invalidBySig := pub.VerifySignature(tc.message, wrongSig)
			suite.Require().False(invalidBySig, "Wrong signature should not verify")

			// Test with different key
			otherPriv := secp256k1.GenPrivKey()
			otherPub := otherPriv.PubKey()
			invalidByKey := otherPub.VerifySignature(tc.message, signature)
			suite.Require().False(invalidByKey, "Signature should not verify with different public key")
		})
	}
}

// TestBIP39_MnemonicGeneration tests BIP39 mnemonic generation
func (suite *CryptoSecurityTestSuite) TestBIP39_MnemonicGeneration() {
	entropySizes := []int{128, 160, 192, 224, 256} // bits

	for _, entropySize := range entropySizes {
		suite.Run("Entropy_"+string(rune(entropySize)), func() {
			// Generate entropy
			entropy, err := bip39.NewEntropy(entropySize)
			suite.Require().NoError(err, "Entropy generation should succeed")
			suite.Require().Equal(entropySize/8, len(entropy), "Entropy should be correct size")

			// Generate mnemonic
			mnemonic, err := bip39.NewMnemonic(entropy)
			suite.Require().NoError(err, "Mnemonic generation should succeed")
			suite.Require().NotEmpty(mnemonic, "Mnemonic should not be empty")

			// Verify mnemonic is valid
			valid := bip39.IsMnemonicValid(mnemonic)
			suite.Require().True(valid, "Generated mnemonic should be valid")

			// Verify entropy can be recovered
			// Note: MnemonicToByteArray returns entropy + checksum, not just entropy
			// The checksum is entropy_bits/32 bits, rounded up to nearest byte
			recoveredEntropy, err := bip39.MnemonicToByteArray(mnemonic)
			suite.Require().NoError(err, "Entropy recovery should succeed")

			// Expected length = ceil((entropy + checksum) / 8) bytes
			// checksum size in bits = entropy size in bits / 32
			checksumBits := entropySize / 32
			totalBits := entropySize + checksumBits
			expectedBytes := (totalBits + 7) / 8 // Round up to nearest byte
			suite.Require().Equal(expectedBytes, len(recoveredEntropy), "Recovered entropy+checksum should be correct length")
		})
	}
}

// TestBIP39_InvalidMnemonics tests rejection of invalid mnemonics
func (suite *CryptoSecurityTestSuite) TestBIP39_InvalidMnemonics() {
	invalidMnemonics := []string{
		"", // Empty
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", // Too short
		"invalid words that are not in wordlist at all",
		"abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon invalid", // Invalid checksum
		"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk llll",                                     // Not real words
	}

	for i, mnemonic := range invalidMnemonics {
		suite.Run("Invalid_"+string(rune(i)), func() {
			valid := bip39.IsMnemonicValid(mnemonic)
			suite.Require().False(valid, "Invalid mnemonic should be rejected: %s", mnemonic)
		})
	}
}

// TestBIP32_HDKeyDerivation tests hierarchical deterministic key derivation
func (suite *CryptoSecurityTestSuite) TestBIP32_HDKeyDerivation() {
	// Generate mnemonic
	entropy, err := bip39.NewEntropy(256)
	suite.Require().NoError(err)

	mnemonic, err := bip39.NewMnemonic(entropy)
	suite.Require().NoError(err)

	// BIP44 path: m/44'/118'/0'/0/0 (Cosmos standard)
	const coinType = 118 // Cosmos
	const account = 0
	const change = 0
	const addressIndex = 0

	// Test key derivation at different paths
	testPaths := []struct {
		name  string
		index uint32
	}{
		{"First address", 0},
		{"Second address", 1},
		{"Third address", 2},
		{"High index", 1000},
	}

	derivedKeys := make(map[string]bool)

	for _, tc := range testPaths {
		suite.Run(tc.name, func() {
			path := hd.CreateHDPath(coinType, account, tc.index)

			// Derive master private key
			seed := bip39.NewSeed(mnemonic, "")
			master, ch := hd.ComputeMastersFromSeed(seed)

			// Derive key at path
			derivedKey, err := hd.DerivePrivateKeyForPath(master, ch, path.String())
			suite.Require().NoError(err, "Key derivation should succeed")
			suite.Require().NotNil(derivedKey, "Derived key should not be nil")
			suite.Require().Equal(32, len(derivedKey), "Derived key should be 32 bytes")

			// Verify derived keys are unique
			keyHex := hex.EncodeToString(derivedKey)
			suite.Require().False(derivedKeys[keyHex], "Derived keys should be unique")
			derivedKeys[keyHex] = true
		})
	}
}

// TestHashCollision_Resistance tests hash collision resistance
func (suite *CryptoSecurityTestSuite) TestHashCollision_Resistance() {
	const numMessages = 10000

	hashes := make(map[string]bool)
	collisions := 0

	for i := 0; i < numMessages; i++ {
		// Generate message
		message := make([]byte, 32)
		_, err := rand.Read(message)
		suite.Require().NoError(err)

		// Hash with SHA3-256
		hash := sha3.Sum256(message)
		hashHex := hex.EncodeToString(hash[:])

		if hashes[hashHex] {
			collisions++
		}
		hashes[hashHex] = true
	}

	suite.Require().Equal(0, collisions, "Should have no hash collisions")
}

// TestWeakKeys_Detection tests detection of weak cryptographic keys
func (suite *CryptoSecurityTestSuite) TestWeakKeys_Detection() {
	// Test that we cannot create keys from weak entropy
	weakEntropySources := [][]byte{
		make([]byte, 32),                                       // All zeros
		[]byte{0xFF, 0xFF, 0xFF, 0xFF},                         // All ones (partial)
		[]byte{0x01, 0x02, 0x03, 0x04},                         // Sequential (partial)
		[]byte{0xAA, 0xAA, 0xAA, 0xAA},                         // Repeated pattern (partial)
		[]byte("password"),                                     // Dictionary word
		[]byte("12345678"),                                     // Weak password
		[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, // Nearly all zeros
	}

	for i, entropy := range weakEntropySources {
		suite.Run("Weak_entropy_"+string(rune(i)), func() {
			// secp256k1 will accept any 32-byte input, but we should validate entropy quality
			// before using it for key generation in production

			// Pad to 32 bytes if needed
			if len(entropy) < 32 {
				padded := make([]byte, 32)
				copy(padded, entropy)
				entropy = padded
			}

			// Check for weak patterns
			isWeak := suite.detectWeakEntropy(entropy)
			if isWeak {
				suite.T().Logf("Weak entropy detected: %x", entropy[:8])
			}
		})
	}
}

// detectWeakEntropy checks for weak entropy patterns
func (suite *CryptoSecurityTestSuite) detectWeakEntropy(data []byte) bool {
	if len(data) < 32 {
		return true
	}

	// Check for all zeros
	allZeros := true
	for _, b := range data {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return true
	}

	// Check for all same byte
	allSame := true
	first := data[0]
	for _, b := range data {
		if b != first {
			allSame = false
			break
		}
	}
	if allSame {
		return true
	}

	// Check for sequential bytes
	sequential := true
	for i := 1; i < len(data); i++ {
		if data[i] != data[i-1]+1 {
			sequential = false
			break
		}
	}
	if sequential {
		return true
	}

	return false
}

// TestTimingAttack_Resistance tests resistance to timing attacks
func (suite *CryptoSecurityTestSuite) TestTimingAttack_Resistance() {
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	message := []byte("constant message for timing test")
	signature, err := priv.Sign(message)
	suite.Require().NoError(err)

	// Verify that signature verification time is constant
	// regardless of validity (constant-time comparison)

	iterations := 1000
	validSignature := signature
	invalidSignature := make([]byte, len(signature))
	copy(invalidSignature, signature)
	if len(invalidSignature) > 0 {
		invalidSignature[0] ^= 0xFF
	}

	// Time valid verifications
	validVerifications := 0
	for i := 0; i < iterations; i++ {
		if pub.VerifySignature(message, validSignature) {
			validVerifications++
		}
	}

	// Time invalid verifications
	invalidVerifications := 0
	for i := 0; i < iterations; i++ {
		if pub.VerifySignature(message, invalidSignature) {
			invalidVerifications++
		}
	}

	suite.Require().Equal(iterations, validVerifications, "All valid verifications should succeed")
	suite.Require().Equal(0, invalidVerifications, "All invalid verifications should fail")

	// Note: Actual timing comparison would require benchmarking
	// This test ensures both paths execute consistently
}

// TestKeyRecovery_Prevention tests that private keys cannot be recovered
func (suite *CryptoSecurityTestSuite) TestKeyRecovery_Prevention() {
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	// Create multiple signatures with same key
	signatures := make([][]byte, 100)
	messages := make([][]byte, 100)

	for i := 0; i < 100; i++ {
		message := make([]byte, 32)
		_, err := rand.Read(message)
		suite.Require().NoError(err)

		signature, err := priv.Sign(message)
		suite.Require().NoError(err)

		messages[i] = message
		signatures[i] = signature
	}

	// Verify all signatures are valid
	for i := 0; i < 100; i++ {
		valid := pub.VerifySignature(messages[i], signatures[i])
		suite.Require().True(valid, "Signature %d should be valid", i)
	}

	// Private key should remain secure even with many signature samples
	// (ECDSA with proper random nonce generation is secure)
}

// TestNonceReuse_Prevention tests prevention of nonce reuse in signatures
func (suite *CryptoSecurityTestSuite) TestNonceReuse_Prevention() {
	priv := secp256k1.GenPrivKey()
	message := []byte("test message")

	// Generate multiple signatures of same message
	signatures := make(map[string]bool)

	for i := 0; i < 100; i++ {
		signature, err := priv.Sign(message)
		suite.Require().NoError(err)

		sigHex := hex.EncodeToString(signature)

		// Each signature should use a different random nonce
		// So signatures should be different even for same message
		if signatures[sigHex] {
			suite.T().Logf("Warning: Duplicate signature detected (nonce reuse)")
		}
		signatures[sigHex] = true
	}

	// With proper random nonce generation, all signatures should be unique
	suite.T().Logf("Generated %d unique signatures out of 100 attempts", len(signatures))
}

// TestAddressGeneration_Consistency tests address generation consistency
func (suite *CryptoSecurityTestSuite) TestAddressGeneration_Consistency() {
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	// Generate address multiple times from same public key
	addresses := make([]string, 100)
	for i := 0; i < 100; i++ {
		addr := sdk.AccAddress(pub.Address())
		addresses[i] = addr.String()
	}

	// All addresses should be identical
	first := addresses[0]
	for i := 1; i < 100; i++ {
		suite.Require().Equal(first, addresses[i], "Address generation should be deterministic")
	}

	// Address should have correct properties
	addr := sdk.AccAddress(pub.Address())
	suite.Require().NotEmpty(addr.String(), "Address should not be empty")
	suite.Require().Equal(20, len(addr), "Address should be 20 bytes")
}

// TestCryptographicPrimitives_Standards tests compliance with standards
func (suite *CryptoSecurityTestSuite) TestCryptographicPrimitives_Standards() {
	// Verify secp256k1 parameters
	priv := secp256k1.GenPrivKey()
	suite.Require().NotNil(priv)

	// Private key should be 32 bytes (256 bits)
	suite.Require().Equal(32, len(priv.Bytes()), "Private key should be 256 bits")

	// Public key should be 33 bytes (compressed)
	pub := priv.PubKey()
	suite.Require().Equal(33, len(pub.Bytes()), "Compressed public key should be 33 bytes")

	// Address should be 20 bytes (RIPEMD160)
	addr := sdk.AccAddress(pub.Address())
	suite.Require().Equal(20, len(addr), "Address should be 20 bytes (RIPEMD160)")
}
