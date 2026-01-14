package dex

import (
	"fmt"

	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the DEX module.
// This enables cross-chain liquidity aggregation and atomic swaps.
type IBCModule struct {
	keeper           keeper.Keeper
	cdc              codec.Codec
	channelValidator *sharedibc.ChannelOpenValidator
	packetValidator  *sharedibc.PacketValidator
	ackHelper        *sharedibc.AcknowledgementHelper
	eventEmitter     *sharedibc.EventEmitter
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
			channeltypes.UNORDERED, // DEX uses unordered channels for better throughput
			adapter,
		),
		packetValidator: sharedibc.NewPacketValidator(adapter, adapter),
		ackHelper:       sharedibc.NewAcknowledgementHelper(),
		eventEmitter:    sharedibc.NewEventEmitter(),
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
	// Disallow user-initiated channel closing for DEX
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Use keeper methods directly (adapter pattern)
	// The shared utilities would require interface conversion which adds complexity
	// This is an acceptable deviation for better type safety
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

	// Use shared event emitter
	im.eventEmitter.EmitChannelCloseEvent(
		ctx,
		types.EventTypeChannelClose,
		channelID,
		portID,
		refunded,
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
	amountOut, err := im.keeper.ExecuteSwap(ctx, moduleAddr, poolID, req.TokenIn, req.TokenOut, req.AmountIn, req.MinAmountOut)
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
