package ibcutil

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthorizedChannel represents a whitelisted IBC port/channel pair.
// This is used across DEX, Oracle, and Compute modules to authorize
// incoming IBC packets from trusted counterparty chains.
type AuthorizedChannel struct {
	PortId    string
	ChannelId string
}

// ChannelStore defines the interface that module keepers must implement
// to support shared channel authorization logic. This interface abstracts
// the underlying parameter storage mechanism.
type ChannelStore interface {
	// GetAuthorizedChannels retrieves the current list of authorized port/channel pairs.
	// Returns an error if params cannot be loaded from state.
	GetAuthorizedChannels(ctx context.Context) ([]AuthorizedChannel, error)

	// SetAuthorizedChannels persists the updated list of authorized port/channel pairs.
	// Returns an error if params cannot be saved to state.
	SetAuthorizedChannels(ctx context.Context, channels []AuthorizedChannel) error
}

// Error codes for IBC channel authorization failures.
// These are registered in each module with module-specific error codes.
var (
	// ErrInvalidChannel is returned when port_id or channel_id is empty after trimming.
	ErrInvalidChannel = errorsmod.Register("ibcutil", 1, "invalid channel")

	// ErrUnauthorizedChannel is returned when checking authorization for a port/channel
	// that is not in the allowlist.
	ErrUnauthorizedChannel = errorsmod.Register("ibcutil", 2, "unauthorized channel")
)

// AuthorizeChannel adds a port/channel pair to the allowlist, preventing duplicates.
//
// This function is used by governance proposals or admin messages to whitelist
// IBC channels for cross-chain communication.
//
// Security considerations:
//   - Input validation: port_id and channel_id are trimmed and must be non-empty
//   - Deduplication: existing entries are not duplicated in the list
//   - Atomicity: uses the store's SetAuthorizedChannels to ensure atomic updates
//
// Parameters:
//   - ctx: SDK context for state access
//   - store: Module keeper implementing the ChannelStore interface
//   - portID: IBC port identifier (e.g., "transfer", "oracle", "compute")
//   - channelID: IBC channel identifier (e.g., "channel-0", "channel-42")
//
// Returns:
//   - nil on success (including if already authorized)
//   - ErrInvalidChannel if port_id or channel_id is empty
//   - error from store if params cannot be loaded or saved
func AuthorizeChannel(ctx context.Context, store ChannelStore, portID, channelID string) error {
	// Normalize inputs: trim whitespace to prevent accidental mismatches
	portID = strings.TrimSpace(portID)
	channelID = strings.TrimSpace(channelID)

	// Validate: both identifiers must be non-empty
	if portID == "" || channelID == "" {
		return ErrInvalidChannel.Wrap("port_id and channel_id must be non-empty")
	}

	// Load current authorized channels from state
	channels, err := store.GetAuthorizedChannels(ctx)
	if err != nil {
		return err
	}

	// Check for duplicates: if already authorized, return success (idempotent)
	for _, ch := range channels {
		if ch.PortId == portID && ch.ChannelId == channelID {
			return nil // Already authorized
		}
	}

	// Append new channel to allowlist
	channels = append(channels, AuthorizedChannel{
		PortId:    portID,
		ChannelId: channelID,
	})

	// Persist updated list
	return store.SetAuthorizedChannels(ctx, channels)
}

// IsAuthorizedChannel checks whether the provided port/channel pair is whitelisted.
//
// This function is called during IBC packet reception to validate that incoming
// packets are from trusted channels. It is a critical security function.
//
// Security considerations:
//   - Must be called before processing any IBC packet data
//   - Returns false on any error (fail-safe behavior)
//   - Case-sensitive exact match required
//
// Parameters:
//   - ctx: SDK context for state access
//   - store: Module keeper implementing the ChannelStore interface
//   - portID: IBC port identifier of the incoming packet
//   - channelID: IBC channel identifier of the incoming packet
//
// Returns:
//   - true if the channel is authorized
//   - false if not authorized or if an error occurs loading params
func IsAuthorizedChannel(ctx context.Context, store ChannelStore, portID, channelID string) bool {
	channels, err := store.GetAuthorizedChannels(ctx)
	if err != nil {
		// Fail-safe: if we can't load params, deny access
		if sdkCtx, ok := ctx.(sdk.Context); ok {
			sdkCtx.Logger().Error("failed to load authorized channels", "error", err)
		}
		return false
	}

	// Linear search through allowlist (typically small, <10 entries)
	for _, ch := range channels {
		if ch.PortId == portID && ch.ChannelId == channelID {
			return true
		}
	}

	return false
}

// SetAuthorizedChannelsWithValidation replaces the entire channel allowlist,
// normalizing and deduplicating entries.
//
// This function is typically used by governance proposals to bulk-update
// the authorized channel list.
//
// Security considerations:
//   - Input validation: all port_id and channel_id must be non-empty
//   - Deduplication: prevents duplicate entries in the allowlist
//   - Normalization: trims whitespace to prevent accidental mismatches
//
// Parameters:
//   - ctx: SDK context for state access
//   - store: Module keeper implementing the ChannelStore interface
//   - channels: New list of authorized port/channel pairs
//
// Returns:
//   - nil on success
//   - ErrInvalidChannel if any port_id or channel_id is empty
//   - error from store if params cannot be saved
func SetAuthorizedChannelsWithValidation(ctx context.Context, store ChannelStore, channels []AuthorizedChannel) error {
	normalized := make([]AuthorizedChannel, 0, len(channels))
	seen := make(map[string]struct{}, len(channels))

	for _, ch := range channels {
		// Normalize: trim whitespace
		portID := strings.TrimSpace(ch.PortId)
		channelID := strings.TrimSpace(ch.ChannelId)

		// Validate: both must be non-empty
		if portID == "" || channelID == "" {
			return ErrInvalidChannel.Wrap("port_id and channel_id must be non-empty")
		}

		// Deduplicate: skip if already seen
		key := portID + "/" + channelID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		// Add to normalized list
		normalized = append(normalized, AuthorizedChannel{
			PortId:    portID,
			ChannelId: channelID,
		})
	}

	// Persist normalized, deduplicated list
	return store.SetAuthorizedChannels(ctx, normalized)
}
