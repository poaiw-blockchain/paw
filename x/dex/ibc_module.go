package dex

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"strconv"
	"strings"

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
	pending := im.keeper.GetPendingOperations(ctx, channelID)
	refunded := 0
	for _, op := range pending {
		if err := im.keeper.RefundOnChannelClose(ctx, op); err != nil {
			ctx.Logger().Error("failed to cleanup dex channel operation",
				"channel", channelID,
				"sequence", op.Sequence,
				"type", op.PacketType,
				"error", err)
			continue
		}
		refunded++
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelClose,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyPendingOperations, fmt.Sprintf("%d", refunded)),
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
	if !im.keeper.IsAuthorizedChannel(ctx, packet.SourcePort, packet.SourceChannel) {
		err := errorsmod.Wrapf(types.ErrUnauthorizedChannel, "port %s channel %s not authorized", packet.SourcePort, packet.SourceChannel)
		ctx.Logger().Error("unauthorized dex packet source", "port", packet.SourcePort, "channel", packet.SourceChannel)
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
	case types.QueryPoolsType:
		// Handle pool query from remote chain
		ack = im.handleQueryPools(ctx, packet, packetData, packetNonce)

	case types.ExecuteSwapType:
		// Handle swap execution request
		ack = im.handleExecuteSwap(ctx, packet, packetData, packetNonce)

	case types.CrossChainSwapType:
		// Handle multi-hop cross-chain swap
		ack = im.handleCrossChainSwap(ctx, packet, packetData, packetNonce)

	case types.PoolUpdateType:
		// Handle pool state update broadcast
		ack = im.handlePoolUpdate(ctx, packet, packetData, packetNonce)

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
	const maxAcknowledgementSize = 1024 * 1024 // 1MB cap to avoid malicious memory pressure
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
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.QueryPoolsPacketData)

	pools := im.lookupPools(ctx, req.TokenA, req.TokenB)
	if len(pools) == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrapf(types.ErrInvalidPacket, "no pools for %s/%s", req.TokenA, req.TokenB))
	}

	ackData := types.QueryPoolsAcknowledgement{
		Nonce:   nonce,
		Success: true,
		Pools:   pools,
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
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.ExecuteSwapPacketData)

	amountOut, fee, err := im.executeSwapFromIBC(ctx, req)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	ackData := types.ExecuteSwapAcknowledgement{
		Nonce:     nonce,
		Success:   true,
		AmountOut: amountOut,
		SwapFee:   fee,
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
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.CrossChainSwapPacketData)

	finalOut, fees, hops, err := im.executeCrossChainRoute(ctx, req)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	ackData := types.CrossChainSwapAcknowledgement{
		Nonce:        nonce,
		Success:      true,
		FinalAmount:  finalOut,
		HopsExecuted: hops,
		TotalFees:    fees,
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
	nonce uint64,
) ibcexported.Acknowledgement {
	ackData := types.PoolUpdateAcknowledgement{
		Nonce:   nonce,
		Success: true,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

// lookupPools fetches pools for token pairs (both directions) and builds PoolInfo slices.
func (im IBCModule) lookupPools(ctx sdk.Context, tokenA, tokenB string) []types.PoolInfo {
	pools := []types.PoolInfo{}

	tryPool := func(a, b string) {
		pool, err := im.keeper.GetPoolByTokens(ctx, a, b)
		if err != nil || pool == nil {
			return
		}
		pools = append(pools, types.PoolInfo{
			ChainID:     ctx.ChainID(),
			PoolID:      fmt.Sprintf("pool-%d", pool.Id),
			TokenA:      pool.TokenA,
			TokenB:      pool.TokenB,
			ReserveA:    pool.ReserveA,
			ReserveB:    pool.ReserveB,
			SwapFee:     im.defaultSwapFee(ctx),
			TotalShares: pool.TotalShares,
		})
	}

	tryPool(tokenA, tokenB)
	if tokenA != tokenB {
		tryPool(tokenB, tokenA)
	}

	return pools
}

// executeCrossChainRoute executes local hops and estimates remote hops for cross-chain route ACKs.
func (im IBCModule) executeCrossChainRoute(ctx sdk.Context, req types.CrossChainSwapPacketData) (math.Int, math.Int, int, error) {
	totalFees := math.ZeroInt()
	currentAmount := req.AmountIn
	hopsExecuted := 0

	for _, hop := range req.Route {
		if hop.ChainID != ctx.ChainID() {
			// Remote hops: conservatively assume min out to avoid overstating liquidity.
			currentAmount = hop.MinAmountOut
			hopsExecuted++
			continue
		}

		amountOut, fee, err := im.executeSwapFromIBC(ctx, types.ExecuteSwapPacketData{
			PoolID:       hop.PoolID,
			TokenIn:      hop.TokenIn,
			TokenOut:     hop.TokenOut,
			AmountIn:     currentAmount,
			MinAmountOut: hop.MinAmountOut,
		})
		if err != nil {
			return math.ZeroInt(), math.ZeroInt(), hopsExecuted, err
		}
		currentAmount = amountOut
		totalFees = totalFees.Add(fee)
		hopsExecuted++
	}

	return currentAmount, totalFees, hopsExecuted, nil
}

func (im IBCModule) executeSwapFromIBC(ctx sdk.Context, req types.ExecuteSwapPacketData) (math.Int, math.Int, error) {
	poolID, ok := im.parsePoolID(req.PoolID)
	pool := (*types.Pool)(nil)

	if ok {
		if p, err := im.keeper.GetPool(ctx, poolID); err == nil && p != nil {
			if (p.TokenA == req.TokenIn && p.TokenB == req.TokenOut) || (p.TokenA == req.TokenOut && p.TokenB == req.TokenIn) {
				pool = p
			}
		}
	}
	if pool == nil {
		if p, err := im.keeper.GetPoolByTokens(ctx, req.TokenIn, req.TokenOut); err == nil && p != nil {
			pool = p
			poolID = p.Id
		}
	}
	if pool == nil {
		return math.ZeroInt(), math.ZeroInt(), errorsmod.Wrap(types.ErrInvalidPacket, "pool not found for swap")
	}

	moduleAddr := im.keeper.GetModuleAddress()
	amountOut, err := im.keeper.ExecuteSwapSecure(ctx, moduleAddr, poolID, req.TokenIn, req.TokenOut, req.AmountIn, req.MinAmountOut)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	swapFeeDec := im.defaultSwapFee(ctx)
	fee := swapFeeDec.MulInt(amountOut).TruncateInt()
	return amountOut, fee, nil
}

func (im IBCModule) parsePoolID(raw string) (uint64, bool) {
	if raw == "" {
		return 0, false
	}
	if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return id, true
	}
	if strings.HasPrefix(raw, "pool-") {
		if id, err := strconv.ParseUint(strings.TrimPrefix(raw, "pool-"), 10, 64); err == nil {
			return id, true
		}
	}
	return 0, false
}

func (im IBCModule) defaultSwapFee(ctx sdk.Context) math.LegacyDec {
	params, err := im.keeper.GetParams(ctx)
	if err != nil {
		return math.LegacyMustNewDecFromStr("0.003")
	}
	return params.SwapFee
}

func (im IBCModule) packetNonce(packetData types.IBCPacketData) uint64 {
	switch req := packetData.(type) {
	case types.QueryPoolsPacketData:
		return req.Nonce
	case types.ExecuteSwapPacketData:
		return req.Nonce
	case types.CrossChainSwapPacketData:
		return req.Nonce
	case types.PoolUpdatePacketData:
		return req.Nonce
	default:
		return 0
	}
}

func (im IBCModule) packetSender(packet channeltypes.Packet, packetData types.IBCPacketData) string {
	switch req := packetData.(type) {
	case types.ExecuteSwapPacketData:
		if req.Sender != "" {
			return req.Sender
		}
	case types.CrossChainSwapPacketData:
		if req.Sender != "" {
			return req.Sender
		}
	}
	if packet.SourcePort != "" {
		return packet.SourcePort
	}
	return packet.SourceChannel
}

func (im IBCModule) packetTimestamp(packetData types.IBCPacketData) int64 {
	switch req := packetData.(type) {
	case types.QueryPoolsPacketData:
		return req.Timestamp
	case types.ExecuteSwapPacketData:
		return req.Timestamp
	case types.CrossChainSwapPacketData:
		return req.Timestamp
	case types.PoolUpdatePacketData:
		return req.Timestamp
	default:
		return 0
	}
}
