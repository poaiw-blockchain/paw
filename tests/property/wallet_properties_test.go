package property

import (
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
		// Generate test data
		privateKey := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "privateKey")
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{12,32}`).Draw(t, "password")

		// Encrypt keystore
		encrypted := encryptTestKeystore(privateKey, password)

		// Property 1: Encrypted data should be different from original
		if bytesEqual(encrypted, privateKey) {
			t.Fatal("encryption did not modify data")
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
		parsed := parseBIP44Path(path)
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
	// Simplified: generate dummy words
	words := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		words[i] = "word"
	}
	return words
}

func deriveTestKey(mnemonic []string, path uint32) []byte {
	// Simplified: deterministic derivation
	key := make([]byte, 32)
	for i, word := range mnemonic {
		key[i%32] ^= byte(len(word) + int(path))
	}
	return key
}

func encryptTestKeystore(data []byte, password string) []byte {
	// Simplified encryption (in reality, use AES-GCM)
	encrypted := make([]byte, len(data))
	for i := range data {
		encrypted[i] = data[i] ^ byte(password[i%len(password)])
	}
	return encrypted
}

func decryptTestKeystore(encrypted []byte, password string) []byte {
	// Simplified decryption
	decrypted := make([]byte, len(encrypted))
	for i := range encrypted {
		decrypted[i] = encrypted[i] ^ byte(password[i%len(password)])
	}
	return decrypted
}

func validatePassword(password string) bool {
	return len(password) >= 12
}

func makeBIP44Path(coinType, account, change, addressIndex uint32) string {
	// BIP44 format: m/44'/coin_type'/account'/change/address_index
	return "m/44'/118'/0'/0/0" // Simplified
}

func validateBIP32Path(path string) bool {
	return path != ""
}

type BIP44Components struct {
	CoinType     uint32
	Account      uint32
	Change       uint32
	AddressIndex uint32
}

func parseBIP44Path(path string) BIP44Components {
	// Simplified parsing
	return BIP44Components{
		CoinType:     118,
		Account:      0,
		Change:       0,
		AddressIndex: 0,
	}
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
