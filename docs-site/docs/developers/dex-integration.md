# DEX Integration Guide

Build trading applications on PAW's decentralized exchange.

## Overview

PAW DEX is an automated market maker (AMM) built into the blockchain that enables:
- Token swaps with constant product formula (x * y = k)
- Liquidity provision and farming
- Limit orders with expiration
- Cross-chain DEX aggregation via IBC
- Advanced trading features (slippage protection, multi-hop routing)

## Getting Started

### Prerequisites

- PAW daemon installed and running
- Wallet with PAW tokens (`upaw`)
- Node access (local or remote RPC endpoint)

### Check Your Balance

```bash
pawd query bank balances $(pawd keys show alice -a)
```

### View Available Pools

```bash
pawd query dex pools
```

## Core Operations

### Creating a Liquidity Pool

Create a new trading pair with initial liquidity:

```bash
pawd tx dex create-pool upaw 1000000000000 uusdt 2000000000000 \
  --from alice \
  --chain-id paw-mvp-1 \
  --gas auto \
  --yes
```

This creates a PAW/USDT pool with 1,000,000 PAW and 2,000,000 USDT.

**Via REST API:**

```bash
curl -X POST http://localhost:1317/paw/dex/v1/create_pool \
  -H "Content-Type: application/json" \
  -d '{
    "creator": "paw1abc123...",
    "token_a": "upaw",
    "amount_a": "1000000000000",
    "token_b": "uusdt",
    "amount_b": "2000000000000"
  }'
```

### Adding Liquidity

Add liquidity to an existing pool:

```bash
# Find the pool ID first
pawd query dex pool-by-tokens upaw uusdt

# Add liquidity to pool 1
pawd tx dex add-liquidity 1 100000000 200000000 \
  --from alice \
  --chain-id paw-mvp-1 \
  --yes
```

**Balanced Liquidity Addition:**

Automatically calculate the correct ratio:

```bash
pawd tx dex advanced add-liquidity-balanced 1 1000000000 \
  --from alice \
  --yes
```

**Via TypeScript SDK:**

```typescript
import { PawClient } from '@paw-chain/sdk';

const client = new PawClient({
  rpcEndpoint: 'http://localhost:26657',
});

const result = await client.dex.addLiquidity({
  poolId: 1,
  amountA: '100000000',
  amountB: '200000000',
  signer: wallet,
});
```

### Executing Swaps

#### Basic Swap

```bash
pawd tx dex swap 1 upaw 1000000000 uusdt 1900000000 \
  --from alice \
  --chain-id paw-mvp-1 \
  --yes
```

This swaps 1,000 PAW for at least 1,900 USDT.

#### Swap with Slippage Protection

```bash
pawd tx dex advanced swap-with-slippage 1 upaw 1000000000 uusdt 0.5 \
  --from alice \
  --deadline 300 \
  --yes
```

Swaps with 0.5% slippage tolerance and 5-minute deadline.

#### Quick Swap (Auto Pool Discovery)

```bash
pawd tx dex advanced quick-swap upaw 1000000000 uusdt \
  --from alice \
  --yes
```

Automatically finds the best pool for your trade.

**Via Python SDK:**

```python
from paw_sdk import PawClient

client = PawClient(rpc_endpoint='http://localhost:26657')

result = client.dex.swap(
    pool_id=1,
    token_in='upaw',
    amount_in='1000000000',
    token_out='uusdt',
    min_amount_out='1900000000',
    signer=wallet,
)
```

### Simulating Trades

Always simulate before executing large trades:

```bash
pawd query dex simulate-swap 1 upaw uusdt 1000000000
```

Returns expected output amount and price impact.

**Via REST API:**

```bash
curl "http://localhost:1317/paw/dex/v1/simulate_swap?pool_id=1&token_in=upaw&token_out=uusdt&amount_in=1000000000"
```

### Removing Liquidity

```bash
# Check your LP shares
pawd query dex liquidity 1 $(pawd keys show alice -a)

# Remove 50% of shares (500,000 out of 1,000,000)
pawd tx dex remove-liquidity 1 500000 \
  --from alice \
  --yes
```

## Advanced Features

### Limit Orders

Place buy/sell orders at specific prices:

```bash
# Check current price
pawd query dex advanced price upaw uusdt

# Place limit order: sell 1,000 PAW for USDT at minimum 2.0 USDT per PAW
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 \
  --from alice \
  --yes

# With 24-hour expiration
pawd tx dex limit-order place 1 upaw 1000000000 uusdt 2.0 \
  --expiration 86400 \
  --from alice \
  --yes
```

**Manage Orders:**

```bash
# View your orders
pawd query dex orders-by-owner $(pawd keys show alice -a)

# Cancel specific order
pawd tx dex limit-order cancel 123 --from alice --yes

# Cancel all orders
pawd tx dex limit-order cancel-all --from alice --yes
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

## Building a Trading Application

### Example: Trading Bot

```typescript
import { PawClient } from '@paw-chain/sdk';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';

// Initialize client
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(process.env.MNEMONIC);
const client = new PawClient({
  rpcEndpoint: 'http://localhost:26657',
  wallet: wallet,
});

// Monitor pools
async function monitorPools() {
  const pools = await client.dex.getPools();

  for (const pool of pools) {
    const stats = await client.dex.getPoolStats(pool.id);

    // Check for arbitrage opportunities
    if (stats.priceImpact > 0.05) {
      console.log(`Arbitrage opportunity in pool ${pool.id}`);
      await executeArbitrage(pool.id);
    }
  }
}

// Execute arbitrage
async function executeArbitrage(poolId: number) {
  // Simulate swap first
  const simulation = await client.dex.simulateSwap({
    poolId: poolId,
    tokenIn: 'upaw',
    tokenOut: 'uusdt',
    amountIn: '1000000',
  });

  if (simulation.expectedReturn > simulation.amountIn * 1.02) {
    // Execute if profitable
    await client.dex.swap({
      poolId: poolId,
      tokenIn: 'upaw',
      amountIn: '1000000',
      tokenOut: 'uusdt',
      minAmountOut: simulation.expectedReturn.toString(),
      signer: wallet,
    });
  }
}

// Run every 10 seconds
setInterval(monitorPools, 10000);
```

### Example: Liquidity Provider Dashboard

```python
from paw_sdk import PawClient

client = PawClient(rpc_endpoint='http://localhost:26657')

def get_user_positions(address):
    # Get all LP positions
    portfolio = client.dex.get_portfolio(address)

    positions = []
    for position in portfolio:
        # Get detailed position info
        details = client.dex.get_lp_position(position['pool_id'], address)

        # Calculate APY
        apy = client.dex.get_fee_apy(position['pool_id'])

        positions.append({
            'pool_id': position['pool_id'],
            'shares': details['shares'],
            'value_usd': details['value_usd'],
            'apy': apy,
            'fees_earned': details['fees_earned'],
        })

    return positions

# Display positions
for pos in get_user_positions('paw1abc123...'):
    print(f"Pool {pos['pool_id']}: ${pos['value_usd']} @ {pos['apy']}% APY")
```

## WebSocket Integration

Real-time DEX events:

```javascript
const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:26657/websocket');

ws.on('open', () => {
  // Subscribe to DEX swap events
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    method: 'subscribe',
    id: 1,
    params: {
      query: "message.action='swap'"
    }
  }));

  // Subscribe to liquidity events
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    method: 'subscribe',
    id: 2,
    params: {
      query: "message.action='add_liquidity' OR message.action='remove_liquidity'"
    }
  }));
});

ws.on('message', (data) => {
  const event = JSON.parse(data);

  if (event.result.events) {
    console.log('New DEX event:', event.result.events);
    // Process event data
  }
});
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

- Never share your mnemonic or private keys
- Verify addresses before executing transactions
- Start with small amounts when testing
- Use hardware wallets for production funds
- Set minimum output amounts to protect against front-running

## Error Handling

### Common Errors

**"insufficient funds"**
```bash
pawd query bank balances $(pawd keys show alice -a)
```
Check your balance includes gas fees.

**"slippage exceeded"**
- Increase slippage tolerance
- Reduce trade size
- Wait for better market conditions

**"pool not found"**
```bash
pawd query dex pools
```
Verify pool exists and token denominations are correct.

**"deadline exceeded"**

Transaction took too long. Increase `--deadline` parameter or retry.

## Performance Tips

- Use `--gas auto --gas-adjustment 1.3` for automatic gas estimation
- Batch operations when possible
- Use `quick-swap` instead of manual pool lookup + swap
- Cache pool data to reduce queries
- Use gRPC for high-frequency trading

## Next Steps

- [IBC Channels Guide](ibc-channels.md) - Cross-chain DEX features
- [REST API Reference](https://docs.paw-chain.com/api/dex)
- [Example Trading Bot](https://github.com/poaiw-blockchain/examples/tree/main/dex-bot)
