# cURL API Examples

Complete cURL examples for interacting with the PAW Blockchain API.

## Table of Contents
- [Setup](#setup)
- [DEX Module](#dex-module)
- [Oracle Module](#oracle-module)
- [Compute Module](#compute-module)
- [Bank Module](#bank-module)
- [Staking Module](#staking-module)
- [Governance Module](#governance-module)

## Setup

```bash
# Set your API endpoint
export API_URL="http://localhost:1317"
# Or for testnet
# export API_URL="https://testnet-api.paw.network"

# Set your wallet address
export MY_ADDRESS="paw1abc123..."
```

## DEX Module

### List All Pools

```bash
curl -X GET "$API_URL/paw/dex/v1/pools" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Pool

```bash
curl -X GET "$API_URL/paw/dex/v1/pools/1" \
  -H "Content-Type: application/json" | jq
```

### Get Pool Price

```bash
curl -X GET "$API_URL/paw/dex/v1/pools/1/price" \
  -H "Content-Type: application/json" | jq
```

### Estimate Swap Output

```bash
curl -X POST "$API_URL/paw/dex/v1/estimate_swap" \
  -H "Content-Type: application/json" \
  -d '{
    "pool_id": 1,
    "token_in": "uapaw",
    "amount_in": "1000000"
  }' | jq
```

### Create Pool (Requires Signed Transaction)

```bash
# First, create the transaction
curl -X POST "$API_URL/paw/dex/v1/create_pool" \
  -H "Content-Type: application/json" \
  -d '{
    "creator": "paw1abc123...",
    "token_a": "uapaw",
    "token_b": "ubtc",
    "amount_a": "1000000000",
    "amount_b": "500000000"
  }' | jq
```

### Swap Tokens

```bash
curl -X POST "$API_URL/paw/dex/v1/swap" \
  -H "Content-Type: application/json" \
  -d '{
    "sender": "paw1abc123...",
    "pool_id": 1,
    "token_in": "uapaw",
    "amount_in": "1000000",
    "min_amount_out": "450000"
  }' | jq
```

### Add Liquidity

```bash
curl -X POST "$API_URL/paw/dex/v1/add_liquidity" \
  -H "Content-Type: application/json" \
  -d '{
    "sender": "paw1abc123...",
    "pool_id": 1,
    "amount_a": "1000000",
    "amount_b": "500000",
    "min_shares": "700000"
  }' | jq
```

### Remove Liquidity

```bash
curl -X POST "$API_URL/paw/dex/v1/remove_liquidity" \
  -H "Content-Type: application/json" \
  -d '{
    "sender": "paw1abc123...",
    "pool_id": 1,
    "shares": "700000",
    "min_amount_a": "900000",
    "min_amount_b": "450000"
  }' | jq
```

## Oracle Module

### List All Price Feeds

```bash
curl -X GET "$API_URL/paw/oracle/v1/prices" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Price Feed

```bash
curl -X GET "$API_URL/paw/oracle/v1/prices/BTC%2FUSD" \
  -H "Content-Type: application/json" | jq
```

### Get Oracle Parameters

```bash
curl -X GET "$API_URL/paw/oracle/v1/params" \
  -H "Content-Type: application/json" | jq
```

### Submit Price (Validator Only)

```bash
curl -X POST "$API_URL/paw/oracle/v1/submit_price" \
  -H "Content-Type: application/json" \
  -d '{
    "validator": "pawvaloper1abc...",
    "asset": "BTC/USD",
    "price": "45000.50"
  }' | jq
```

## Compute Module

### List All Tasks

```bash
curl -X GET "$API_URL/paw/compute/v1/tasks" \
  -H "Content-Type: application/json" | jq
```

### List Tasks by Status

```bash
curl -X GET "$API_URL/paw/compute/v1/tasks?status=completed" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Task

```bash
curl -X GET "$API_URL/paw/compute/v1/tasks/1" \
  -H "Content-Type: application/json" | jq
```

### List Compute Providers

```bash
curl -X GET "$API_URL/paw/compute/v1/providers" \
  -H "Content-Type: application/json" | jq
```

### Submit Compute Task

```bash
curl -X POST "$API_URL/paw/compute/v1/submit_task" \
  -H "Content-Type: application/json" \
  -d '{
    "requester": "paw1abc123...",
    "task_type": "api_call",
    "task_data": {
      "url": "https://api.github.com/data",
      "method": "GET"
    },
    "fee": {
      "denom": "uapaw",
      "amount": "100000"
    }
  }' | jq
```

## Bank Module

### Get Account Balance

```bash
curl -X GET "$API_URL/cosmos/bank/v1beta1/balances/$MY_ADDRESS" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Denomination Balance

```bash
curl -X GET "$API_URL/cosmos/bank/v1beta1/balances/$MY_ADDRESS/by_denom?denom=uapaw" \
  -H "Content-Type: application/json" | jq
```

### Get Total Supply

```bash
curl -X GET "$API_URL/cosmos/bank/v1beta1/supply" \
  -H "Content-Type: application/json" | jq
```

### Get Supply of Specific Denomination

```bash
curl -X GET "$API_URL/cosmos/bank/v1beta1/supply/by_denom?denom=uapaw" \
  -H "Content-Type: application/json" | jq
```

### Send Tokens

```bash
curl -X POST "$API_URL/cosmos/bank/v1beta1/send" \
  -H "Content-Type: application/json" \
  -d '{
    "from_address": "paw1abc123...",
    "to_address": "paw1def456...",
    "amount": [
      {
        "denom": "uapaw",
        "amount": "1000000"
      }
    ]
  }' | jq
```

## Staking Module

### List All Validators

```bash
curl -X GET "$API_URL/cosmos/staking/v1beta1/validators" \
  -H "Content-Type: application/json" | jq
```

### List Bonded Validators Only

```bash
curl -X GET "$API_URL/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Validator

```bash
export VALIDATOR_ADDR="pawvaloper1abc..."
curl -X GET "$API_URL/cosmos/staking/v1beta1/validators/$VALIDATOR_ADDR" \
  -H "Content-Type: application/json" | jq
```

### Get Delegations for Address

```bash
curl -X GET "$API_URL/cosmos/staking/v1beta1/delegations/$MY_ADDRESS" \
  -H "Content-Type: application/json" | jq
```

### Get Unbonding Delegations

```bash
curl -X GET "$API_URL/cosmos/staking/v1beta1/delegators/$MY_ADDRESS/unbonding_delegations" \
  -H "Content-Type: application/json" | jq
```

### Get Staking Pool

```bash
curl -X GET "$API_URL/cosmos/staking/v1beta1/pool" \
  -H "Content-Type: application/json" | jq
```

### Delegate Tokens

```bash
curl -X POST "$API_URL/cosmos/staking/v1beta1/delegate" \
  -H "Content-Type: application/json" \
  -d '{
    "delegator_address": "paw1abc123...",
    "validator_address": "pawvaloper1abc...",
    "amount": {
      "denom": "uapaw",
      "amount": "1000000"
    }
  }' | jq
```

## Governance Module

### List All Proposals

```bash
curl -X GET "$API_URL/cosmos/gov/v1beta1/proposals" \
  -H "Content-Type: application/json" | jq
```

### List Active Proposals

```bash
curl -X GET "$API_URL/cosmos/gov/v1beta1/proposals?proposal_status=2" \
  -H "Content-Type: application/json" | jq
```

### Get Specific Proposal

```bash
curl -X GET "$API_URL/cosmos/gov/v1beta1/proposals/1" \
  -H "Content-Type: application/json" | jq
```

### Get Proposal Votes

```bash
curl -X GET "$API_URL/cosmos/gov/v1beta1/proposals/1/votes" \
  -H "Content-Type: application/json" | jq
```

### Get Proposal Tally

```bash
curl -X GET "$API_URL/cosmos/gov/v1beta1/proposals/1/tally" \
  -H "Content-Type: application/json" | jq
```

### Get Governance Parameters

```bash
# Deposit params
curl -X GET "$API_URL/cosmos/gov/v1beta1/params/deposit" \
  -H "Content-Type: application/json" | jq

# Voting params
curl -X GET "$API_URL/cosmos/gov/v1beta1/params/voting" \
  -H "Content-Type: application/json" | jq

# Tally params
curl -X GET "$API_URL/cosmos/gov/v1beta1/params/tallying" \
  -H "Content-Type: application/json" | jq
```

## Auth Module

### Get Account Information

```bash
curl -X GET "$API_URL/cosmos/auth/v1beta1/accounts/$MY_ADDRESS" \
  -H "Content-Type: application/json" | jq
```

### Get Auth Parameters

```bash
curl -X GET "$API_URL/cosmos/auth/v1beta1/params" \
  -H "Content-Type: application/json" | jq
```

## Tendermint RPC

### Get Node Status

```bash
curl -X GET "$API_URL/status" \
  -H "Content-Type: application/json" | jq
```

### Health Check

```bash
curl -X GET "$API_URL/health" \
  -H "Content-Type: application/json" | jq
```

### Get Latest Block

```bash
curl -X GET "$API_URL/block" \
  -H "Content-Type: application/json" | jq
```

### Get Block by Height

```bash
curl -X GET "$API_URL/block?height=100" \
  -H "Content-Type: application/json" | jq
```

### Query Transaction

```bash
export TX_HASH="ABC123..."
curl -X GET "$API_URL/tx?hash=0x$TX_HASH" \
  -H "Content-Type: application/json" | jq
```

### Get Validators

```bash
curl -X GET "$API_URL/validators" \
  -H "Content-Type: application/json" | jq
```

## Advanced Examples

### Pagination

```bash
# Get first 10 pools
curl -X GET "$API_URL/paw/dex/v1/pools?pagination.limit=10&pagination.offset=0" | jq

# Get next 10 pools
curl -X GET "$API_URL/paw/dex/v1/pools?pagination.limit=10&pagination.offset=10" | jq
```

### Filtering and Sorting

```bash
# Get pools with specific token
curl -X GET "$API_URL/paw/dex/v1/pools" | jq '.pools[] | select(.token_a == "uapaw")'

# Get only active validators
curl -X GET "$API_URL/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED" | jq
```

### Batch Requests

```bash
# Get multiple pieces of data in parallel
{
  curl -X GET "$API_URL/cosmos/bank/v1beta1/balances/$MY_ADDRESS" &
  curl -X GET "$API_URL/cosmos/staking/v1beta1/delegations/$MY_ADDRESS" &
  curl -X GET "$API_URL/paw/dex/v1/pools" &
  wait
} | jq -s '.'
```

### Error Handling

```bash
# Handle errors gracefully
RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/paw/dex/v1/pools/999")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -eq 200 ]; then
  echo "Success: $BODY" | jq
else
  echo "Error (HTTP $HTTP_CODE): $BODY" | jq
fi
```

## Tips

1. **Use jq for JSON processing**: Install jq to pretty-print and filter JSON responses
2. **Set environment variables**: Define common values like API_URL and MY_ADDRESS
3. **Check HTTP status codes**: Always verify the response status before processing data
4. **Use verbose mode for debugging**: Add `-v` flag to curl for detailed output
5. **Save responses**: Use `-o output.json` to save responses to files

## Common Issues

### CORS Errors
If accessing from a browser, ensure the API server has CORS enabled.

### Rate Limiting
If you hit rate limits, add delays between requests:
```bash
for i in {1..10}; do
  curl "$API_URL/paw/dex/v1/pools/$i" | jq
  sleep 1
done
```

### Authentication
For write operations, you need to sign transactions with your private key. Use the PAW SDK or CLI tools for transaction signing.

## See Also

- [JavaScript Examples](./javascript.md)
- [Python Examples](./python.md)
- [Go Examples](./go.md)
- [Authentication Guide](../guides/authentication.md)
