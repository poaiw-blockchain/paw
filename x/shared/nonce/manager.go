// Package nonce provides shared IBC packet nonce management for replay attack prevention.
// This package consolidates identical nonce validation logic used across dex, oracle, and compute modules.
package nonce

import (
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// IncomingNoncePrefix is the prefix for incoming packet nonces
	IncomingNoncePrefix = "nonce"
	// SendNoncePrefix is the prefix for outgoing packet nonces
	SendNoncePrefix = "nonce_send"
	// NonceTimestampPrefix is the prefix for nonce creation timestamps
	NonceTimestampPrefix = "nonce_ts"

	// MaxTimestampAge is the maximum age of a packet timestamp (24 hours in seconds)
	MaxTimestampAge = int64(86400)
	// MaxFutureDrift is the maximum allowed clock drift into the future (5 minutes in seconds)
	MaxFutureDrift = int64(300)
	// DefaultNonceTTLSeconds is the default TTL for nonces (7 days)
	DefaultNonceTTLSeconds = int64(604800)
)

// ErrorProvider allows modules to provide their own error types while using shared nonce logic.
// Each module implements this interface to wrap errors with their module-specific error types.
type ErrorProvider interface {
	// InvalidNonceError returns an error for invalid nonce with the given message
	InvalidNonceError(msg string) error
	// InvalidPacketError returns an error for invalid packet with the given message
	InvalidPacketError(msg string) error
}

// Manager handles IBC packet nonce validation and generation.
// It provides replay attack prevention through monotonically increasing nonces
// and timestamp validation.
type Manager struct {
	storeKey      storetypes.StoreKey
	errorProvider ErrorProvider
	moduleName    string
}

// NewManager creates a new nonce manager for a module.
// storeKey: the module's store key for persistence
// errorProvider: module-specific error type provider
// moduleName: the module name (used as default sender for normalization)
func NewManager(storeKey storetypes.StoreKey, errorProvider ErrorProvider, moduleName string) *Manager {
	return &Manager{
		storeKey:      storeKey,
		errorProvider: errorProvider,
		moduleName:    moduleName,
	}
}

// encodeNonce encodes a uint64 nonce to bytes
func encodeNonce(n uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	return bz
}

// decodeNonce decodes bytes to a uint64 nonce
func decodeNonce(bz []byte) uint64 {
	if len(bz) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// incomingNonceKey generates the store key for incoming packet nonces
func (m *Manager) incomingNonceKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", IncomingNoncePrefix, channel, m.normalizeSender(sender)))
}

// sendNonceKey generates the store key for outgoing packet nonces
func (m *Manager) sendNonceKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", SendNoncePrefix, channel, m.normalizeSender(sender)))
}

// nonceTimestampKey generates the store key for nonce creation timestamps
func (m *Manager) nonceTimestampKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", NonceTimestampPrefix, channel, m.normalizeSender(sender)))
}

// normalizeSender ensures sender is never empty (uses module name as default)
func (m *Manager) normalizeSender(sender string) string {
	if sender == "" {
		return m.moduleName
	}
	return sender
}

// getIncomingNonce retrieves the stored incoming nonce for a channel/sender pair
func (m *Manager) getIncomingNonce(ctx sdk.Context, channel, sender string) uint64 {
	store := ctx.KVStore(m.storeKey)
	key := m.incomingNonceKey(channel, sender)
	return decodeNonce(store.Get(key))
}

// setIncomingNonce stores the incoming nonce for a channel/sender pair
func (m *Manager) setIncomingNonce(ctx sdk.Context, channel, sender string, nonce uint64) {
	store := ctx.KVStore(m.storeKey)
	key := m.incomingNonceKey(channel, sender)
	store.Set(key, encodeNonce(nonce))
}

// getNonceTimestamp retrieves the stored timestamp for a channel/sender pair
func (m *Manager) getNonceTimestamp(ctx sdk.Context, channel, sender string) int64 {
	store := ctx.KVStore(m.storeKey)
	key := m.nonceTimestampKey(channel, sender)
	bz := store.Get(key)
	if len(bz) != 8 {
		return 0
	}
	return int64(decodeNonce(bz))
}

// setNonceTimestamp stores the timestamp when a nonce was created
func (m *Manager) setNonceTimestamp(ctx sdk.Context, channel, sender string, timestamp int64) {
	store := ctx.KVStore(m.storeKey)
	key := m.nonceTimestampKey(channel, sender)
	store.Set(key, encodeNonce(uint64(timestamp)))
}

// ValidateIncomingPacketNonce validates packet nonce and timestamp to prevent replay attacks.
// It enforces:
// 1. Nonce must be greater than zero
// 2. Channel must not be empty
// 3. Timestamp must be positive
// 4. Timestamp must be within 24 hours of current block time (prevents old packet replay)
// 5. Timestamp must not be more than 5 minutes in the future (clock drift tolerance)
// 6. Nonce must be monotonically increasing per channel/sender pair
//
// After successful validation, the new nonce is stored.
func (m *Manager) ValidateIncomingPacketNonce(ctx sdk.Context, channel, sender string, packetNonce uint64, timestamp int64) error {
	if packetNonce == 0 {
		return m.errorProvider.InvalidNonceError("nonce must be greater than zero")
	}
	if channel == "" {
		return m.errorProvider.InvalidPacketError("source channel missing")
	}
	if timestamp <= 0 {
		return m.errorProvider.InvalidPacketError("timestamp must be positive")
	}

	// Check timestamp is within 24 hours
	currentTime := ctx.BlockTime().Unix()
	timeDiff := currentTime - timestamp

	if timeDiff > MaxTimestampAge {
		return m.errorProvider.InvalidPacketError(fmt.Sprintf(
			"packet timestamp too old: %d seconds ago (max: %d seconds)",
			timeDiff, MaxTimestampAge))
	}

	// Allow small clock drift into the future
	if timeDiff < -MaxFutureDrift {
		return m.errorProvider.InvalidPacketError(fmt.Sprintf(
			"packet timestamp too far in future: %d seconds ahead (max: %d seconds)",
			-timeDiff, MaxFutureDrift))
	}

	// Enforce monotonically increasing nonce
	stored := m.getIncomingNonce(ctx, channel, sender)
	if packetNonce <= stored {
		return m.errorProvider.InvalidNonceError(fmt.Sprintf(
			"replay attack detected: packet nonce %d not greater than stored %d",
			packetNonce, stored))
	}

	// Store the new nonce after successful validation
	m.setIncomingNonce(ctx, channel, sender, packetNonce)
	// Store timestamp for pruning
	m.setNonceTimestamp(ctx, channel, sender, ctx.BlockTime().Unix())
	return nil
}

// getSendNonce retrieves the stored outgoing nonce for a channel/sender pair
func (m *Manager) getSendNonce(ctx sdk.Context, channel, sender string) uint64 {
	store := ctx.KVStore(m.storeKey)
	key := m.sendNonceKey(channel, sender)
	return decodeNonce(store.Get(key))
}

// setSendNonce stores the outgoing nonce for a channel/sender pair
func (m *Manager) setSendNonce(ctx sdk.Context, channel, sender string, nonce uint64) {
	store := ctx.KVStore(m.storeKey)
	key := m.sendNonceKey(channel, sender)
	store.Set(key, encodeNonce(nonce))
}

// NextOutboundNonce generates the next monotonically increasing nonce for outgoing packets.
// It atomically increments and returns the next nonce for the given channel/sender pair.
func (m *Manager) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	if channel == "" {
		channel = "unknown"
	}
	current := m.getSendNonce(ctx, channel, sender)
	next := current + 1
	m.setSendNonce(ctx, channel, sender, next)
	return next
}

// PruneExpiredNonces removes nonces older than the specified TTL.
// Returns the number of nonces pruned and any error encountered.
//
// This implements amortized cleanup to prevent unbounded state growth while
// avoiding O(n) gas spikes. Uses batch processing with configurable limits.
//
// Performance: O(k) where k = maxPrunePerCall, distributed across multiple blocks
func (m *Manager) PruneExpiredNonces(ctx sdk.Context, ttlSeconds int64, maxPrunePerCall int) (int, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = DefaultNonceTTLSeconds
	}
	if maxPrunePerCall <= 0 {
		maxPrunePerCall = 100 // Default batch size
	}

	store := ctx.KVStore(m.storeKey)
	currentTime := ctx.BlockTime().Unix()
	cutoffTime := currentTime - ttlSeconds

	prunedCount := 0
	keysToDelete := make([][]byte, 0, maxPrunePerCall*3) // 3 keys per nonce (nonce, send, timestamp)

	// Iterate through timestamp entries to find expired nonces
	// Timestamp keys have format: NonceTimestampPrefix/channel/sender
	iterator := storetypes.KVStorePrefixIterator(store, []byte(NonceTimestampPrefix))
	defer iterator.Close()

	for ; iterator.Valid() && prunedCount < maxPrunePerCall; iterator.Next() {
		timestampKey := iterator.Key()
		timestamp := int64(decodeNonce(iterator.Value()))

		// Skip nonces that haven't expired yet
		if timestamp > cutoffTime {
			continue
		}

		// Extract channel and sender from the timestamp key
		// Format: NonceTimestampPrefix/channel/sender
		channel, sender := extractChannelSenderFromKey(timestampKey, NonceTimestampPrefix)
		if channel == "" {
			continue // Invalid key format
		}

		// Mark all related keys for deletion (atomic cleanup)
		keysToDelete = append(keysToDelete,
			m.incomingNonceKey(channel, sender), // Incoming nonce
			m.sendNonceKey(channel, sender),     // Outgoing nonce
			timestampKey,                        // Timestamp
		)

		prunedCount++
	}

	// Delete all marked keys atomically
	for _, key := range keysToDelete {
		store.Delete(key)
	}

	return prunedCount, nil
}

// extractChannelSenderFromKey parses channel and sender from a prefixed key.
// Key format: prefix/channel/sender
// Returns empty strings if key format is invalid.
func extractChannelSenderFromKey(key []byte, prefix string) (channel, sender string) {
	prefixBytes := []byte(prefix)
	if len(key) <= len(prefixBytes)+2 { // Need at least prefix + "/" + minimal channel/sender
		return "", ""
	}

	// Skip prefix and first separator
	remainder := string(key[len(prefixBytes)+1:]) // +1 to skip the "/"

	// Split by "/"
	parts := []string{}
	current := ""
	for _, ch := range remainder {
		if ch == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}
