package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"
)

func TestKeyHelpers(t *testing.T) {
	// Catastrophic failure key/id round trip
	id := uint64(42)
	key := CatastrophicFailureKey(id)
	require.Equal(t, id, GetCatastrophicFailureIDFromBytes(key[len(CatastrophicFailureKeyPrefix):]))

	// IBC packet nonce key includes channel and sender
	channel := "channel-7"
	sender := "paw1sender"
	nonceKey := IBCPacketNonceKey(channel, sender)
	require.Contains(t, string(nonceKey), channel)
	require.Contains(t, string(nonceKey), sender)

	// Escrow locked height key encodes height
	heightKey := EscrowLockedHeightKey(99)
	require.True(t, len(heightKey) > len(EscrowLockedHeightPrefix))

	// Migration key helpers
	old := []byte{ModuleNamespace, 0x02}
	require.Equal(t, []byte{0x02}, GetOldKey(old))
	require.Equal(t, old, GetNewKey([]byte{0x02}))

	// ProviderByReputationKey sorts descending; ensure key contains address
	addr := address.Module("provider-test") // 20 bytes deterministic
	repKey := ProviderByReputationKey(200, addr)
	require.Contains(t, string(repKey), string(addr))
}
