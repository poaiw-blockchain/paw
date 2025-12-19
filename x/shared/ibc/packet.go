package ibc

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hashicorp/go-metrics"
)

// PacketData defines the interface that all IBC packet data types must implement.
// This allows the shared utilities to work with any module's packet types.
type PacketData interface {
	// ValidateBasic performs basic validation of the packet data
	ValidateBasic() error
	// GetType returns the packet type identifier
	GetType() string
}

// NonceValidator defines the interface for validating packet nonces.
// Each module must implement this to validate incoming packet nonces.
type NonceValidator interface {
	// ValidateIncomingPacketNonce validates the nonce of an incoming packet
	ValidateIncomingPacketNonce(ctx sdk.Context, sourceChannel, sender string, nonce uint64, timestamp int64) error
}

// ChannelAuthorizer defines the interface for channel authorization.
// Each module must implement this to control which channels are authorized.
type ChannelAuthorizer interface {
	// IsAuthorizedChannel checks if a source port/channel is authorized
	IsAuthorizedChannel(ctx sdk.Context, sourcePort, sourceChannel string) error
}

// CapabilityClaimer defines the interface for claiming channel capabilities.
type CapabilityClaimer interface {
	// ClaimCapability claims a channel capability
	ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error
}

// PendingOperation represents a pending IBC operation that needs cleanup on channel close.
type PendingOperation struct {
	Sequence   uint64
	PacketType string
	Data       interface{}
}

// ChannelCloseHandler defines the interface for handling channel close operations.
type ChannelCloseHandler interface {
	// GetPendingOperations retrieves pending operations for a channel
	GetPendingOperations(ctx sdk.Context, channelID string) []PendingOperation
	// RefundOnChannelClose processes refunds/cleanup for a pending operation
	RefundOnChannelClose(ctx sdk.Context, op PendingOperation) error
}

// PacketHandler defines the interface for module-specific packet processing.
type PacketHandler interface {
	// ProcessPacket handles the module-specific packet processing
	ProcessPacket(ctx sdk.Context, packet channeltypes.Packet, packetData PacketData, nonce uint64) ibcexported.Acknowledgement
}

// AcknowledgementHandler defines the interface for handling packet acknowledgements.
type AcknowledgementHandler interface {
	// OnAcknowledgementPacket processes a packet acknowledgement
	OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, ack channeltypes.Acknowledgement) error
}

// TimeoutHandler defines the interface for handling packet timeouts.
type TimeoutHandler interface {
	// OnTimeoutPacket processes a packet timeout
	OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error
}

// PacketValidator provides common packet validation logic.
type PacketValidator struct {
	nonceValidator NonceValidator
	authorizer     ChannelAuthorizer
}

// NewPacketValidator creates a new packet validator.
func NewPacketValidator(nonceValidator NonceValidator, authorizer ChannelAuthorizer) *PacketValidator {
	return &PacketValidator{
		nonceValidator: nonceValidator,
		authorizer:     authorizer,
	}
}

// ValidateIncomingPacket performs comprehensive validation of an incoming packet.
// This consolidates validation logic that is duplicated across all three modules.
//
// Security considerations:
//   - Channel authorization prevents unauthorized packet sources
//   - Nonce validation prevents replay attacks
//   - Timestamp validation prevents time-based attacks
//   - Basic packet validation prevents malformed data
func (pv *PacketValidator) ValidateIncomingPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData PacketData,
	nonce uint64,
	timestamp int64,
	sender string,
) error {
	// Validate channel authorization
	if err := pv.authorizer.IsAuthorizedChannel(ctx, packet.SourcePort, packet.SourceChannel); err != nil {
		if logger := ctx.Logger(); logger != nil {
			logger.Error("unauthorized packet source",
				"port", packet.SourcePort,
				"channel", packet.SourceChannel,
				"error", err)
		}
		emitValidationFailure(ctx, packet.SourcePort, packet.SourceChannel, err.Error())
		return errorsmod.Wrapf(err, "port %s channel %s not authorized", packet.SourcePort, packet.SourceChannel)
	}

	// Validate packet data
	if err := packetData.ValidateBasic(); err != nil {
		emitValidationFailure(ctx, packet.SourcePort, packet.SourceChannel, err.Error())
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	// Validate nonce
	if nonce == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "packet nonce missing")
	}

	// Validate timestamp
	if timestamp == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "packet timestamp missing")
	}

	// Validate nonce against replay attacks
	if err := pv.nonceValidator.ValidateIncomingPacketNonce(ctx, packet.SourceChannel, sender, nonce, timestamp); err != nil {
		emitValidationFailure(ctx, packet.SourcePort, packet.SourceChannel, err.Error())
		return err
	}

	return nil
}

// ChannelOpenValidator provides common channel opening validation logic.
type ChannelOpenValidator struct {
	expectedVersion  string
	expectedPort     string
	expectedOrdering channeltypes.Order
	claimer          CapabilityClaimer
}

// NewChannelOpenValidator creates a new channel open validator.
func NewChannelOpenValidator(
	version string,
	port string,
	ordering channeltypes.Order,
	claimer CapabilityClaimer,
) *ChannelOpenValidator {
	return &ChannelOpenValidator{
		expectedVersion:  version,
		expectedPort:     port,
		expectedOrdering: ordering,
		claimer:          claimer,
	}
}

// ValidateChannelOpenInit validates channel opening initialization.
// This consolidates validation logic from OnChanOpenInit across all modules.
func (cov *ChannelOpenValidator) ValidateChannelOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	version string,
) error {
	// Validate channel ordering
	if order != cov.expectedOrdering {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", cov.expectedOrdering, order)
	}

	// Validate version
	if version != cov.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"expected version %s, got %s", cov.expectedVersion, version)
	}

	// Validate port
	if portID != cov.expectedPort {
		return errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", cov.expectedPort, portID)
	}

	// Claim channel capability
	if err := cov.claimer.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return errorsmod.Wrap(err, "failed to claim channel capability")
	}

	return nil
}

// ValidateChannelOpenTry validates channel opening on the counterparty side.
// This consolidates validation logic from OnChanOpenTry across all modules.
func (cov *ChannelOpenValidator) ValidateChannelOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterpartyVersion string,
) error {
	// Validate channel ordering
	if order != cov.expectedOrdering {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", cov.expectedOrdering, order)
	}

	// Validate counterparty version
	if counterpartyVersion != cov.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid counterparty version: expected %s, got %s", cov.expectedVersion, counterpartyVersion)
	}

	// Validate port
	if portID != cov.expectedPort {
		return errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", cov.expectedPort, portID)
	}

	// Claim channel capability
	if err := cov.claimer.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return errorsmod.Wrap(err, "failed to claim channel capability")
	}

	return nil
}

// ValidateChannelOpenAck validates channel opening acknowledgement.
func (cov *ChannelOpenValidator) ValidateChannelOpenAck(counterpartyVersion string) error {
	if counterpartyVersion != cov.expectedVersion {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid counterparty version: expected %s, got %s", cov.expectedVersion, counterpartyVersion)
	}
	return nil
}

// HandleChannelClose handles channel close operations including cleanup of pending operations.
// This consolidates the OnChanCloseConfirm logic from all modules.
//
// Security considerations:
//   - Ensures all pending operations are properly cleaned up
//   - Refunds escrowed funds on channel close
//   - Logs failures for manual intervention
func HandleChannelClose(
	ctx sdk.Context,
	channelID string,
	handler ChannelCloseHandler,
) (cleanedUp int, err error) {
	pending := handler.GetPendingOperations(ctx, channelID)
	cleaned := 0

	for _, op := range pending {
		if err := handler.RefundOnChannelClose(ctx, op); err != nil {
			if logger := ctx.Logger(); logger != nil {
				logger.Error("failed to cleanup channel operation",
					"channel", channelID,
					"sequence", op.Sequence,
					"type", op.PacketType,
					"error", err,
				)
			}
			continue
		}
		cleaned++
	}

	return cleaned, nil
}

// AcknowledgementHelper provides common acknowledgement handling logic.
type AcknowledgementHelper struct{}

// NewAcknowledgementHelper creates a new acknowledgement helper.
func NewAcknowledgementHelper() *AcknowledgementHelper {
	return &AcknowledgementHelper{}
}

func emitValidationFailure(ctx sdk.Context, port, channel, reason string) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"ibc_packet_validation_failed",
			sdk.NewAttribute("port", port),
			sdk.NewAttribute("channel", channel),
			sdk.NewAttribute("reason", reason),
		),
	)
	telemetry.IncrCounterWithLabels(
		[]string{"ibc", "packet_validation_failed"},
		1,
		[]metrics.Label{
			telemetry.NewLabel("port", port),
			telemetry.NewLabel("channel", channel),
			telemetry.NewLabel("reason", reason),
		},
	)
}

// ValidateAndUnmarshalAck validates and unmarshals an acknowledgement.
// This consolidates the OnAcknowledgementPacket validation logic.
//
// Security considerations:
//   - Maximum acknowledgement size prevents memory exhaustion attacks
//   - Proper error handling prevents malformed acknowledgements from causing issues
func (ah *AcknowledgementHelper) ValidateAndUnmarshalAck(acknowledgement []byte) (channeltypes.Acknowledgement, error) {
	const maxAcknowledgementSize = 1024 * 1024 // 1MB limit to prevent memory exhaustion
	if len(acknowledgement) > maxAcknowledgementSize {
		return channeltypes.Acknowledgement{}, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"ack too large: %d > %d bytes", len(acknowledgement), maxAcknowledgementSize)
	}

	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return channeltypes.Acknowledgement{}, errorsmod.Wrapf(
			sdkerrors.ErrUnknownRequest,
			"cannot unmarshal packet acknowledgement: %v", err)
	}

	return ack, nil
}

// CreateSuccessAck creates a successful acknowledgement with the given data.
func CreateSuccessAck(data []byte) channeltypes.Acknowledgement {
	return channeltypes.NewResultAcknowledgement(data)
}

// CreateErrorAck creates an error acknowledgement with the given error.
func CreateErrorAck(err error) channeltypes.Acknowledgement {
	return channeltypes.NewErrorAcknowledgement(err)
}

// EventEmitter provides common event emission logic for IBC operations.
type EventEmitter struct{}

// NewEventEmitter creates a new event emitter.
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{}
}

// EmitChannelOpenEvent emits a channel open event.
func (ee *EventEmitter) EmitChannelOpenEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
	counterpartyPortID string,
	counterpartyChannelID string,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
			sdk.NewAttribute("counterparty_port_id", counterpartyPortID),
			sdk.NewAttribute("counterparty_channel_id", counterpartyChannelID),
		),
	)
}

// EmitChannelOpenAckEvent emits a channel open acknowledgement event.
func (ee *EventEmitter) EmitChannelOpenAckEvent(
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

// EmitChannelOpenConfirmEvent emits a channel open confirm event.
func (ee *EventEmitter) EmitChannelOpenConfirmEvent(
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

// EmitChannelCloseEvent emits a channel close event with cleanup stats.
func (ee *EventEmitter) EmitChannelCloseEvent(
	ctx sdk.Context,
	eventType string,
	channelID string,
	portID string,
	cleanedUpCount int,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute("channel_id", channelID),
			sdk.NewAttribute("port_id", portID),
			sdk.NewAttribute("pending_operations", fmt.Sprintf("%d", cleanedUpCount)),
		),
	)
}

// EmitPacketReceiveEvent emits a packet receive event.
func (ee *EventEmitter) EmitPacketReceiveEvent(
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

// EmitPacketAckEvent emits a packet acknowledgement event.
func (ee *EventEmitter) EmitPacketAckEvent(
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

// EmitPacketTimeoutEvent emits a packet timeout event.
func (ee *EventEmitter) EmitPacketTimeoutEvent(
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
