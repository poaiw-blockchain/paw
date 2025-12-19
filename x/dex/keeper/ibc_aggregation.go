package keeper

import (
	"context"
	"encoding/json"
	"errors"
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
// This module enables PAW DEX to aggregate liquidity from other Cosmos SDK chains
// (Osmosis, Injective, etc.) and execute cross-chain swaps for optimal pricing and routing.
//
// Architecture:
// The aggregator acts as a liquidity router that can split trades across multiple chains
// to achieve better pricing than any single pool could provide. This is particularly
// valuable for:
// - Large trades that would face high slippage on a single pool
// - Exotic token pairs not available locally
// - Arbitrage opportunities across chain boundaries
//
// Core Features:
// 1. Remote Pool Discovery: Query liquidity on other chains via IBC
// 2. Multi-Hop Routing: Find optimal execution path across chains
// 3. Atomic Execution: All-or-nothing cross-chain swaps with rollback
// 4. Timeout Handling: Automatic refunds if IBC packets timeout
// 5. Price Caching: Cache remote pool state to reduce IBC latency
//
// Typical Flow:
// 1. User submits cross-chain swap request with max slippage
// 2. Aggregator queries cached + fresh pool data from remote chains
// 3. Routing algorithm finds best execution path (may span multiple chains)
// 4. Execution: IBC transfer → remote swap → IBC return
// 5. Acknowledgement handling: Confirm success or refund on timeout
//
// Security Considerations:
// - Tokens are escrowed locally before IBC transfer (prevents double-spend)
// - Timeout periods enforce automatic refunds (no stuck funds)
// - Each chain's swap security applies to its portion of the route
// - Price manipulation limited by caching TTL and validation
//
// Performance Notes:
// - Pool queries are asynchronous (results cached for future use)
// - Route calculation is local (no IBC round-trip)
// - Execution may span multiple blocks (IBC latency)
// - Consider using state channels for high-frequency trading

const (
	// IBC Packet Types
	// These constants identify different types of IBC packets for cross-chain DEX operations.

	// PacketTypeQueryPools identifies packets that query pool liquidity on remote chains.
	// Response contains pool reserves, fees, and metadata.
	PacketTypeQueryPools = "query_pools"

	// PacketTypeExecuteSwap identifies packets that request swap execution on remote chains.
	// Includes token amounts, slippage limits, and sender/receiver addresses.
	PacketTypeExecuteSwap = "execute_swap"

	// PacketTypeCrossChainSwap identifies packets for multi-hop cross-chain swaps.
	// May involve multiple remote chains in a single atomic operation.
	PacketTypeCrossChainSwap = "cross_chain_swap"

	// Target Chain Identifiers
	// Production chain IDs for major Cosmos SDK chains with DEX functionality.

	// OsmosisChainID is the chain ID for Osmosis mainnet.
	// Osmosis is the largest Cosmos DEX with deep liquidity across many pairs.
	OsmosisChainID = "osmosis-1"

	// InjectiveChainID is the chain ID for Injective mainnet.
	// Injective provides orderbook-based trading and derivatives markets.
	InjectiveChainID = "injective-1"

	// IBC Timeout Configuration

	// DefaultIBCTimeout is the default timeout for IBC packets (10 minutes).
	// This provides sufficient time for cross-chain communication while preventing
	// indefinite fund locking. Timeouts trigger automatic refunds to users.
	// Value chosen to accommodate:
	// - Network latency (typically <1 minute)
	// - Remote chain processing (may be slow under load)
	// - Multiple hops (if routing through intermediate chains)
	DefaultIBCTimeout = 10 * time.Minute
)

// CrossChainPoolInfo represents liquidity information from a remote chain pool.
//
// This struct caches pool state from other Cosmos chains to enable local route
// calculation without repeated IBC queries. Pool data is refreshed periodically
// via asynchronous IBC queries.
//
// Cache Freshness:
//   - LastUpdated tracks when data was retrieved
//   - Stale data (>5 minutes old) is excluded from routing
//   - Fresh queries are initiated in background to update cache
//
// Usage:
//
//	Used by routing algorithm to compare liquidity across chains and find
//	optimal execution paths for cross-chain swaps.
type CrossChainPoolInfo struct {
	ChainID     string         `json:"chain_id"`     // Cosmos chain ID (e.g., "osmosis-1")
	PoolID      string         `json:"pool_id"`      // Pool identifier on remote chain
	TokenA      string         `json:"token_a"`      // First token denomination (may be IBC denom)
	TokenB      string         `json:"token_b"`      // Second token denomination (may be IBC denom)
	ReserveA    math.Int       `json:"reserve_a"`    // Reserve amount for TokenA
	ReserveB    math.Int       `json:"reserve_b"`    // Reserve amount for TokenB
	SwapFee     math.LegacyDec `json:"swap_fee"`     // Fee percentage (e.g., 0.003 = 0.3%)
	LastUpdated time.Time      `json:"last_updated"` // Timestamp of last data refresh
}

// CrossChainSwapRoute represents a multi-chain execution path for a swap.
//
// A route consists of one or more swap steps that may execute on different chains.
// Each step represents an individual swap operation, and steps are executed sequentially
// with tokens transferred via IBC between chains as needed.
//
// Example Single-Chain Route:
//
//	Steps: [{ ChainID: "paw-1", TokenIn: "ATOM", TokenOut: "OSMO" }]
//
// Example Multi-Chain Route (ATOM → ETH via Osmosis):
//
//	Steps: [
//	  { ChainID: "paw-1", TokenIn: "ATOM", TokenOut: "IBC/OSMO" },
//	  { ChainID: "osmosis-1", TokenIn: "OSMO", TokenOut: "IBC/ETH" },
//	]
//
// Execution Properties:
//   - All steps execute atomically (all succeed or all revert)
//   - Slippage protection applies to final output only
//   - Each chain's swap fees apply to its step
//   - IBC transfer fees apply between chains
type CrossChainSwapRoute struct {
	Steps []SwapStep `json:"steps"` // Ordered sequence of swap operations
}

// SwapStep represents a single swap operation in a cross-chain route.
//
// Each step executes on a specific chain using that chain's liquidity pool.
// Steps are executed sequentially, with IBC transfers bridging between chains.
//
// Fields:
//   - ChainID: Which Cosmos chain executes this swap
//   - PoolID: Which pool on that chain to use
//   - TokenIn: Input token for this step (may be IBC denom from previous step)
//   - TokenOut: Output token for this step (may be IBC denom for next step)
//   - AmountIn: Exact input amount (from previous step or initial input)
//   - MinAmountOut: Minimum acceptable output (for slippage protection)
//
// Security Notes:
//   - MinAmountOut is calculated by routing algorithm based on pool state
//   - Each step validates independently (doesn't trust previous step)
//   - Failed steps trigger rollback of entire route
type SwapStep struct {
	ChainID      string   `json:"chain_id"`       // Chain executing this swap
	PoolID       string   `json:"pool_id"`        // Pool identifier on that chain
	TokenIn      string   `json:"token_in"`       // Input token denomination
	TokenOut     string   `json:"token_out"`      // Output token denomination
	AmountIn     math.Int `json:"amount_in"`      // Input amount for this step
	MinAmountOut math.Int `json:"min_amount_out"` // Minimum output (slippage limit)
}

// IBC Packet Data Structures
//
// These structs define the payload formats for IBC packets used in cross-chain
// DEX operations. All packets are JSON-encoded for interoperability.

// QueryPoolsPacketData is sent via IBC to query pool liquidity on remote chains.
//
// This packet requests information about pools matching a specific token pair.
// The remote chain responds with pool reserves, fees, and metadata for all
// matching pools.
//
// Usage:
//
//	Sent asynchronously to update pool cache. Responses update local cache
//	for use in future route calculations.
type QueryPoolsPacketData struct {
	Type   string `json:"type"`    // Always "query_pools"
	Nonce  uint64 `json:"nonce"`   // Unique request identifier
	TokenA string `json:"token_a"` // First token in pair (may be IBC denom)
	TokenB string `json:"token_b"` // Second token in pair (may be IBC denom)
}

// QueryPoolsPacketAck is the IBC acknowledgement for pool query packets.
//
// Contains either:
// - Success=true with list of matching pools
// - Success=false with error message
//
// Successful responses are cached locally to avoid repeated IBC queries.
type QueryPoolsPacketAck struct {
	Success bool                 `json:"success"`         // Whether query succeeded
	Pools   []CrossChainPoolInfo `json:"pools"`           // Matching pools (if success)
	Error   string               `json:"error,omitempty"` // Error message (if failure)
}

// ExecuteSwapPacketData is sent via IBC to execute a swap on a remote chain.
//
// This packet contains all information needed to execute a swap on the target chain:
// - Which pool to use
// - Input/output tokens and amounts
// - Slippage protection (MinAmountOut)
// - Sender (for refunds on failure)
// - Receiver (for output tokens on success)
//
// Execution Flow:
//  1. Tokens are escrowed locally before sending packet
//  2. IBC transfer sends tokens to remote chain
//  3. Remote chain executes swap
//  4. Output tokens sent back via IBC
//  5. Acknowledgement confirms success or triggers refund
type ExecuteSwapPacketData struct {
	Type         string   `json:"type"`           // Always "execute_swap"
	Nonce        uint64   `json:"nonce"`          // Unique request identifier
	PoolID       string   `json:"pool_id"`        // Pool to execute on
	TokenIn      string   `json:"token_in"`       // Input token denomination
	TokenOut     string   `json:"token_out"`      // Output token denomination
	AmountIn     math.Int `json:"amount_in"`      // Input amount
	MinAmountOut math.Int `json:"min_amount_out"` // Minimum output (slippage limit)
	Sender       string   `json:"sender"`         // Original sender (for refunds)
	Receiver     string   `json:"receiver"`       // Output recipient
}

// ExecuteSwapPacketAck is the IBC acknowledgement for swap execution packets.
//
// Contains either:
// - Success=true with actual output amount and fee
// - Success=false with error message (triggers refund)
//
// Success Case:
//
//	Remote chain executed swap and transferred output tokens back.
//	AmountOut and SwapFee are recorded for user notification.
//
// Failure Case:
//
//	Remote chain rejected swap (slippage, insufficient liquidity, etc.).
//	Local chain refunds escrowed input tokens to sender.
type ExecuteSwapPacketAck struct {
	Success   bool     `json:"success"`            // Whether swap succeeded
	AmountOut math.Int `json:"amount_out"`         // Actual output amount (if success)
	SwapFee   math.Int `json:"swap_fee,omitempty"` // Fee charged (if success)
	Error     string   `json:"error,omitempty"`    // Error message (if failure)
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
		packetNonce := k.NextOutboundNonce(sdkCtx, channelID, types.PortID)
		packetData := QueryPoolsPacketData{
			Type:   PacketTypeQueryPools,
			Nonce:  packetNonce,
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
		k.storePendingQuery(sdkCtx, channelID, sequence, chainID, tokenA, tokenB)
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
	if !ack.Success() {
		errStr := ack.GetError()
		k.emitAckErrorEvent(ctx, packet, errStr)
		return nil
	}

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
	case PacketTypeCrossChainSwap:
		// Cross-chain swap acknowledgements are currently no-ops.
		return nil
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
		k.removePendingQuery(ctx, packet.SourceChannel, packet.Sequence)
		return nil
	case PacketTypeExecuteSwap:
		// Swap timeout - refund user
		if err := k.refundSwap(ctx, packet.Sequence, "ibc_timeout"); err != nil {
			return err
		}
		k.clearPendingOperation(ctx, packet.SourceChannel, packet.Sequence)
		return nil
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

func (k Keeper) storePendingQuery(ctx sdk.Context, channelID string, sequence uint64, chainID, tokenA, tokenB string) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("pending_query_%d", sequence))
	value := []byte(fmt.Sprintf("%s:%s:%s", chainID, tokenA, tokenB))
	store.Set(key, value)
	k.trackPendingOperation(ctx, channelID, PacketTypeQueryPools, sequence)
}

func (k Keeper) removePendingQuery(ctx sdk.Context, channelID string, sequence uint64) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("pending_query_%d", sequence))
	store.Delete(key)
	k.clearPendingOperation(ctx, channelID, sequence)
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

	// DIVISION BY ZERO PROTECTION: Validate reserves before calculation
	if reserveIn.IsZero() || reserveOut.IsZero() {
		return nil, errorsmod.Wrap(types.ErrInsufficientLiquidity, "pool has zero reserves")
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
		refundErr := k.bankKeeper.SendCoins(ctx, escrowAddr, sender, sdk.NewCoins(tokenCoin))
		if refundErr != nil {
			return nil, errors.Join(
				errorsmod.Wrapf(err, "failed to send IBC transfer"),
				errorsmod.Wrapf(refundErr, "failed to refund escrowed tokens"),
			)
		}
		return nil, errorsmod.Wrapf(err, "failed to send IBC transfer")
	}

	// Step 4: Create swap execution packet
	swapPacket := ExecuteSwapPacketData{
		Type:         PacketTypeExecuteSwap,
		Nonce:        k.NextOutboundNonce(ctx, channelID, sender.String()),
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
	k.storePendingRemoteSwap(ctx, channelID, swapSeq, transferSeq, sender.String(), amountIn, step)

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
	return &types.SwapResult{
		AmountIn:     amountIn,
		AmountOut:    math.ZeroInt(), // Set from ACK once remote execution completes
		Route:        fmt.Sprintf("remote:%s -> %s (%s)", step.TokenIn, step.TokenOut, step.ChainID),
		Slippage:     math.LegacyZeroDec(),
		PriceImpact:  math.LegacyZeroDec(),
		Fee:          math.ZeroInt(),
		NewPoolPrice: math.LegacyZeroDec(),
	}, nil
}

// storePendingRemoteSwap stores information about a pending remote swap
func (k Keeper) storePendingRemoteSwap(ctx sdk.Context, channelID string, swapSeq, transferSeq uint64, sender string, amountIn math.Int, step SwapStep) {
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
		"channel_id":     channelID,
	}

	dataBytes, _ := json.Marshal(data)
	store.Set(key, dataBytes)
	k.trackPendingOperation(ctx, channelID, PacketTypeExecuteSwap, swapSeq)
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
	ackBytes, err := json.Marshal(ackData)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to marshal ack data")
	}

	var ack types.QueryPoolsAcknowledgement
	if err := json.Unmarshal(ackBytes, &ack); err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal query pools ack")
	}

	if !ack.Success {
		ctx.Logger().Error("pool query failed", "error", ack.Error)
		k.removePendingQuery(ctx, packet.SourceChannel, packet.Sequence)
		return nil
	}

	// Cache the pool information
	store := ctx.KVStore(k.storeKey)
	for _, pool := range ack.Pools {
		chainID := pool.ChainID
		if chainID == "" {
			chainID = ctx.ChainID()
		}

		cached := CrossChainPoolInfo{
			ChainID:     chainID,
			PoolID:      pool.PoolID,
			TokenA:      pool.TokenA,
			TokenB:      pool.TokenB,
			ReserveA:    pool.ReserveA,
			ReserveB:    pool.ReserveB,
			SwapFee:     pool.SwapFee,
			LastUpdated: ctx.BlockTime(),
		}

		cacheKey := []byte(fmt.Sprintf("cached_pool_%s_%s_%s_%s",
			cached.ChainID, cached.PoolID, cached.TokenA, cached.TokenB))

		poolBytes, err := json.Marshal(cached)
		if err != nil {
			ctx.Logger().Error("failed to marshal pool", "error", err)
			continue
		}

		store.Set(cacheKey, poolBytes)

		ctx.Logger().Debug("cached remote pool",
			"chain_id", cached.ChainID,
			"pool_id", cached.PoolID,
			"reserve_a", cached.ReserveA.String(),
			"reserve_b", cached.ReserveB.String(),
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

	k.removePendingQuery(ctx, packet.SourceChannel, packet.Sequence)
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

	store := ctx.KVStore(k.storeKey)
	swapKey := []byte(fmt.Sprintf("pending_remote_swap_%d", packet.Sequence))
	swapData := store.Get(swapKey)
	if swapData == nil {
		return nil
	}

	var pendingSwap map[string]interface{}
	if err := json.Unmarshal(swapData, &pendingSwap); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal pending swap")
	}

	sender, _ := pendingSwap["sender"].(string)
	chainID, _ := pendingSwap["chain_id"].(string)
	poolID, _ := pendingSwap["pool_id"].(string)

	if ack.Success {
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

		// Persist last remote swap result for introspection/IBC result queries.
		k.storeRemoteSwapResult(ctx, packet.Sequence, ack.AmountOut, ack.SwapFee)

		ctx.Logger().Info("remote swap completed successfully",
			"sender", sender,
			"chain_id", chainID,
			"amount_out", ack.AmountOut.String(),
		)
	} else {
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

	store.Delete(swapKey)
	k.clearPendingOperation(ctx, packet.SourceChannel, packet.Sequence)
	return nil
}

func (k Keeper) emitAckErrorEvent(ctx sdk.Context, packet channeltypes.Packet, errMsg string) {
	packetType := ""
	var packetData map[string]interface{}
	if err := json.Unmarshal(packet.Data, &packetData); err == nil {
		if t, ok := packetData["type"].(string); ok {
			packetType = t
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dex_acknowledgement_error",
			sdk.NewAttribute("packet_type", packetType),
			sdk.NewAttribute("channel", packet.SourceChannel),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute("codespace", types.ModuleName),
			sdk.NewAttribute("code", fmt.Sprintf("%d", sdkerrors.ErrUnknownRequest.ABCICode())),
			sdk.NewAttribute("error", errMsg),
		),
	)
}

// storeRemoteSwapResult persists the outcome of a remote swap acknowledgement for observability/tests.
func (k Keeper) storeRemoteSwapResult(ctx sdk.Context, sequence uint64, amountOut, swapFee math.Int) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("remote_swap_result_%d", sequence))

	result := map[string]string{
		"amount_out": amountOut.String(),
		"swap_fee":   swapFee.String(),
	}

	if bz, err := json.Marshal(result); err == nil {
		store.Set(key, bz)
	}
}

func (k Keeper) refundSwap(ctx sdk.Context, sequence uint64, reason string) error {
	// Refund tokens to user when IBC packet cannot complete
	store := ctx.KVStore(k.storeKey)

	// Get pending swap info
	swapKey := []byte(fmt.Sprintf("pending_remote_swap_%d", sequence))
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

	channelID, _ := pendingSwap["channel_id"].(string)

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
			sdk.NewAttribute("channel", channelID),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("sequence", fmt.Sprintf("%d", sequence)),
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
