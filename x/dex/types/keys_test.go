package types

import (
	"bytes"
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestModuleNamespace(t *testing.T) {
	if ModuleNamespace != byte(0x02) {
		t.Errorf("Expected ModuleNamespace to be 0x02, got %x", ModuleNamespace)
	}
}

func TestKeyPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		expected []byte
	}{
		{
			name:     "PoolKeyPrefix",
			key:      PoolKeyPrefix,
			expected: []byte{0x02, 0x01},
		},
		{
			name:     "PoolCountKey",
			key:      PoolCountKey,
			expected: []byte{0x02, 0x02},
		},
		{
			name:     "PoolByTokensKeyPrefix",
			key:      PoolByTokensKeyPrefix,
			expected: []byte{0x02, 0x03},
		},
		{
			name:     "LiquidityKeyPrefix",
			key:      LiquidityKeyPrefix,
			expected: []byte{0x02, 0x04},
		},
		{
			name:     "ParamsKey",
			key:      ParamsKey,
			expected: []byte{0x02, 0x05},
		},
		{
			name:     "CircuitBreakerKeyPrefix",
			key:      CircuitBreakerKeyPrefix,
			expected: []byte{0x02, 0x06},
		},
		{
			name:     "LastLiquidityActionKeyPrefix",
			key:      LastLiquidityActionKeyPrefix,
			expected: []byte{0x02, 0x07},
		},
		{
			name:     "ReentrancyLockKeyPrefix",
			key:      ReentrancyLockKeyPrefix,
			expected: []byte{0x02, 0x08},
		},
		{
			name:     "PoolLPFeeKeyPrefix",
			key:      PoolLPFeeKeyPrefix,
			expected: []byte{0x02, 0x09},
		},
		{
			name:     "ProtocolFeeKeyPrefix",
			key:      ProtocolFeeKeyPrefix,
			expected: []byte{0x02, 0x0A},
		},
		{
			name:     "LiquidityShareKeyPrefix",
			key:      LiquidityShareKeyPrefix,
			expected: []byte{0x02, 0x0B},
		},
		{
			name:     "IBCPacketNonceKeyPrefix",
			key:      IBCPacketNonceKeyPrefix,
			expected: []byte{0x02, 0x16},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Equal(tt.key, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, tt.key)
			}
		})
	}
}

func TestGetPoolLPFeeKey(t *testing.T) {
	tests := []struct {
		name   string
		poolID uint64
		token  string
	}{
		{
			name:   "pool 1 with upaw",
			poolID: 1,
			token:  "upaw",
		},
		{
			name:   "pool 100 with uatom",
			poolID: 100,
			token:  "uatom",
		},
		{
			name:   "max pool ID",
			poolID: ^uint64(0),
			token:  "token",
		},
		{
			name:   "pool 0",
			poolID: 0,
			token:  "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetPoolLPFeeKey(tt.poolID, tt.token)

			// Verify prefix
			if !bytes.HasPrefix(key, PoolLPFeeKeyPrefix) {
				t.Error("Key does not start with PoolLPFeeKeyPrefix")
			}

			// Verify pool ID encoding
			poolIDBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(poolIDBytes, tt.poolID)
			if !bytes.Contains(key, poolIDBytes) {
				t.Errorf("Key does not contain correctly encoded pool ID")
			}

			// Verify token suffix
			if !bytes.HasSuffix(key, []byte(tt.token)) {
				t.Errorf("Key does not end with token: %s", tt.token)
			}

			// Verify expected length
			expectedLen := len(PoolLPFeeKeyPrefix) + 8 + len(tt.token)
			if len(key) != expectedLen {
				t.Errorf("Expected key length %d, got %d", expectedLen, len(key))
			}
		})
	}
}

func TestGetProtocolFeeKey(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "upaw token",
			token: "upaw",
		},
		{
			name:  "uatom token",
			token: "uatom",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "long token name",
			token: "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetProtocolFeeKey(tt.token)

			// Verify prefix
			if !bytes.HasPrefix(key, ProtocolFeeKeyPrefix) {
				t.Error("Key does not start with ProtocolFeeKeyPrefix")
			}

			// Verify token suffix
			if !bytes.HasSuffix(key, []byte(tt.token)) {
				t.Errorf("Key does not end with token: %s", tt.token)
			}

			// Verify expected length
			expectedLen := len(ProtocolFeeKeyPrefix) + len(tt.token)
			if len(key) != expectedLen {
				t.Errorf("Expected key length %d, got %d", expectedLen, len(key))
			}
		})
	}
}

func TestGetLiquidityShareKey(t *testing.T) {
	addr1, _ := sdk.AccAddressFromBech32("cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q")
	addr2, _ := sdk.AccAddressFromBech32("cosmos1h8h9h6gjgj7h8h9h6gjgj7h8h9h6gjgjslv6qe")

	tests := []struct {
		name     string
		poolID   uint64
		provider sdk.AccAddress
	}{
		{
			name:     "pool 1 with addr1",
			poolID:   1,
			provider: addr1,
		},
		{
			name:     "pool 100 with addr2",
			poolID:   100,
			provider: addr2,
		},
		{
			name:     "pool 0 with addr1",
			poolID:   0,
			provider: addr1,
		},
		{
			name:     "max pool ID with addr2",
			poolID:   ^uint64(0),
			provider: addr2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetLiquidityShareKey(tt.poolID, tt.provider)

			// Verify prefix
			if !bytes.HasPrefix(key, LiquidityShareKeyPrefix) {
				t.Error("Key does not start with LiquidityShareKeyPrefix")
			}

			// Verify pool ID encoding
			poolIDBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(poolIDBytes, tt.poolID)
			if !bytes.Contains(key, poolIDBytes) {
				t.Errorf("Key does not contain correctly encoded pool ID")
			}

			// Verify address suffix
			if !bytes.HasSuffix(key, tt.provider.Bytes()) {
				t.Error("Key does not end with provider address")
			}

			// Verify expected length
			expectedLen := len(LiquidityShareKeyPrefix) + 8 + len(tt.provider.Bytes())
			if len(key) != expectedLen {
				t.Errorf("Expected key length %d, got %d", expectedLen, len(key))
			}
		})
	}
}

func TestGetIBCPacketNonceKey(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		sender    string
	}{
		{
			name:      "channel-0 with sender1",
			channelID: "channel-0",
			sender:    "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		},
		{
			name:      "channel-100 with sender2",
			channelID: "channel-100",
			sender:    "cosmos1h8h9h6gjgj7h8h9h6gjgj7h8h9h6gjgjslv6qe",
		},
		{
			name:      "empty channel",
			channelID: "",
			sender:    "sender",
		},
		{
			name:      "empty sender",
			channelID: "channel-1",
			sender:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetIBCPacketNonceKey(tt.channelID, tt.sender)

			// Verify prefix
			if !bytes.HasPrefix(key, IBCPacketNonceKeyPrefix) {
				t.Error("Key does not start with IBCPacketNonceKeyPrefix")
			}

			// Verify channel ID is present
			if tt.channelID != "" && !bytes.Contains(key, []byte(tt.channelID)) {
				t.Errorf("Key does not contain channel ID: %s", tt.channelID)
			}

			// Verify separator
			if !bytes.Contains(key, []byte("/")) {
				t.Error("Key does not contain separator '/'")
			}

			// Verify sender is present
			if tt.sender != "" && !bytes.Contains(key, []byte(tt.sender)) {
				t.Errorf("Key does not contain sender: %s", tt.sender)
			}
		})
	}
}

func TestDefaultAuthority(t *testing.T) {
	authority := DefaultAuthority()

	if authority == "" {
		t.Error("DefaultAuthority() returned empty string")
	}

	// Verify it's a valid bech32 address
	_, err := sdk.AccAddressFromBech32(authority)
	if err != nil {
		t.Errorf("DefaultAuthority() returned invalid bech32 address: %v", err)
	}
}

func TestGetLiquidityShareKey_Uniqueness(t *testing.T) {
	addr1, _ := sdk.AccAddressFromBech32("cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q")
	addr2, _ := sdk.AccAddressFromBech32("cosmos1h8h9h6gjgj7h8h9h6gjgj7h8h9h6gjgjslv6qe")

	// Different pool IDs should produce different keys
	key1 := GetLiquidityShareKey(1, addr1)
	key2 := GetLiquidityShareKey(2, addr1)
	if bytes.Equal(key1, key2) {
		t.Error("Different pool IDs produced same key")
	}

	// Different providers should produce different keys
	key3 := GetLiquidityShareKey(1, addr1)
	key4 := GetLiquidityShareKey(1, addr2)
	if bytes.Equal(key3, key4) {
		t.Error("Different providers produced same key")
	}

	// Same pool ID and provider should produce same key
	key5 := GetLiquidityShareKey(1, addr1)
	key6 := GetLiquidityShareKey(1, addr1)
	if !bytes.Equal(key5, key6) {
		t.Error("Same pool ID and provider produced different keys")
	}
}

func TestGetPoolLPFeeKey_Uniqueness(t *testing.T) {
	// Different pool IDs should produce different keys
	key1 := GetPoolLPFeeKey(1, "upaw")
	key2 := GetPoolLPFeeKey(2, "upaw")
	if bytes.Equal(key1, key2) {
		t.Error("Different pool IDs produced same key")
	}

	// Different tokens should produce different keys
	key3 := GetPoolLPFeeKey(1, "upaw")
	key4 := GetPoolLPFeeKey(1, "uatom")
	if bytes.Equal(key3, key4) {
		t.Error("Different tokens produced same key")
	}

	// Same pool ID and token should produce same key
	key5 := GetPoolLPFeeKey(1, "upaw")
	key6 := GetPoolLPFeeKey(1, "upaw")
	if !bytes.Equal(key5, key6) {
		t.Error("Same pool ID and token produced different keys")
	}
}

func TestGetIBCPacketNonceKey_Uniqueness(t *testing.T) {
	// Different channel IDs should produce different keys
	key1 := GetIBCPacketNonceKey("channel-0", "sender1")
	key2 := GetIBCPacketNonceKey("channel-1", "sender1")
	if bytes.Equal(key1, key2) {
		t.Error("Different channel IDs produced same key")
	}

	// Different senders should produce different keys
	key3 := GetIBCPacketNonceKey("channel-0", "sender1")
	key4 := GetIBCPacketNonceKey("channel-0", "sender2")
	if bytes.Equal(key3, key4) {
		t.Error("Different senders produced same key")
	}

	// Same channel ID and sender should produce same key
	key5 := GetIBCPacketNonceKey("channel-0", "sender1")
	key6 := GetIBCPacketNonceKey("channel-0", "sender1")
	if !bytes.Equal(key5, key6) {
		t.Error("Same channel ID and sender produced different keys")
	}
}
