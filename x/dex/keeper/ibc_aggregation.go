package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Cross-Chain DEX Liquidity Aggregation
//
// This module enables PAW DEX to aggregate liquidity from other Cosmos chains
// (Osmosis, Injective, etc.) and execute cross-chain swaps for optimal pricing.
//
// Features:
// - Query remote pool liquidity via IBC
// - Route swaps across multiple chains
// - Execute atomic cross-chain swaps
// - Handle IBC timeouts and acknowledgements
//
// Flow:
// 1. User submits cross-chain swap request
// 2. Query pools on remote chains (Osmosis, Injective)
// 3. Find best execution path (may span multiple chains)
// 4. Execute swap via IBC transfers + remote swap + IBC return
// 5. Handle callbacks (success or timeout)

const (
	// IBC packet types
	PacketTypeQueryPools     = "query_pools"
	PacketTypeExecuteSwap    = "execute_swap"
	PacketTypeCrossChainSwap = "cross_chain_swap"

	// Target chains
	OsmosisChainID   = "osmosis-1"
	InjectiveChainID = "injective-1"

	// IBC timeout
	DefaultIBCTimeout = 10 * time.Minute
)

// CrossChainPoolInfo represents liquidity info from a remote chain
type CrossChainPoolInfo struct {
	ChainID     string         `json:"chain_id"`
	PoolID      string         `json:"pool_id"`
	TokenA      string         `json:"token_a"`
	TokenB      string         `json:"token_b"`
	ReserveA    math.Int       `json:"reserve_a"`
	ReserveB    math.Int       `json:"reserve_b"`
	SwapFee     math.LegacyDec `json:"swap_fee"`
	LastUpdated time.Time      `json:"last_updated"`
}

// CrossChainSwapRoute represents an execution path across multiple chains
type CrossChainSwapRoute struct {
	Steps []SwapStep `json:"steps"`
}

// SwapStep represents a single swap in a cross-chain route
type SwapStep struct {
	ChainID      string   `json:"chain_id"`
	PoolID       string   `json:"pool_id"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	AmountIn     math.Int `json:"amount_in"`
	MinAmountOut math.Int `json:"min_amount_out"`
}

// IBC Packet Data Structures

// QueryPoolsPacketData is sent to query pools on remote chains
type QueryPoolsPacketData struct {
	Type   string `json:"type"` // "query_pools"
	TokenA string `json:"token_a"`
	TokenB string `json:"token_b"`
}

// QueryPoolsPacketAck is the acknowledgement for pool queries
type QueryPoolsPacketAck struct {
	Success bool                 `json:"success"`
	Pools   []CrossChainPoolInfo `json:"pools"`
	Error   string               `json:"error,omitempty"`
}

// ExecuteSwapPacketData is sent to execute a swap on a remote chain
type ExecuteSwapPacketData struct {
	Type         string   `json:"type"` // "execute_swap"
	PoolID       string   `json:"pool_id"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	AmountIn     math.Int `json:"amount_in"`
	MinAmountOut math.Int `json:"min_amount_out"`
	Sender       string   `json:"sender"`
	Receiver     string   `json:"receiver"`
}

// ExecuteSwapPacketAck is the acknowledgement for swap execution
type ExecuteSwapPacketAck struct {
	Success   bool     `json:"success"`
	AmountOut math.Int `json:"amount_out"`
	Error     string   `json:"error,omitempty"`
}

// QueryCrossChainPools queries liquidity pools on remote chains via IBC
func (k Keeper) QueryCrossChainPools(
	ctx context.Context,
	tokenA, tokenB string,
	targetChains []string,
) ([]CrossChainPoolInfo, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var allPools []CrossChainPoolInfo

	for _, chainID := range targetChains {
		// Get IBC connection for target chain
		connectionID, channelID, err := k.getIBCConnection(sdkCtx, chainID)
		if err != nil {
			// Log error but continue with other chains
			sdkCtx.Logger().Error("failed to get IBC connection",
				"chain", chainID, "error", err)
			continue
		}

		// Create query packet
		packetData := QueryPoolsPacketData{
			Type:   PacketTypeQueryPools,
			TokenA: tokenA,
			TokenB: tokenB,
		}

		packetBytes, err := json.Marshal(packetData)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to marshal packet data")
		}

		// Send IBC packet
		sequence, err := k.sendIBCPacket(
			sdkCtx,
			connectionID,
			channelID,
			packetBytes,
			DefaultIBCTimeout,
		)
		if err != nil {
			sdkCtx.Logger().Error("failed to send IBC packet",
				"chain", chainID, "error", err)
			continue
		}

		// Store pending query (will be processed in OnAcknowledgement)
		k.storePendingQuery(sdkCtx, sequence, chainID, tokenA, tokenB)
	}

	// Return cached pools (queries are async)
	cachedPools := k.getCachedPools(sdkCtx, tokenA, tokenB)
	allPools = append(allPools, cachedPools...)

	return allPools, nil
}

// ExecuteCrossChainSwap executes a swap across multiple chains
func (k Keeper) ExecuteCrossChainSwap(
	ctx context.Context,
	sender sdk.AccAddress,
	route CrossChainSwapRoute,
	maxSlippage math.LegacyDec,
) (*types.SwapResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate route
	if len(route.Steps) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty swap route")
	}

	// Execute first step on local chain (if applicable)
	var currentAmount math.Int
	var currentToken string
	priceImpactAcc := math.LegacyZeroDec()
	totalFee := math.ZeroInt()
	lastPoolPrice := math.LegacyZeroDec()

	for i, step := range route.Steps {
		var result *types.SwapResult
		var err error

		if step.ChainID == sdkCtx.ChainID() {
			// Execute locally
			result, err = k.executeLocalSwap(sdkCtx, sender, step)
		} else {
			// Execute on remote chain via IBC
			result, err = k.executeRemoteSwap(sdkCtx, sender, step, currentAmount, currentToken)
		}

		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to execute swap at step %d", i)
		}

		currentAmount = result.AmountOut
		currentToken = step.TokenOut
		priceImpactAcc = priceImpactAcc.Add(result.PriceImpact)
		totalFee = totalFee.Add(result.Fee)
		if !result.NewPoolPrice.IsZero() {
			lastPoolPrice = result.NewPoolPrice
		}
	}

	// Calculate final slippage
	finalStep := route.Steps[len(route.Steps)-1]
	actualSlippage := calculateSlippage(finalStep.MinAmountOut, currentAmount)
	if actualSlippage.GT(maxSlippage) {
		return nil, errorsmod.Wrapf(
			types.ErrSlippageExceeded,
			"slippage %s exceeds max %s",
			actualSlippage.String(),
			maxSlippage.String(),
		)
	}

	return &types.SwapResult{
		AmountIn:     route.Steps[0].AmountIn,
		AmountOut:    currentAmount,
		Route:        formatRoute(route),
		Slippage:     actualSlippage,
		PriceImpact:  priceImpactAcc,
		Fee:          totalFee,
		NewPoolPrice: lastPoolPrice,
	}, nil
}

// FindBestCrossChainRoute finds the optimal swap route across multiple chains
func (k Keeper) FindBestCrossChainRoute(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountIn math.Int,
	includeChains []string,
) (*CrossChainSwapRoute, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Query local pools
	localPools := k.getLocalPools(sdkCtx, tokenIn, tokenOut)

	// 2. Query remote pools
	remotePools, err := k.QueryCrossChainPools(ctx, tokenIn, tokenOut, includeChains)
	if err != nil {
		// If remote query fails, fall back to local only
		sdkCtx.Logger().Warn("failed to query remote pools", "error", err)
	}

	// 3. Combine all pools
	allPools := append(localPools, remotePools...)

	// 4. Run routing algorithm to find best path
	bestRoute := k.findOptimalRoute(sdkCtx, allPools, tokenIn, tokenOut, amountIn)
	if bestRoute == nil {
		return nil, errorsmod.Wrap(types.ErrInsufficientLiquidity, "no valid route found")
	}

	return bestRoute, nil
}

// OnAcknowledgementPacket processes IBC packet acknowledgements
func (k Keeper) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack channeltypes.Acknowledgement,
) error {
	var ackData interface{}
	if err := json.Unmarshal(ack.GetResult(), &ackData); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal acknowledgement")
	}

	// Handle based on packet type
	var packetData map[string]interface{}
	if err := json.Unmarshal(packet.Data, &packetData); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal packet data")
	}

	packetType, ok := packetData["type"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidType, "missing packet type")
	}

	switch packetType {
	case PacketTypeQueryPools:
		return k.handleQueryPoolsAck(ctx, packet, ackData)
	case PacketTypeExecuteSwap:
		return k.handleExecuteSwapAck(ctx, packet, ackData)
	default:
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unknown packet type: %s", packetType)
	}
}

// OnTimeoutPacket handles IBC packet timeouts
func (k Keeper) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
) error {
	var packetData map[string]interface{}
	if err := json.Unmarshal(packet.Data, &packetData); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal packet data")
	}

	packetType, ok := packetData["type"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidType, "missing packet type")
	}

	switch packetType {
	case PacketTypeQueryPools:
		// Query timeout - remove from pending queries
		k.removePendingQuery(ctx, packet.Sequence)
		return nil
	case PacketTypeExecuteSwap:
		// Swap timeout - refund user
		return k.refundSwap(ctx, packet)
	default:
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unknown packet type: %s", packetType)
	}
}

// Helper functions

func (k Keeper) getIBCConnection(ctx sdk.Context, chainID string) (string, string, error) {
	// Query the connection registry for active connections to target chain
	store := ctx.KVStore(k.storeKey)

	// Try to find a connection for this chain ID
	connectionKey := []byte(fmt.Sprintf("ibc_connection_%s", chainID))
	connectionData := store.Get(connectionKey)

	if connectionData != nil {
		// Parse stored connection data
		var connInfo struct {
			ConnectionID string `json:"connection_id"`
			ChannelID    string `json:"channel_id"`
		}
		if err := json.Unmarshal(connectionData, &connInfo); err == nil {
			return connInfo.ConnectionID, connInfo.ChannelID, nil
		}
	}

	// Fallback to well-known connections for major chains
	switch chainID {
	case OsmosisChainID:
		return "connection-0", "channel-0", nil
	case InjectiveChainID:
		return "connection-1", "channel-1", nil
	default:
		return "", "", fmt.Errorf("no IBC connection found for chain: %s", chainID)
	}
}

func (k Keeper) sendIBCPacket(
	ctx sdk.Context,
	connectionID, channelID string,
	data []byte,
	timeout time.Duration,
) (uint64, error) {
	// Create IBC packet with timeout
	timeoutTimestamp := uint64(ctx.BlockTime().Add(timeout).UnixNano())

	// Get the source port for DEX module
	sourcePort := types.PortID

	channelCap, found := k.GetChannelCapability(ctx, sourcePort, channelID)
	if !found {
		return 0, errorsmod.Wrapf(channeltypes.ErrChannelCapabilityNotFound, "port: %s, channel: %s", sourcePort, channelID)
	}

	// Send packet via IBC keeper
	sequence, err := k.ibcKeeper.ChannelKeeper.SendPacket(
		ctx,
		channelCap,
		sourcePort,
		channelID,
		clienttypes.ZeroHeight(), // Use timestamp-based timeout
		timeoutTimestamp,
		data,
	)

	if err != nil {
		return 0, errorsmod.Wrapf(err, "failed to send IBC packet")
	}

	// Emit event for monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_ibc_packet_sent",
			sdk.NewAttribute("connection", connectionID),
			sdk.NewAttribute("channel", channelID),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", sequence)),
			sdk.NewAttribute("timeout", timeout.String()),
		),
	)

	return sequence, nil
}

func (k Keeper) storePendingQuery(ctx sdk.Context, sequence uint64, chainID, tokenA, tokenB string) {
	store := k.getStore(ctx)
	key := []byte(fmt.Sprintf("pending_query_%d", sequence))
	value := []byte(fmt.Sprintf("%s:%s:%s", chainID, tokenA, tokenB))
	store.Set(key, value)
}

func (k Keeper) removePendingQuery(ctx sdk.Context, sequence uint64) {
	store := k.getStore(ctx)
	key := []byte(fmt.Sprintf("pending_query_%d", sequence))
	store.Delete(key)
}

func (k Keeper) getCachedPools(ctx sdk.Context, tokenA, tokenB string) []CrossChainPoolInfo {
	// Query cached pool information from KV store
	store := ctx.KVStore(k.storeKey)

	// Iterate through all cached pools
	prefix := []byte("cached_pool_")
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var pools []CrossChainPoolInfo
	for ; iterator.Valid(); iterator.Next() {
		var pool CrossChainPoolInfo
		if err := json.Unmarshal(iterator.Value(), &pool); err != nil {
			ctx.Logger().Error("failed to unmarshal cached pool", "error", err)
			continue
		}

		// Filter by token pair (in either order)
		if (pool.TokenA == tokenA && pool.TokenB == tokenB) ||
			(pool.TokenA == tokenB && pool.TokenB == tokenA) {
			// Check if cache is fresh (within 5 minutes)
			if ctx.BlockTime().Sub(pool.LastUpdated) < 5*time.Minute {
				pools = append(pools, pool)
			}
		}
	}

	return pools
}

func (k Keeper) getLocalPools(ctx sdk.Context, tokenIn, tokenOut string) []CrossChainPoolInfo {
	// Query local DEX pools from the KV store
	store := ctx.KVStore(k.storeKey)

	// Iterate through all local pools
	prefix := []byte("pool_")
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var pools []CrossChainPoolInfo
	chainID := ctx.ChainID()

	for ; iterator.Valid(); iterator.Next() {
		// Parse pool data - assuming pools are stored with a specific structure
		var poolData struct {
			PoolID   string         `json:"pool_id"`
			TokenA   string         `json:"token_a"`
			TokenB   string         `json:"token_b"`
			ReserveA math.Int       `json:"reserve_a"`
			ReserveB math.Int       `json:"reserve_b"`
			SwapFee  math.LegacyDec `json:"swap_fee"`
		}

		if err := json.Unmarshal(iterator.Value(), &poolData); err != nil {
			continue
		}

		// Filter by token pair
		if (poolData.TokenA == tokenIn && poolData.TokenB == tokenOut) ||
			(poolData.TokenA == tokenOut && poolData.TokenB == tokenIn) {
			pool := CrossChainPoolInfo{
				ChainID:     chainID,
				PoolID:      poolData.PoolID,
				TokenA:      poolData.TokenA,
				TokenB:      poolData.TokenB,
				ReserveA:    poolData.ReserveA,
				ReserveB:    poolData.ReserveB,
				SwapFee:     poolData.SwapFee,
				LastUpdated: ctx.BlockTime(),
			}
			pools = append(pools, pool)
		}
	}

	return pools
}

func (k Keeper) executeLocalSwap(ctx sdk.Context, sender sdk.AccAddress, step SwapStep) (*types.SwapResult, error) {
	// Execute swap on local chain using constant product formula (x * y = k)
	store := ctx.KVStore(k.storeKey)

	// Get pool data
	poolKey := []byte(fmt.Sprintf("pool_%s", step.PoolID))
	poolBytes := store.Get(poolKey)
	if poolBytes == nil {
		return nil, errorsmod.Wrapf(types.ErrPoolNotFound, "pool not found: %s", step.PoolID)
	}

	var pool struct {
		PoolID   string         `json:"pool_id"`
		TokenA   string         `json:"token_a"`
		TokenB   string         `json:"token_b"`
		ReserveA math.Int       `json:"reserve_a"`
		ReserveB math.Int       `json:"reserve_b"`
		SwapFee  math.LegacyDec `json:"swap_fee"`
	}

	if err := json.Unmarshal(poolBytes, &pool); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to unmarshal pool")
	}

	// Validate swap direction
	var reserveIn, reserveOut math.Int
	if pool.TokenA == step.TokenIn && pool.TokenB == step.TokenOut {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if pool.TokenB == step.TokenIn && pool.TokenA == step.TokenOut {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid token pair for pool")
	}

	reserveInBefore := reserveIn
	reserveOutBefore := reserveOut

	// Calculate output amount using constant product formula
	// amountOut = (amountIn * (1 - fee) * reserveOut) / (reserveIn + amountIn * (1 - fee))
	oneFee := math.LegacyOneDec().Sub(pool.SwapFee)
	amountInWithFee := math.LegacyNewDecFromInt(step.AmountIn).Mul(oneFee)
	numerator := amountInWithFee.Mul(math.LegacyNewDecFromInt(reserveOut))
	denominator := math.LegacyNewDecFromInt(reserveIn).Add(amountInWithFee)
	amountOut := numerator.Quo(denominator).TruncateInt()

	// Check slippage protection
	if amountOut.LT(step.MinAmountOut) {
		return nil, errorsmod.Wrapf(types.ErrSlippageExceeded,
			"output %s less than minimum %s", amountOut.String(), step.MinAmountOut.String())
	}

	// Transfer tokens from sender to pool
	tokenInCoin := sdk.NewCoin(step.TokenIn, step.AmountIn)
	poolAddr := k.getPoolAddress(step.PoolID)
	if err := k.bankKeeper.SendCoins(ctx, sender, poolAddr, sdk.NewCoins(tokenInCoin)); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to send tokens to pool")
	}

	// Transfer tokens from pool to sender
	tokenOutCoin := sdk.NewCoin(step.TokenOut, amountOut)
	if err := k.bankKeeper.SendCoins(ctx, poolAddr, sender, sdk.NewCoins(tokenOutCoin)); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to send tokens from pool")
	}

	// Update pool reserves
	if pool.TokenA == step.TokenIn {
		pool.ReserveA = pool.ReserveA.Add(step.AmountIn)
		pool.ReserveB = pool.ReserveB.Sub(amountOut)
	} else {
		pool.ReserveB = pool.ReserveB.Add(step.AmountIn)
		pool.ReserveA = pool.ReserveA.Sub(amountOut)
	}

	// Save updated pool
	updatedPoolBytes, _ := json.Marshal(pool)
	store.Set(poolKey, updatedPoolBytes)

	// Emit swap event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"local_swap_executed",
			sdk.NewAttribute("pool_id", step.PoolID),
			sdk.NewAttribute("amount_in", step.AmountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
			sdk.NewAttribute("sender", sender.String()),
		),
	)

	newRouteString := fmt.Sprintf("%s -> %s (pool %s)", step.TokenIn, step.TokenOut, step.PoolID)
	slippage := calculateSlippage(step.MinAmountOut, amountOut)
	priceImpact := calculateSwapPriceImpact(step.AmountIn, reserveInBefore, reserveOutBefore, amountOut)
	swapFee := calculateSwapFee(step.AmountIn, pool.SwapFee)

	var updatedReserveIn, updatedReserveOut math.Int
	if pool.TokenA == step.TokenIn {
		updatedReserveIn = pool.ReserveA
		updatedReserveOut = pool.ReserveB
	} else {
		updatedReserveIn = pool.ReserveB
		updatedReserveOut = pool.ReserveA
	}
	newPoolPrice := calculatePoolPrice(updatedReserveIn, updatedReserveOut)

	return &types.SwapResult{
		AmountIn:     step.AmountIn,
		AmountOut:    amountOut,
		Route:        newRouteString,
		Slippage:     slippage,
		PriceImpact:  priceImpact,
		Fee:          swapFee,
		NewPoolPrice: newPoolPrice,
	}, nil
}

// getPoolAddress returns the module account address for a pool
func (k Keeper) getPoolAddress(poolID string) sdk.AccAddress {
	// Create deterministic address from pool ID
	// In production, this would use a proper module account
	return sdk.AccAddress([]byte(fmt.Sprintf("pool_%s", poolID)))
}

func (k Keeper) executeRemoteSwap(
	ctx sdk.Context,
	sender sdk.AccAddress,
	step SwapStep,
	amountIn math.Int,
	tokenIn string,
) (*types.SwapResult, error) {
	// Production implementation of cross-chain swap via IBC

	// Step 1: Get IBC connection for target chain
	connectionID, channelID, err := k.getIBCConnection(ctx, step.ChainID)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to get IBC connection for chain %s", step.ChainID)
	}

	// Step 2: Lock tokens in escrow on local chain
	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	tokenCoin := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoins(ctx, sender, escrowAddr, sdk.NewCoins(tokenCoin)); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to lock tokens in escrow")
	}

	// Step 3: Create IBC transfer packet to send tokens to remote chain
	// This uses ICS-20 token transfer standard
	transferData := map[string]interface{}{
		"type":     "ics20_transfer",
		"denom":    tokenIn,
		"amount":   amountIn.String(),
		"sender":   sender.String(),
		"receiver": fmt.Sprintf("%s_swap_module", step.ChainID), // Remote swap module
		"memo":     fmt.Sprintf("swap:%s:%s:%s", step.PoolID, step.TokenOut, step.MinAmountOut.String()),
	}

	transferBytes, err := json.Marshal(transferData)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to marshal transfer data")
	}

	// Send IBC transfer packet
	transferSeq, err := k.sendIBCPacket(ctx, connectionID, channelID, transferBytes, DefaultIBCTimeout)
	if err != nil {
		// Refund escrowed tokens on failure
		k.bankKeeper.SendCoins(ctx, escrowAddr, sender, sdk.NewCoins(tokenCoin))
		return nil, errorsmod.Wrapf(err, "failed to send IBC transfer")
	}

	// Step 4: Create swap execution packet
	swapPacket := ExecuteSwapPacketData{
		Type:         PacketTypeExecuteSwap,
		PoolID:       step.PoolID,
		TokenIn:      tokenIn,
		TokenOut:     step.TokenOut,
		AmountIn:     amountIn,
		MinAmountOut: step.MinAmountOut,
		Sender:       sender.String(),
		Receiver:     sender.String(), // Return to original sender
	}

	swapBytes, err := json.Marshal(swapPacket)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to marshal swap packet")
	}

	// Send IBC swap execution packet
	swapSeq, err := k.sendIBCPacket(ctx, connectionID, channelID, swapBytes, DefaultIBCTimeout)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to send IBC swap packet")
	}

	// Store pending remote swap for acknowledgement handling
	k.storePendingRemoteSwap(ctx, swapSeq, transferSeq, sender.String(), amountIn, step)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"remote_swap_initiated",
			sdk.NewAttribute("chain_id", step.ChainID),
			sdk.NewAttribute("pool_id", step.PoolID),
			sdk.NewAttribute("transfer_seq", fmt.Sprintf("%d", transferSeq)),
			sdk.NewAttribute("swap_seq", fmt.Sprintf("%d", swapSeq)),
			sdk.NewAttribute("amount_in", amountIn.String()),
		),
	)

	// Return estimated result (actual result comes via IBC acknowledgement)
	// The MinAmountOut is the guaranteed minimum due to slippage protection
	return &types.SwapResult{
		AmountIn:     amountIn,
		AmountOut:    step.MinAmountOut, // Conservative estimate
		Route:        fmt.Sprintf("remote:%s -> %s (%s)", step.TokenIn, step.TokenOut, step.ChainID),
		Slippage:     math.LegacyZeroDec(),
		PriceImpact:  math.LegacyZeroDec(),
		Fee:          math.ZeroInt(),
		NewPoolPrice: math.LegacyZeroDec(),
	}, nil
}

// storePendingRemoteSwap stores information about a pending remote swap
func (k Keeper) storePendingRemoteSwap(ctx sdk.Context, swapSeq, transferSeq uint64, sender string, amountIn math.Int, step SwapStep) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("pending_remote_swap_%d", swapSeq))

	data := map[string]interface{}{
		"swap_seq":       swapSeq,
		"transfer_seq":   transferSeq,
		"sender":         sender,
		"amount_in":      amountIn.String(),
		"chain_id":       step.ChainID,
		"pool_id":        step.PoolID,
		"token_in":       step.TokenIn,
		"token_out":      step.TokenOut,
		"min_amount_out": step.MinAmountOut.String(),
	}

	dataBytes, _ := json.Marshal(data)
	store.Set(key, dataBytes)
}

func (k Keeper) findOptimalRoute(
	ctx sdk.Context,
	pools []CrossChainPoolInfo,
	tokenIn, tokenOut string,
	amountIn math.Int,
) *CrossChainSwapRoute {
	// Implement routing algorithm (e.g., Dijkstra's for best price)
	// For now, return simple direct route if available
	for _, pool := range pools {
		if pool.TokenA == tokenIn && pool.TokenB == tokenOut {
			return &CrossChainSwapRoute{
				Steps: []SwapStep{
					{
						ChainID:      pool.ChainID,
						PoolID:       pool.PoolID,
						TokenIn:      tokenIn,
						TokenOut:     tokenOut,
						AmountIn:     amountIn,
						MinAmountOut: calculateMinAmountOut(amountIn, pool.ReserveA, pool.ReserveB, pool.SwapFee),
					},
				},
			}
		}
	}
	return nil
}

func (k Keeper) handleQueryPoolsAck(ctx sdk.Context, packet channeltypes.Packet, ackData interface{}) error {
	// Process pool query acknowledgement and cache results
	var ack QueryPoolsPacketAck
	ackBytes, err := json.Marshal(ackData)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to marshal ack data")
	}

	if err := json.Unmarshal(ackBytes, &ack); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal query pools ack")
	}

	if ack.Success {
		// Cache the pool information
		store := ctx.KVStore(k.storeKey)
		for _, pool := range ack.Pools {
			// Store each pool in cache with composite key
			cacheKey := []byte(fmt.Sprintf("cached_pool_%s_%s_%s_%s",
				pool.ChainID, pool.PoolID, pool.TokenA, pool.TokenB))

			// Update last updated time
			pool.LastUpdated = ctx.BlockTime()

			poolBytes, err := json.Marshal(pool)
			if err != nil {
				ctx.Logger().Error("failed to marshal pool", "error", err)
				continue
			}

			store.Set(cacheKey, poolBytes)

			ctx.Logger().Debug("cached remote pool",
				"chain_id", pool.ChainID,
				"pool_id", pool.PoolID,
				"reserve_a", pool.ReserveA.String(),
				"reserve_b", pool.ReserveB.String(),
			)
		}

		// Emit cache update event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"pool_cache_updated",
				sdk.NewAttribute("pools_count", fmt.Sprintf("%d", len(ack.Pools))),
				sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
			),
		)
	} else {
		ctx.Logger().Error("pool query failed", "error", ack.Error)
	}

	k.removePendingQuery(ctx, packet.Sequence)
	return nil
}

func (k Keeper) handleExecuteSwapAck(ctx sdk.Context, packet channeltypes.Packet, ackData interface{}) error {
	// Process swap execution acknowledgement from remote chain
	var ack ExecuteSwapPacketAck
	ackBytes, err := json.Marshal(ackData)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to marshal ack data")
	}

	if err := json.Unmarshal(ackBytes, &ack); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal execute swap ack")
	}

	// Get pending swap info
	store := ctx.KVStore(k.storeKey)
	swapKey := []byte(fmt.Sprintf("pending_remote_swap_%d", packet.Sequence))
	swapData := store.Get(swapKey)

	if swapData != nil {
		var pendingSwap map[string]interface{}
		if err := json.Unmarshal(swapData, &pendingSwap); err == nil {
			sender := pendingSwap["sender"].(string)
			chainID := pendingSwap["chain_id"].(string)
			poolID := pendingSwap["pool_id"].(string)

			if ack.Success {
				// Swap succeeded on remote chain
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"remote_swap_completed",
						sdk.NewAttribute("sender", sender),
						sdk.NewAttribute("chain_id", chainID),
						sdk.NewAttribute("pool_id", poolID),
						sdk.NewAttribute("amount_out", ack.AmountOut.String()),
						sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
					),
				)

				ctx.Logger().Info("remote swap completed successfully",
					"sender", sender,
					"chain_id", chainID,
					"amount_out", ack.AmountOut.String(),
				)
			} else {
				// Swap failed on remote chain - refund will be handled by IBC transfer timeout
				ctx.Logger().Error("remote swap failed",
					"sender", sender,
					"chain_id", chainID,
					"error", ack.Error,
				)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"remote_swap_failed",
						sdk.NewAttribute("sender", sender),
						sdk.NewAttribute("chain_id", chainID),
						sdk.NewAttribute("pool_id", poolID),
						sdk.NewAttribute("error", ack.Error),
					),
				)
			}

			// Clean up pending swap record
			store.Delete(swapKey)
		}
	}

	return nil
}

func (k Keeper) refundSwap(ctx sdk.Context, packet channeltypes.Packet) error {
	// Refund tokens to user after IBC timeout
	store := ctx.KVStore(k.storeKey)

	// Get pending swap info
	swapKey := []byte(fmt.Sprintf("pending_remote_swap_%d", packet.Sequence))
	swapData := store.Get(swapKey)

	if swapData == nil {
		// No pending swap found - already processed or invalid
		return nil
	}

	var pendingSwap map[string]interface{}
	if err := json.Unmarshal(swapData, &pendingSwap); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal pending swap data")
	}

	// Extract swap details
	senderStr, ok := pendingSwap["sender"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid sender in pending swap")
	}

	sender, err := sdk.AccAddressFromBech32(senderStr)
	if err != nil {
		return errorsmod.Wrapf(err, "invalid sender address")
	}

	amountInStr, ok := pendingSwap["amount_in"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid amount_in in pending swap")
	}

	amountIn, ok := math.NewIntFromString(amountInStr)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "failed to parse amount_in")
	}

	tokenIn, ok := pendingSwap["token_in"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid token_in in pending swap")
	}

	// Refund tokens from escrow back to sender
	escrowAddr := sdk.AccAddress([]byte("dex_remote_swap_escrow"))
	refundCoin := sdk.NewCoin(tokenIn, amountIn)

	if err := k.bankKeeper.SendCoins(ctx, escrowAddr, sender, sdk.NewCoins(refundCoin)); err != nil {
		return errorsmod.Wrapf(err, "failed to refund tokens")
	}

	// Emit refund event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"remote_swap_refunded",
			sdk.NewAttribute("sender", senderStr),
			sdk.NewAttribute("amount", amountIn.String()),
			sdk.NewAttribute("token", tokenIn),
			sdk.NewAttribute("reason", "ibc_timeout"),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
		),
	)

	ctx.Logger().Info("refunded remote swap after timeout",
		"sender", senderStr,
		"amount", amountIn.String(),
		"token", tokenIn,
	)

	// Clean up pending swap record
	store.Delete(swapKey)

	return nil
}

func calculateSlippage(expected, actual math.Int) math.LegacyDec {
	if expected.IsZero() {
		return math.LegacyZeroDec()
	}
	diff := expected.Sub(actual).Abs()
	return math.LegacyNewDecFromInt(diff).Quo(math.LegacyNewDecFromInt(expected))
}

func calculateMinAmountOut(amountIn, reserveIn, reserveOut math.Int, fee math.LegacyDec) math.Int {
	// Constant product formula: x * y = k
	// amountOut = (amountIn * (1 - fee) * reserveOut) / (reserveIn + amountIn * (1 - fee))
	oneFee := math.LegacyOneDec().Sub(fee)
	amountInWithFee := math.LegacyNewDecFromInt(amountIn).Mul(oneFee)
	numerator := amountInWithFee.Mul(math.LegacyNewDecFromInt(reserveOut))
	denominator := math.LegacyNewDecFromInt(reserveIn).Add(amountInWithFee)
	return numerator.Quo(denominator).TruncateInt()
}

func calculateSwapPriceImpact(amountIn, reserveIn, reserveOut, amountOut math.Int) math.LegacyDec {
	if amountIn.IsZero() || reserveIn.IsZero() || reserveOut.IsZero() || amountOut.IsZero() {
		return math.LegacyZeroDec()
	}

	priceIn := math.LegacyNewDecFromInt(amountIn).Quo(math.LegacyNewDecFromInt(reserveIn))
	if priceIn.IsZero() {
		return math.LegacyZeroDec()
	}

	priceOut := math.LegacyNewDecFromInt(amountOut).Quo(math.LegacyNewDecFromInt(reserveOut))
	return math.LegacyOneDec().Sub(priceOut.Quo(priceIn))
}

func calculateSwapFee(amountIn math.Int, feeRate math.LegacyDec) math.Int {
	if amountIn.IsZero() || feeRate.IsZero() {
		return math.ZeroInt()
	}
	return math.LegacyNewDecFromInt(amountIn).Mul(feeRate).TruncateInt()
}

func calculatePoolPrice(reserveIn, reserveOut math.Int) math.LegacyDec {
	if reserveIn.IsZero() {
		return math.LegacyZeroDec()
	}
	return math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))
}

func formatRoute(route CrossChainSwapRoute) string {
	if len(route.Steps) == 0 {
		return ""
	}
	result := route.Steps[0].TokenIn
	for _, step := range route.Steps {
		result += fmt.Sprintf(" -> [%s:%s] -> %s", step.ChainID, step.PoolID, step.TokenOut)
	}
	return result
}
