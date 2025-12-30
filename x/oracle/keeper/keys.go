package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
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

	// ValidatorPriceByAssetKeyPrefix is the prefix for secondary index of validator prices by asset
	// PERF-6: Enables O(v) iteration for asset-specific price lookups instead of O(V) full scan
	// Key format: prefix + asset + 0x00 + validator
	ValidatorPriceByAssetKeyPrefix = []byte{0x03, 0x0F}

	// DATA-8: VotingPowerSnapshotPrefix stores validator voting power snapshots per vote period
	// This ensures consistent voting power is used throughout a vote period, preventing manipulation
	// Key format: prefix + vote_period_number (8 bytes) + 0x00 + validator_addr
	VotingPowerSnapshotPrefix = []byte{0x03, 0x10}

	// DATA-8: VotingPowerSnapshotTotalKey stores total voting power snapshot per vote period
	// Key format: prefix + vote_period_number (8 bytes)
	VotingPowerSnapshotTotalKey = []byte{0x03, 0x11}

	// DATA-8: CurrentVotePeriodKey stores the current vote period number for snapshot lookups
	CurrentVotePeriodKey = []byte{0x03, 0x12}
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

// GetValidatorPriceByAssetKey returns the secondary index key for a validator's price by asset
// PERF-6: This enables efficient iteration of all validator prices for a specific asset
func GetValidatorPriceByAssetKey(asset string, validatorAddr sdk.ValAddress) []byte {
	key := append(ValidatorPriceByAssetKeyPrefix, []byte(asset)...)
	key = append(key, byte(0x00)) // separator
	return append(key, []byte(validatorAddr.String())...)
}

// GetValidatorPricesByAssetPrefix returns the prefix for all validator prices for an asset
// PERF-6: Used for efficient iteration by asset
func GetValidatorPricesByAssetPrefix(asset string) []byte {
	key := append(ValidatorPriceByAssetKeyPrefix, []byte(asset)...)
	return append(key, byte(0x00)) // include separator for exact asset match
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
	return sharedibc.GetIBCPacketNonceKey(IBCPacketNonceKeyPrefix, channelID, sender)
}

// DATA-8: GetVotingPowerSnapshotKey returns the key for a validator's voting power snapshot
func GetVotingPowerSnapshotKey(votePeriod uint64, validatorAddr sdk.ValAddress) []byte {
	key := append(VotingPowerSnapshotPrefix, make([]byte, 8)...)
	binary.BigEndian.PutUint64(key[len(VotingPowerSnapshotPrefix):], votePeriod)
	key = append(key, byte(0x00)) // separator
	return append(key, []byte(validatorAddr.String())...)
}

// DATA-8: GetVotingPowerSnapshotTotalKey returns the key for total voting power snapshot
func GetVotingPowerSnapshotTotalKey(votePeriod uint64) []byte {
	key := append(VotingPowerSnapshotTotalKey, make([]byte, 8)...)
	binary.BigEndian.PutUint64(key[len(VotingPowerSnapshotTotalKey):], votePeriod)
	return key
}

// DATA-8: GetVotingPowerSnapshotPrefixForPeriod returns the prefix for all snapshots in a vote period
func GetVotingPowerSnapshotPrefixForPeriod(votePeriod uint64) []byte {
	key := append(VotingPowerSnapshotPrefix, make([]byte, 8)...)
	binary.BigEndian.PutUint64(key[len(VotingPowerSnapshotPrefix):], votePeriod)
	return append(key, byte(0x00)) // separator
}
