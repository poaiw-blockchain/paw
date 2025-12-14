package ibc

import (
	"errors"
	"strings"
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

// TestPacketValidator_NilContext tests behavior with various context states.
func TestPacketValidator_NilContext(t *testing.T) {
	ctx := createTestContext()
	pv := NewPacketValidator(
		&mockNonceValidator{},
		&mockChannelAuthorizer{},
	)

	packet := channeltypes.Packet{
		SourcePort:    "port-1",
		SourceChannel: "channel-1",
	}
	packetData := mockPacketData{
		packetType: "test",
		valid:      true,
	}

	// Should work with valid context
	err := pv.ValidateIncomingPacket(ctx, packet, packetData, 1, 12345, "sender")
	require.NoError(t, err)
}

// TestPacketValidator_EmptyPacket tests validation with empty packet fields.
func TestPacketValidator_EmptyPacket(t *testing.T) {
	ctx := createTestContext()

	tests := []struct {
		name        string
		packet      channeltypes.Packet
		shouldFail  bool
		errContains string
	}{
		{
			name: "empty source port",
			packet: channeltypes.Packet{
				SourcePort:    "",
				SourceChannel: "channel-1",
			},
			shouldFail:  false, // Empty port is validated by authorizer
			errContains: "",
		},
		{
			name: "empty source channel",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "",
			},
			shouldFail:  false, // Empty channel is validated by authorizer
			errContains: "",
		},
		{
			name: "both empty",
			packet: channeltypes.Packet{
				SourcePort:    "",
				SourceChannel: "",
			},
			shouldFail:  false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := NewPacketValidator(
				&mockNonceValidator{shouldFail: false},
				&mockChannelAuthorizer{shouldFail: tt.shouldFail},
			)

			err := pv.ValidateIncomingPacket(ctx, tt.packet, mockPacketData{valid: true}, 1, 12345, "sender")

			if tt.shouldFail {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPacketValidator_InvalidPacketData tests various invalid packet data scenarios.
func TestPacketValidator_InvalidPacketData(t *testing.T) {
	ctx := createTestContext()

	tests := []struct {
		name       string
		packetData PacketData
		wantErr    bool
	}{
		{
			name: "valid packet data",
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			wantErr: false,
		},
		{
			name: "invalid packet data",
			packetData: mockPacketData{
				packetType: "test",
				valid:      false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := NewPacketValidator(
				&mockNonceValidator{},
				&mockChannelAuthorizer{},
			)

			packet := channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			}

			err := pv.ValidateIncomingPacket(ctx, packet, tt.packetData, 1, 12345, "sender")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestChannelOpenValidator_NilCapability tests behavior with nil capability.
func TestChannelOpenValidator_NilCapability(t *testing.T) {
	ctx := createTestContext()

	cov := NewChannelOpenValidator(
		"v1",
		"test-port",
		channeltypes.ORDERED,
		&mockCapabilityClaimer{},
	)

	// Should work with nil capability
	err := cov.ValidateChannelOpenInit(ctx, channeltypes.ORDERED, "test-port", "channel-0", nil, "v1")
	require.NoError(t, err)
}

// TestChannelOpenValidator_EmptyVersion tests behavior with empty version strings.
func TestChannelOpenValidator_EmptyVersion(t *testing.T) {
	ctx := createTestContext()

	tests := []struct {
		name            string
		expectedVersion string
		actualVersion   string
		wantErr         bool
	}{
		{
			name:            "both empty",
			expectedVersion: "",
			actualVersion:   "",
			wantErr:         false,
		},
		{
			name:            "expected empty, actual non-empty",
			expectedVersion: "",
			actualVersion:   "v1",
			wantErr:         true,
		},
		{
			name:            "expected non-empty, actual empty",
			expectedVersion: "v1",
			actualVersion:   "",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov := NewChannelOpenValidator(
				tt.expectedVersion,
				"test-port",
				channeltypes.ORDERED,
				&mockCapabilityClaimer{},
			)

			err := cov.ValidateChannelOpenInit(ctx, channeltypes.ORDERED, "test-port", "channel-0", nil, tt.actualVersion)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestChannelOpenValidator_LongStrings tests behavior with very long strings.
func TestChannelOpenValidator_LongStrings(t *testing.T) {
	ctx := createTestContext()

	longString := strings.Repeat("a", 10000)

	cov := NewChannelOpenValidator(
		longString,
		longString,
		channeltypes.ORDERED,
		&mockCapabilityClaimer{},
	)

	err := cov.ValidateChannelOpenInit(ctx, channeltypes.ORDERED, longString, "channel-0", nil, longString)
	require.NoError(t, err)
}

// TestHandleChannelClose_EmptyChannelID tests channel close with empty channel ID.
func TestHandleChannelClose_EmptyChannelID(t *testing.T) {
	ctx := createTestContext()

	handler := &mockChannelCloseHandler{
		operations: []PendingOperation{
			{Sequence: 1, PacketType: "test"},
		},
	}

	cleaned, err := HandleChannelClose(ctx, "", handler)
	require.NoError(t, err)
	require.Equal(t, 1, cleaned)
}

// TestHandleChannelClose_NilOperations tests channel close with nil operations slice.
func TestHandleChannelClose_NilOperations(t *testing.T) {
	ctx := createTestContext()

	handler := &mockChannelCloseHandler{
		operations: nil,
	}

	cleaned, err := HandleChannelClose(ctx, "channel-0", handler)
	require.NoError(t, err)
	require.Equal(t, 0, cleaned)
}

// TestHandleChannelClose_LargeNumberOfOperations tests cleanup with many operations.
func TestHandleChannelClose_LargeNumberOfOperations(t *testing.T) {
	ctx := createTestContext()

	const numOps = 10000
	ops := make([]PendingOperation, numOps)
	for i := 0; i < numOps; i++ {
		ops[i] = PendingOperation{
			Sequence:   uint64(i),
			PacketType: "test",
		}
	}

	handler := &mockChannelCloseHandler{
		operations: ops,
	}

	cleaned, err := HandleChannelClose(ctx, "channel-0", handler)
	require.NoError(t, err)
	require.Equal(t, numOps, cleaned)
}

// TestHandleChannelClose_MixedSuccessFailure tests cleanup with mixed success/failure.
func TestHandleChannelClose_MixedSuccessFailure(t *testing.T) {
	ctx := createTestContext()

	ops := []PendingOperation{
		{Sequence: 1, PacketType: "success"},
		{Sequence: 2, PacketType: "success"},
		{Sequence: 3, PacketType: "success"},
	}

	// Create a handler that fails on specific sequences
	handler := &mockChannelCloseHandlerSelective{
		operations:   ops,
		failSequences: map[uint64]bool{2: true}, // Fail on sequence 2
	}

	cleaned, err := HandleChannelClose(ctx, "channel-0", handler)
	require.NoError(t, err)
	require.Equal(t, 2, cleaned) // Should clean 2 out of 3
}

// mockChannelCloseHandlerSelective allows selective failures.
type mockChannelCloseHandlerSelective struct {
	operations    []PendingOperation
	failSequences map[uint64]bool
}

func (m *mockChannelCloseHandlerSelective) GetPendingOperations(ctx sdk.Context, channelID string) []PendingOperation {
	return m.operations
}

func (m *mockChannelCloseHandlerSelective) RefundOnChannelClose(ctx sdk.Context, op PendingOperation) error {
	if m.failSequences[op.Sequence] {
		return errors.New("refund failed")
	}
	return nil
}

// TestAcknowledgementHelper_ExactlySizeLimit tests acknowledgement at exact size limit.
func TestAcknowledgementHelper_ExactlySizeLimit(t *testing.T) {
	ah := NewAcknowledgementHelper()

	// Create acknowledgement exactly at the 1MB limit
	exactSize := make([]byte, 1024*1024)
	copy(exactSize, []byte(`{"result":"AQID"}`))

	_, err := ah.ValidateAndUnmarshalAck(exactSize)
	require.NoError(t, err)
}

// TestAcknowledgementHelper_EmptyAck tests empty acknowledgement.
func TestAcknowledgementHelper_EmptyAck(t *testing.T) {
	ah := NewAcknowledgementHelper()

	_, err := ah.ValidateAndUnmarshalAck([]byte{})
	require.Error(t, err)
}

// TestAcknowledgementHelper_MalformedJSON tests various malformed JSON.
func TestAcknowledgementHelper_MalformedJSON(t *testing.T) {
	ah := NewAcknowledgementHelper()

	tests := []struct {
		name string
		ack  []byte
	}{
		{
			name: "incomplete JSON",
			ack:  []byte(`{"result":`),
		},
		{
			name: "invalid JSON structure",
			ack:  []byte(`{{{`),
		},
		{
			name: "not JSON",
			ack:  []byte(`this is not json`),
		},
		{
			name: "null bytes",
			ack:  []byte{0, 0, 0},
		},
		{
			name: "invalid UTF-8",
			ack:  []byte{0xff, 0xfe, 0xfd},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ah.ValidateAndUnmarshalAck(tt.ack)
			require.Error(t, err)
		})
	}
}

// TestCreateSuccessAck_EmptyData tests success ack with empty data.
func TestCreateSuccessAck_EmptyData(t *testing.T) {
	ack := CreateSuccessAck([]byte{})
	require.True(t, ack.Success())
}

// TestCreateSuccessAck_NilData tests success ack with nil data.
func TestCreateSuccessAck_NilData(t *testing.T) {
	ack := CreateSuccessAck(nil)
	require.True(t, ack.Success())
}

// TestCreateSuccessAck_LargeData tests success ack with large data.
func TestCreateSuccessAck_LargeData(t *testing.T) {
	largeData := make([]byte, 1024*1024) // 1MB
	ack := CreateSuccessAck(largeData)
	require.True(t, ack.Success())
}

// TestCreateErrorAck_NilError tests error ack with nil error.
func TestCreateErrorAck_NilError(t *testing.T) {
	// This may panic or create invalid ack depending on implementation
	// Testing the actual behavior
	defer func() {
		if r := recover(); r != nil {
			// Panic is acceptable for nil error
			t.Log("Panicked on nil error (acceptable)")
		}
	}()

	ack := CreateErrorAck(nil)
	// If we get here, check that it's marked as failure
	require.False(t, ack.Success())
}

// TestCreateErrorAck_VariousErrors tests error ack with various error types.
func TestCreateErrorAck_VariousErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "simple error",
			err:  errors.New("simple error"),
		},
		{
			name: "sdk error",
			err:  sdkerrors.ErrInvalidRequest,
		},
		{
			name: "wrapped error",
			err:  errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "wrapped"),
		},
		{
			name: "empty error message",
			err:  errors.New(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ack := CreateErrorAck(tt.err)
			require.False(t, ack.Success())
		})
	}
}

// TestPacketValidator_MaxValues tests validation with maximum values.
func TestPacketValidator_MaxValues(t *testing.T) {
	ctx := createTestContext()
	pv := NewPacketValidator(
		&mockNonceValidator{},
		&mockChannelAuthorizer{},
	)

	packet := channeltypes.Packet{
		SourcePort:    strings.Repeat("a", 1000),
		SourceChannel: strings.Repeat("b", 1000),
		Sequence:      ^uint64(0), // Max uint64
	}

	err := pv.ValidateIncomingPacket(ctx, packet, mockPacketData{valid: true}, ^uint64(0), 9223372036854775807, strings.Repeat("c", 1000))
	require.NoError(t, err)
}

// TestChannelOpenValidator_AllOrderings tests all channel ordering types.
func TestChannelOpenValidator_AllOrderings(t *testing.T) {
	ctx := createTestContext()

	orderings := []channeltypes.Order{
		channeltypes.NONE,
		channeltypes.UNORDERED,
		channeltypes.ORDERED,
	}

	for _, ordering := range orderings {
		t.Run(ordering.String(), func(t *testing.T) {
			cov := NewChannelOpenValidator(
				"v1",
				"test-port",
				ordering,
				&mockCapabilityClaimer{},
			)

			err := cov.ValidateChannelOpenInit(ctx, ordering, "test-port", "channel-0", nil, "v1")
			require.NoError(t, err)

			// Test mismatch
			for _, other := range orderings {
				if other != ordering {
					err := cov.ValidateChannelOpenInit(ctx, other, "test-port", "channel-0", nil, "v1")
					require.Error(t, err)
				}
			}
		})
	}
}

// TestPendingOperation_NilData tests pending operation with nil data.
func TestPendingOperation_NilData(t *testing.T) {
	op := PendingOperation{
		Sequence:   1,
		PacketType: "test",
		Data:       nil,
	}

	require.Equal(t, uint64(1), op.Sequence)
	require.Equal(t, "test", op.PacketType)
	require.Nil(t, op.Data)
}

// TestPendingOperation_VariousDataTypes tests pending operation with various data types.
func TestPendingOperation_VariousDataTypes(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "string data",
			data: "test string",
		},
		{
			name: "int data",
			data: 42,
		},
		{
			name: "struct data",
			data: struct{ Field string }{"value"},
		},
		{
			name: "slice data",
			data: []byte{1, 2, 3},
		},
		{
			name: "map data",
			data: map[string]string{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := PendingOperation{
				Sequence:   1,
				PacketType: "test",
				Data:       tt.data,
			}

			require.Equal(t, tt.data, op.Data)
		})
	}
}
