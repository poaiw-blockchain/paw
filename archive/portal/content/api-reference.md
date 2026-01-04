# API Reference

Complete API documentation for PAW Blockchain.

## Base URLs

```
Mainnet API: https://api.pawchain.io
Mainnet RPC: https://rpc.pawchain.io
Mainnet WS:  wss://ws.pawchain.io

Testnet API: https://testnet-api.pawchain.io
Testnet RPC: https://testnet-rpc.pawchain.io
Testnet WS:  wss://testnet-ws.pawchain.io

Local API:   http://localhost:1317
Local RPC:   http://localhost:26657
```

## Authentication

PAW API is mostly public. Transaction broadcasting requires signing with your private key.

## Rate Limits

- **Public endpoints**: 100 requests/minute per IP
- **WebSocket**: 50 connections per IP
- **Heavy queries**: 10 requests/minute

## Bank Module

### Get All Balances

```http
GET /cosmos/bank/v1beta1/balances/{address}
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

### Get Balance by Denom

```http
GET /cosmos/bank/v1beta1/balances/{address}/by_denom?denom=upaw
```

### Get Total Supply

```http
GET /cosmos/bank/v1beta1/supply
```

### Send Tokens

```http
POST /cosmos/tx/v1beta1/txs
```

**Request Body:**
```json
{
  "tx": {
    "body": {
      "messages": [{
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "paw1abc...xyz",
        "to_address": "paw1def...uvw",
        "amount": [
          {
            "denom": "upaw",
            "amount": "1000000"
          }
        ]
      }],
      "memo": "Payment",
      "timeout_height": "0",
      "extension_options": [],
      "non_critical_extension_options": []
    },
    "auth_info": {
      "signer_infos": [],
      "fee": {
        "amount": [
          {
            "denom": "upaw",
            "amount": "1000"
          }
        ],
        "gas_limit": "200000",
        "payer": "",
        "granter": ""
      }
    },
    "signatures": []
  },
  "mode": "BROADCAST_MODE_SYNC"
}
```

## DEX Module

### Get All Pools

```http
GET /paw/dex/v1/pools
```

**Response:**
```json
{
  "pools": [
    {
      "id": "1",
      "assets": [
        {"denom": "upaw", "amount": "1000000000"},
        {"denom": "uusdc", "amount": "1000000000"}
      ],
      "total_shares": "1000000000",
      "swap_fee": "0.003"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "10"
  }
}
```

### Get Pool by ID

```http
GET /paw/dex/v1/pools/{poolId}
```

### Get Pool Liquidity

```http
GET /paw/dex/v1/pools/{poolId}/liquidity
```

### Swap Tokens

```http
POST /paw/dex/v1/swap
```

**Request:**
```json
{
  "sender": "paw1abc...xyz",
  "pool_id": "1",
  "token_in": {
    "denom": "upaw",
    "amount": "1000000"
  },
  "token_out_min_amount": "950000",
  "token_out_denom": "uusdc"
}
```

### Add Liquidity

```http
POST /paw/dex/v1/add-liquidity
```

### Remove Liquidity

```http
POST /paw/dex/v1/remove-liquidity
```

## Staking Module

### Get All Validators

```http
GET /cosmos/staking/v1beta1/validators
```

**Response:**
```json
{
  "validators": [
    {
      "operator_address": "pawvaloper1abc...xyz",
      "consensus_pubkey": {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": "..."
      },
      "jailed": false,
      "status": "BOND_STATUS_BONDED",
      "tokens": "1000000000",
      "delegator_shares": "1000000000.000000000000000000",
      "description": {
        "moniker": "Validator Name",
        "identity": "",
        "website": "https://validator.com",
        "security_contact": "security@validator.com",
        "details": "Description..."
      },
      "unbonding_height": "0",
      "unbonding_time": "1970-01-01T00:00:00Z",
      "commission": {
        "commission_rates": {
          "rate": "0.100000000000000000",
          "max_rate": "0.200000000000000000",
          "max_change_rate": "0.010000000000000000"
        },
        "update_time": "2025-01-01T00:00:00Z"
      },
      "min_self_delegation": "1"
    }
  ]
}
```

### Get Validator

```http
GET /cosmos/staking/v1beta1/validators/{validatorAddr}
```

### Get Delegations

```http
GET /cosmos/staking/v1beta1/delegations/{delegatorAddr}
```

### Get Delegation to Validator

```http
GET /cosmos/staking/v1beta1/validators/{validatorAddr}/delegations/{delegatorAddr}
```

### Delegate

```http
POST /cosmos/staking/v1beta1/delegate
```

**Request:**
```json
{
  "delegator_address": "paw1abc...xyz",
  "validator_address": "pawvaloper1abc...xyz",
  "amount": {
    "denom": "upaw",
    "amount": "1000000"
  }
}
```

### Undelegate

```http
POST /cosmos/staking/v1beta1/undelegate
```

### Redelegate

```http
POST /cosmos/staking/v1beta1/redelegate
```

## Distribution Module

### Get Rewards

```http
GET /cosmos/distribution/v1beta1/delegators/{delegatorAddr}/rewards
```

**Response:**
```json
{
  "rewards": [
    {
      "validator_address": "pawvaloper1abc...xyz",
      "reward": [
        {
          "denom": "upaw",
          "amount": "123456.789000000000000000"
        }
      ]
    }
  ],
  "total": [
    {
      "denom": "upaw",
      "amount": "123456.789000000000000000"
    }
  ]
}
```

### Withdraw Rewards

```http
POST /cosmos/distribution/v1beta1/withdraw-rewards
```

### Get Community Pool

```http
GET /cosmos/distribution/v1beta1/community_pool
```

## Governance Module

### Get Proposals

```http
GET /cosmos/gov/v1beta1/proposals
```

**Query Parameters:**
- `proposal_status`: PROPOSAL_STATUS_VOTING_PERIOD, etc.
- `voter`: Filter by voter address
- `depositor`: Filter by depositor

**Response:**
```json
{
  "proposals": [
    {
      "proposal_id": "1",
      "content": {
        "@type": "/cosmos.gov.v1beta1.TextProposal",
        "title": "Proposal Title",
        "description": "Proposal description..."
      },
      "status": "PROPOSAL_STATUS_VOTING_PERIOD",
      "final_tally_result": {
        "yes": "1000000",
        "abstain": "0",
        "no": "0",
        "no_with_veto": "0"
      },
      "submit_time": "2025-11-01T00:00:00Z",
      "deposit_end_time": "2025-11-03T00:00:00Z",
      "total_deposit": [
        {
          "denom": "upaw",
          "amount": "100000000"
        }
      ],
      "voting_start_time": "2025-11-03T00:00:00Z",
      "voting_end_time": "2025-11-10T00:00:00Z"
    }
  ]
}
```

### Get Proposal

```http
GET /cosmos/gov/v1beta1/proposals/{proposalId}
```

### Submit Proposal

```http
POST /cosmos/gov/v1beta1/proposals
```

### Vote

```http
POST /cosmos/gov/v1beta1/proposals/{proposalId}/votes
```

**Request:**
```json
{
  "voter": "paw1abc...xyz",
  "option": "VOTE_OPTION_YES"
}
```

**Vote Options:**
- `VOTE_OPTION_YES`
- `VOTE_OPTION_NO`
- `VOTE_OPTION_ABSTAIN`
- `VOTE_OPTION_NO_WITH_VETO`

### Deposit

```http
POST /cosmos/gov/v1beta1/proposals/{proposalId}/deposits
```

## Tendermint RPC

### Get Node Info

```http
GET /cosmos/base/tendermint/v1beta1/node_info
```

### Get Latest Block

```http
GET /cosmos/base/tendermint/v1beta1/blocks/latest
```

### Get Block by Height

```http
GET /cosmos/base/tendermint/v1beta1/blocks/{height}
```

### Get Validators

```http
GET /cosmos/base/tendermint/v1beta1/validatorsets/latest
```

## Transaction Queries

### Get Transaction

```http
GET /cosmos/tx/v1beta1/txs/{hash}
```

### Search Transactions

```http
GET /cosmos/tx/v1beta1/txs?events=message.action='send'
```

**Query Parameters:**
- `events`: Event query string
- `pagination.limit`: Number of results
- `pagination.offset`: Offset for pagination
- `order_by`: ORDER_BY_ASC or ORDER_BY_DESC

### Simulate Transaction

```http
POST /cosmos/tx/v1beta1/simulate
```

**Use Case:** Estimate gas before broadcasting

### Broadcast Transaction

```http
POST /cosmos/tx/v1beta1/txs
```

**Modes:**
- `BROADCAST_MODE_SYNC`: Return after CheckTx
- `BROADCAST_MODE_ASYNC`: Return immediately
- `BROADCAST_MODE_BLOCK`: Wait for block inclusion (deprecated)

## WebSocket Subscriptions

### Connect

```javascript
const ws = new WebSocket('wss://ws.pawchain.io');
```

### Subscribe to New Blocks

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "id": 1,
  "params": {
    "query": "tm.event='NewBlock'"
  }
}
```

### Subscribe to Transactions

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "id": 2,
  "params": {
    "query": "tm.event='Tx'"
  }
}
```

### Subscribe to Specific Events

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "id": 3,
  "params": {
    "query": "message.action='send' AND transfer.recipient='paw1abc...xyz'"
  }
}
```

### Unsubscribe

```json
{
  "jsonrpc": "2.0",
  "method": "unsubscribe",
  "id": 1,
  "params": {
    "query": "tm.event='NewBlock'"
  }
}
```

## Error Codes

| Code | Description | Solution |
|------|-------------|----------|
| 400 | Bad Request | Check request format |
| 404 | Not Found | Verify resource exists |
| 429 | Too Many Requests | Reduce request rate |
| 500 | Internal Server Error | Retry or contact support |
| 503 | Service Unavailable | Service temporarily down |

### Common Error Responses

```json
{
  "code": 5,
  "message": "insufficient funds",
  "details": []
}
```

## SDK Code Examples

### JavaScript

```javascript
import { PAWClient } from '@paw-chain/sdk';

const client = new PAWClient({
  rpcEndpoint: 'https://rpc.pawchain.io',
  chainId: 'paw-1'
});

// Get balance
const balance = await client.bank.getBalance(address, 'upaw');

// Send transaction
const result = await client.bank.send({
  from: senderAddress,
  to: recipientAddress,
  amount: '1000000upaw'
});
```

### Python

```python
from paw import PAWClient

client = PAWClient(
    rpc_endpoint='https://rpc.pawchain.io',
    chain_id='paw-1'
)

# Get balance
balance = await client.bank.get_balance(address, 'upaw')

# Send transaction
result = await client.bank.send(
    from_address=sender,
    to_address=recipient,
    amount='1000000upaw'
)
```

### Go

```go
import "github.com/paw-chain/paw/sdk/client"

c := client.NewPAWClient("https://rpc.pawchain.io")

// Get balance
balance, err := c.Bank.GetBalance(ctx, address, "upaw")

// Send transaction
result, err := c.Bank.Send(ctx, &client.SendRequest{
    From:   sender,
    To:     recipient,
    Amount: "1000000upaw",
})
```

## Best Practices

1. **Caching**: Cache responses for frequently accessed data
2. **Pagination**: Use pagination for large result sets
3. **Error Handling**: Implement retry logic with exponential backoff
4. **Rate Limiting**: Respect rate limits, implement client-side limiting
5. **WebSockets**: Use WebSockets for real-time data instead of polling
6. **Gas Estimation**: Simulate transactions before broadcasting
7. **Security**: Never expose private keys, use environment variables

## Support

- **Documentation**: [docs.pawchain.io](https://docs.pawchain.io)
- **Discord**: [discord.gg/DBHTc2QV](https://discord.gg/DBHTc2QV)
- ****: [github.com/paw-chain/paw](https://github.com/paw-chain/paw)
- **Status**: [status.pawchain.io](https://status.pawchain.io)
