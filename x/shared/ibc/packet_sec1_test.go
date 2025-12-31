package ibc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSEC1_4_PacketTypeWhitelist tests that the packet type whitelist
// correctly validates packet types (SEC-1.4 fix)
func TestSEC1_4_PacketTypeWhitelist(t *testing.T) {
	testCases := []struct {
		name        string
		packetType  string
		shouldError bool
	}{
		// Valid compute packet types
		{name: "compute_request valid", packetType: "compute_request", shouldError: false},
		{name: "compute_result valid", packetType: "compute_result", shouldError: false},
		{name: "compute_cancel valid", packetType: "compute_cancel", shouldError: false},
		{name: "compute_status valid", packetType: "compute_status", shouldError: false},
		{name: "compute_ack valid", packetType: "compute_ack", shouldError: false},
		{name: "compute_timeout valid", packetType: "compute_timeout", shouldError: false},

		// Valid oracle packet types
		{name: "oracle_price valid", packetType: "oracle_price", shouldError: false},
		{name: "oracle_price_feed valid", packetType: "oracle_price_feed", shouldError: false},
		{name: "oracle_price_batch valid", packetType: "oracle_price_batch", shouldError: false},
		{name: "oracle_subscribe valid", packetType: "oracle_subscribe", shouldError: false},
		{name: "oracle_unsubscribe valid", packetType: "oracle_unsubscribe", shouldError: false},
		{name: "oracle_ack valid", packetType: "oracle_ack", shouldError: false},

		// Valid DEX packet types
		{name: "dex_swap valid", packetType: "dex_swap", shouldError: false},
		{name: "dex_liquidity valid", packetType: "dex_liquidity", shouldError: false},
		{name: "dex_order valid", packetType: "dex_order", shouldError: false},
		{name: "dex_cancel valid", packetType: "dex_cancel", shouldError: false},
		{name: "dex_settlement valid", packetType: "dex_settlement", shouldError: false},
		{name: "dex_ack valid", packetType: "dex_ack", shouldError: false},

		// Valid shared packet types
		{name: "heartbeat valid", packetType: "heartbeat", shouldError: false},
		{name: "ping valid", packetType: "ping", shouldError: false},
		{name: "pong valid", packetType: "pong", shouldError: false},

		// Invalid packet types
		{name: "empty string", packetType: "", shouldError: true},
		{name: "unknown type", packetType: "unknown_packet", shouldError: true},
		{name: "malicious type", packetType: "evil_packet", shouldError: true},
		{name: "sql injection attempt", packetType: "'; DROP TABLE packets;--", shouldError: true},
		{name: "null byte", packetType: "packet\x00type", shouldError: true},
		{name: "very long type", packetType: "a_very_long_packet_type_that_should_not_exist_in_the_whitelist", shouldError: true},
		{name: "case mismatch", packetType: "COMPUTE_REQUEST", shouldError: true},
		{name: "partial match", packetType: "compute", shouldError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePacketType(tc.packetType)
			if tc.shouldError {
				require.Error(t, err)
				if tc.packetType == "" {
					require.Contains(t, err.Error(), "cannot be empty")
				} else {
					require.Contains(t, err.Error(), "unknown packet type")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSEC1_4_RegisterPacketType tests that custom packet types can be registered
func TestSEC1_4_RegisterPacketType(t *testing.T) {
	customType := "custom_module_packet"

	// Should fail before registration
	err := ValidatePacketType(customType)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown packet type")

	// Register the custom type
	RegisterPacketType(customType)

	// Should pass after registration
	err = ValidatePacketType(customType)
	require.NoError(t, err)

	// Clean up - remove from whitelist to not affect other tests
	delete(ValidPacketTypes, customType)
}

// TestSEC1_4_RegisterEmptyPacketType tests that empty types cannot be registered
func TestSEC1_4_RegisterEmptyPacketType(t *testing.T) {
	// Registering empty string should be a no-op
	initialCount := len(ValidPacketTypes)
	RegisterPacketType("")
	require.Equal(t, initialCount, len(ValidPacketTypes))

	// Empty string should still fail validation
	err := ValidatePacketType("")
	require.Error(t, err)
}

// TestSEC1_4_AllValidTypesPresent ensures all expected packet types are in the whitelist
func TestSEC1_4_AllValidTypesPresent(t *testing.T) {
	expectedTypes := []string{
		// Compute module
		"compute_request", "compute_result", "compute_cancel",
		"compute_status", "compute_ack", "compute_timeout",
		// Oracle module
		"oracle_price", "oracle_price_feed", "oracle_price_batch",
		"oracle_subscribe", "oracle_unsubscribe", "oracle_ack",
		// DEX module
		"dex_swap", "dex_liquidity", "dex_order",
		"dex_cancel", "dex_settlement", "dex_ack",
		// Shared
		"heartbeat", "ping", "pong",
	}

	for _, expected := range expectedTypes {
		require.True(t, ValidPacketTypes[expected], "expected packet type %s not in whitelist", expected)
	}
}
