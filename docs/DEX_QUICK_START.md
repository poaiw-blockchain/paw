# PAW DEX Quick Start Guide

Get started with PAW's decentralized exchange in minutes.

## What is PAW DEX?

PAW DEX is an automated market maker (AMM) built into the PAW blockchain that enables:
- Token swaps with constant product formula (x * y = k)
- Liquidity provision and farming
- Limit orders with expiration
- Cross-chain DEX aggregation via IBC
- Advanced trading features (slippage protection, multi-hop routing)

## Prerequisites

- PAW daemon installed (`pawd`)
- Wallet with PAW tokens (`upaw`)
- Node running or access to RPC endpoint

## Basic Setup

### 1. Check Your Balance

```bash
pawd query bank balances $(pawd keys show alice -a)
```

### 2. View Available Pools

```bash
pawd query dex pools
```

## Common Tasks

### Creating a Liquidity Pool

Create a new trading pair with initial liquidity:

```bash
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 \
  --from alice \
  --chain-id paw-mvp-1 \
  --gas auto
```

This creates a PAW/USDT pool with 1,000,000 PAW and 2,000,000 USDT.

### Adding Liquidity

Add liquidity to an existing pool:

```bash
# Find the pool ID first
pawd query dex pool-by-tokens upaw uusdt

# Add liquidity to pool 1
pawd tx dex add-liquidity 1 100000000 200000000 \
  --from alice \
  --chain-id paw-mvp-1
```

**Pro tip:** Use `add-liquidity-balanced` for automatic ratio calculation:

```bash
pawd tx dex advanced add-liquidity-balanced 1 1000000000 \
  --from alice
```

### Executing Swaps

#### Basic Swap

```bash
pawd tx dex swap 1 upaw 1000000000 uusdt 1900000000 \
  --from alice \
  --chain-id paw-mvp-1
```

This swaps 1,000 PAW for at least 1,900 USDT.

#### Swap with Slippage Protection

```bash
pawd tx dex advanced swap-with-slippage 1 upaw 1000000000 uusdt 0.5 \
  --from alice \
  --deadline 300
```

Swaps with 0.5% slippage tolerance and 5-minute deadline.

#### Quick Swap (Auto Pool Discovery)

```bash
pawd tx dex advanced quick-swap upaw 1000000000 uusdt \
  --from alice
```

Automatically finds the best pool for your trade.

### Simulating Trades

Always simulate before executing large trades:

```bash
pawd query dex simulate-swap 1 upaw uusdt 1000000000
```

Returns expected output amount and price impact.

### Removing Liquidity

```bash
# Check your LP shares
pawd query dex liquidity 1 $(pawd keys show alice -a)

# Remove 50% of shares (500,000 out of 1,000,000)
pawd tx dex remove-liquidity 1 500000 \
  --from alice
```

## Advanced Features

### Limit Orders

Place buy/sell orders at specific prices:

```bash
# Check current price
pawd query dex advanced price upaw uusdt

# Place limit order: sell 1,000 PAW for USDT at minimum 2.0 USDT per PAW
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 \
  --from alice

# With 24-hour expiration
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 \
  --expiration 86400 \
  --from alice
```

**Manage orders:**

```bash
# View your orders
pawd query dex orders-by-owner $(pawd keys show alice -a)

# Cancel specific order
pawd tx dex limit-order cancel 123 --from alice

# Cancel all orders
pawd tx dex limit-order cancel-all --from alice
```

### Portfolio Management

```bash
# View all LP positions
pawd query dex advanced portfolio $(pawd keys show alice -a)

# Get detailed position info
pawd query dex advanced lp-position 1 $(pawd keys show alice -a)

# View user DEX statistics
pawd query dex stats user $(pawd keys show alice -a)
```

### Market Analysis

```bash
# Market overview
pawd query dex stats overview

# Top pools by TVL
pawd query dex stats top-pools --limit 10

# Pool statistics
pawd query dex advanced pool-stats 1

# Token information
pawd query dex stats token upaw

# Find arbitrage opportunities
pawd query dex advanced arbitrage --min-profit 1.0
```

## Complete Trading Workflow

### Example: Provide Liquidity and Earn Fees

```bash
# Step 1: Check existing pools
pawd query dex pools

# Step 2: Create pool if needed (or skip to step 3)
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 --from alice

# Step 3: Add liquidity
pawd tx dex add-liquidity 1 100000000 200000000 --from alice

# Step 4: Monitor your position
pawd query dex advanced lp-position 1 $(pawd keys show alice -a)

# Step 5: Check fee APY
pawd query dex advanced fee-apy 1

# Step 6: Remove liquidity when desired
pawd tx dex remove-liquidity 1 <shares> --from alice
```

### Example: Execute Market Trade

```bash
# Step 1: Find pool for token pair
pawd query dex pool-by-tokens upaw uusdt

# Step 2: Check current price
pawd query dex advanced price upaw uusdt

# Step 3: Simulate the swap
pawd query dex simulate-swap 1 upaw uusdt 1000000000

# Step 4: Execute with slippage protection
pawd tx dex advanced swap-with-slippage 1 upaw 1000000000 uusdt 0.5 --from alice
```

## Best Practices

### For Liquidity Providers

1. **Check APY before providing liquidity:**
   ```bash
   pawd query dex advanced fee-apy <pool-id>
   ```

2. **Monitor impermanent loss:**
   ```bash
   pawd query dex advanced lp-position <pool-id> <address>
   ```

3. **Use balanced liquidity addition** to avoid manual ratio calculations

4. **Remove liquidity gradually** during high volatility

### For Traders

1. **Always simulate large trades** to check price impact
2. **Use slippage protection** (0.5-5% depending on market conditions)
3. **Set deadlines** (300-600 seconds) to prevent stale transactions
4. **Check order book depth** before large market orders:
   ```bash
   pawd query dex order-book <pool-id> --limit 50
   ```

### Security

- **Never share your mnemonic or private keys**
- **Verify addresses** before executing transactions
- **Start with small amounts** when testing
- **Use hardware wallets** for production funds
- **Set minimum output amounts** to protect against front-running

## Common Issues

### "insufficient funds"

Check your balance includes gas fees:
```bash
pawd query bank balances $(pawd keys show alice -a)
```

### "slippage exceeded"

- Increase slippage tolerance
- Reduce trade size
- Wait for better market conditions

### "pool not found"

Verify pool exists and token denominations are correct:
```bash
pawd query dex pools
```

### "deadline exceeded"

Transaction took too long. Increase `--deadline` parameter or retry.

## Tips for Gas Optimization

- Use `--gas auto --gas-adjustment 1.3` for automatic gas estimation
- Batch operations when possible
- Use `quick-swap` instead of manual pool lookup + swap

## Next Steps

- **Cross-chain swaps:** See [IBC_QUICK_START.md](IBC_QUICK_START.md) for cross-chain DEX features
- **Full CLI reference:** [CLI_DEX.md](CLI_DEX.md)
- **Technical details:** [TECHNICAL_SPECIFICATION.md](TECHNICAL_SPECIFICATION.md)
- **DEX module code:** `/x/dex/`

## Getting Help

- Discord: https://discord.gg/DBHTc2QV
- Documentation: https://docs.paw-chain.com/dex
- GitHub Issues: https://github.com/paw-chain/paw
