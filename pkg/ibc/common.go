package ibc

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ChannelCapabilityManager defines the interface for claiming channel capabilities
type ChannelCapabilityManager interface {
	ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error
}

// ChannelValidator performs common IBC channel validation
type ChannelValidator struct {
	expectedOrder   channeltypes.Order
	expectedVersion string
	expectedPortID  string
}

// NewChannelValidator creates a new channel validator with expected parameters
func NewChannelValidator(order channeltypes.Order, version string, portID string) *ChannelValidator {
	return &ChannelValidator{
		expectedOrder:   order,
		expectedVersion: version,
		expectedPortID:  portID,
	}
}

// ValidateChannelInit validates channel initialization parameters
//
// This performs common validation that all IBC modules need:
// - Channel ordering must match expected ordering
// - Version must match expected version
// - Port ID must match expected port
//
// Security considerations:
//   - Prevents unauthorized channel creation with wrong parameters
//   - Ensures consistency across channel lifecycle
//
// Returns error if any validation fails
func (v *ChannelValidator) ValidateChannelInit(
	order channeltypes.Order,
	portID string,
	version string,
) error {
	// Validate channel ordering
	if order != v.expectedOrder {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", v.expectedOrder, order)
	}

	// Validate version
	if version != v.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidVersion,
			"expected version %s, got %s", v.expectedVersion, version)
	}

	// Validate port
	if portID != v.expectedPortID {
		return errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", v.expectedPortID, portID)
	}

	return nil
}

// ValidateChannelTry validates channel handshake try parameters
func (v *ChannelValidator) ValidateChannelTry(
	order channeltypes.Order,
	counterpartyVersion string,
) error {
	// Validate channel ordering
	if order != v.expectedOrder {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", v.expectedOrder, order)
	}

	// Validate counterparty version
	if counterpartyVersion != v.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidVersion,
			"invalid counterparty version: expected %s, got %s", v.expectedVersion, counterpartyVersion)
	}

	return nil
}

// ValidateChannelAck validates channel acknowledgement parameters
func (v *ChannelValidator) ValidateChannelAck(counterpartyVersion string) error {
	// Validate counterparty version
	if counterpartyVersion != v.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidVersion,
			"invalid counterparty version: expected %s, got %s", v.expectedVersion, counterpartyVersion)
	}

	return nil
}

// ClaimChannelCapability claims the channel capability and returns error if it fails
func ClaimChannelCapability(
	ctx sdk.Context,
	capManager ChannelCapabilityManager,
	chanCap *capabilitytypes.Capability,
	portID string,
	channelID string,
) error {
	if err := capManager.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return errorsmod.Wrap(err, "failed to claim channel capability")
	}
	return nil
}

// EmitChannelOpenEvent emits a standardized channel open event
func EmitChannelOpenEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
	counterparty channeltypes.Counterparty,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
			sdk.NewAttribute("counterparty_port_id", counterparty.PortId),
			sdk.NewAttribute("counterparty_channel_id", counterparty.ChannelId),
		),
	)
}

// EmitChannelOpenAckEvent emits a standardized channel open acknowledgement event
func EmitChannelOpenAckEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
	counterpartyChannelID string,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
			sdk.NewAttribute("counterparty_channel_id", counterpartyChannelID),
		),
	)
}

// EmitChannelOpenConfirmEvent emits a standardized channel open confirm event
func EmitChannelOpenConfirmEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
		),
	)
}

// EmitChannelCloseEvent emits a standardized channel close event with pending operations count
func EmitChannelCloseEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
	pendingOperations int,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
			sdk.NewAttribute("pending_operations", fmt.Sprintf("%d", pendingOperations)),
		),
	)
}

// EmitPacketReceiveEvent emits a standardized packet receive event
func EmitPacketReceiveEvent(
	ctx sdk.Context,
	eventType string,
	packetType string,
	channelID string,
	sequence uint64,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("packet_type", packetType),
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", sequence)),
		),
	)
}

// EmitPacketAckEvent emits a standardized packet acknowledgement event
func EmitPacketAckEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	sequence uint64,
	success bool,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", sequence)),
			sdk.NewAttribute("ack_success", fmt.Sprintf("%t", success)),
		),
	)
}

// EmitPacketTimeoutEvent emits a standardized packet timeout event
func EmitPacketTimeoutEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	sequence uint64,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", sequence)),
		),
	)
}

// PacketDataValidator defines common packet validation interface
type PacketDataValidator interface {
	ValidateBasic() error
	GetType() string
}

// ValidateIncomingPacket performs common packet validation
//
// Security considerations:
//   - Checks packet size limits to prevent DOS
//   - Validates basic packet structure
//   - Ensures required fields are present
//
// Parameters:
//   - packet: The IBC packet to validate
//   - maxPacketSize: Maximum allowed packet size in bytes
//
// Returns error if validation fails
func ValidateIncomingPacket(
	packet channeltypes.Packet,
	maxPacketSize int,
) error {
	// Validate packet size to prevent DOS attacks
	if len(packet.Data) > maxPacketSize {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"packet data too large: %d > %d", len(packet.Data), maxPacketSize)
	}

	// Validate required fields
	if packet.SourcePort == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty source port")
	}
	if packet.SourceChannel == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty source channel")
	}
	if packet.DestinationPort == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty destination port")
	}
	if packet.DestinationChannel == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty destination channel")
	}

	return nil
}

// CreateErrorAck creates a standardized error acknowledgement
func CreateErrorAck(err error) ibcexported.Acknowledgement {
	return channeltypes.NewErrorAcknowledgement(err)
}

// CreateSuccessAck creates a standardized success acknowledgement
func CreateSuccessAck(result []byte) ibcexported.Acknowledgement {
	return channeltypes.NewResultAcknowledgement(result)
}

// ValidateAcknowledgement validates and unmarshals an acknowledgement
//
// Security considerations:
//   - Limits acknowledgement size to prevent memory DOS
//   - Validates JSON structure before unmarshalling
//
// Parameters:
//   - acknowledgement: Raw acknowledgement bytes
//   - maxAckSize: Maximum allowed acknowledgement size
//
// Returns unmarshalled acknowledgement or error
func ValidateAcknowledgement(acknowledgement []byte, maxAckSize int) (*channeltypes.Acknowledgement, error) {
	// Validate size to prevent malicious memory pressure
	if len(acknowledgement) > maxAckSize {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"acknowledgement too large: %d > %d", len(acknowledgement), maxAckSize)
	}

	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest,
			"cannot unmarshal packet acknowledgement: %v", err)
	}

	return &ack, nil
}

// DisallowUserChannelClose returns error for user-initiated channel closing
//
// Most IBC modules should not allow users to close channels as this can
// disrupt ongoing operations and lead to fund loss.
func DisallowUserChannelClose() error {
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}
