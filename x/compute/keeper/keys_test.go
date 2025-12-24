package keeper

import (
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestProviderKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ProviderKey(addr)
		key2 := ProviderKey(addr)
		require.Equal(t, key1, key2)
	})

	t.Run("different addresses produce different keys", func(t *testing.T) {
		addr2 := sdk.AccAddress([]byte("other_provider_addr_"))
		key1 := ProviderKey(addr)
		key2 := ProviderKey(addr2)
		require.NotEqual(t, key1, key2)
	})
}

func TestRequestKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := RequestKey(123)
		key2 := RequestKey(123)
		require.Equal(t, key1, key2)
	})

	t.Run("different IDs produce different keys", func(t *testing.T) {
		key1 := RequestKey(1)
		key2 := RequestKey(2)
		require.NotEqual(t, key1, key2)
	})
}

func TestResultKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ResultKey(123)
		key2 := ResultKey(123)
		require.Equal(t, key1, key2)
	})

	t.Run("different IDs produce different keys", func(t *testing.T) {
		key1 := ResultKey(1)
		key2 := ResultKey(2)
		require.NotEqual(t, key1, key2)
	})
}

func TestRequestByRequesterKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_requester_addr_"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := RequestByRequesterKey(addr, 123)
		key2 := RequestByRequesterKey(addr, 123)
		require.Equal(t, key1, key2)
	})
}

func TestRequestByProviderKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := RequestByProviderKey(addr, 123)
		key2 := RequestByProviderKey(addr, 123)
		require.Equal(t, key1, key2)
	})
}

func TestRequestByStatusKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := RequestByStatusKey(uint32(types.REQUEST_STATUS_PENDING), 123)
		key2 := RequestByStatusKey(uint32(types.REQUEST_STATUS_PENDING), 123)
		require.Equal(t, key1, key2)
	})

	t.Run("different statuses produce different keys", func(t *testing.T) {
		key1 := RequestByStatusKey(uint32(types.REQUEST_STATUS_PENDING), 123)
		key2 := RequestByStatusKey(uint32(types.REQUEST_STATUS_COMPLETED), 123)
		require.NotEqual(t, key1, key2)
	})
}

func TestActiveProviderKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ActiveProviderKey(addr)
		key2 := ActiveProviderKey(addr)
		require.Equal(t, key1, key2)
	})
}

func TestGetRequestIDFromBytes(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		// Create a key and extract ID
		key := RequestKey(123)
		// Key format is prefix (2 bytes) + 8 bytes
		idBytes := key[2:] // Skip 2-byte prefix
		id := GetRequestIDFromBytes(idBytes)
		require.Equal(t, uint64(123), id)
	})
}

func TestGetStatusFromBytes(t *testing.T) {
	t.Run("extracts status correctly", func(t *testing.T) {
		statusBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(statusBytes, uint32(types.REQUEST_STATUS_PENDING))
		status := GetStatusFromBytes(statusBytes)
		require.Equal(t, uint32(types.REQUEST_STATUS_PENDING), status)
	})
}

func TestNonceKey(t *testing.T) {
	provider := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := NonceKey(provider, 123)
		key2 := NonceKey(provider, 123)
		require.Equal(t, key1, key2)
	})

	t.Run("different nonces produce different keys", func(t *testing.T) {
		key1 := NonceKey(provider, 1)
		key2 := NonceKey(provider, 2)
		require.NotEqual(t, key1, key2)
	})
}

func TestProofHashKey(t *testing.T) {
	provider := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ProofHashKey(provider, []byte("hash123"))
		key2 := ProofHashKey(provider, []byte("hash123"))
		require.Equal(t, key1, key2)
	})
}

func TestProviderSigningKeyKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ProviderSigningKeyKey(addr)
		key2 := ProviderSigningKeyKey(addr)
		require.Equal(t, key1, key2)
	})
}

func TestRequestFinalizedKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := RequestFinalizedKey(123)
		key2 := RequestFinalizedKey(123)
		require.Equal(t, key1, key2)
	})
}

func TestDisputeKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := DisputeKey(123)
		key2 := DisputeKey(123)
		require.Equal(t, key1, key2)
	})
}

func TestEvidenceKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := EvidenceKey(1, 2)
		key2 := EvidenceKey(1, 2)
		require.Equal(t, key1, key2)
	})

	t.Run("different dispute IDs produce different keys", func(t *testing.T) {
		key1 := EvidenceKey(1, 2)
		key2 := EvidenceKey(3, 2)
		require.NotEqual(t, key1, key2)
	})
}

func TestEvidenceKeyPrefixForDispute(t *testing.T) {
	t.Run("generates consistent prefix", func(t *testing.T) {
		prefix1 := EvidenceKeyPrefixForDispute(123)
		prefix2 := EvidenceKeyPrefixForDispute(123)
		require.Equal(t, prefix1, prefix2)
	})
}

func TestSlashRecordKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := SlashRecordKey(123)
		key2 := SlashRecordKey(123)
		require.Equal(t, key1, key2)
	})
}

func TestAppealKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := AppealKey(123)
		key2 := AppealKey(123)
		require.Equal(t, key1, key2)
	})
}

func TestDisputeByRequestKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := DisputeByRequestKey(1, 2)
		key2 := DisputeByRequestKey(1, 2)
		require.Equal(t, key1, key2)
	})
}

func TestDisputeByStatusKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := DisputeByStatusKey(uint32(types.DISPUTE_STATUS_VOTING), 123)
		key2 := DisputeByStatusKey(uint32(types.DISPUTE_STATUS_VOTING), 123)
		require.Equal(t, key1, key2)
	})
}

func TestSlashRecordByProviderKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := SlashRecordByProviderKey(addr, 123)
		key2 := SlashRecordByProviderKey(addr, 123)
		require.Equal(t, key1, key2)
	})
}

func TestAppealByStatusKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := AppealByStatusKey(uint32(types.APPEAL_STATUS_PENDING), 123)
		key2 := AppealByStatusKey(uint32(types.APPEAL_STATUS_PENDING), 123)
		require.Equal(t, key1, key2)
	})
}

func TestGetDisputeIDFromBytes(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		key := DisputeKey(456)
		idBytes := key[2:] // Skip 2-byte prefix
		id := GetDisputeIDFromBytes(idBytes)
		require.Equal(t, uint64(456), id)
	})
}

func TestGetSlashIDFromBytes(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		key := SlashRecordKey(789)
		idBytes := key[2:] // Skip 2-byte prefix
		id := GetSlashIDFromBytes(idBytes)
		require.Equal(t, uint64(789), id)
	})
}

func TestGetAppealIDFromBytes(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		key := AppealKey(321)
		idBytes := key[2:] // Skip 2-byte prefix
		id := GetAppealIDFromBytes(idBytes)
		require.Equal(t, uint64(321), id)
	})
}

func TestCircuitParamsKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := CircuitParamsKey("circuit_id")
		key2 := CircuitParamsKey("circuit_id")
		require.Equal(t, key1, key2)
	})
}

func TestNonceByHeightKey(t *testing.T) {
	provider := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := NonceByHeightKey(100, provider, 1)
		key2 := NonceByHeightKey(100, provider, 1)
		require.Equal(t, key1, key2)
	})
}

func TestNonceByHeightPrefixForHeight(t *testing.T) {
	t.Run("generates consistent prefix", func(t *testing.T) {
		prefix1 := NonceByHeightPrefixForHeight(100)
		prefix2 := NonceByHeightPrefixForHeight(100)
		require.Equal(t, prefix1, prefix2)
	})
}

func TestProviderStatsKey(t *testing.T) {
	addr := sdk.AccAddress([]byte("test_provider_addr__"))

	t.Run("generates consistent key", func(t *testing.T) {
		key1 := ProviderStatsKey(addr.String())
		key2 := ProviderStatsKey(addr.String())
		require.Equal(t, key1, key2)
	})
}

func TestEscrowTimeoutReverseKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		key1 := EscrowTimeoutReverseKey(123)
		key2 := EscrowTimeoutReverseKey(123)
		require.Equal(t, key1, key2)
	})
}
