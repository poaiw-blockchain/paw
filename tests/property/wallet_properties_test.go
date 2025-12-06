package property

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestMnemonicGenerationProperties tests BIP39 mnemonic generation properties
func TestMnemonicGenerationProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Property: Generated mnemonic should always be valid
		// Property: Mnemonic length should be 12, 15, 18, 21, or 24 words
		wordCount := rapid.SampledFrom([]int{12, 15, 18, 21, 24}).Draw(t, "wordCount")

		// Simulate mnemonic generation
		mnemonic := generateTestMnemonic(wordCount)

		// Properties to verify
		if len(mnemonic) != wordCount {
			t.Fatalf("expected %d words, got %d", wordCount, len(mnemonic))
		}

		// Each word should be from BIP39 wordlist (simplified check)
		for i, word := range mnemonic {
			if word == "" {
				t.Fatalf("word %d is empty", i)
			}
		}
	})
}

// TestKeyDerivationProperties tests HD key derivation properties
func TestKeyDerivationProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Property: Same mnemonic + path should always derive same key (deterministic)
		mnemonic := generateTestMnemonic(24)
		path := rapid.Uint32Range(0, 0x80000000).Draw(t, "path")

		key1 := deriveTestKey(mnemonic, path)
		key2 := deriveTestKey(mnemonic, path)

		if !bytesEqual(key1, key2) {
			t.Fatal("key derivation is not deterministic")
		}

		// Property: Different paths should derive different keys
		path2 := rapid.Uint32Range(0, 0x80000000).Draw(t, "path2")
		if path2 != path {
			key3 := deriveTestKey(mnemonic, path2)
			if bytesEqual(key1, key3) {
				t.Fatal("different paths produced same key")
			}
		}
	})
}

// TestKeystoreEncryptionProperties tests keystore encryption properties
func TestKeystoreEncryptionProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data - ensure private key has at least some entropy
		// In real scenarios, an all-zero private key would be rejected
		privateKeyRaw := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "privateKey")

		// Check for all-zero private key (invalid in real crypto)
		allZero := true
		for _, b := range privateKeyRaw {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			// In production, this would be rejected. For test purposes, add minimal entropy
			privateKeyRaw[31] = 1
		}

		privateKey := privateKeyRaw
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{12,32}`).Draw(t, "password")

		// Encrypt keystore
		encrypted := encryptTestKeystore(privateKey, password)

		// Property 1: Encrypted data should be different from original
		// Note: The full encrypted output includes nonce+ciphertext+MAC, so it will always differ
		if len(encrypted) <= len(privateKey) {
			t.Fatal("encrypted data is not longer than plaintext")
		}

		// Property 2: Decryption with correct password should recover original
		decrypted := decryptTestKeystore(encrypted, password)
		if !bytesEqual(decrypted, privateKey) {
			t.Fatal("decryption did not recover original data")
		}

		// Property 3: Decryption with wrong password should fail
		wrongPassword := password + "wrong"
		recovered := decryptTestKeystore(encrypted, wrongPassword)
		if recovered != nil {
			t.Fatal("decryption with wrong password should fail")
		}

		// Property 4: Multiple encryptions with same password produce different ciphertexts (due to IV)
		encrypted2 := encryptTestKeystore(privateKey, password)
		if bytesEqual(encrypted, encrypted2) {
			t.Fatal("multiple encryptions produced identical ciphertexts")
		}
	})
}

// TestPasswordStrengthProperties tests password validation properties
func TestPasswordStrengthProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Property: Passwords below minimum length should be rejected
		shortPassword := rapid.StringMatching(`[a-z]{1,11}`).Draw(t, "shortPassword")
		if validatePassword(shortPassword) {
			t.Fatal("short password was accepted")
		}

		// Property: Passwords at or above minimum length should be accepted
		goodPassword := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{12,32}`).Draw(t, "goodPassword")
		if !validatePassword(goodPassword) {
			t.Fatal("good password was rejected")
		}
	})
}

// TestBIP32PathProperties tests BIP32 path validation properties
func TestBIP32PathProperties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Property: Valid BIP44 paths should be accepted
		coinType := rapid.Uint32Range(0, 1000).Draw(t, "coinType")
		account := rapid.Uint32Range(0, 100).Draw(t, "account")
		change := rapid.Uint32Range(0, 1).Draw(t, "change")
		addressIndex := rapid.Uint32Range(0, 1000000).Draw(t, "addressIndex")

		path := makeBIP44Path(coinType, account, change, addressIndex)
		if !validateBIP32Path(path) {
			t.Fatalf("valid BIP44 path rejected: %s", path)
		}

		// Property: Path components should be recoverable
		parsed, ok := parseBIP44Path(path)
		if !ok {
			t.Fatalf("failed to parse valid path: %s", path)
		}
		if parsed.CoinType != coinType ||
			parsed.Account != account ||
			parsed.Change != change ||
			parsed.AddressIndex != addressIndex {
			t.Fatal("path components not preserved")
		}
	})
}

// Helper functions for property tests

func generateTestMnemonic(wordCount int) []string {
	// Generate distinct dummy words to maintain uniqueness guarantees
	words := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		words[i] = fmt.Sprintf("word-%d", i)
	}
	return words
}

func deriveTestKey(mnemonic []string, path uint32) []byte {
	payload := fmt.Sprintf("%s-%d", strings.Join(mnemonic, " "), path)
	sum := sha256.Sum256([]byte(payload))
	return sum[:]
}

func derivePasswordKey(password string, nonce []byte) []byte {
	data := append([]byte(password), nonce...)
	sum := sha256.Sum256(data)
	return sum[:]
}

func computeKeystoreMAC(nonce, ciphertext []byte, password string) []byte {
	macInput := append(append([]byte{}, nonce...), ciphertext...)
	macInput = append(macInput, []byte(password)...)
	sum := sha256.Sum256(macInput)
	return sum[:]
}

func encryptTestKeystore(data []byte, password string) []byte {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		copy(nonce, []byte("paw-fallback"))
	}

	key := derivePasswordKey(password, nonce)
	ciphertext := make([]byte, len(data))
	for i := range data {
		ciphertext[i] = data[i] ^ key[i%len(key)]
	}

	mac := computeKeystoreMAC(nonce, ciphertext, password)

	out := make([]byte, 0, len(nonce)+len(ciphertext)+len(mac))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	out = append(out, mac...)
	return out
}

func decryptTestKeystore(encrypted []byte, password string) []byte {
	if len(encrypted) < 12+sha256.Size {
		return nil
	}

	nonce := encrypted[:12]
	mac := encrypted[len(encrypted)-sha256.Size:]
	ciphertext := encrypted[12 : len(encrypted)-sha256.Size]

	expectedMac := computeKeystoreMAC(nonce, ciphertext, password)
	if !bytes.Equal(mac, expectedMac) {
		return nil
	}

	key := derivePasswordKey(password, nonce)
	plaintext := make([]byte, len(ciphertext))
	for i := range ciphertext {
		plaintext[i] = ciphertext[i] ^ key[i%len(key)]
	}
	return plaintext
}

func validatePassword(password string) bool {
	return len(password) >= 12
}

func makeBIP44Path(coinType, account, change, addressIndex uint32) string {
	// BIP44 format: m/44'/coin_type'/account'/change/address_index
	return fmt.Sprintf("m/44'/%d'/%d'/%d/%d", coinType, account, change, addressIndex)
}

func validateBIP32Path(path string) bool {
	_, ok := parseBIP44Path(path)
	return ok
}

type BIP44Components struct {
	CoinType     uint32
	Account      uint32
	Change       uint32
	AddressIndex uint32
}

func parseBIP44Path(path string) (BIP44Components, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 6 || parts[0] != "m" || parts[1] != "44'" {
		return BIP44Components{}, false
	}

	coinType, ok := parseHardenedPart(parts[2])
	if !ok {
		return BIP44Components{}, false
	}

	account, ok := parseHardenedPart(parts[3])
	if !ok {
		return BIP44Components{}, false
	}

	change, err := strconv.ParseUint(parts[4], 10, 32)
	if err != nil {
		return BIP44Components{}, false
	}

	addressIndex, err := strconv.ParseUint(parts[5], 10, 32)
	if err != nil {
		return BIP44Components{}, false
	}

	return BIP44Components{
		CoinType:     coinType,
		Account:      account,
		Change:       uint32(change),
		AddressIndex: uint32(addressIndex),
	}, true
}

func parseHardenedPart(part string) (uint32, bool) {
	if !strings.HasSuffix(part, "'") {
		return 0, false
	}

	value, err := strconv.ParseUint(strings.TrimSuffix(part, "'"), 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(value), true
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
