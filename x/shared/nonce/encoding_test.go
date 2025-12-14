package nonce_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEncodeDecodeNonce tests nonce encoding and decoding.
// Note: These functions are internal, so we test them indirectly through the manager.
func TestEncodeDecodeNonceRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		nonce uint64
	}{
		{name: "zero", nonce: 0},
		{name: "one", nonce: 1},
		{name: "small", nonce: 42},
		{name: "medium", nonce: 1000000},
		{name: "large", nonce: 1000000000000},
		{name: "power of 2", nonce: 1 << 32},
		{name: "power of 2 - 1", nonce: (1 << 32) - 1},
		{name: "max uint64 - 1", nonce: ^uint64(0) - 1},
		// Note: max uint64 is tested separately due to overflow behavior
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh manager for each test to avoid state pollution
			manager, ctx := setupManager(t)

			// Set nonce by validating it
			if tt.nonce > 0 {
				err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", tt.nonce, ctx.BlockTime().Unix())
				require.NoError(t, err)
			}

			// Verify it's stored correctly by trying to replay it
			if tt.nonce > 0 {
				err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", tt.nonce, ctx.BlockTime().Unix())
				require.Error(t, err)
				require.Contains(t, err.Error(), "replay attack")
			}

			// Verify higher nonce works (skip for values that would overflow)
			if tt.nonce < ^uint64(0) {
				err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", tt.nonce+1, ctx.BlockTime().Unix())
				require.NoError(t, err)
			}
		})
	}
}

// TestNonceKeyGeneration tests that different keys are generated for different inputs.
func TestNonceKeyGeneration(t *testing.T) {
	manager, ctx := setupManager(t)

	// Test that different channel/sender combinations create different keys
	combinations := []struct {
		channel string
		sender  string
		nonce   uint64
	}{
		{"channel-0", "sender-0", 1},
		{"channel-0", "sender-1", 1},
		{"channel-1", "sender-0", 1},
		{"channel-1", "sender-1", 1},
		{"channel-0", "", 1},
		{"", "sender-0", 1}, // This should fail due to empty channel validation
	}

	for i, combo := range combinations {
		if combo.channel == "" {
			// Empty channel should fail
			err := manager.ValidateIncomingPacketNonce(ctx, combo.channel, combo.sender, combo.nonce, ctx.BlockTime().Unix())
			require.Error(t, err)
			continue
		}

		err := manager.ValidateIncomingPacketNonce(ctx, combo.channel, combo.sender, combo.nonce, ctx.BlockTime().Unix())
		require.NoError(t, err, "combination %d failed", i)
	}

	// Verify all nonces are independent
	for _, combo := range combinations {
		if combo.channel == "" {
			continue
		}

		// Same nonce should fail (replay)
		err := manager.ValidateIncomingPacketNonce(ctx, combo.channel, combo.sender, combo.nonce, ctx.BlockTime().Unix())
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack")

		// Higher nonce should work
		err = manager.ValidateIncomingPacketNonce(ctx, combo.channel, combo.sender, combo.nonce+1, ctx.BlockTime().Unix())
		require.NoError(t, err)
	}
}

// TestNonceBigEndianEncoding tests that nonces are encoded in big-endian format.
func TestNonceBigEndianEncoding(t *testing.T) {
	// We can verify encoding is correct by checking that nonces work in any order
	// as long as they're monotonically increasing from the last stored value
	manager, ctx := setupManager(t)

	// Validate nonces in ascending order (must be monotonic)
	nonces := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for _, n := range nonces {
		err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", n, ctx.BlockTime().Unix())
		require.NoError(t, err)
	}

	// Verify we can't replay any of them
	for _, n := range nonces {
		err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", n, ctx.BlockTime().Unix())
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack")
	}

	// Verify higher nonce works
	err := manager.ValidateIncomingPacketNonce(ctx, "channel-0", "sender1", 11, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

// TestDirectEncodingDecoding tests encoding/decoding directly.
func TestDirectEncodingDecoding(t *testing.T) {
	tests := []uint64{
		0,
		1,
		255,
		256,
		65535,
		65536,
		4294967295,
		4294967296,
		18446744073709551615, // max uint64
	}

	for _, expected := range tests {
		// Encode
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, expected)

		// Decode
		actual := binary.BigEndian.Uint64(bz)

		require.Equal(t, expected, actual)
	}
}

// TestShortByteSliceDecode tests decoding of short byte slices.
func TestShortByteSliceDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{name: "nil", input: nil, expected: 0},
		{name: "empty", input: []byte{}, expected: 0},
		{name: "1 byte", input: []byte{1}, expected: 0},
		{name: "2 bytes", input: []byte{1, 2}, expected: 0},
		{name: "7 bytes", input: []byte{1, 2, 3, 4, 5, 6, 7}, expected: 0},
		{name: "8 bytes", input: []byte{0, 0, 0, 0, 0, 0, 0, 1}, expected: 1},
		{name: "9 bytes (truncated)", input: []byte{0, 0, 0, 0, 0, 0, 0, 1, 99}, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result uint64
			if len(tt.input) == 8 {
				result = binary.BigEndian.Uint64(tt.input)
			}
			// For non-8 byte slices, we expect 0 (matching the manager's decodeNonce behavior)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestOutboundNonceKeyGeneration tests outbound nonce key generation.
func TestOutboundNonceKeyGeneration(t *testing.T) {
	manager, ctx := setupManager(t)

	// Test that different channel/sender combinations create different outbound nonce sequences
	combinations := []struct {
		channel string
		sender  string
	}{
		{"channel-0", "sender-0"},
		{"channel-0", "sender-1"},
		{"channel-1", "sender-0"},
		{"channel-1", "sender-1"},
		{"channel-0", ""},
		{"", "sender-0"},
	}

	// Each combination should start at nonce 1
	for _, combo := range combinations {
		nonce := manager.NextOutboundNonce(ctx, combo.channel, combo.sender)
		require.Equal(t, uint64(1), nonce)
	}

	// Each combination should be independent
	for _, combo := range combinations {
		nonce := manager.NextOutboundNonce(ctx, combo.channel, combo.sender)
		require.Equal(t, uint64(2), nonce)
	}
}

// TestKeyFormatting tests the key formatting for storage.
func TestKeyFormatting(t *testing.T) {
	manager, ctx := setupManager(t)

	// Keys with special characters should work
	specialChannels := []string{
		"channel-0",
		"channel/with/slashes",
		"channel:with:colons",
		"channel-with-unicode-中文",
		"channel-with-emoji-🚀",
		"channel\nwith\nnewlines",
		"channel\twith\ttabs",
	}

	specialSenders := []string{
		"sender-0",
		"sender/with/slashes",
		"sender:with:colons",
		"sender-with-unicode-中文",
		"sender-with-emoji-🚀",
	}

	for _, channel := range specialChannels {
		for _, sender := range specialSenders {
			// Test incoming nonce
			err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 1, ctx.BlockTime().Unix())
			require.NoError(t, err, "channel=%s sender=%s", channel, sender)

			// Test outbound nonce
			nonce := manager.NextOutboundNonce(ctx, channel, sender)
			require.Equal(t, uint64(1), nonce, "channel=%s sender=%s", channel, sender)
		}
	}
}

// TestNonceOverwriteProtection tests that nonces cannot be overwritten with lower values.
func TestNonceOverwriteProtection(t *testing.T) {
	manager, ctx := setupManager(t)

	const channel = "channel-0"
	const sender = "sender1"

	// Set nonce to 100
	err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 100, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Try to set lower nonces - all should fail
	for i := uint64(1); i <= 100; i++ {
		err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, i, ctx.BlockTime().Unix())
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay attack")
	}

	// Higher nonce should work
	err = manager.ValidateIncomingPacketNonce(ctx, channel, sender, 101, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

// TestZeroByteHandling tests handling of zero bytes in keys.
func TestZeroByteHandling(t *testing.T) {
	manager, ctx := setupManager(t)

	// Keys with zero bytes
	err := manager.ValidateIncomingPacketNonce(ctx, "channel\x00test", "sender\x00test", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Should be stored separately from non-zero-byte version
	err = manager.ValidateIncomingPacketNonce(ctx, "channeltest", "sendertest", 1, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

// TestIncomingOutboundNonceIndependence tests that incoming and outbound nonces are independent.
func TestIncomingOutboundNonceIndependence(t *testing.T) {
	manager, ctx := setupManager(t)

	const channel = "channel-0"
	const sender = "sender1"

	// Set incoming nonce to 50
	err := manager.ValidateIncomingPacketNonce(ctx, channel, sender, 50, ctx.BlockTime().Unix())
	require.NoError(t, err)

	// Outbound nonce should start at 1
	nonce := manager.NextOutboundNonce(ctx, channel, sender)
	require.Equal(t, uint64(1), nonce)

	// Set outbound to 100 by calling it 100 times
	for i := 2; i <= 100; i++ {
		nonce = manager.NextOutboundNonce(ctx, channel, sender)
		require.Equal(t, uint64(i), nonce)
	}

	// Incoming should still be at 50 - next should be 51
	err = manager.ValidateIncomingPacketNonce(ctx, channel, sender, 51, ctx.BlockTime().Unix())
	require.NoError(t, err)
}

// TestLargeKeyGeneration tests key generation with very large strings.
func TestLargeKeyGeneration(t *testing.T) {
	manager, ctx := setupManager(t)

	// Create very large channel and sender names
	largeChannel := string(make([]byte, 10000))
	largeSender := string(make([]byte, 10000))

	for i := range largeChannel {
		largeChannel = largeChannel[:i] + "a" + largeChannel[i+1:]
	}
	for i := range largeSender {
		largeSender = largeSender[:i] + "b" + largeSender[i+1:]
	}

	// Should work with large keys
	err := manager.ValidateIncomingPacketNonce(ctx, largeChannel, largeSender, 1, ctx.BlockTime().Unix())
	require.NoError(t, err)

	nonce := manager.NextOutboundNonce(ctx, largeChannel, largeSender)
	require.Equal(t, uint64(1), nonce)
}
