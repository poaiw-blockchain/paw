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
	// NonceEpochPrefix is the prefix for nonce epoch tracking
	NonceEpochPrefix = "nonce_epoch"

	// MaxTimestampAge is the maximum age of a packet timestamp (24 hours in seconds)
	MaxTimestampAge = int64(86400)
	// MaxFutureDrift is the maximum allowed clock drift into the future (5 minutes in seconds)
	MaxFutureDrift = int64(300)
	// DefaultNonceTTLSeconds is the default TTL for nonces (7 days)
	DefaultNonceTTLSeconds = int64(604800)

	// SEC-1.5 FIX: Nonce overflow protection constants
	// NonceRotationThreshold is the threshold at which we rotate to a new epoch
	// Set to 90% of max uint64 to provide margin before overflow
	NonceRotationThreshold = uint64(16602069666338596454) // ~90% of MaxUint64
	// MaxNonceValue is the maximum value a nonce can reach before epoch rotation
	MaxNonceValue = ^uint64(0) // MaxUint64
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

// nonceEpochKey generates the store key for nonce epoch tracking
// SEC-1.5 FIX: Epoch key allows tracking nonce version for overflow prevention
func (m *Manager) nonceEpochKey(channel, sender string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", NonceEpochPrefix, channel, m.normalizeSender(sender)))
}

// getEpoch retrieves the current epoch for a channel/sender pair
// SEC-1.5 FIX: Epochs increment when nonces approach overflow threshold
func (m *Manager) getEpoch(ctx sdk.Context, channel, sender string) uint64 {
	store := ctx.KVStore(m.storeKey)
	key := m.nonceEpochKey(channel, sender)
	bz := store.Get(key)
	if len(bz) != 8 {
		return 0
	}
	return decodeNonce(bz)
}

// setEpoch stores the epoch for a channel/sender pair
func (m *Manager) setEpoch(ctx sdk.Context, channel, sender string, epoch uint64) {
	store := ctx.KVStore(m.storeKey)
	key := m.nonceEpochKey(channel, sender)
	store.Set(key, encodeNonce(epoch))
}

// rotateEpoch increments the epoch and resets the nonce counter
// SEC-1.5 FIX: Called when nonce approaches overflow threshold
func (m *Manager) rotateEpoch(ctx sdk.Context, channel, sender string) uint64 {
	currentEpoch := m.getEpoch(ctx, channel, sender)
	newEpoch := currentEpoch + 1
	m.setEpoch(ctx, channel, sender, newEpoch)
	// Reset the nonce counter for the new epoch
	m.setSendNonce(ctx, channel, sender, 0)
	return newEpoch
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
//
// SEC-1.5 FIX: Includes overflow protection via epoch rotation. When the nonce
// approaches the uint64 maximum (at 90% threshold), the epoch is incremented
// and the nonce counter resets to 1. The combined (epoch, nonce) pair provides
// unique identification without risk of overflow.
func (m *Manager) NextOutboundNonce(ctx sdk.Context, channel, sender string) uint64 {
	if channel == "" {
		channel = "unknown"
	}
	current := m.getSendNonce(ctx, channel, sender)

	// SEC-1.5 FIX: Check for approaching overflow and rotate epoch if needed
	if current >= NonceRotationThreshold {
		// Rotate to new epoch and reset nonce
		newEpoch := m.rotateEpoch(ctx, channel, sender)
		// Log the epoch rotation for monitoring
		if logger := ctx.Logger(); logger != nil {
			logger.Info("nonce epoch rotated due to approaching overflow",
				"channel", channel,
				"sender", sender,
				"new_epoch", newEpoch,
				"previous_nonce", current,
			)
		}
		// Return 1 as the first nonce in the new epoch
		m.setSendNonce(ctx, channel, sender, 1)
		return 1
	}

	next := current + 1
	m.setSendNonce(ctx, channel, sender, next)
	return next
}

// GetCurrentEpoch returns the current epoch for a channel/sender pair.
// SEC-1.5 FIX: Allows callers to include epoch in packet data for cross-epoch validation.
func (m *Manager) GetCurrentEpoch(ctx sdk.Context, channel, sender string) uint64 {
	return m.getEpoch(ctx, channel, sender)
}

// VersionedNonce represents a nonce with its epoch for cross-chain communication.
// SEC-1.5 FIX: This structure should be included in IBC packets to ensure
// unique identification even after epoch rotations.
type VersionedNonce struct {
	Epoch uint64 // The epoch number (increments on overflow)
	Nonce uint64 // The nonce within this epoch
}

// NextVersionedNonce returns both the epoch and nonce for use in IBC packets.
// SEC-1.5 FIX: This is the recommended function for getting nonces to include in packets.
func (m *Manager) NextVersionedNonce(ctx sdk.Context, channel, sender string) VersionedNonce {
	nonce := m.NextOutboundNonce(ctx, channel, sender)
	epoch := m.getEpoch(ctx, channel, sender)
	return VersionedNonce{
		Epoch: epoch,
		Nonce: nonce,
	}
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
