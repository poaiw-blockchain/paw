package dex

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the DEX module.
// This enables cross-chain liquidity aggregation and atomic swaps.
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
// Validates the channel creation for DEX operations
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
	// DEX can use unordered channels for better throughput
	if order != channeltypes.UNORDERED {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.UNORDERED, order)
	}

	// Validate version
	if version != types.IBCVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket,
			"expected version %s, got %s", types.IBCVersion, version)
	}

	// Validate port
	if portID != types.PortID {
		return "", errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", types.PortID, portID)
	}

	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", errorsmod.Wrap(err, "failed to claim channel capability")
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
	if order != channeltypes.UNORDERED {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.UNORDERED, order)
	}

	// Validate version
	if counterpartyVersion != types.IBCVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket,
			"invalid counterparty version: expected %s, got %s", types.IBCVersion, counterpartyVersion)
	}

	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", errorsmod.Wrap(err, "failed to claim channel capability")
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
		return errorsmod.Wrapf(types.ErrInvalidPacket,
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
	// Disallow user-initiated channel closing for DEX
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
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
// Handles incoming DEX packets (pool queries, swap execution, updates)
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// Parse packet data
	packetData, err := types.ParsePacketData(packet.Data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrInvalidPacket, "failed to parse packet data: %s", err.Error()))
	}

	// Validate packet
	if err := packetData.ValidateBasic(); err != nil {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrap(types.ErrInvalidPacket, err.Error()))
	}

	// Route packet based on type
	var ack ibcexported.Acknowledgement
	switch packetData.GetType() {
	case types.QueryPoolsType:
		// Handle pool query from remote chain
		ack = im.handleQueryPools(ctx, packet, packetData)

	case types.ExecuteSwapType:
		// Handle swap execution request
		ack = im.handleExecuteSwap(ctx, packet, packetData)

	case types.CrossChainSwapType:
		// Handle multi-hop cross-chain swap
		ack = im.handleCrossChainSwap(ctx, packet, packetData)

	case types.PoolUpdateType:
		// Handle pool state update broadcast
		ack = im.handlePoolUpdate(ctx, packet, packetData)

	default:
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrInvalidPacket, "unknown packet type: %s", packetData.GetType()))
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
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest,
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
// Handles packet timeout (refunds swaps, cleanup)
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

func (im IBCModule) handleQueryPools(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Query local pools and return info
	// In production, this would query actual pool state
	ackData := types.QueryPoolsAcknowledgement{
		Success: true,
		Pools:   []types.PoolInfo{},
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleExecuteSwap(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Execute swap on local pools
	// In production, this would perform actual swap
	ackData := types.ExecuteSwapAcknowledgement{
		Success: true,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleCrossChainSwap(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Handle multi-hop cross-chain swap
	ackData := types.CrossChainSwapAcknowledgement{
		Success: true,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handlePoolUpdate(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
) ibcexported.Acknowledgement {
	// Store pool update from remote chain
	return channeltypes.NewResultAcknowledgement([]byte("{\"success\":true}"))
}
