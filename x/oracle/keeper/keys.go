package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// ModuleNamespace is the namespace byte for the Oracle module (0x03)
	// All store keys are prefixed with this byte to prevent collisions with other modules
	ModuleNamespace = byte(0x03)

	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x03, 0x01}

	// PriceKeyPrefix is the prefix for price storage
	PriceKeyPrefix = []byte{0x03, 0x02}

	// ValidatorPriceKeyPrefix is the prefix for validator price submissions
	ValidatorPriceKeyPrefix = []byte{0x03, 0x03}

	// ValidatorOracleKeyPrefix is the prefix for validator oracle info
	ValidatorOracleKeyPrefix = []byte{0x03, 0x04}

	// PriceSnapshotKeyPrefix is the prefix for price snapshots
	PriceSnapshotKeyPrefix = []byte{0x03, 0x05}

	// FeederDelegationKeyPrefix is the prefix for feeder delegations
	FeederDelegationKeyPrefix = []byte{0x03, 0x06}

	// SubmissionByHeightPrefix is the prefix for indexing submissions by block height for cleanup
	SubmissionByHeightPrefix = []byte{0x03, 0x07}


	// OutlierHistoryKeyPrefix is the prefix for outlier history storage
	// Tracks validator outlier submissions for reputation and slashing
	OutlierHistoryKeyPrefix = []byte{0x03, 0x0C}

	// IBCPacketNonceKeyPrefix is the prefix for IBC packet nonce tracking (replay protection)
	// Now properly namespaced under Oracle module
	IBCPacketNonceKeyPrefix = []byte{0x03, 0x0D}

	// CachedTotalVotingPowerKey stores the cached total voting power of all bonded validators
	// PERF-2: Updated in BeginBlocker to avoid O(n*m) recalculation per asset per block
	CachedTotalVotingPowerKey = []byte{0x03, 0x0E}
)

// GetPriceKey returns the store key for a price by asset
func GetPriceKey(asset string) []byte {
	return append(PriceKeyPrefix, []byte(asset)...)
}

// GetValidatorPriceKey returns the store key for a validator's price submission
func GetValidatorPriceKey(validatorAddr sdk.ValAddress, asset string) []byte {
	key := append(ValidatorPriceKeyPrefix, []byte(validatorAddr.String())...)
	key = append(key, byte(0x00)) // separator
	return append(key, []byte(asset)...)
}

// GetValidatorPricesByValidatorKey returns the prefix for all prices from a validator
func GetValidatorPricesByValidatorKey(validatorAddr sdk.ValAddress) []byte {
	return append(ValidatorPriceKeyPrefix, []byte(validatorAddr.String())...)
}

// GetValidatorOracleKey returns the store key for validator oracle info
func GetValidatorOracleKey(validatorAddr sdk.ValAddress) []byte {
	return append(ValidatorOracleKeyPrefix, []byte(validatorAddr.String())...)
}

// GetPriceSnapshotKey returns the store key for a price snapshot
func GetPriceSnapshotKey(asset string, blockHeight int64) []byte {
	key := append(PriceSnapshotKeyPrefix, []byte(asset)...)
	key = append(key, byte(0x00)) // separator
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(blockHeight))
	return append(key, heightBz...)
}

// GetPriceSnapshotsByAssetKey returns the prefix for all snapshots of an asset
func GetPriceSnapshotsByAssetKey(asset string) []byte {
	return append(PriceSnapshotKeyPrefix, []byte(asset)...)
}

// GetFeederDelegationKey returns the store key for a feeder delegation
func GetFeederDelegationKey(validatorAddr sdk.ValAddress) []byte {
	return append(FeederDelegationKeyPrefix, []byte(validatorAddr.String())...)
}

// GetSubmissionByHeightKey returns the index key for submissions by height for cleanup
func GetSubmissionByHeightKey(height int64, validatorAddr sdk.ValAddress, asset string) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(height))
	key := append(SubmissionByHeightPrefix, heightBz...)
	key = append(key, []byte(validatorAddr.String())...)
	key = append(key, byte(0x00)) // separator
	return append(key, []byte(asset)...)
}

// GetSubmissionByHeightPrefixForHeight returns the prefix for all submissions at a specific height
func GetSubmissionByHeightPrefixForHeight(height int64) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(height))
	return append(SubmissionByHeightPrefix, heightBz...)
}

// IBCPacketNonceKey returns the store key for IBC packet nonce tracking
// Used for replay attack prevention by tracking nonce per channel/sender pair
func IBCPacketNonceKey(channelID, sender string) []byte {
	return append(append(IBCPacketNonceKeyPrefix, []byte(channelID+"/")...), []byte(sender)...)
}
