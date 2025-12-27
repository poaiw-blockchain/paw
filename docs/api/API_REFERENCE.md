# PAW Blockchain API Reference

Complete API documentation for all PAW modules.

## Endpoints

- **gRPC**: `localhost:9090`
- **REST**: `localhost:1317`
- **RPC**: `localhost:26657`

---

## DEX Module (`/paw/dex/v1`)

### Queries

#### Get Pool
```
GET /paw/dex/v1/pools/{pool_id}
```
Returns pool details including reserves, token pair, and total shares.

**Response:**
```json
{
  "pool": {
    "id": "1",
    "token_a": "upaw",
    "token_b": "uatom",
    "reserve_a": "1000000",
    "reserve_b": "500000",
    "total_shares": "707106"
  }
}
```

#### List All Pools
```
GET /paw/dex/v1/pools
```
Returns paginated list of all pools.

#### Get Pool by Tokens
```
GET /paw/dex/v1/pools/by-tokens/{token_a}/{token_b}
```

#### Get User Liquidity
```
GET /paw/dex/v1/liquidity/{pool_id}/{provider}
```

#### Simulate Swap
```
GET /paw/dex/v1/simulate-swap/{pool_id}?token_in=upaw&token_out=uatom&amount_in=1000
```
Simulates swap without execution. Returns expected output.

#### Get Order Book
```
GET /paw/dex/v1/order-book/{pool_id}?limit=50
```
Returns buy and sell orders for a pool.

### Transactions

#### Create Pool
```bash
pawd tx dex create-pool upaw uatom 1000000 500000 --from alice
```

#### Swap Tokens
```bash
pawd tx dex swap 1 upaw uatom 1000 900 --from alice
```
Parameters: pool_id, token_in, token_out, amount_in, min_amount_out

#### Add Liquidity
```bash
pawd tx dex add-liquidity 1 1000000 500000 --from alice
```

#### Remove Liquidity
```bash
pawd tx dex remove-liquidity 1 100000 --from alice
```
Parameters: pool_id, shares_to_remove

#### Place Limit Order
```bash
pawd tx dex place-limit-order 1 buy upaw 1000 "0.5" --from alice
```

#### Cancel Limit Order
```bash
pawd tx dex cancel-limit-order 42 --from alice
```

---

## Compute Module (`/paw/compute/v1`)

### Queries

#### Get Provider
```
GET /paw/compute/v1/providers/{address}
```

#### List Providers
```
GET /paw/compute/v1/providers
```

#### Get Request
```
GET /paw/compute/v1/requests/{request_id}
```

#### Get Result
```
GET /paw/compute/v1/results/{request_id}
```

### Transactions

#### Register Provider
```bash
pawd tx compute register-provider \
  --moniker "My Provider" \
  --endpoint "https://compute.example.com" \
  --stake 1000000upaw \
  --from provider
```

#### Submit Compute Request
```bash
pawd tx compute submit-request \
  --image "docker.io/library/python:3.11" \
  --command "python,script.py" \
  --max-payment 100000upaw \
  --from alice
```

#### Submit Result (Provider)
```bash
pawd tx compute submit-result \
  --request-id 1 \
  --output-hash "abc123..." \
  --output-url "https://storage.example.com/result" \
  --from provider
```

#### Register Signing Key
```bash
pawd tx compute register-signing-key \
  --pubkey "base64_encoded_ed25519_pubkey" \
  --from provider
```

---

## Oracle Module (`/paw/oracle/v1`)

### Queries

#### Get Price
```
GET /paw/oracle/v1/prices/{asset}
```

#### Get All Prices
```
GET /paw/oracle/v1/prices
```

#### Get Validator Prices
```
GET /paw/oracle/v1/validator-prices/{validator}
```

### Transactions

#### Submit Price (Validator)
```bash
pawd tx oracle submit-price upaw "1.5" --from validator
```

---

## Common Response Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Internal error |
| 2 | Invalid request |
| 3 | Unauthorized |
| 4 | Not found |
| 5 | Already exists |

## Pagination

All list endpoints support pagination:
```
?pagination.limit=100
?pagination.offset=0
?pagination.count_total=true
```

## gRPC Examples

```go
import "github.com/paw-chain/paw/x/dex/types"

conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
client := types.NewQueryClient(conn)

resp, _ := client.Pool(ctx, &types.QueryPoolRequest{PoolId: 1})
fmt.Println(resp.Pool)
```

## Events

### DEX Events
- `dex_pool_created`: New pool created
- `dex_swap_executed`: Swap completed
- `dex_liquidity_added`: Liquidity deposited
- `dex_liquidity_removed`: Liquidity withdrawn
- `dex_limit_order_placed`: Limit order created
- `dex_limit_order_cancelled`: Limit order cancelled

### Compute Events
- `compute_provider_registered`: Provider registered
- `compute_request_submitted`: New compute request
- `compute_result_submitted`: Result submitted
- `compute_dispute_created`: Dispute filed

### Oracle Events
- `oracle_price_submitted`: Validator submitted price
- `oracle_price_aggregated`: Price finalized
