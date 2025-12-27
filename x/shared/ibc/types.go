package ibc

// ChannelOperation tracks an outbound IBC packet that still has funds or state locked locally.
// This is a unified struct used by all modules (dex, oracle, compute) to track pending
// operations that need cleanup when an IBC channel closes unexpectedly.
//
// Fields:
//   - ChannelID: The IBC channel identifier
//   - Sequence: The packet sequence number
//   - PacketType: Module-specific packet type identifier
//   - ChainID: Optional chain identifier (used by oracle module)
//   - JobID: Optional job identifier (used by compute module)
//   - TargetChain: Optional target chain (used by compute module)
type ChannelOperation struct {
	ChannelID   string `json:"channel_id"`
	Sequence    uint64 `json:"sequence"`
	PacketType  string `json:"packet_type"`
	ChainID     string `json:"chain_id,omitempty"`
	JobID       string `json:"job_id,omitempty"`
	TargetChain string `json:"target_chain,omitempty"`
}

// GetIBCPacketNonceKey returns the store key for IBC packet nonce tracking.
// Used for replay attack prevention by tracking nonce per channel/sender pair.
// The prefix parameter allows each module to use its own namespace.
func GetIBCPacketNonceKey(prefix []byte, channelID, sender string) []byte {
	return append(append(prefix, []byte(channelID+"/")...), []byte(sender)...)
}
