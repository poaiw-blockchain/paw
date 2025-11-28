# Using the PAW DEX

The PAW Decentralized Exchange (DEX) allows you to trade tokens directly from your wallet without intermediaries.

## What is the PAW DEX?

The PAW DEX is a built-in decentralized exchange featuring:

- **Atomic Swaps**: Trustless token exchanges
- **AMM Pools**: Automated market maker liquidity pools
- **Low Fees**: Minimal trading fees (0.3% per swap)
- **Instant Settlement**: Sub-second trade finality
- **No KYC**: Trade without identity verification
- **Self-Custody**: You always control your keys

## Getting Started with Trading

### Prerequisites

- PAW wallet with funds
- Sufficient PAW for transaction fees
- Understanding of trading risks

### Accessing the DEX

**Option 1: Web Interface**
```bash
https://dex.paw.network
```

**Option 2: Desktop Wallet**
- Open PAW Desktop Wallet
- Navigate to "DEX" tab

**Option 3: Command Line**
```bash
pawd tx dex swap --help
```

## Available Trading Pairs

### Current Pairs

The DEX currently supports:

| Pair | Liquidity | 24h Volume | Fee |
|------|-----------|------------|-----|
| PAW/USDC | $2.5M | $450K | 0.3% |
| PAW/ATOM | $1.2M | $280K | 0.3% |
| PAW/ETH | $800K | $150K | 0.3% |
| USDC/ATOM | $600K | $95K | 0.3% |

::: tip
New trading pairs can be added through governance proposals.
:::

## Making a Swap

### Web Interface

1. **Connect Your Wallet**
   - Click "Connect Wallet"
   - Select wallet type (Keplr, Ledger, etc.)
   - Approve connection

2. **Select Trading Pair**
   - Choose tokens to swap
   - Enter amount to trade
   - View exchange rate and fees

3. **Review and Confirm**
   - Check slippage tolerance
   - Review total cost
   - Click "Swap"
   - Confirm in wallet

4. **Transaction Complete**
   - View transaction hash
   - Check updated balances

### Command Line

```bash
# Swap 1000 PAW for USDC
pawd tx dex swap \
  --amount 1000000000upaw \
  --min-out 950000000uusdc \
  --pair upaw:uusdc \
  --from my-wallet \
  --fees 500upaw \
  --gas auto

# Swap with specific slippage tolerance (1%)
pawd tx dex swap \
  --amount 1000000000upaw \
  --slippage 0.01 \
  --pair upaw:uusdc \
  --from my-wallet
```

### Understanding Slippage

**Slippage** is the difference between expected and actual trade price.

```bash
# Example with 1% slippage tolerance
Expected: 1000 PAW → 1000 USDC
Minimum accepted: 1000 PAW → 990 USDC (1% slippage)

# If price moves more than 1%, transaction reverts
```

::: warning
Higher slippage = more likely to execute but worse price
Lower slippage = better price but may fail if market moves
:::

## Providing Liquidity

Earn fees by providing liquidity to trading pools.

### How Liquidity Pools Work

- **Add Equal Value**: Deposit tokens in 50/50 ratio
- **Receive LP Tokens**: Proof of your pool share
- **Earn Fees**: Get 0.3% of all trades in your pool
- **Impermanent Loss**: Risk if prices diverge significantly

### Adding Liquidity

**Web Interface:**

1. Navigate to "Pools" tab
2. Select pool to join
3. Enter amount to add
4. Approve transaction
5. Receive LP tokens

**Command Line:**

```bash
# Add liquidity to PAW/USDC pool
pawd tx dex add-liquidity \
  --token-a 1000000000upaw \
  --token-b 1000000000uusdc \
  --pool-id 1 \
  --from my-wallet \
  --fees 500upaw

# Pool automatically calculates optimal ratio
```

### Removing Liquidity

```bash
# Remove liquidity from pool
pawd tx dex remove-liquidity \
  --lp-tokens 500000000 \
  --pool-id 1 \
  --min-token-a 450000000 \
  --min-token-b 450000000 \
  --from my-wallet \
  --fees 500upaw
```

### Calculating Returns

**Example Pool Returns:**

Initial Investment:
- 1000 PAW ($1000)
- 1000 USDC ($1000)
- Total: $2000

After 30 days:
- Trading fees earned: $45 (0.3% × $500K volume)
- Your pool share: 10%
- Your earnings: $4.50/day = $135/month
- APR: ~81% (before impermanent loss)

::: tip APR Calculator
Use the [Pool Calculator](https://dex.paw.network/calculator) to estimate returns.
:::

## Understanding Impermanent Loss

### What is Impermanent Loss?

Loss occurs when token prices diverge from your entry point.

**Example:**

```
Initial: 1000 PAW ($1 each) + 1000 USDC = $2000

PAW doubles to $2:
- Holding: 1000 PAW × $2 + 1000 USDC = $3000
- In Pool: 707 PAW × $2 + 1414 USDC = $2828
- Impermanent Loss: $172 (5.7%)
```

### Mitigating Impermanent Loss

- Choose stable pairs (USDC/USDT)
- Provide liquidity long-term to earn more fees
- Monitor price ratios
- Consider single-sided staking instead

## Advanced Trading Features

### Limit Orders (Coming Soon)

```bash
# Place limit order (future feature)
pawd tx dex limit-order \
  --pair upaw:uusdc \
  --side buy \
  --price 1.05 \
  --amount 1000000000upaw
```

### Price Charts

Access real-time charts:
- **TradingView Integration**: [dex.paw.network/charts](https://dex.paw.network/charts)
- **API Access**: [developer/api](/developer/api)

### Market Data

```bash
# Get current pool info
pawd query dex pool 1

# Get pool reserves
pawd query dex reserves 1

# Get 24h volume
pawd query dex volume 1 --period 24h

# Get current price
pawd query dex price upaw uusdc
```

## DEX Security Features

### Circuit Breakers

Automatic trading halts if:
- Single trade exceeds 10% of pool
- Price moves more than 20% in one block
- Unusual volume detected

### Emergency Pause

Governance can pause trading in emergencies:

```bash
# Check if DEX is paused
pawd query dex params

# Emergency pause (governance only)
pawd tx gov submit-proposal param-change \
  --title "Pause DEX Trading" \
  --param dex.trading_enabled=false
```

### Audit Reports

- ✅ Trail of Bits (2024-10)
- ✅ CertiK (2024-11)
- ✅ Halborn (2024-12)

## Fee Structure

### Trading Fees

| Action | Fee | Recipient |
|--------|-----|-----------|
| Swap | 0.3% | Liquidity Providers |
| Add Liquidity | Network Fee | Validators |
| Remove Liquidity | Network Fee | Validators |

### Example Calculations

```bash
# Swap 1000 PAW for USDC
Amount: 1000 PAW
Trading Fee (0.3%): 3 PAW
Network Fee: 0.0005 PAW
You Receive: ~997 PAW worth of USDC

# Add Liquidity
Amount: 1000 PAW + 1000 USDC
Trading Fee: None
Network Fee: 0.001 PAW
```

## Common Use Cases

### Arbitrage Trading

Take advantage of price differences:

```bash
# Buy PAW on DEX at $1.00
# Sell PAW on CEX at $1.02
# Profit: $0.02 per PAW (minus fees)
```

### Portfolio Rebalancing

```bash
# Monthly rebalance script
# If PAW > 60% of portfolio, swap for stablecoins
# If PAW < 40% of portfolio, buy more PAW
```

### Dollar-Cost Averaging

```bash
# Automated weekly buys
pawd tx dex swap \
  --amount 100000000uusdc \
  --pair uusdc:upaw \
  --from my-wallet
# Run weekly via cron job
```

## Monitoring Your Positions

### Track Pool Performance

```bash
# View your LP positions
pawd query dex positions paw1xxxxx...

# Calculate current value
pawd query dex position-value paw1xxxxx... --pool-id 1

# View accumulated fees
pawd query dex fees-earned paw1xxxxx... --pool-id 1
```

### Analytics Dashboard

Visit [analytics.paw.network](https://analytics.paw.network) for:
- Personal trading history
- Pool performance charts
- Fee earnings tracker
- Impermanent loss calculator
- Portfolio overview

## Troubleshooting

### Transaction Failed

**Common issues:**

1. **Insufficient Funds**
   ```bash
   Error: insufficient funds for transaction fees
   Solution: Add more PAW to cover fees
   ```

2. **Slippage Too Low**
   ```bash
   Error: price moved beyond slippage tolerance
   Solution: Increase slippage or retry
   ```

3. **Pool Not Found**
   ```bash
   Error: pool does not exist
   Solution: Verify pool ID or create pool via governance
   ```

### Price Impact Too High

If warning shows "Price Impact > 5%":

- Your trade is large relative to pool size
- Consider splitting into smaller trades
- Wait for more liquidity to be added

### Liquidity Issues

If can't add liquidity:

```bash
# Check pool ratio
pawd query dex pool 1 | jq '.reserves'

# Your tokens must match pool ratio
# Example: If ratio is 2:1, deposit 2000 PAW per 1000 USDC
```

## Best Practices

### Trading Tips

1. **Always set slippage tolerance appropriately**
   - Stablecoins: 0.1-0.5%
   - Volatile pairs: 1-3%

2. **Check liquidity before large trades**
   ```bash
   pawd query dex pool 1 | jq '.total_liquidity'
   ```

3. **Monitor for MEV attacks**
   - Use private RPC endpoints
   - Set reasonable gas prices

4. **Keep transaction fees in wallet**
   - Always maintain 1-2 PAW for fees

### Liquidity Provision Tips

1. **Understand impermanent loss**
2. **Choose pairs wisely**
   - Correlated assets = less IL
   - Stable pairs = minimal IL
3. **Long-term commitment**
   - Fees accumulate over time
4. **Diversify across pools**
5. **Monitor pool health**

## Video Tutorials

### DEX Trading Tutorial

<div class="video-container">
  <iframe
    src="https://www.youtube.com/embed/DEX_TRADING_VIDEO_ID"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen>
  </iframe>
</div>

### Liquidity Provider Guide

<div class="video-container">
  <iframe
    src="https://www.youtube.com/embed/LP_GUIDE_VIDEO_ID"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen>
  </iframe>
</div>

## Next Steps

- **[Staking Guide](/guide/staking)** - Earn rewards by staking PAW
- **[Governance](/guide/governance)** - Vote on DEX parameters
- **[API Reference](/developer/api)** - Build trading bots

---

**Previous:** [Wallets](/guide/wallets) | **Next:** [Staking](/guide/staking) →
