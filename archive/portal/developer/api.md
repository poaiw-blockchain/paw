# API Reference

Complete REST and gRPC API documentation for PAW Blockchain.

## Base URLs

- **Mainnet REST**: `https://api.paw.network`
- **Mainnet RPC**: `https://rpc.paw.network`
- **Mainnet gRPC**: `grpc.paw.network:9090`
- **Testnet REST**: `https://api-testnet.paw.network`
- **Testnet RPC**: `https://rpc-testnet.paw.network`

## Authentication

Most read operations don't require authentication. Transactions must be signed.

## Bank Module

### Get Balance

```http
GET /cosmos/bank/v1beta1/balances/{address}
```

**Example:**
```bash
curl https://api.paw.network/cosmos/bank/v1beta1/balances/paw1xxxxx...
```

**Response:**
```json
{
  "balances": [
    {
      "denom": "upaw",
      "amount": "1000000000"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "1"
  }
}
```

### Send Tokens

```http
POST /cosmos/tx/v1beta1/txs
```

**Body:**
```json
{
  "tx": {
    "body": {
      "messages": [{
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "paw1...",
        "to_address": "paw2...",
        "amount": [{"denom": "upaw", "amount": "1000000"}]
      }]
    },
    "signatures": ["..."]
  },
  "mode": "BROADCAST_MODE_SYNC"
}
```

## Staking Module

### Get Validators

```http
GET /cosmos/staking/v1beta1/validators
```

### Get Delegations

```http
GET /cosmos/staking/v1beta1/delegations/{delegator_addr}
```

### Delegate

```http
POST /cosmos/tx/v1beta1/txs
```

## DEX Module

### Get Pools

```http
GET /paw/dex/v1beta1/pools
```

**Response:**
```json
{
  "pools": [
    {
      "id": "1",
      "token_a": "upaw",
      "token_b": "uusdc",
      "reserve_a": "1000000000",
      "reserve_b": "1000000000",
      "total_liquidity": "1000000000"
    }
  ]
}
```

### Swap Tokens

```http
POST /paw/dex/v1beta1/swap
```

**Body:**
```json
{
  "pool_id": "1",
  "token_in": {"denom": "upaw", "amount": "1000000"},
  "min_token_out": "950000",
  "sender": "paw1..."
}
```

## Governance

### Get Proposals

```http
GET /cosmos/gov/v1beta1/proposals
```

### Vote

```http
POST /cosmos/gov/v1beta1/proposals/{proposal_id}/votes
```

## WebSocket API

Connect to real-time events:

```javascript
const ws = new WebSocket('wss://rpc.paw.network/websocket');

ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  id: 1,
  params: {
    query: "tm.event='NewBlock'"
  }
}));

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('New block:', data.result.data.value.block.header.height);
};
```

## Rate Limits

- **Free tier**: 100 requests/minute
- **Authenticated**: 1000 requests/minute
- **Enterprise**: Custom limits

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request |
| 401 | Unauthorized |
| 404 | Not Found |
| 429 | Rate Limit Exceeded |
| 500 | Internal Server Error |

## Interactive API Explorer

Try the API at [api-explorer.paw.network](https://api-explorer.paw.network)

---

**Previous:** [Module Development](/developer/module-development) | **Next:** [Validator Setup](/validator/setup) →
