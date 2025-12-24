package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

var (
	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x01}

	// ProviderKeyPrefix is the prefix for provider storage
	ProviderKeyPrefix = []byte{0x02}

	// RequestKeyPrefix is the prefix for request storage
	RequestKeyPrefix = []byte{0x03}

	// ResultKeyPrefix is the prefix for result storage
	ResultKeyPrefix = []byte{0x04}

	// NextRequestIDKey is the key for the next request ID counter
	NextRequestIDKey = []byte{0x05}

	// RequestsByRequesterPrefix is the prefix for indexing requests by requester
	RequestsByRequesterPrefix = []byte{0x06}

	// RequestsByProviderPrefix is the prefix for indexing requests by provider
	RequestsByProviderPrefix = []byte{0x07}

	// RequestsByStatusPrefix is the prefix for indexing requests by status
	RequestsByStatusPrefix = []byte{0x08}

	// ActiveProvidersPrefix is the prefix for indexing active providers
	ActiveProvidersPrefix = []byte{0x09}

	// NonceKeyPrefix is the prefix for nonce tracking (replay attack prevention)
	NonceKeyPrefix = []byte{0x0A}

	// GovernanceParamsKey is the key for governance parameters
	GovernanceParamsKey = []byte{0x0B}

	// DisputeKeyPrefix is the prefix for dispute storage
	DisputeKeyPrefix = []byte{0x0C}

	// EvidenceKeyPrefix is the prefix for evidence storage
	EvidenceKeyPrefix = []byte{0x0D}

	// SlashRecordKeyPrefix is the prefix for slash record storage
	SlashRecordKeyPrefix = []byte{0x0E}

	// AppealKeyPrefix is the prefix for appeal storage
	AppealKeyPrefix = []byte{0x0F}

	// NextDisputeIDKey is the key for the next dispute ID counter
	NextDisputeIDKey = []byte{0x10}

	// NextSlashIDKey is the key for the next slash ID counter
	NextSlashIDKey = []byte{0x11}

	// NextAppealIDKey is the key for the next appeal ID counter
	NextAppealIDKey = []byte{0x12}

	// DisputesByRequestPrefix is the prefix for indexing disputes by request
	DisputesByRequestPrefix = []byte{0x13}

	// DisputesByStatusPrefix is the prefix for indexing disputes by status
	DisputesByStatusPrefix = []byte{0x14}

	// SlashRecordsByProviderPrefix is the prefix for indexing slash records by provider
	SlashRecordsByProviderPrefix = []byte{0x15}

	// AppealsByStatusPrefix is the prefix for indexing appeals by status
	AppealsByStatusPrefix = []byte{0x16}

	// CircuitParamsKeyPrefix is the prefix for ZK circuit parameters
	CircuitParamsKeyPrefix = []byte{0x17}

	// ZKMetricsKey is the key for ZK proof verification metrics
	ZKMetricsKey = []byte{0x18}

	// VerificationProofHashPrefix stores digests of verification proofs to prevent reuse
	VerificationProofHashPrefix = []byte{0x19}

	// ProviderSigningKeyPrefix stores fallback signing keys registered by providers
	ProviderSigningKeyPrefix = []byte{0x1A}

	// RequestFinalizedPrefix tracks whether a request has already settled funds
	RequestFinalizedPrefix = []byte{0x1B}

	// NonceByHeightPrefix is the prefix for indexing nonces by block height for cleanup
	NonceByHeightPrefix = []byte{0x1E}

	// ProviderStatsKeyPrefix is the prefix for provider performance statistics
	ProviderStatsKeyPrefix = []byte{0x1C}

	// EscrowTimeoutReversePrefix is the reverse index for escrow timeout lookup by request ID.
	// Key: prefix + requestID -> timeout timestamp
	// This enables O(1) lookup when removing timeout indexes.
	EscrowTimeoutReversePrefix = []byte{0x1D}

	// ProvidersByReputationPrefix is the prefix for reputation-sorted provider index
	// Key: prefix + reputation (inverted for descending order) + provider address
	// This enables O(log n) provider selection by reputation score
	ProvidersByReputationPrefix = []byte{0x1F}

	// CatastrophicFailureKeyPrefix is the prefix for catastrophic failure records
	CatastrophicFailureKeyPrefix = []byte{0x23}

	// NextCatastrophicFailureIDKey is the key for the next catastrophic failure ID counter
	NextCatastrophicFailureIDKey = []byte{0x24}

	// IBCPacketKeyPrefix is the prefix for IBC packet sequence tracking
	// Key: prefix + channelID + sequence -> requestID
	// Used to track pending IBC compute requests for validation
	IBCPacketKeyPrefix = []byte{0x25}
)

// ProviderKey returns the store key for a provider
func ProviderKey(address sdk.AccAddress) []byte {
	return append(ProviderKeyPrefix, address.Bytes()...)
}

// RequestKey returns the store key for a request
func RequestKey(requestID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, requestID)
	return append(RequestKeyPrefix, bz...)
}

// ResultKey returns the store key for a result
func ResultKey(requestID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, requestID)
	return append(ResultKeyPrefix, bz...)
}

// RequestByRequesterKey returns the index key for requests by requester
func RequestByRequesterKey(requester sdk.AccAddress, requestID uint64) []byte {
	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)
	return append(append(RequestsByRequesterPrefix, requester.Bytes()...), idBz...)
}

// RequestByProviderKey returns the index key for requests by provider
func RequestByProviderKey(provider sdk.AccAddress, requestID uint64) []byte {
	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)
	return append(append(RequestsByProviderPrefix, provider.Bytes()...), idBz...)
}

// RequestByStatusKey returns the index key for requests by status
func RequestByStatusKey(status uint32, requestID uint64) []byte {
	statusBz := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBz, status)
	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, requestID)
	return append(append(RequestsByStatusPrefix, statusBz...), idBz...)
}

// ActiveProviderKey returns the index key for active providers
func ActiveProviderKey(address sdk.AccAddress) []byte {
	return append(ActiveProvidersPrefix, address.Bytes()...)
}

// GetRequestIDFromBytes converts bytes to request ID
func GetRequestIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// GetStatusFromBytes converts bytes to status
func GetStatusFromBytes(bz []byte) uint32 {
	return binary.BigEndian.Uint32(bz)
}

// NonceKey returns the store key for a nonce tracking entry
func NonceKey(provider sdk.AccAddress, nonce uint64) []byte {
	nonceBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBz, nonce)
	return append(append(NonceKeyPrefix, provider.Bytes()...), nonceBz...)
}

// ProofHashKey returns the store key for a verification proof digest
func ProofHashKey(provider sdk.AccAddress, hash []byte) []byte {
	key := make([]byte, 0, len(VerificationProofHashPrefix)+len(provider.Bytes())+len(hash))
	key = append(key, VerificationProofHashPrefix...)
	key = append(key, provider.Bytes()...)
	key = append(key, hash...)
	return key
}

// ProviderSigningKeyKey returns the store key for a provider's registered signing key.
func ProviderSigningKeyKey(provider sdk.AccAddress) []byte {
	return append(ProviderSigningKeyPrefix, provider.Bytes()...)
}

// RequestFinalizedKey returns the store key for a request settlement flag
func RequestFinalizedKey(requestID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, requestID)
	return append(RequestFinalizedPrefix, bz...)
}

// DisputeKey returns the store key for a dispute
func DisputeKey(disputeID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, disputeID)
	return append(DisputeKeyPrefix, bz...)
}

// EvidenceKey returns the store key for evidence
func EvidenceKey(disputeID uint64, evidenceIndex uint64) []byte {
	disputeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(disputeBz, disputeID)
	indexBz := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBz, evidenceIndex)
	return append(append(EvidenceKeyPrefix, disputeBz...), indexBz...)
}

// EvidenceKeyPrefix returns the prefix for all evidence for a dispute
func EvidenceKeyPrefixForDispute(disputeID uint64) []byte {
	disputeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(disputeBz, disputeID)
	return append(EvidenceKeyPrefix, disputeBz...)
}

// SlashRecordKey returns the store key for a slash record
func SlashRecordKey(slashID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, slashID)
	return append(SlashRecordKeyPrefix, bz...)
}

// AppealKey returns the store key for an appeal
func AppealKey(appealID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, appealID)
	return append(AppealKeyPrefix, bz...)
}

// DisputeByRequestKey returns the index key for disputes by request
func DisputeByRequestKey(requestID uint64, disputeID uint64) []byte {
	requestBz := make([]byte, 8)
	binary.BigEndian.PutUint64(requestBz, requestID)
	disputeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(disputeBz, disputeID)
	return append(append(DisputesByRequestPrefix, requestBz...), disputeBz...)
}

// DisputeByStatusKey returns the index key for disputes by status
func DisputeByStatusKey(status uint32, disputeID uint64) []byte {
	statusBz := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBz, status)
	disputeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(disputeBz, disputeID)
	return append(append(DisputesByStatusPrefix, statusBz...), disputeBz...)
}

// SlashRecordByProviderKey returns the index key for slash records by provider
func SlashRecordByProviderKey(provider sdk.AccAddress, slashID uint64) []byte {
	slashBz := make([]byte, 8)
	binary.BigEndian.PutUint64(slashBz, slashID)
	return append(append(SlashRecordsByProviderPrefix, provider.Bytes()...), slashBz...)
}

// AppealByStatusKey returns the index key for appeals by status
func AppealByStatusKey(status uint32, appealID uint64) []byte {
	statusBz := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBz, status)
	appealBz := make([]byte, 8)
	binary.BigEndian.PutUint64(appealBz, appealID)
	return append(append(AppealsByStatusPrefix, statusBz...), appealBz...)
}

// GetDisputeIDFromBytes converts bytes to dispute ID
func GetDisputeIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// GetSlashIDFromBytes converts bytes to slash ID
func GetSlashIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// GetAppealIDFromBytes converts bytes to appeal ID
func GetAppealIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// CircuitParamsKey returns the store key for circuit parameters
func CircuitParamsKey(circuitID string) []byte {
	return append(CircuitParamsKeyPrefix, []byte(circuitID)...)
}

// NonceByHeightKey returns the index key for nonces by height for cleanup
func NonceByHeightKey(height int64, provider sdk.AccAddress, nonce uint64) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, types.SaturateInt64ToUint64(height))
	nonceBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBz, nonce)
	return append(append(append(NonceByHeightPrefix, heightBz...), provider.Bytes()...), nonceBz...)
}

// NonceByHeightPrefixForHeight returns the prefix for all nonces at a specific height
func NonceByHeightPrefixForHeight(height int64) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, types.SaturateInt64ToUint64(height))
	return append(NonceByHeightPrefix, heightBz...)
}

// ProviderStatsKey returns the store key for provider statistics
func ProviderStatsKey(providerAddr string) []byte {
	return append(ProviderStatsKeyPrefix, []byte(providerAddr)...)
}

// EscrowTimeoutReverseKey returns the reverse index key for escrow timeout lookup.
// This enables O(1) deletion of timeout entries by request ID.
func EscrowTimeoutReverseKey(requestID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, requestID)
	return append(EscrowTimeoutReversePrefix, bz...)
}

// ProviderByReputationKey returns the index key for providers sorted by reputation (descending)
// We invert the reputation score (255 - reputation) to achieve descending order in the iterator
func ProviderByReputationKey(reputation uint32, address sdk.AccAddress) []byte {
	// Invert reputation for descending order (higher reputation comes first)
	invertedRep := uint32(255) - reputation
	if invertedRep > 255 {
		invertedRep = 255
	}

	repBz := make([]byte, 4)
	binary.BigEndian.PutUint32(repBz, invertedRep)
	return append(append(ProvidersByReputationPrefix, repBz...), address.Bytes()...)
}

// CatastrophicFailureKey returns the store key for a catastrophic failure record
func CatastrophicFailureKey(failureID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, failureID)
	return append(CatastrophicFailureKeyPrefix, bz...)
}

// GetCatastrophicFailureIDFromBytes converts bytes to catastrophic failure ID
func GetCatastrophicFailureIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
