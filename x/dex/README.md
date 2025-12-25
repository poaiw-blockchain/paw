# DEX Module

## Purpose

The DEX module implements a production-grade automated market maker (AMM) with constant product formula (x * y = k), providing trustless token swaps, liquidity provision, and advanced MEV protection mechanisms for the PAW blockchain.

## Key Features

- **AMM Pools**: Constant product market maker (x * y = k) for decentralized token swaps
- **Liquidity Provision**: Users provide liquidity and earn fees via LP tokens
- **Limit Orders**: Order book functionality with partial fills and expiration
- **MEV Protection**: Commit-reveal scheme to prevent front-running and sandwich attacks
- **TWAP Oracle**: Time-weighted average price for manipulation-resistant pricing
- **Circuit Breaker**: Automatic pause on excessive price volatility or anomalies
- **Flash Loan Protection**: Minimum delay between liquidity operations
- **IBC Integration**: Cross-chain swaps via authorized IBC channels

## Key Types

### Pool
- `id`: Unique pool identifier
- `token_a`: First token denomination
- `token_b`: Second token denomination
- `reserve_a`: Reserve amount of token A
- `reserve_b`: Reserve amount of token B
- `total_shares`: Total LP tokens issued
- `creator`: Pool creator address

### LimitOrder
- `id`: Order identifier
- `owner`: Order creator
- `pool_id`: Associated pool
- `order_type`: BUY or SELL
- `token_in`: Input token
- `token_out`: Desired output token
- `amount_in`: Input amount
- `min_amount_out`: Minimum output (slippage protection)
- `limit_price`: Execution price limit
- `status`: OPEN, PARTIALLY_FILLED, FILLED, CANCELLED, EXPIRED

### SwapCommit
- `trader`: Address committing to swap
- `swap_hash`: keccak256(trader, pool_id, token_in, token_out, amount_in, min_amount_out, deadline, nonce)
- `commit_height`: Block height of commitment
- `expiry_height`: When commit expires if not revealed

## Key Messages

- **MsgCreatePool**: Create new liquidity pool with initial reserves
- **MsgAddLiquidity**: Add liquidity to pool, receive LP tokens
- **MsgRemoveLiquidity**: Burn LP tokens, receive underlying assets
- **MsgSwap**: Execute instant swap with slippage protection (public mempool, vulnerable to MEV)
- **MsgCommitSwap**: Commit to swap without revealing parameters (MEV protection phase 1)
- **MsgRevealSwap**: Reveal and execute committed swap (MEV protection phase 2)

## Configuration Parameters

### Fee Structure
- `swap_fee`: Total swap fee (default: 0.3%)
- `lp_fee`: Portion to liquidity providers (default: 0.25%)
- `protocol_fee`: Portion to protocol treasury (default: 0.05%)

### Security Limits
- `min_liquidity`: Minimum liquidity for pool creation (default: 1000)
- `max_slippage_percent`: Maximum allowed slippage (default: 5%)
- `max_pool_drain_percent`: Max single-swap drain (default: 30%)
- `flash_loan_protection_blocks`: Delay between LP operations (default: 10 blocks)

### MEV Protection (Commit-Reveal)
- `enable_commit_reveal`: Enable commit-reveal scheme (default: false for testnet)
- `commit_reveal_delay`: Blocks between commit and reveal (default: 10 blocks ≈ 60s)
- `commit_timeout_blocks`: Commit expiration window (default: 100 blocks ≈ 10 minutes)
- `recommended_max_slippage`: UI warning threshold (default: 3%)

### Gas Metering
- `pool_creation_gas`: Gas for pool creation validation (default: 1000)
- `swap_validation_gas`: Gas for swap validation (default: 1500)
- `liquidity_gas`: Gas for liquidity operations (default: 1200)

### Circuit Breaker
- `upgrade_preserve_circuit_breaker_state`: Preserve pause state across upgrades (default: true)

## MEV Risk and Protections

### Current Testnet Protections
- **Slippage Limits**: `min_amount_out` enforces minimum acceptable output
- **Deadline**: Transaction expires if not executed by deadline
- **Pool Drain Limit**: Max 30% of reserves per swap

### Mainnet Commit-Reveal Scheme
When `enable_commit_reveal` is true:

1. **Commit Phase**: Submit `MsgCommitSwap` with hash of swap parameters
   - Hash: `keccak256(trader, pool_id, token_in, token_out, amount_in, min_amount_out, deadline, nonce)`
   - Nonce prevents hash grinding attacks

2. **Delay Period**: Wait minimum `commit_reveal_delay` blocks

3. **Reveal Phase**: Submit `MsgRevealSwap` with actual parameters
   - Parameters must hash to original commit
   - Swap executes if validation passes

**Benefits**: Front-runners cannot see swap details during commit phase, eliminating sandwich attacks.

**Tradeoffs**: Two transactions required, adds latency, higher gas cost.

---

**Module Path:** `github.com/paw-chain/paw/x/dex`
**Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-25
