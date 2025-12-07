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

	// MaxTimestampAge is the maximum age of a packet timestamp (24 hours in seconds)
	MaxTimestampAge = int64(86400)
	// MaxFutureDrift is the maximum allowed clock drift into the future (5 minutes in seconds)
	MaxFutureDrift = int64(300)
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
