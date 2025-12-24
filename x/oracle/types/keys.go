package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	// ModuleNamespace is the namespace byte for the Oracle module (0x03)
	ModuleNamespace = byte(0x03)

	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x03, 0x01}

	// PriceKeyPrefix is the prefix for price store keys
	PriceKeyPrefix = []byte{0x03, 0x02}

	// ValidatorKeyPrefix is the prefix for validator store keys
	ValidatorKeyPrefix = []byte{0x03, 0x03}

	// FeederDelegationKeyPrefix is the prefix for feeder delegation store keys
	FeederDelegationKeyPrefix = []byte{0x03, 0x04}

	// MissCounterKeyPrefix is the prefix for miss counter store keys
	MissCounterKeyPrefix = []byte{0x03, 0x05}

	// AggregateVoteKeyPrefix is the prefix for aggregate vote store keys
	AggregateVoteKeyPrefix = []byte{0x03, 0x06}

	// PrevoteKeyPrefix is the prefix for prevote store keys
	PrevoteKeyPrefix = []byte{0x03, 0x07}

	// VoteKeyPrefix is the prefix for vote store keys
	VoteKeyPrefix = []byte{0x03, 0x08}

	// DelegateKeyPrefix is the prefix for delegate store keys
	DelegateKeyPrefix = []byte{0x03, 0x09}

	// SlashingKeyPrefix is the prefix for slashing store keys
	SlashingKeyPrefix = []byte{0x03, 0x0A}

	// TWAPKeyPrefix is the prefix for TWAP store keys
	TWAPKeyPrefix = []byte{0x03, 0x0B}

	// IBCPacketNonceKeyPrefix is the prefix for IBC packet nonce tracking (replay protection)
	IBCPacketNonceKeyPrefix = []byte{0x03, 0x0D}

	// EmergencyPauseStateKey is the key for emergency pause state
	EmergencyPauseStateKey = []byte{0x03, 0x0E}
)

// DefaultAuthority returns the governance module address as the only allowed
// authority for oracle parameter updates.
func DefaultAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// GetIBCPacketNonceKey returns the store key for IBC packet nonce tracking
// Used for replay attack prevention by tracking nonce per channel/sender pair
func GetIBCPacketNonceKey(channelID, sender string) []byte {
	return append(append(IBCPacketNonceKeyPrefix, []byte(channelID+"/")...), []byte(sender)...)
}
