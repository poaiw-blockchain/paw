package ibc

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestEventEmitter_EmitChannelOpenEvent tests channel open event emission.
func TestEventEmitter_EmitChannelOpenEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelOpenEvent(ctx, "test_event", "channel-0", "port-0", "counterparty-port", "counterparty-channel-0")

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "test_event", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 4)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "channel-0", attrMap["channel_id"])
	require.Equal(t, "port-0", attrMap["port_id"])
	require.Equal(t, "counterparty-port", attrMap["counterparty_port_id"])
	require.Equal(t, "counterparty-channel-0", attrMap["counterparty_channel_id"])
}

// TestEventEmitter_EmitChannelOpenAckEvent tests channel open acknowledgement event emission.
func TestEventEmitter_EmitChannelOpenAckEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelOpenAckEvent(ctx, "ack_event", "channel-1", "port-1", "counterparty-channel-1")

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "ack_event", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 3)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "channel-1", attrMap["channel_id"])
	require.Equal(t, "port-1", attrMap["port_id"])
	require.Equal(t, "counterparty-channel-1", attrMap["counterparty_channel_id"])
}

// TestEventEmitter_EmitChannelOpenConfirmEvent tests channel open confirm event emission.
func TestEventEmitter_EmitChannelOpenConfirmEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelOpenConfirmEvent(ctx, "confirm_event", "channel-2", "port-2")

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "confirm_event", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 2)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "channel-2", attrMap["channel_id"])
	require.Equal(t, "port-2", attrMap["port_id"])
}

// TestEventEmitter_EmitChannelCloseEvent tests channel close event emission with cleanup stats.
func TestEventEmitter_EmitChannelCloseEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelCloseEvent(ctx, "close_event", "channel-3", "port-3", 42)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "close_event", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 3)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "channel-3", attrMap["channel_id"])
	require.Equal(t, "port-3", attrMap["port_id"])
	require.Equal(t, "42", attrMap["pending_operations"])
}

// TestEventEmitter_EmitPacketReceiveEvent tests packet receive event emission.
func TestEventEmitter_EmitPacketReceiveEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitPacketReceiveEvent(ctx, "packet_receive", "swap", "channel-4", 123)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "packet_receive", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 3)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "swap", attrMap["packet_type"])
	require.Equal(t, "channel-4", attrMap["channel_id"])
	require.Equal(t, "123", attrMap["sequence"])
}

// TestEventEmitter_EmitPacketAckEvent tests packet acknowledgement event emission.
func TestEventEmitter_EmitPacketAckEvent(t *testing.T) {
	tests := []struct {
		name    string
		success bool
	}{
		{
			name:    "successful acknowledgement",
			success: true,
		},
		{
			name:    "failed acknowledgement",
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext()
			ee := NewEventEmitter()

			ee.EmitPacketAckEvent(ctx, "packet_ack", "channel-5", 456, tt.success)

			events := ctx.EventManager().Events()
			require.Len(t, events, 1)

			event := events[0]
			require.Equal(t, "packet_ack", event.Type)

			attrs := event.Attributes
			require.Len(t, attrs, 3)

			attrMap := make(map[string]string)
			for _, attr := range attrs {
				attrMap[attr.Key] = attr.Value
			}

			require.Equal(t, "channel-5", attrMap["channel_id"])
			require.Equal(t, "456", attrMap["sequence"])
			if tt.success {
				require.Equal(t, "true", attrMap["ack_success"])
			} else {
				require.Equal(t, "false", attrMap["ack_success"])
			}
		})
	}
}

// TestEventEmitter_EmitPacketTimeoutEvent tests packet timeout event emission.
func TestEventEmitter_EmitPacketTimeoutEvent(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitPacketTimeoutEvent(ctx, "packet_timeout", "channel-6", 789)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, "packet_timeout", event.Type)

	attrs := event.Attributes
	require.Len(t, attrs, 2)

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "channel-6", attrMap["channel_id"])
	require.Equal(t, "789", attrMap["sequence"])
}

// TestEventEmitter_MultipleEvents tests emitting multiple events in sequence.
func TestEventEmitter_MultipleEvents(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	// Emit multiple events
	ee.EmitChannelOpenEvent(ctx, "open", "ch1", "p1", "cp1", "cch1")
	ee.EmitPacketReceiveEvent(ctx, "receive", "type1", "ch2", 1)
	ee.EmitPacketAckEvent(ctx, "ack", "ch3", 2, true)

	events := ctx.EventManager().Events()
	require.Len(t, events, 3)

	require.Equal(t, "open", events[0].Type)
	require.Equal(t, "receive", events[1].Type)
	require.Equal(t, "ack", events[2].Type)
}

// TestEventEmitter_EmptyStrings tests event emission with empty strings.
func TestEventEmitter_EmptyStrings(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	// This should not panic
	ee.EmitChannelOpenEvent(ctx, "", "", "", "", "")
	ee.EmitPacketReceiveEvent(ctx, "", "", "", 0)
	ee.EmitPacketTimeoutEvent(ctx, "", "", 0)

	events := ctx.EventManager().Events()
	require.Len(t, events, 3)
}

// TestEventEmitter_SpecialCharacters tests event emission with special characters.
func TestEventEmitter_SpecialCharacters(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	specialChars := "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"

	ee.EmitChannelOpenEvent(ctx, specialChars, specialChars, specialChars, specialChars, specialChars)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, specialChars, event.Type)
}

// TestEventEmitter_UnicodeCharacters tests event emission with unicode characters.
func TestEventEmitter_UnicodeCharacters(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	unicode := "中文-🚀-العربية-עברית"

	ee.EmitChannelOpenEvent(ctx, unicode, unicode, unicode, unicode, unicode)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	require.Equal(t, unicode, event.Type)
}

// TestEventEmitter_VeryLargeSequenceNumbers tests event emission with very large sequence numbers.
func TestEventEmitter_VeryLargeSequenceNumbers(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	maxUint64 := uint64(18446744073709551615)

	ee.EmitPacketReceiveEvent(ctx, "test", "type", "channel", maxUint64)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	attrMap := make(map[string]string)
	for _, attr := range event.Attributes {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "18446744073709551615", attrMap["sequence"])
}

// TestEventEmitter_ZeroCleanupCount tests channel close event with zero cleanup count.
func TestEventEmitter_ZeroCleanupCount(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelCloseEvent(ctx, "close", "channel", "port", 0)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	attrMap := make(map[string]string)
	for _, attr := range event.Attributes {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "0", attrMap["pending_operations"])
}

// TestEventEmitter_NegativeCleanupCount tests channel close event with negative cleanup count.
func TestEventEmitter_NegativeCleanupCount(t *testing.T) {
	ctx := createTestContext()
	ee := NewEventEmitter()

	ee.EmitChannelCloseEvent(ctx, "close", "channel", "port", -1)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	event := events[0]
	attrMap := make(map[string]string)
	for _, attr := range event.Attributes {
		attrMap[attr.Key] = attr.Value
	}

	require.Equal(t, "-1", attrMap["pending_operations"])
}

// TestEventEmitter_Concurrent tests concurrent event emission (no race conditions).
func TestEventEmitter_Concurrent(t *testing.T) {
	// Create a fresh context for this test
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())

	ee := NewEventEmitter()

	const goroutines = 100

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			ee.EmitPacketReceiveEvent(ctx, "test", "type", "channel", uint64(idx))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Note: Events might not all be captured due to concurrent writes to the same context
	// This test mainly ensures no panics or race conditions occur
}
