# PAW DEX CLI Reference

Complete command-line interface reference for the PAW decentralized exchange.

## Quick Start

```bash
# Build the daemon
cd /home/decri/blockchain-projects/paw
go build -o pawd ./cmd/pawd

# View DEX commands
./pawd tx dex --help
./pawd query dex --help
```

## Transaction Commands

### Pool Management

#### Create Pool
```bash
pawd tx dex create-pool [token-a] [amount-a] [token-b] [amount-b] --from [key]

# Example: Create PAW/USDT pool with 1M PAW and 2M USDT
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 --from alice --chain-id paw-mvp-1
```

#### Add Liquidity
```bash
pawd tx dex add-liquidity [pool-id] [amount-a] [amount-b] --from [key]

# Example: Add liquidity to pool 1
pawd tx dex add-liquidity 1 100000000 200000000 --from alice
```

#### Remove Liquidity
```bash
pawd tx dex remove-liquidity [pool-id] [shares] --from [key]

# Example: Remove 50% of your shares from pool 1
pawd tx dex remove-liquidity 1 500000 --from alice
```

### Token Swaps

#### Basic Swap
```bash
pawd tx dex swap [pool-id] [token-in] [amount-in] [token-out] [min-amount-out] --from [key]

# Example: Swap 1000 PAW for at least 1900 USDT
pawd tx dex swap 1 upaw 1000000000 uusdt 1900000000 --from alice
```

### Limit Orders

#### Place Limit Order
```bash
pawd tx dex limit-order place [pool-id] [token-in] [amount-in] [token-out] [min-price] --from [key]

# Example: Sell 1000 PAW for USDT at minimum 2.0 USDT per PAW
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 --from alice

# With expiration (order expires in 24 hours)
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 --expiration 86400 --from alice
```

#### Cancel Limit Order
```bash
pawd tx dex limit-order cancel [order-id] --from [key]

# Example: Cancel order 123
pawd tx dex limit-order cancel 123 --from alice
```

#### Cancel All Orders
```bash
pawd tx dex limit-order cancel-all --from [key]

# Cancel only orders in pool 1
pawd tx dex limit-order cancel-all --pool-id 1 --from alice
```

### Advanced Trading

#### Swap with Automatic Slippage
```bash
pawd tx dex advanced swap-with-slippage [pool-id] [token-in] [amount-in] [token-out] [slippage-%] --from [key]

# Example: Swap with 0.5% slippage tolerance
pawd tx dex advanced swap-with-slippage 1 upaw 1000000000 uusdt 0.5 --from alice --deadline 300
```

#### Quick Swap (Auto Pool Discovery)
```bash
pawd tx dex advanced quick-swap [token-in] [amount-in] [token-out] --from [key]

# Example: Quick swap PAW to USDT (finds best pool automatically)
pawd tx dex advanced quick-swap upaw 1000000000 uusdt --from alice
```

#### Balanced Liquidity Addition
```bash
pawd tx dex advanced add-liquidity-balanced [pool-id] [total-value-token-a] --from [key]

# Example: Add liquidity with automatic ratio calculation
pawd tx dex advanced add-liquidity-balanced 1 1000000000 --from alice
```

## Query Commands

### Pool Queries

#### Query Single Pool
```bash
pawd query dex pool [pool-id]

# Example
pawd query dex pool 1
```

#### Query All Pools
```bash
pawd query dex pools

# With pagination
pawd query dex pools --limit 10 --offset 20
```

#### Find Pool by Token Pair
```bash
pawd query dex pool-by-tokens [token-a] [token-b]

# Example: Find PAW/USDT pool
pawd query dex pool-by-tokens upaw uusdt
```

### Liquidity Queries

#### Query User Liquidity
```bash
pawd query dex liquidity [pool-id] [address]

# Example
pawd query dex liquidity 1 paw1abcdefghijklmnopqrstuvwxyz...
```

### Swap Simulation

#### Simulate Swap
```bash
pawd query dex simulate-swap [pool-id] [token-in] [token-out] [amount-in]

# Example: Simulate swapping 1000 PAW to USDT
pawd query dex simulate-swap 1 upaw uusdt 1000000000
```

### Limit Order Queries

#### Query Single Order
```bash
pawd query dex limit-order [order-id]

# Example
pawd query dex limit-order 123
```

#### Query All Orders
```bash
pawd query dex limit-orders --limit 50
```

#### Query Orders by Owner
```bash
pawd query dex orders-by-owner [address]

# Example
pawd query dex orders-by-owner paw1abcdefghijklmnopqrstuvwxyz...
```

#### Query Orders by Pool
```bash
pawd query dex orders-by-pool [pool-id]

# Example
pawd query dex orders-by-pool 1
```

#### Query Order Book
```bash
pawd query dex order-book [pool-id]

# Example: View order book for pool 1
pawd query dex order-book 1 --limit 50
```

### Advanced Analytics

#### Pool Statistics
```bash
pawd query dex advanced pool-stats [pool-id]

# Example: Get comprehensive stats for pool 1
pawd query dex advanced pool-stats 1
```

#### All Pool Statistics
```bash
pawd query dex advanced all-pool-stats

# Sort by APY, show top 10
pawd query dex advanced all-pool-stats --sort-by apy --limit 10

# Filter by minimum TVL
pawd query dex advanced all-pool-stats --min-tvl 1000000
```

#### Current Price
```bash
pawd query dex advanced price [token-in] [token-out]

# Example: Get PAW/USDT price
pawd query dex advanced price upaw uusdt
```

#### User Portfolio
```bash
pawd query dex advanced portfolio [address]

# Example: View all LP positions for an address
pawd query dex advanced portfolio paw1abcdefghijklmnopqrstuvwxyz...
```

#### LP Position Details
```bash
pawd query dex advanced lp-position [pool-id] [address]

# Example
pawd query dex advanced lp-position 1 paw1abcdefghijklmnopqrstuvwxyz...
```

#### Arbitrage Detection
```bash
pawd query dex advanced arbitrage

# Filter by minimum profit percentage
pawd query dex advanced arbitrage --min-profit 0.5
```

#### Optimal Swap Route
```bash
pawd query dex advanced route [token-in] [token-out] [amount]

# Example: Find best route for PAW to ATOM
pawd query dex advanced route upaw uatom 1000000000
```

#### Fee APY Calculation
```bash
pawd query dex advanced fee-apy [pool-id]

# Example
pawd query dex advanced fee-apy 1
```

### DEX Statistics

#### Market Overview
```bash
pawd query dex stats overview

# Shows: total pools, TVL, active tokens, etc.
```

#### Top Pools
```bash
pawd query dex stats top-pools

# Show top 5 pools by TVL
pawd query dex stats top-pools --limit 5
```

#### User Statistics
```bash
pawd query dex stats user [address]

# Example: View comprehensive DEX stats for a user
pawd query dex stats user paw1abcdefghijklmnopqrstuvwxyz...
```

#### Token Information
```bash
pawd query dex stats token [denom]

# Example: Get all info about PAW token
pawd query dex stats token upaw
```

### Module Parameters

#### Query DEX Parameters
```bash
pawd query dex params

# Shows: swap fees, limits, enabled features
```

## Common Use Cases

### 1. Provide Liquidity

```bash
# Step 1: Check existing pools
pawd query dex pools

# Step 2: Create new pool if needed
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 --from alice

# Step 3: Or add to existing pool
pawd tx dex add-liquidity 1 100000000 200000000 --from alice
```

### 2. Execute a Trade

```bash
# Step 1: Find the pool
pawd query dex pool-by-tokens upaw uusdt

# Step 2: Simulate the swap
pawd query dex simulate-swap 1 upaw uusdt 1000000000

# Step 3: Execute with slippage protection
pawd tx dex advanced swap-with-slippage 1 upaw 1000000000 uusdt 0.5 --from alice
```

### 3. Set Limit Order

```bash
# Step 1: Check current price
pawd query dex advanced price upaw uusdt

# Step 2: Place limit order above/below market
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.1 --from alice

# Step 3: Monitor your orders
pawd query dex orders-by-owner $(pawd keys show alice -a)
```

### 4. Analyze Market

```bash
# Get market overview
pawd query dex stats overview

# Find best pools
pawd query dex stats top-pools --limit 10

# Check specific token
pawd query dex stats token upaw

# Detect arbitrage opportunities
pawd query dex advanced arbitrage --min-profit 1.0
```

### 5. Manage Portfolio

```bash
# View all positions
pawd query dex advanced portfolio $(pawd keys show alice -a)

# Check specific LP position
pawd query dex advanced lp-position 1 $(pawd keys show alice -a)

# View user DEX stats
pawd query dex stats user $(pawd keys show alice -a)
```

## Tips & Best Practices

### Slippage Protection
Always set appropriate `min-amount-out` or use `swap-with-slippage` to protect against:
- Front-running attacks
- High volatility periods
- Large trades with high price impact

### Deadline Usage
Set reasonable deadlines (300-600 seconds) for time-sensitive operations to prevent:
- Stale transactions executing at bad prices
- Blockchain congestion issues

### Gas Optimization
- Use `quick-swap` for simple trades (auto-discovers pool)
- Batch multiple operations when possible
- Check simulations before executing expensive operations

### Liquidity Management
- Use `add-liquidity-balanced` for automatic ratio calculation
- Monitor impermanent loss with `lp-position` queries
- Check fee APY before providing liquidity

### Order Book Strategy
- Place limit orders with expiration for conditional trades
- Monitor order book depth before large trades
- Use `cancel-all` for quick exit from all positions

## Output Formats

Most query commands support JSON output:

```bash
pawd query dex pool 1 --output json | jq '.'
pawd query dex stats overview --output json
```

## Common Errors

### "insufficient funds"
- Check balance: `pawd query bank balances $(pawd keys show alice -a)`
- Ensure you have enough tokens + gas fees

### "slippage exceeded"
- Increase your slippage tolerance
- Wait for better market conditions
- Reduce trade size

### "pool not found"
- Verify pool exists: `pawd query dex pools`
- Check token denominations are correct

### "deadline exceeded"
- Transaction took too long to process
- Increase deadline parameter or retry

## Advanced Topics

### Interactive Mode
Some commands support interactive mode for easier UX (implementation in `interactive.go`).

### Price Calculation
Prices are calculated using constant product formula: `x * y = k`
- Includes 0.3% swap fee (configurable)
- Price impact increases with trade size relative to pool

### Limit Order Execution
Orders execute automatically when:
- Pool price reaches limit price
- Sufficient liquidity available
- Order has not expired

### Security Features
- Reentrancy protection on all state-changing operations
- Deadline enforcement for time-sensitive operations
- Minimum output validation (slippage protection)
- Access control on order cancellation

---

**For more information:**
- [PAW Documentation](../README.md)
- [DEX Module Specification](./DEX_SPEC.md)
- [Keeper Implementation](../x/dex/keeper/)
