package types

import (
	"bytes"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestModuleNamespace(t *testing.T) {
	if ModuleNamespace != byte(0x03) {
		t.Errorf("Expected ModuleNamespace 0x03, got %x", ModuleNamespace)
	}
}

func TestParamsKey(t *testing.T) {
	expected := []byte{0x03, 0x01}
	if !bytes.Equal(ParamsKey, expected) {
		t.Errorf("Expected ParamsKey %v, got %v", expected, ParamsKey)
	}

	// Verify it starts with module namespace
	if len(ParamsKey) < 1 || ParamsKey[0] != ModuleNamespace {
		t.Error("ParamsKey must start with ModuleNamespace")
	}
}

func TestKeyPrefixes_UniqueAndStartWithNamespace(t *testing.T) {
	prefixes := map[string][]byte{
		"ParamsKey":                 ParamsKey,
		"PriceKeyPrefix":            PriceKeyPrefix,
		"ValidatorKeyPrefix":        ValidatorKeyPrefix,
		"FeederDelegationKeyPrefix": FeederDelegationKeyPrefix,
		"MissCounterKeyPrefix":      MissCounterKeyPrefix,
		"AggregateVoteKeyPrefix":    AggregateVoteKeyPrefix,
		"PrevoteKeyPrefix":          PrevoteKeyPrefix,
		"VoteKeyPrefix":             VoteKeyPrefix,
		"DelegateKeyPrefix":         DelegateKeyPrefix,
		"SlashingKeyPrefix":         SlashingKeyPrefix,
		"TWAPKeyPrefix":             TWAPKeyPrefix,
		"IBCPacketNonceKeyPrefix":   IBCPacketNonceKeyPrefix,
		"EmergencyPauseStateKey":    EmergencyPauseStateKey,
	}

	// Check all start with module namespace
	for name, prefix := range prefixes {
		if len(prefix) < 1 {
			t.Errorf("%s is empty", name)
			continue
		}
		if prefix[0] != ModuleNamespace {
			t.Errorf("%s does not start with ModuleNamespace (0x03), got %x", name, prefix[0])
		}
	}

	// Check all are unique
	seen := make(map[string]string)
	for name, prefix := range prefixes {
		key := string(prefix)
		if other, exists := seen[key]; exists {
			t.Errorf("Duplicate key prefix: %s and %s both have %v", name, other, prefix)
		}
		seen[key] = name
	}
}

func TestKeyPrefixes_ExpectedValues(t *testing.T) {
	tests := []struct {
		name     string
		prefix   []byte
		expected []byte
	}{
		{"PriceKeyPrefix", PriceKeyPrefix, []byte{0x03, 0x02}},
		{"ValidatorKeyPrefix", ValidatorKeyPrefix, []byte{0x03, 0x03}},
		{"FeederDelegationKeyPrefix", FeederDelegationKeyPrefix, []byte{0x03, 0x04}},
		{"MissCounterKeyPrefix", MissCounterKeyPrefix, []byte{0x03, 0x05}},
		{"AggregateVoteKeyPrefix", AggregateVoteKeyPrefix, []byte{0x03, 0x06}},
		{"PrevoteKeyPrefix", PrevoteKeyPrefix, []byte{0x03, 0x07}},
		{"VoteKeyPrefix", VoteKeyPrefix, []byte{0x03, 0x08}},
		{"DelegateKeyPrefix", DelegateKeyPrefix, []byte{0x03, 0x09}},
		{"SlashingKeyPrefix", SlashingKeyPrefix, []byte{0x03, 0x0A}},
		{"TWAPKeyPrefix", TWAPKeyPrefix, []byte{0x03, 0x0B}},
		{"IBCPacketNonceKeyPrefix", IBCPacketNonceKeyPrefix, []byte{0x03, 0x0D}},
		{"EmergencyPauseStateKey", EmergencyPauseStateKey, []byte{0x03, 0x0E}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Equal(tt.prefix, tt.expected) {
				t.Errorf("Expected %s = %v, got %v", tt.name, tt.expected, tt.prefix)
			}
		})
	}
}

func TestDefaultAuthority(t *testing.T) {
	authority := DefaultAuthority()

	if authority == "" {
		t.Error("DefaultAuthority returned empty string")
	}

	// Verify it's the governance module address
	expectedAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	if authority != expectedAddr.String() {
		t.Errorf("Expected authority %s, got %s", expectedAddr.String(), authority)
	}

	// Verify it's a valid bech32 address
	if len(authority) < 10 {
		t.Error("Authority string seems too short to be valid bech32")
	}
}

func TestGetIBCPacketNonceKey(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		sender    string
	}{
		{
			name:      "standard channel and sender",
			channelID: "channel-0",
			sender:    "cosmos1abc",
		},
		{
			name:      "empty channel ID",
			channelID: "",
			sender:    "cosmos1abc",
		},
		{
			name:      "empty sender",
			channelID: "channel-0",
			sender:    "",
		},
		{
			name:      "both empty",
			channelID: "",
			sender:    "",
		},
		{
			name:      "long channel ID",
			channelID: "channel-123456789",
			sender:    "cosmos1verylongaddress",
		},
		{
			name:      "special characters in sender",
			channelID: "channel-0",
			sender:    "cosmos1abc/def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetIBCPacketNonceKey(tt.channelID, tt.sender)

			// Verify key starts with IBCPacketNonceKeyPrefix
			if !bytes.HasPrefix(key, IBCPacketNonceKeyPrefix) {
				t.Errorf("Key does not start with IBCPacketNonceKeyPrefix")
			}

			// Verify key contains channel ID and sender
			keyStr := string(key)
			if tt.channelID != "" && !bytes.Contains(key, []byte(tt.channelID)) {
				t.Errorf("Key does not contain channel ID: %s", tt.channelID)
			}
			if tt.sender != "" && !bytes.Contains(key, []byte(tt.sender)) {
				t.Errorf("Key does not contain sender: %s", tt.sender)
			}

			// Verify format includes separator
			if !bytes.Contains(key, []byte("/")) {
				t.Error("Key does not contain expected separator '/'")
			}

			// Verify key construction
			expectedSuffix := []byte(tt.channelID + "/" + tt.sender)
			expectedKey := append(IBCPacketNonceKeyPrefix, expectedSuffix...)
			if !bytes.Equal(key, expectedKey) {
				t.Errorf("Expected key %v, got %v", expectedKey, key)
			}

			// Log key for debugging
			t.Logf("Channel: %q, Sender: %q => Key: %q (%v)", tt.channelID, tt.sender, keyStr, key)
		})
	}
}

func TestGetIBCPacketNonceKey_Uniqueness(t *testing.T) {
	// Test that different channel/sender combinations produce different keys
	key1 := GetIBCPacketNonceKey("channel-0", "cosmos1abc")
	key2 := GetIBCPacketNonceKey("channel-1", "cosmos1abc")
	key3 := GetIBCPacketNonceKey("channel-0", "cosmos1xyz")
	key4 := GetIBCPacketNonceKey("channel-1", "cosmos1xyz")

	keys := [][]byte{key1, key2, key3, key4}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if bytes.Equal(keys[i], keys[j]) {
				t.Errorf("Keys %d and %d are equal but should be unique", i, j)
			}
		}
	}
}

func TestGetIBCPacketNonceKey_SameInputsSameKey(t *testing.T) {
	// Test that same inputs always produce same key
	channelID := "channel-42"
	sender := "cosmos1test"

	key1 := GetIBCPacketNonceKey(channelID, sender)
	key2 := GetIBCPacketNonceKey(channelID, sender)

	if !bytes.Equal(key1, key2) {
		t.Errorf("Same inputs produced different keys: %v vs %v", key1, key2)
	}
}

func TestKeyPrefixes_NoGaps(t *testing.T) {
	// Verify key prefixes use sequential bytes without gaps (for efficient storage)
	// Note: 0x0C is intentionally skipped (reserved or not yet used)
	expectedSequence := [][]byte{
		{0x03, 0x01}, // ParamsKey
		{0x03, 0x02}, // PriceKeyPrefix
		{0x03, 0x03}, // ValidatorKeyPrefix
		{0x03, 0x04}, // FeederDelegationKeyPrefix
		{0x03, 0x05}, // MissCounterKeyPrefix
		{0x03, 0x06}, // AggregateVoteKeyPrefix
		{0x03, 0x07}, // PrevoteKeyPrefix
		{0x03, 0x08}, // VoteKeyPrefix
		{0x03, 0x09}, // DelegateKeyPrefix
		{0x03, 0x0A}, // SlashingKeyPrefix
		{0x03, 0x0B}, // TWAPKeyPrefix
		// 0x0C intentionally skipped
		{0x03, 0x0D}, // IBCPacketNonceKeyPrefix
		{0x03, 0x0E}, // EmergencyPauseStateKey
	}

	actual := [][]byte{
		ParamsKey,
		PriceKeyPrefix,
		ValidatorKeyPrefix,
		FeederDelegationKeyPrefix,
		MissCounterKeyPrefix,
		AggregateVoteKeyPrefix,
		PrevoteKeyPrefix,
		VoteKeyPrefix,
		DelegateKeyPrefix,
		SlashingKeyPrefix,
		TWAPKeyPrefix,
		IBCPacketNonceKeyPrefix,
		EmergencyPauseStateKey,
	}

	for i, expected := range expectedSequence {
		if !bytes.Equal(actual[i], expected) {
			t.Errorf("Index %d: expected %v, got %v", i, expected, actual[i])
		}
	}
}

func TestKeyPrefixes_Length(t *testing.T) {
	// All key prefixes should be 2 bytes (namespace + identifier)
	prefixes := []struct {
		name   string
		prefix []byte
	}{
		{"ParamsKey", ParamsKey},
		{"PriceKeyPrefix", PriceKeyPrefix},
		{"ValidatorKeyPrefix", ValidatorKeyPrefix},
		{"FeederDelegationKeyPrefix", FeederDelegationKeyPrefix},
		{"MissCounterKeyPrefix", MissCounterKeyPrefix},
		{"AggregateVoteKeyPrefix", AggregateVoteKeyPrefix},
		{"PrevoteKeyPrefix", PrevoteKeyPrefix},
		{"VoteKeyPrefix", VoteKeyPrefix},
		{"DelegateKeyPrefix", DelegateKeyPrefix},
		{"SlashingKeyPrefix", SlashingKeyPrefix},
		{"TWAPKeyPrefix", TWAPKeyPrefix},
		{"IBCPacketNonceKeyPrefix", IBCPacketNonceKeyPrefix},
		{"EmergencyPauseStateKey", EmergencyPauseStateKey},
	}

	for _, p := range prefixes {
		if len(p.prefix) != 2 {
			t.Errorf("%s has length %d, expected 2", p.name, len(p.prefix))
		}
	}
}

func TestGetIBCPacketNonceKey_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		sender    string
		expectLen int // minimum expected length
	}{
		{
			name:      "minimal inputs",
			channelID: "a",
			sender:    "b",
			expectLen: len(IBCPacketNonceKeyPrefix) + 3, // prefix + "a/b"
		},
		{
			name:      "unicode in sender",
			channelID: "channel-0",
			sender:    "cosmos1测试",
			expectLen: len(IBCPacketNonceKeyPrefix) + len("channel-0/") + len("cosmos1测试"),
		},
		{
			name:      "numbers only",
			channelID: "123",
			sender:    "456",
			expectLen: len(IBCPacketNonceKeyPrefix) + len("123/456"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetIBCPacketNonceKey(tt.channelID, tt.sender)

			if len(key) < tt.expectLen {
				t.Errorf("Expected key length >= %d, got %d", tt.expectLen, len(key))
			}

			// Verify it's still properly prefixed
			if !bytes.HasPrefix(key, IBCPacketNonceKeyPrefix) {
				t.Error("Key does not start with IBCPacketNonceKeyPrefix")
			}
		})
	}
}
