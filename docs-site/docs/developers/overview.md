# Developer Overview

Build applications on PAW blockchain with our comprehensive developer tools and APIs.

## What You Can Build

### Trading Applications
- DEX frontends and trading bots
- Portfolio management tools
- Liquidity analytics dashboards
- Arbitrage systems

### Compute Applications
- AI/ML job submission platforms
- Verifiable computation services
- Cross-chain compute orchestration

### Oracle Applications
- Price feed aggregators
- Custom oracle implementations
- Data provider services

### Cross-Chain Applications
- IBC-enabled wallets
- Cross-chain swap interfaces
- Multi-chain asset management

## Development Tools

### CLI (Command Line Interface)

The `pawd` CLI provides complete blockchain interaction:

```bash
# Query operations
pawd query dex pools
pawd query bank balances <address>
pawd query oracle price BTC/USD

# Transaction operations
pawd tx dex swap <pool-id> <amount-in> <min-out>
pawd tx compute submit-job <job-data>
pawd tx oracle report-price <symbol> <price>
```

Full CLI reference: Run `pawd --help` or see our [CLI documentation](https://github.com/poaiw-blockchain/paw/tree/main/docs).

### REST API

Query blockchain data via HTTP:

```bash
# Get DEX pools
curl http://localhost:1317/paw/dex/v1/pools

# Get account balance
curl http://localhost:1317/cosmos/bank/v1beta1/balances/{address}

# Get oracle prices
curl http://localhost:1317/paw/oracle/v1/prices
```

API documentation: `http://localhost:1317/swagger/` (when node is running)

### gRPC API

For high-performance applications:

```go
import (
    "google.golang.org/grpc"
    dextypes "github.com/poaiw-blockchain/paw/x/dex/types"
)

conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
client := dextypes.NewQueryClient(conn)

// Query pools
resp, _ := client.Pools(context.Background(), &dextypes.QueryPoolsRequest{})
```

### WebSocket

Real-time blockchain events:

```javascript
const ws = new WebSocket('ws://localhost:26657/websocket');

// Subscribe to new blocks
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  id: 1,
  params: {
    query: "tm.event='NewBlock'"
  }
}));

// Subscribe to DEX events
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  id: 2,
  params: {
    query: "message.module='dex'"
  }
}));
```

## SDKs and Libraries

### JavaScript/TypeScript

```bash
npm install @paw-chain/sdk
```

```typescript
import { PawClient } from '@paw-chain/sdk';

const client = new PawClient({
  rpcEndpoint: 'http://localhost:26657',
  restEndpoint: 'http://localhost:1317',
});

// Query DEX pools
const pools = await client.dex.getPools();

// Execute swap
const result = await client.dex.swap({
  poolId: 1,
  tokenIn: 'upaw',
  amountIn: '1000000',
  tokenOut: 'uusdt',
  minAmountOut: '1900000',
  signer: wallet,
});
```

### Python

```bash
pip install paw-sdk
```

```python
from paw_sdk import PawClient

client = PawClient(
    rpc_endpoint='http://localhost:26657',
    rest_endpoint='http://localhost:1317',
)

# Query pools
pools = client.dex.get_pools()

# Execute swap
result = client.dex.swap(
    pool_id=1,
    token_in='upaw',
    amount_in='1000000',
    token_out='uusdt',
    min_amount_out='1900000',
    signer=wallet,
)
```

### Go

```go
import (
    "github.com/poaiw-blockchain/paw/app"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// Native Go integration
func main() {
    encodingConfig := app.MakeEncodingConfig()
    // Use PAW types directly
}
```

## Key Concepts

### Accounts

PAW uses Bech32 addresses with prefix `paw`:

```
paw1abc123...  // Regular account
pawvaloper1abc123...  // Validator operator account
```

Generate addresses:
```bash
pawd keys add mykey
pawd keys show mykey -a
```

### Gas and Fees

All transactions require gas fees paid in `upaw`:

```bash
# Auto-calculate gas
pawd tx ... --gas auto --gas-adjustment 1.5 --fees 1000upaw

# Manual gas
pawd tx ... --gas 200000 --fees 2000upaw
```

Minimum gas price: `0.001upaw` per unit

### Transactions

Transaction lifecycle:
1. Build transaction message
2. Sign with private key
3. Broadcast to network
4. Wait for confirmation (1-2 blocks)

```bash
# Build, sign, and broadcast in one command
pawd tx bank send alice bob 1000000upaw --yes

# Or do separately
pawd tx bank send alice bob 1000000upaw --generate-only > unsigned.json
pawd tx sign unsigned.json --from alice > signed.json
pawd tx broadcast signed.json
```

### Queries

Queries are read-only and don't require gas:

```bash
pawd query bank balances paw1abc123...
pawd query dex pool 1
pawd query oracle price BTC/USD
```

## Module-Specific Guides

### DEX Development

Build trading applications with the DEX module:

- [DEX Integration Guide](dex-integration.md) - Complete DEX development guide
- Swap tokens, manage liquidity, place limit orders
- Query pools, simulate swaps, track portfolio

### IBC Development

Enable cross-chain features:

- [IBC Channels Guide](ibc-channels.md) - Cross-chain integration
- Token transfers, cross-chain swaps
- Oracle price feeds, compute jobs

### Compute Development

Submit and verify AI/ML jobs:

```bash
# Submit job
pawd tx compute submit-job \
  --job-type wasm \
  --job-data @model.wasm \
  --payment 1000000upaw \
  --from requester

# Query job status
pawd query compute job-status <job-id>
```

### Oracle Development

Report and consume price feeds:

```bash
# Report price (validators/oracles only)
pawd tx oracle report-price BTC/USD 45000.00 --from oracle

# Query aggregated price
pawd query oracle aggregated-price BTC/USD
```

## Testing

### Local Development

Use a local node for development:

```bash
# Start local node
pawd start --home ./localnode

# Use in another terminal
export NODE="http://localhost:26657"
pawd query ... --node $NODE
```

### Testnet

Deploy to public testnet:

- Testnet RPC: `https://rpc.paw-testnet.com`
- Testnet REST: `https://api.paw-testnet.com`
- Testnet Explorer: `https://explorer.paw-testnet.com`

Request testnet tokens from faucet:
```bash
curl -X POST https://faucet.paw-testnet.com/request \
  -H "Content-Type: application/json" \
  -d '{"address":"paw1abc123..."}'
```

### Testing Tools

```bash
# Simulate transactions
pawd tx ... --dry-run

# Estimate gas
pawd tx ... --gas auto

# Query simulation endpoint
curl -X POST http://localhost:1317/cosmos/tx/v1beta1/simulate \
  -d '{"tx_bytes":"..."}'
```

## Best Practices

### Security

1. Never commit private keys or mnemonics
2. Use environment variables for sensitive data
3. Validate all user input
4. Use hardware wallets for production funds
5. Set minimum output amounts to prevent slippage attacks

### Performance

1. Use gRPC for high-throughput applications
2. Batch queries when possible
3. Cache frequently accessed data
4. Use state sync for faster node setup
5. Implement proper error handling and retries

### Error Handling

```typescript
try {
  await client.dex.swap({ ... });
} catch (error) {
  if (error.code === 'INSUFFICIENT_FUNDS') {
    // Handle insufficient balance
  } else if (error.code === 'SLIPPAGE_EXCEEDED') {
    // Handle slippage error
  } else {
    // Handle other errors
  }
}
```

## Examples

See our example repositories:

- [DEX Trading Bot](https://github.com/poaiw-blockchain/examples/tree/main/dex-bot)
- [Portfolio Tracker](https://github.com/poaiw-blockchain/examples/tree/main/portfolio-tracker)
- [IBC Transfer Demo](https://github.com/poaiw-blockchain/examples/tree/main/ibc-transfer)
- [Oracle Price Feed](https://github.com/poaiw-blockchain/examples/tree/main/oracle-feed)

## Next Steps

- [DEX Integration Guide](dex-integration.md)
- [IBC Channels Guide](ibc-channels.md)
- [API Reference](https://docs.paw-chain.com/api)
- [GitHub Repository](https://github.com/poaiw-blockchain/paw)
