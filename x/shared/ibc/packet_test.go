package ibc

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockPacketData struct {
	packetType string
	valid      bool
}

func (m mockPacketData) ValidateBasic() error {
	if !m.valid {
		return errors.New("invalid packet data")
	}
	return nil
}

func (m mockPacketData) GetType() string {
	return m.packetType
}

type mockNonceValidator struct {
	shouldFail bool
}

func (m *mockNonceValidator) ValidateIncomingPacketNonce(ctx sdk.Context, sourceChannel, sender string, nonce uint64, timestamp int64) error {
	if m.shouldFail {
		return errors.New("invalid nonce")
	}
	return nil
}

type mockChannelAuthorizer struct {
	shouldFail bool
}

func (m *mockChannelAuthorizer) IsAuthorizedChannel(ctx sdk.Context, sourcePort, sourceChannel string) error {
	if m.shouldFail {
		return sdkerrors.ErrUnauthorized
	}
	return nil
}

type mockCapabilityClaimer struct {
	shouldFail bool
}

func (m *mockCapabilityClaimer) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	if m.shouldFail {
		return errors.New("failed to claim capability")
	}
	return nil
}

type mockChannelCloseHandler struct {
	operations      []PendingOperation
	refundShouldFail bool
}

func (m *mockChannelCloseHandler) GetPendingOperations(ctx sdk.Context, channelID string) []PendingOperation {
	return m.operations
}

func (m *mockChannelCloseHandler) RefundOnChannelClose(ctx sdk.Context, op PendingOperation) error {
	if m.refundShouldFail {
		return errors.New("refund failed")
	}
	return nil
}

func TestPacketValidator_ValidateIncomingPacket(t *testing.T) {
	ctx := sdk.Context{}

	tests := []struct {
		name             string
		packet           channeltypes.Packet
		packetData       PacketData
		nonce            uint64
		timestamp        int64
		sender           string
		authorizerFails  bool
		nonceValidatorFails bool
		wantErr          bool
		errContains      string
	}{
		{
			name: "valid packet",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			nonce:     1,
			timestamp: 12345,
			sender:    "sender-1",
			wantErr:   false,
		},
		{
			name: "unauthorized channel",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			nonce:           1,
			timestamp:       12345,
			sender:          "sender-1",
			authorizerFails: true,
			wantErr:         true,
			errContains:     "not authorized",
		},
		{
			name: "invalid packet data",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      false,
			},
			nonce:     1,
			timestamp: 12345,
			sender:    "sender-1",
			wantErr:   true,
			errContains: "invalid packet data",
		},
		{
			name: "missing nonce",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			nonce:       0,
			timestamp:   12345,
			sender:      "sender-1",
			wantErr:     true,
			errContains: "nonce missing",
		},
		{
			name: "missing timestamp",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			nonce:       1,
			timestamp:   0,
			sender:      "sender-1",
			wantErr:     true,
			errContains: "timestamp missing",
		},
		{
			name: "invalid nonce",
			packet: channeltypes.Packet{
				SourcePort:    "port-1",
				SourceChannel: "channel-1",
			},
			packetData: mockPacketData{
				packetType: "test",
				valid:      true,
			},
			nonce:               1,
			timestamp:           12345,
			sender:              "sender-1",
			nonceValidatorFails: true,
			wantErr:             true,
			errContains:         "invalid nonce",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := NewPacketValidator(
				&mockNonceValidator{shouldFail: tt.nonceValidatorFails},
				&mockChannelAuthorizer{shouldFail: tt.authorizerFails},
			)

			err := pv.ValidateIncomingPacket(ctx, tt.packet, tt.packetData, tt.nonce, tt.timestamp, tt.sender)

			if tt.wantErr {
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

func TestChannelOpenValidator_ValidateChannelOpenInit(t *testing.T) {
	ctx := sdk.Context{}

	tests := []struct {
		name         string
		order        channeltypes.Order
		portID       string
		channelID    string
		version      string
		claimerFails bool
		wantErr      bool
		errContains  string
	}{
		{
			name:      "valid channel open init - ordered",
			order:     channeltypes.ORDERED,
			portID:    "test-port",
			channelID: "channel-0",
			version:   "v1",
			wantErr:   false,
		},
		{
			name:      "valid channel open init - unordered",
			order:     channeltypes.UNORDERED,
			portID:    "test-port",
			channelID: "channel-0",
			version:   "v1",
			wantErr:   false,
		},
		{
			name:        "invalid channel ordering",
			order:       channeltypes.UNORDERED,
			portID:      "test-port",
			channelID:   "channel-0",
			version:     "v1",
			wantErr:     true,
			errContains: "expected ORDERED channel",
		},
		{
			name:        "invalid version",
			order:       channeltypes.ORDERED,
			portID:      "test-port",
			channelID:   "channel-0",
			version:     "v2",
			wantErr:     true,
			errContains: "expected version v1, got v2",
		},
		{
			name:        "invalid port",
			order:       channeltypes.ORDERED,
			portID:      "wrong-port",
			channelID:   "channel-0",
			version:     "v1",
			wantErr:     true,
			errContains: "expected port test-port, got wrong-port",
		},
		{
			name:         "claim capability fails",
			order:        channeltypes.ORDERED,
			portID:       "test-port",
			channelID:    "channel-0",
			version:      "v1",
			claimerFails: true,
			wantErr:      true,
			errContains:  "failed to claim channel capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with ORDERED expectation
			cov := NewChannelOpenValidator(
				"v1",
				"test-port",
				channeltypes.ORDERED,
				&mockCapabilityClaimer{shouldFail: tt.claimerFails},
			)

			err := cov.ValidateChannelOpenInit(ctx, tt.order, tt.portID, tt.channelID, nil, tt.version)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}

	// Test UNORDERED validator
	t.Run("unordered validator accepts unordered channels", func(t *testing.T) {
		cov := NewChannelOpenValidator(
			"v1",
			"test-port",
			channeltypes.UNORDERED,
			&mockCapabilityClaimer{},
		)

		err := cov.ValidateChannelOpenInit(ctx, channeltypes.UNORDERED, "test-port", "channel-0", nil, "v1")
		require.NoError(t, err)
	})
}

func TestChannelOpenValidator_ValidateChannelOpenTry(t *testing.T) {
	ctx := sdk.Context{}

	tests := []struct {
		name                string
		order               channeltypes.Order
		portID              string
		channelID           string
		counterpartyVersion string
		claimerFails        bool
		wantErr             bool
		errContains         string
	}{
		{
			name:                "valid channel open try",
			order:               channeltypes.ORDERED,
			portID:              "test-port",
			channelID:           "channel-0",
			counterpartyVersion: "v1",
			wantErr:             false,
		},
		{
			name:                "invalid channel ordering",
			order:               channeltypes.UNORDERED,
			portID:              "test-port",
			channelID:           "channel-0",
			counterpartyVersion: "v1",
			wantErr:             true,
			errContains:         "expected ORDERED channel",
		},
		{
			name:                "invalid counterparty version",
			order:               channeltypes.ORDERED,
			portID:              "test-port",
			channelID:           "channel-0",
			counterpartyVersion: "v2",
			wantErr:             true,
			errContains:         "expected v1, got v2",
		},
		{
			name:                "claim capability fails",
			order:               channeltypes.ORDERED,
			portID:              "test-port",
			channelID:           "channel-0",
			counterpartyVersion: "v1",
			claimerFails:        true,
			wantErr:             true,
			errContains:         "failed to claim channel capability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov := NewChannelOpenValidator(
				"v1",
				"test-port",
				channeltypes.ORDERED,
				&mockCapabilityClaimer{shouldFail: tt.claimerFails},
			)

			err := cov.ValidateChannelOpenTry(ctx, tt.order, tt.portID, tt.channelID, nil, tt.counterpartyVersion)

			if tt.wantErr {
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

func TestChannelOpenValidator_ValidateChannelOpenAck(t *testing.T) {
	tests := []struct {
		name                string
		counterpartyVersion string
		wantErr             bool
		errContains         string
	}{
		{
			name:                "valid counterparty version",
			counterpartyVersion: "v1",
			wantErr:             false,
		},
		{
			name:                "invalid counterparty version",
			counterpartyVersion: "v2",
			wantErr:             true,
			errContains:         "expected v1, got v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov := NewChannelOpenValidator(
				"v1",
				"test-port",
				channeltypes.ORDERED,
				&mockCapabilityClaimer{},
			)

			err := cov.ValidateChannelOpenAck(tt.counterpartyVersion)

			if tt.wantErr {
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

func TestHandleChannelClose(t *testing.T) {
	ctx := sdk.Context{}

	tests := []struct {
		name             string
		operations       []PendingOperation
		refundShouldFail bool
		wantCleaned      int
	}{
		{
			name: "all operations cleaned successfully",
			operations: []PendingOperation{
				{Sequence: 1, PacketType: "swap"},
				{Sequence: 2, PacketType: "query"},
				{Sequence: 3, PacketType: "update"},
			},
			refundShouldFail: false,
			wantCleaned:      3,
		},
		{
			name: "some operations fail to clean",
			operations: []PendingOperation{
				{Sequence: 1, PacketType: "swap"},
				{Sequence: 2, PacketType: "query"},
				{Sequence: 3, PacketType: "update"},
			},
			refundShouldFail: true,
			wantCleaned:      0,
		},
		{
			name:             "no pending operations",
			operations:       []PendingOperation{},
			refundShouldFail: false,
			wantCleaned:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockChannelCloseHandler{
				operations:      tt.operations,
				refundShouldFail: tt.refundShouldFail,
			}

			cleaned, err := HandleChannelClose(ctx, "channel-0", handler)

			require.NoError(t, err)
			require.Equal(t, tt.wantCleaned, cleaned)
		})
	}
}

func TestAcknowledgementHelper_ValidateAndUnmarshalAck(t *testing.T) {
	tests := []struct {
		name        string
		ack         []byte
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid acknowledgement",
			ack:     []byte(`{"result":"AQID"}`),
			wantErr: false,
		},
		{
			name:        "acknowledgement too large",
			ack:         make([]byte, 1024*1024+1),
			wantErr:     true,
			errContains: "ack too large",
		},
		{
			name:        "invalid JSON",
			ack:         []byte(`{invalid json`),
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah := NewAcknowledgementHelper()

			_, err := ah.ValidateAndUnmarshalAck(tt.ack)

			if tt.wantErr {
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

func TestCreateSuccessAck(t *testing.T) {
	data := []byte("test data")
	ack := CreateSuccessAck(data)

	require.True(t, ack.Success())
}

func TestCreateErrorAck(t *testing.T) {
	err := errors.New("test error")
	ack := CreateErrorAck(err)

	require.False(t, ack.Success())
}
