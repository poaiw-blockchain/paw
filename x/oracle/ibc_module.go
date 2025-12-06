package oracle

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

	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the oracle module.
// This enables cross-chain price feed aggregation and oracle data sharing.
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
// Validates the channel creation for oracle operations
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
	// Oracle can use unordered channels for better performance
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
	// Disallow user-initiated channel closing for oracle
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	pending := im.keeper.GetPendingOperations(ctx, channelID)
	cleanup := 0
	for _, op := range pending {
		if err := im.keeper.RefundOnChannelClose(ctx, op); err != nil {
			ctx.Logger().Error("failed to cleanup oracle channel operation",
				"channel", channelID,
				"sequence", op.Sequence,
				"type", op.PacketType,
				"error", err,
			)
			continue
		}
		cleanup++
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelClose,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyPendingOperations, fmt.Sprintf("%d", cleanup)),
		),
	)

	return nil
}

// OnRecvPacket implements the IBCModule interface
// Handles incoming oracle packets (price updates, heartbeats, queries)
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	if !im.keeper.IsAuthorizedChannel(ctx, packet.SourcePort, packet.SourceChannel) {
		err := errorsmod.Wrapf(types.ErrUnauthorizedChannel, "port %s channel %s not authorized", packet.SourcePort, packet.SourceChannel)
		ctx.Logger().Error("unauthorized oracle packet source", "port", packet.SourcePort, "channel", packet.SourceChannel)
		return channeltypes.NewErrorAcknowledgement(err)
	}

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

	packetNonce := im.packetNonce(packetData)
	if packetNonce == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "packet nonce missing"))
	}

	packetTimestamp := im.packetTimestamp(packetData)
	if packetTimestamp == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "packet timestamp missing"))
	}

	sender := im.packetSender(packet, packetData)
	if err := im.keeper.ValidateIncomingPacketNonce(ctx, packet.SourceChannel, sender, packetNonce, packetTimestamp); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// Route packet based on type
	var ack ibcexported.Acknowledgement
	switch packetData.GetType() {
	case types.SubscribePricesType:
		// Handle price subscription request
		ack = im.handleSubscribePrices(ctx, packet, packetData, packetNonce)

	case types.QueryPriceType:
		// Handle price query
		ack = im.handleQueryPrice(ctx, packet, packetData, packetNonce)

	case types.PriceUpdateType:
		// Handle price update broadcast
		ack = im.handlePriceUpdate(ctx, packet, packetData, packetNonce)

	case types.OracleHeartbeatType:
		// Handle oracle heartbeat
		ack = im.handleOracleHeartbeat(ctx, packet, packetData, packetNonce)

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
	const maxAcknowledgementSize = 1024 * 1024 // 1MB guard against malicious acknowledgements
	if len(acknowledgement) > maxAcknowledgementSize {
		return errorsmod.Wrapf(types.ErrInvalidAck, "ack too large: %d > %d", len(acknowledgement), maxAcknowledgementSize)
	}

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
// Handles packet timeout
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

func (im IBCModule) handleSubscribePrices(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.SubscribePricesPacketData)

	ackData := types.SubscribePricesAcknowledgement{
		Nonce:             nonce,
		Success:           true,
		SubscribedSymbols: req.Symbols,
		SubscriptionID:    fmt.Sprintf("sub-%s-%d", packet.SourceChannel, packet.Sequence),
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleQueryPrice(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.QueryPricePacketData)

	priceData, err := im.keeper.BuildPriceData(ctx, req.Symbol)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	ackData := types.QueryPriceAcknowledgement{
		Nonce:     nonce,
		Success:   true,
		PriceData: priceData,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handlePriceUpdate(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	// Delegate to keeper
	ack, err := im.keeper.OnRecvPacket(ctx, packet, nonce)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	return ack
}

func (im IBCModule) handleOracleHeartbeat(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	// Delegate to keeper
	ack, err := im.keeper.OnRecvPacket(ctx, packet, nonce)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	return ack
}

func (im IBCModule) packetNonce(packetData types.IBCPacketData) uint64 {
	switch req := packetData.(type) {
	case types.SubscribePricesPacketData:
		return req.Nonce
	case types.QueryPricePacketData:
		return req.Nonce
	case types.PriceUpdatePacketData:
		return req.Nonce
	case types.OracleHeartbeatPacketData:
		return req.Nonce
	default:
		return 0
	}
}

func (im IBCModule) packetSender(packet channeltypes.Packet, packetData types.IBCPacketData) string {
	switch req := packetData.(type) {
	case types.SubscribePricesPacketData:
		if req.Subscriber != "" {
			return req.Subscriber
		}
	case types.QueryPricePacketData:
		if req.Sender != "" {
			return req.Sender
		}
	case types.PriceUpdatePacketData:
		if req.Source != "" {
			return req.Source
		}
	case types.OracleHeartbeatPacketData:
		if req.ChainID != "" {
			return req.ChainID
		}
	}
	if packet.SourcePort != "" {
		return packet.SourcePort
	}
	return packet.SourceChannel
}

func (im IBCModule) packetTimestamp(packetData types.IBCPacketData) int64 {
	switch req := packetData.(type) {
	case types.SubscribePricesPacketData:
		return req.Timestamp
	case types.QueryPricePacketData:
		return req.Timestamp
	case types.PriceUpdatePacketData:
		return req.Timestamp
	case types.OracleHeartbeatPacketData:
		return req.Timestamp
	default:
		return 0
	}
}
