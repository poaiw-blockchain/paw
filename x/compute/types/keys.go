package types

var (
	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x01}

	// ComputeRequestKeyPrefix is the prefix for compute request store keys
	ComputeRequestKeyPrefix = []byte{0x02}

	// ProviderKeyPrefix is the prefix for provider store keys
	ProviderKeyPrefix = []byte{0x03}

	// EscrowKeyPrefix is the prefix for escrow store keys
	EscrowKeyPrefix = []byte{0x04}

	// JobStatusKeyPrefix is the prefix for job status store keys
	JobStatusKeyPrefix = []byte{0x05}

	// NonceTrackerKeyPrefix is the prefix for nonce tracker store keys
	NonceTrackerKeyPrefix = []byte{0x06}

	// NonceKeyPrefix is an alias for NonceTrackerKeyPrefix
	NonceKeyPrefix = []byte{0x06}

	// RequestKeyPrefix is an alias for ComputeRequestKeyPrefix
	RequestKeyPrefix = []byte{0x02}

	// ResultKeyPrefix is the prefix for compute result store keys
	ResultKeyPrefix = []byte{0x07}

	// IBCPacketNonceKeyPrefix is the prefix for IBC packet nonce tracking (replay protection)
	IBCPacketNonceKeyPrefix = []byte{0x0D}
)

// GetIBCPacketNonceKey returns the store key for IBC packet nonce tracking
// Used for replay attack prevention by tracking nonce per channel/sender pair
func GetIBCPacketNonceKey(channelID, sender string) []byte {
	return append(append(IBCPacketNonceKeyPrefix, []byte(channelID+"/")...), []byte(sender)...)
}
