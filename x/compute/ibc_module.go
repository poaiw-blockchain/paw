package compute

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the compute module.
// This enables cross-chain compute job distribution and result verification.
type IBCModule struct {
	keeper keeper.Keeper
	cdc    codec.Codec
}

// NewIBCModule creates a new IBCModule given the keeper and codec
func NewIBCModule(keeper keeper.Keeper, cdc codec.Codec) IBCModule {
	return IBCModule{
		keeper: keeper,
		cdc:    cdc,
	}
}

// OnChanOpenInit implements the IBCModule interface
// Validates the channel creation for compute operations
func (im IBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	// Validate channel ordering - compute requires ordered channels for job sequencing
	if order != channeltypes.ORDERED {
		return "", sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.ORDERED, order)
	}

	// Validate version
	if version != types.IBCVersion {
		return "", sdkerrors.Wrapf(types.ErrInvalidPacket,
			"expected version %s, got %s", types.IBCVersion, version)
	}

	// Validate port
	if portID != types.PortID {
		return "", sdkerrors.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", types.PortID, portID)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpen,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyPortID, counterparty.PortId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterparty.ChannelId),
		),
	)

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// Validate channel ordering
	if order != channeltypes.ORDERED {
		return "", sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.ORDERED, order)
	}

	// Validate version
	if counterpartyVersion != types.IBCVersion {
		return "", sdkerrors.Wrapf(types.ErrInvalidPacket,
			"invalid counterparty version: expected %s, got %s", types.IBCVersion, counterpartyVersion)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpen,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyPortID, counterparty.PortId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterparty.ChannelId),
		),
	)

	return types.IBCVersion, nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	// Validate counterparty version
	if counterpartyVersion != types.IBCVersion {
		return sdkerrors.Wrapf(types.ErrInvalidPacket,
			"invalid counterparty version: expected %s, got %s", types.IBCVersion, counterpartyVersion)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpenAck,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterpartyChannelID),
		),
	)

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpenConfirm,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
		),
	)

	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for compute
	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelClose,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
		),
	)

	return nil
}

// OnRecvPacket implements the IBCModule interface
// Handles incoming compute packets (job results, provider info, etc.)
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// Parse packet data
	packetData, err := types.ParsePacketData(packet.Data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(types.ErrInvalidPacket, "failed to parse packet data: %s", err.Error()))
	}

	// Validate packet
	if err := packetData.ValidateBasic(); err != nil {
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrap(types.ErrInvalidPacket, err.Error()))
	}

	// Route packet based on type
	var ack ibcexported.Acknowledgement
	switch packetData.GetType() {
	case types.JobResultType:
		// Handle job result from remote provider
		ack = im.handleJobResult(ctx, packet, packetData)

	case types.DiscoverProvidersType:
		// Handle provider discovery request from remote chain
		ack = im.handleDiscoverProviders(ctx, packet, packetData)

	case types.SubmitJobType:
		// Handle job submission from remote requester
		ack = im.handleSubmitJob(ctx, packet, packetData)

	case types.JobStatusType:
		// Handle job status query
		ack = im.handleJobStatusQuery(ctx, packet, packetData)

	default:
		return channeltypes.NewErrorAcknowledgement(
			sdkerrors.Wrapf(types.ErrInvalidPacket, "unknown packet type: %s", packetData.GetType()))
	}

	// Emit receive event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketReceive,
			sdk.NewAttribute(types.AttributeKeyPacketType, packetData.GetType()),
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
		),
	)

	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
// Handles acknowledgements for sent packets
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	var ack channeltypes.Acknowledgement
	if err := im.cdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest,
			"cannot unmarshal packet acknowledgement: %v", err)
	}

	// Delegate to keeper's acknowledgement handler
	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, ack); err != nil {
		return err
	}

	// Emit acknowledgement event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketAck,
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
		),
	)

	return nil
}

// OnTimeoutPacket implements the IBCModule interface
// Handles packet timeout (refunds, cleanup, etc.)
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// Delegate to keeper's timeout handler
	if err := im.keeper.OnTimeoutPacket(ctx, packet); err != nil {
		return err
	}

	// Emit timeout event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketTimeout,
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
		),
	)

	return nil
}

// Helper functions for packet handling

func (im IBCModule) handleJobResult(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Delegate to keeper
	ack, err := im.keeper.OnRecvPacket(ctx, packet)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	return ack
}

func (im IBCModule) handleDiscoverProviders(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Return empty provider list for now
	// In production, this would query local compute providers
	ackData := types.DiscoverProvidersAcknowledgement{
		Success:   true,
		Providers: []types.ProviderInfo{},
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleSubmitJob(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Job submission from remote chain
	// In production, this would validate and queue the job
	ackData := types.SubmitJobAcknowledgement{
		Success: true,
		Status:  "pending",
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleJobStatusQuery(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Job status query
	ackData := types.JobStatusAcknowledgement{
		Success: true,
		Status:  "unknown",
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}
