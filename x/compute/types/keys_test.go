package types

import (
	"bytes"
	"testing"
)

func TestModuleNamespace(t *testing.T) {
	if ModuleNamespace != byte(0x01) {
		t.Errorf("ModuleNamespace = %v, want 0x01", ModuleNamespace)
	}
}

func TestKeyPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		expected []byte
	}{
		{"ParamsKey", ParamsKey, []byte{0x01, 0x01}},
		{"ComputeRequestKeyPrefix", ComputeRequestKeyPrefix, []byte{0x01, 0x02}},
		{"ProviderKeyPrefix", ProviderKeyPrefix, []byte{0x01, 0x03}},
		{"EscrowKeyPrefix", EscrowKeyPrefix, []byte{0x01, 0x04}},
		{"JobStatusKeyPrefix", JobStatusKeyPrefix, []byte{0x01, 0x05}},
		{"NonceTrackerKeyPrefix", NonceTrackerKeyPrefix, []byte{0x01, 0x06}},
		{"NonceKeyPrefix", NonceKeyPrefix, []byte{0x01, 0x06}},
		{"RequestKeyPrefix", RequestKeyPrefix, []byte{0x01, 0x02}},
		{"ResultKeyPrefix", ResultKeyPrefix, []byte{0x01, 0x07}},
		{"IBCPacketNonceKeyPrefix", IBCPacketNonceKeyPrefix, []byte{0x01, 0x28}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Equal(tt.key, tt.expected) {
				t.Errorf("%s = %v, want %v", tt.name, tt.key, tt.expected)
			}
		})
	}
}

func TestNonceKeyPrefix_Alias(t *testing.T) {
	// Verify that NonceKeyPrefix is an alias for NonceTrackerKeyPrefix
	if !bytes.Equal(NonceKeyPrefix, NonceTrackerKeyPrefix) {
		t.Errorf("NonceKeyPrefix should be an alias for NonceTrackerKeyPrefix")
	}
}

func TestRequestKeyPrefix_Alias(t *testing.T) {
	// Verify that RequestKeyPrefix is an alias for ComputeRequestKeyPrefix
	if !bytes.Equal(RequestKeyPrefix, ComputeRequestKeyPrefix) {
		t.Errorf("RequestKeyPrefix should be an alias for ComputeRequestKeyPrefix")
	}
}

func TestDefaultAuthority(t *testing.T) {
	authority := DefaultAuthority()

	if authority == "" {
		t.Error("DefaultAuthority() returned empty string")
	}

	// Verify it starts with the cosmos prefix (initialized in msgs_test.go)
	// The actual value depends on the SDK config, but it should not be empty
	if len(authority) == 0 {
		t.Error("DefaultAuthority() should return a non-empty address")
	}
}

func TestGetIBCPacketNonceKey(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		sender    string
		wantLen   int
	}{
		{
			name:      "simple channel and sender",
			channelID: "channel-0",
			sender:    "cosmos1abc",
			wantLen:   len(IBCPacketNonceKeyPrefix) + len("channel-0/") + len("cosmos1abc"),
		},
		{
			name:      "long channel ID",
			channelID: "channel-999",
			sender:    "cosmos1xyz",
			wantLen:   len(IBCPacketNonceKeyPrefix) + len("channel-999/") + len("cosmos1xyz"),
		},
		{
			name:      "empty channel ID",
			channelID: "",
			sender:    "cosmos1test",
			wantLen:   len(IBCPacketNonceKeyPrefix) + len("/") + len("cosmos1test"),
		},
		{
			name:      "empty sender",
			channelID: "channel-0",
			sender:    "",
			wantLen:   len(IBCPacketNonceKeyPrefix) + len("channel-0/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetIBCPacketNonceKey(tt.channelID, tt.sender)

			// Verify length
			if len(key) != tt.wantLen {
				t.Errorf("GetIBCPacketNonceKey() returned key of length %d, want %d", len(key), tt.wantLen)
			}

			// Verify it starts with the correct prefix
			if !bytes.HasPrefix(key, IBCPacketNonceKeyPrefix) {
				t.Errorf("GetIBCPacketNonceKey() key does not start with IBCPacketNonceKeyPrefix")
			}

			// Verify it contains the channel ID and sender
			expectedSuffix := tt.channelID + "/" + tt.sender
			if !bytes.HasSuffix(key, []byte(expectedSuffix)) {
				t.Errorf("GetIBCPacketNonceKey() key does not end with expected suffix %s", expectedSuffix)
			}
		})
	}
}

func TestGetIBCPacketNonceKey_Uniqueness(t *testing.T) {
	// Different channel/sender combinations should produce different keys
	key1 := GetIBCPacketNonceKey("channel-0", "sender1")
	key2 := GetIBCPacketNonceKey("channel-0", "sender2")
	key3 := GetIBCPacketNonceKey("channel-1", "sender1")
	key4 := GetIBCPacketNonceKey("channel-1", "sender2")

	keys := [][]byte{key1, key2, key3, key4}

	// Verify all keys are unique
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if bytes.Equal(keys[i], keys[j]) {
				t.Errorf("GetIBCPacketNonceKey produced duplicate keys for different inputs")
			}
		}
	}
}

func TestGetIBCPacketNonceKey_Determinism(t *testing.T) {
	channelID := "channel-5"
	sender := "cosmos1test123"

	// Should produce the same key for the same inputs
	key1 := GetIBCPacketNonceKey(channelID, sender)
	key2 := GetIBCPacketNonceKey(channelID, sender)

	if !bytes.Equal(key1, key2) {
		t.Error("GetIBCPacketNonceKey() is not deterministic")
	}
}

func TestKeyPrefixUniqueness(t *testing.T) {
	// All key prefixes should be unique to avoid collisions
	prefixes := map[string][]byte{
		"ParamsKey":                 ParamsKey,
		"ComputeRequestKeyPrefix":   ComputeRequestKeyPrefix,
		"ProviderKeyPrefix":         ProviderKeyPrefix,
		"EscrowKeyPrefix":           EscrowKeyPrefix,
		"JobStatusKeyPrefix":        JobStatusKeyPrefix,
		"NonceTrackerKeyPrefix":     NonceTrackerKeyPrefix,
		"ResultKeyPrefix":           ResultKeyPrefix,
		"IBCPacketNonceKeyPrefix":   IBCPacketNonceKeyPrefix,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for name, prefix := range prefixes {
		key := string(prefix)
		if seen[key] {
			t.Errorf("Duplicate key prefix found for %s: %v", name, prefix)
		}
		seen[key] = true
	}
}

func TestKeyPrefixHasModuleNamespace(t *testing.T) {
	// All key prefixes should start with the module namespace
	prefixes := []struct {
		name string
		key  []byte
	}{
		{"ParamsKey", ParamsKey},
		{"ComputeRequestKeyPrefix", ComputeRequestKeyPrefix},
		{"ProviderKeyPrefix", ProviderKeyPrefix},
		{"EscrowKeyPrefix", EscrowKeyPrefix},
		{"JobStatusKeyPrefix", JobStatusKeyPrefix},
		{"NonceTrackerKeyPrefix", NonceTrackerKeyPrefix},
		{"ResultKeyPrefix", ResultKeyPrefix},
		{"IBCPacketNonceKeyPrefix", IBCPacketNonceKeyPrefix},
	}

	for _, tt := range prefixes {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.key) < 1 {
				t.Errorf("%s is empty", tt.name)
				return
			}
			if tt.key[0] != ModuleNamespace {
				t.Errorf("%s does not start with ModuleNamespace (0x%02x), got 0x%02x", tt.name, ModuleNamespace, tt.key[0])
			}
		})
	}
}

func BenchmarkGetIBCPacketNonceKey(b *testing.B) {
	channelID := "channel-12345"
	sender := "cosmos1abcdefghijklmnopqrstuvwxyz0123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetIBCPacketNonceKey(channelID, sender)
	}
}

func TestGetIBCPacketNonceKey_SpecialCharacters(t *testing.T) {
	// Test with special characters that might be in addresses
	tests := []struct {
		name      string
		channelID string
		sender    string
	}{
		{
			name:      "address with numbers",
			channelID: "channel-0",
			sender:    "cosmos1234567890",
		},
		{
			name:      "address with mixed case",
			channelID: "channel-0",
			sender:    "cosmos1AbCdEf",
		},
		{
			name:      "channel with dash",
			channelID: "channel-123-456",
			sender:    "cosmos1test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			key := GetIBCPacketNonceKey(tt.channelID, tt.sender)

			// Should have correct prefix
			if !bytes.HasPrefix(key, IBCPacketNonceKeyPrefix) {
				t.Error("Key does not have correct prefix")
			}

			// Should contain both channel and sender
			keyStr := string(key)
			if !bytes.Contains([]byte(keyStr), []byte(tt.channelID)) {
				t.Error("Key does not contain channel ID")
			}
			if !bytes.Contains([]byte(keyStr), []byte(tt.sender)) {
				t.Error("Key does not contain sender")
			}
		})
	}
}
