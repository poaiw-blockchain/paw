// Package keeper implements the DEX (Decentralized Exchange) module keeper.
//
// The DEX module provides an automated market maker (AMM) with IBC-enabled
// cross-chain liquidity aggregation. It enables users to create liquidity pools,
// swap tokens, and execute limit orders across connected blockchains.
//
// # Core Functionality
//
// Liquidity Pools: Create and manage constant-product AMM pools with dual-token
// reserves. Supports adding/removing liquidity with LP token minting and burning.
//
// Token Swaps: Execute token swaps using the constant-product formula (x * y = k).
// Two-tier security architecture with ExecuteSwap (performance-optimized) and
// ExecuteSwapSecure (comprehensive security validations).
//
// Limit Orders: Place on-chain limit orders with maker/taker fee structures,
// partial fills, and automatic order book matching.
//
// IBC Integration: Cross-chain liquidity aggregation via IBC packets. Route swaps
// to remote chains, aggregate prices, and settle cross-chain trades atomically.
//
// Security Features: Circuit breakers, reentrancy guards, slippage protection,
// flash loan prevention, invariant checking, and comprehensive gas metering.
//
// # Key Types
//
// Keeper: Main module keeper managing state, bank operations, and IBC channels.
//
// Pool: Liquidity pool with token reserves, swap fee, and total LP shares.
//
// LimitOrder: Maker order with price, quantity, filled amount, and status.
//
// # Usage Patterns
//
// Creating a pool:
//
//	pool, err := keeper.CreatePool(ctx, creator, "tokenA", "tokenB", amountA, amountB)
//
// Executing a swap:
//
//	amountOut, err := keeper.Swap(ctx, poolID, tokenIn, amountIn, minAmountOut, trader)
//
// Placing a limit order:
//
//	orderID, err := keeper.PlaceLimitOrder(ctx, maker, poolID, tokenIn, amountIn, limitPrice)
//
// # IBC Port
//
// The module binds to the "dex" IBC port for cross-chain operations. All IBC
// channels must be authorized via governance before accepting packets.
//
// # Metrics
//
// The keeper exposes Prometheus metrics for swaps, pools, liquidity changes,
// and IBC operations via DEXMetrics.
package keeper
