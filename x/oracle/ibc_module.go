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
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the oracle module.
// This enables cross-chain price feed aggregation and oracle data sharing.
type IBCModule struct {
	keeper            keeper.Keeper
	cdc               codec.Codec
	channelValidator  *sharedibc.ChannelOpenValidator
	packetValidator   *sharedibc.PacketValidator
	ackHelper         *sharedibc.AcknowledgementHelper
	eventEmitter      *sharedibc.EventEmitter
}

// NewIBCModule creates a new IBCModule given the keeper and codec
func NewIBCModule(keeper keeper.Keeper, cdc codec.Codec) IBCModule {
	// Create adapter to make keeper compatible with shared interfaces
	adapter := newKeeperAdapter(&keeper)

	return IBCModule{
		keeper: keeper,
		cdc:    cdc,
		channelValidator: sharedibc.NewChannelOpenValidator(
			types.IBCVersion,
			types.PortID,
			channeltypes.UNORDERED, // Oracle uses unordered channels for better performance
			adapter,
		),
		packetValidator: sharedibc.NewPacketValidator(adapter, adapter),
		ackHelper:       sharedibc.NewAcknowledgementHelper(),
		eventEmitter:    sharedibc.NewEventEmitter(),
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
	// Use shared validation logic
	if err := im.channelValidator.ValidateChannelOpenInit(ctx, order, portID, channelID, chanCap, version); err != nil {
		return "", err
	}

	// Emit event using shared emitter
	im.eventEmitter.EmitChannelOpenEvent(
		ctx,
		types.EventTypeChannelOpen,
		channelID,
		portID,
		counterparty.PortId,
		counterparty.ChannelId,
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
	// Use shared validation logic
	if err := im.channelValidator.ValidateChannelOpenTry(ctx, order, portID, channelID, chanCap, counterpartyVersion); err != nil {
		return "", err
	}

	// Emit event using shared emitter
	im.eventEmitter.EmitChannelOpenEvent(
		ctx,
		types.EventTypeChannelOpen,
		channelID,
		portID,
		counterparty.PortId,
		counterparty.ChannelId,
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
	// Use shared validation logic
	if err := im.channelValidator.ValidateChannelOpenAck(counterpartyVersion); err != nil {
		return err
	}

	// Emit event using shared emitter
	im.eventEmitter.EmitChannelOpenAckEvent(
		ctx,
		types.EventTypeChannelOpenAck,
		channelID,
		portID,
		counterpartyChannelID,
	)

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Emit event using shared emitter
	im.eventEmitter.EmitChannelOpenConfirmEvent(
		ctx,
		types.EventTypeChannelOpenConfirm,
		channelID,
		portID,
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

	// Use shared event emitter
	im.eventEmitter.EmitChannelCloseEvent(
		ctx,
		types.EventTypeChannelClose,
		channelID,
		portID,
		cleanup,
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
	// Parse packet data
	packetData, err := types.ParsePacketData(packet.Data)
	if err != nil {
		return sharedibc.CreateErrorAck(
			errorsmod.Wrapf(types.ErrInvalidPacket, "failed to parse packet data: %s", err.Error()))
	}

	// Extract nonce, timestamp, and sender for validation
	packetNonce := im.packetNonce(packetData)
	packetTimestamp := im.packetTimestamp(packetData)
	sender := im.packetSender(packet, packetData)

	// Use shared validation logic - this consolidates all the duplicated validation
	if err := im.packetValidator.ValidateIncomingPacket(
		ctx,
		packet,
		packetData,
		packetNonce,
		packetTimestamp,
		sender,
	); err != nil {
		return sharedibc.CreateErrorAck(err)
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

	// Emit receive event using shared emitter
	im.eventEmitter.EmitPacketReceiveEvent(
		ctx,
		types.EventTypePacketReceive,
		packetData.GetType(),
		packet.DestinationChannel,
		packet.Sequence,
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
	// Use shared validation and unmarshaling logic
	ack, err := im.ackHelper.ValidateAndUnmarshalAck(acknowledgement)
	if err != nil {
		return err
	}

	// Delegate to keeper's acknowledgement handler
	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, ack); err != nil {
		return err
	}

	// Emit acknowledgement event using shared emitter
	im.eventEmitter.EmitPacketAckEvent(
		ctx,
		types.EventTypePacketAck,
		packet.SourceChannel,
		packet.Sequence,
		ack.Success(),
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

	// Emit timeout event using shared emitter
	im.eventEmitter.EmitPacketTimeoutEvent(
		ctx,
		types.EventTypePacketTimeout,
		packet.SourceChannel,
		packet.Sequence,
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
