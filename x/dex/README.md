# DEX Module

## Overview

The DEX (Decentralized Exchange) module implements a production-grade automated market maker (AMM) with constant product formula (x * y = k), limit orders, and advanced trading features. It enables trustless token swaps, liquidity provision, and cross-chain trading via IBC.

## Concepts

### Automated Market Maker (AMM)

The DEX uses a constant product market maker model:

```
x * y = k
```

Where:
- `x` = reserve of token X in the pool
- `y` = reserve of token Y in the pool
- `k` = constant product (invariant)

### Liquidity Pools

Each pool represents a trading pair and contains:
- **Reserves**: Token balances locked in the pool
- **LP Tokens**: Represent ownership share in the pool
- **Fee Structure**: Configurable swap fees distributed to LPs
- **Pool Parameters**: Slippage protection, drain limits, etc.

### Limit Orders

Advanced order book functionality:
- **Maker Orders**: Resting orders at specific prices
- **Taker Orders**: Immediate execution against existing orders
- **Order Matching**: Automatic execution when prices match
- **Partial Fills**: Orders can be filled incrementally
- **Time-in-Force**: GTC (Good-Til-Cancelled), IOC (Immediate-Or-Cancel)

### Fee Structure

Three-tier fee distribution:
- **Swap Fee** (default 0.3%): Total fee charged on swaps
- **LP Fee** (default 0.25%): Goes to liquidity providers
- **Protocol Fee** (default 0.05%): Goes to protocol treasury

### Time-Weighted Average Price (TWAP)

Oracle integration for manipulation-resistant pricing:
- Accumulates price over time
- Resistant to single-block manipulation
- Configurable lookback window
- Used for security checks and external integrations

### Security Features

#### Flash Loan Protection
- Tracks intra-block state changes
- Prevents same-block manipulation
- Configurable protection window (default: 10 blocks)

#### Slippage Protection
- Maximum slippage percent (default: 5%)
- Per-transaction slippage limits
- Automatic transaction reversion on excess slippage

#### Pool Drain Protection
- Maximum drain percent (default: 30%)
- Prevents single-transaction pool depletion
- Maintains pool health and stability

### IBC Integration

Cross-chain DEX functionality:
- **Packet Types**: Swap requests, liquidity operations
- **Authorized Channels**: Whitelist of allowed IBC channels
- **Timeout Handling**: Automatic refunds on timeout
- **Channel Security**: Only authorized channels can execute trades

## State

The module stores the following data:

### Pools
- **Key**: `pools/{poolId}`
- **Value**: Pool struct
- **Description**: All active liquidity pools

```go
type Pool struct {
    Id           uint64
    TokenX       string
    TokenY       string
    ReserveX     math.Int
    ReserveY     math.Int
    TotalShares  math.Int
    SwapFee      math.LegacyDec
    CreatedAt    int64
}
```

### Limit Orders
- **Key**: `limit_orders/{orderId}`
- **Value**: LimitOrder struct
- **Description**: Active limit orders in the order book

```go
type LimitOrder struct {
    OrderId      uint64
    Creator      string
    PoolId       uint64
    Direction    OrderDirection  // BUY or SELL
    Price        math.LegacyDec
    Quantity     math.Int
    FilledQty    math.Int
    Status       OrderStatus
    CreatedAt    int64
    ExpiresAt    int64
}
```

### TWAP Accumulators
- **Key**: `twap/{poolId}/{blockHeight}`
- **Value**: TWAPAccumulator struct
- **Description**: Time-weighted price accumulation data

```go
type TWAPAccumulator struct {
    PoolId          uint64
    PriceAccumulator math.LegacyDec
    Timestamp       int64
    BlockHeight     int64
}
```

### Liquidity Provider Shares
- **Key**: `lp_shares/{poolId}/{address}`
- **Value**: math.Int (share amount)
- **Description**: LP token balances per user per pool

### Nonce Tracker
- **Key**: `nonces/{address}`
- **Value**: uint64
- **Description**: Replay protection nonces for each user

### Parameters
- **Key**: `params`
- **Value**: Params struct
- **Description**: Module configuration parameters

```go
type Params struct {
    SwapFee                   math.LegacyDec  // 0.3%
    LpFee                     math.LegacyDec  // 0.25%
    ProtocolFee               math.LegacyDec  // 0.05%
    MinLiquidity              math.Int        // 1000
    MaxSlippagePercent        math.LegacyDec  // 5%
    MaxPoolDrainPercent       math.LegacyDec  // 30%
    FlashLoanProtectionBlocks uint64          // 10
    PoolCreationGas           uint64          // 1000
    SwapValidationGas         uint64          // 1500
    LiquidityGas              uint64          // 1200
    AuthorizedChannels        []AuthorizedChannel
}
```

## Messages

### MsgCreatePool

Create a new liquidity pool.

**CLI Command:**
```bash
pawd tx dex create-pool \
  --token-x upaw \
  --token-y uatom \
  --amount-x 1000000 \
  --amount-y 1000000 \
  --swap-fee 0.003 \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Both tokens must be different
- Initial liquidity must meet minimum (default: 1000)
- Swap fee must be within allowed range
- Token denoms must be valid

### MsgAddLiquidity

Add liquidity to an existing pool.

**CLI Command:**
```bash
pawd tx dex add-liquidity \
  --pool-id 1 \
  --amount-x 100000 \
  --amount-y 100000 \
  --min-shares 95000 \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Pool must exist
- Amounts must maintain current pool ratio (within slippage tolerance)
- User must have sufficient balances
- Min shares protects against front-running

### MsgRemoveLiquidity

Remove liquidity from a pool.

**CLI Command:**
```bash
pawd tx dex remove-liquidity \
  --pool-id 1 \
  --shares 50000 \
  --min-amount-x 40000 \
  --min-amount-y 40000 \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- User must own sufficient LP shares
- Min amounts protect against price manipulation
- Pool must have sufficient reserves

### MsgSwap

Execute a token swap.

**CLI Command:**
```bash
pawd tx dex swap \
  --pool-id 1 \
  --token-in 100000upaw \
  --min-token-out 95000uatom \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Pool must exist and have liquidity
- User must have sufficient input tokens
- Output must meet minimum (slippage protection)
- Swap must not exceed max pool drain percent

### MsgCreateLimitOrder

Create a limit order.

**CLI Command:**
```bash
pawd tx dex create-limit-order \
  --pool-id 1 \
  --direction buy \
  --price 1.05 \
  --quantity 100000 \
  --time-in-force gtc \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Pool must exist
- Price must be positive
- Quantity must be positive
- User must have sufficient funds (escrowed immediately)

### MsgCancelLimitOrder

Cancel an active limit order.

**CLI Command:**
```bash
pawd tx dex cancel-limit-order \
  --order-id 42 \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Order must exist and be active
- Only order creator can cancel
- Unfilled funds are returned immediately

## Queries

### Query Pool

Get pool details.

```bash
pawd query dex pool 1
```

**Response:**
```json
{
  "pool": {
    "id": "1",
    "token_x": "upaw",
    "token_y": "uatom",
    "reserve_x": "1000000000",
    "reserve_y": "500000000",
    "total_shares": "707106781",
    "swap_fee": "0.003000000000000000",
    "created_at": "1234567890"
  }
}
```

### Query Pools

List all pools with pagination.

```bash
pawd query dex pools --page 1 --limit 10
```

### Query Estimate Swap

Estimate swap output (no state changes).

```bash
pawd query dex estimate-swap 1 100000upaw
```

**Response:**
```json
{
  "token_out": "99000uatom",
  "price_impact": "0.010000000000000000",
  "swap_fee": "300upaw",
  "lp_fee": "250upaw",
  "protocol_fee": "50upaw"
}
```

### Query Limit Order

Get order details.

```bash
pawd query dex limit-order 42
```

### Query Limit Orders

List orders by pool or user.

```bash
# By pool
pawd query dex limit-orders-by-pool 1

# By user
pawd query dex limit-orders-by-user paw1abc...xyz
```

### Query TWAP

Get time-weighted average price.

```bash
pawd query dex twap 1 --window 1000
```

### Query User Liquidity

Get user's LP positions.

```bash
pawd query dex user-liquidity paw1abc...xyz
```

### Query Params

Get module parameters.

```bash
pawd query dex params
```

## Events

### EventPoolCreated
Emitted when a new pool is created.

**Attributes:**
- `pool_id`: Unique pool identifier
- `creator`: Pool creator address
- `token_x`: First token denom
- `token_y`: Second token denom
- `initial_reserves`: Initial liquidity amounts

### EventLiquidityAdded
Emitted when liquidity is added to a pool.

**Attributes:**
- `pool_id`: Pool identifier
- `provider`: LP address
- `amount_x`: Amount of token X added
- `amount_y`: Amount of token Y added
- `shares_issued`: LP tokens minted

### EventLiquidityRemoved
Emitted when liquidity is removed from a pool.

**Attributes:**
- `pool_id`: Pool identifier
- `provider`: LP address
- `amount_x`: Amount of token X returned
- `amount_y`: Amount of token Y returned
- `shares_burned`: LP tokens burned

### EventSwap
Emitted when a swap is executed.

**Attributes:**
- `pool_id`: Pool identifier
- `trader`: Trader address
- `token_in`: Input token and amount
- `token_out`: Output token and amount
- `swap_fee`: Total fee charged
- `price_impact`: Percentage price impact

### EventLimitOrderCreated
Emitted when a limit order is created.

**Attributes:**
- `order_id`: Unique order identifier
- `creator`: Order creator address
- `pool_id`: Pool identifier
- `direction`: Buy or sell
- `price`: Order price
- `quantity`: Order quantity

### EventLimitOrderFilled
Emitted when a limit order is filled (fully or partially).

**Attributes:**
- `order_id`: Order identifier
- `filled_quantity`: Amount filled in this execution
- `remaining_quantity`: Amount still unfilled
- `execution_price`: Price at which order was filled

### EventLimitOrderCancelled
Emitted when a limit order is cancelled.

**Attributes:**
- `order_id`: Order identifier
- `creator`: Order creator address
- `refunded_amount`: Amount returned to creator

## Parameters

Module parameters can be updated via governance.

### Governance Update Example

```bash
# Create parameter change proposal
pawd tx gov submit-proposal param-change proposal.json \
  --from validator \
  --chain-id paw-1

# proposal.json
{
  "title": "Update DEX Swap Fee",
  "description": "Reduce swap fee to 0.25%",
  "changes": [
    {
      "subspace": "dex",
      "key": "SwapFee",
      "value": "0.0025"
    }
  ],
  "deposit": "10000000upaw"
}
```

## Security Considerations

### Flash Loan Protection
The module tracks same-block operations and prevents manipulation:
- Swaps in the same block are monitored
- Large price movements trigger protection
- Configurable protection window

### Slippage Protection
All trades include slippage parameters:
- `min_token_out` for swaps
- `min_shares` for liquidity additions
- Automatic reversion on excess slippage

### Pool Drain Protection
Maximum single-transaction drain is limited:
- Default: 30% of pool reserves
- Prevents pool depletion attacks
- Maintains pool stability

### Access Control
- Only authorized IBC channels can execute cross-chain trades
- Governance controls parameter updates
- User authorization for all user-specific operations

## Integration Examples

### JavaScript/TypeScript
```typescript
import { SigningStargateClient } from "@cosmjs/stargate";

const client = await SigningStargateClient.connectWithSigner(
  "https://rpc.paw.network",
  signer
);

// Create a swap transaction
const msg = {
  typeUrl: "/paw.dex.v1.MsgSwap",
  value: {
    creator: "paw1abc...xyz",
    poolId: 1,
    tokenIn: { denom: "upaw", amount: "100000" },
    minTokenOut: "95000"
  }
};

const result = await client.signAndBroadcast(address, [msg], "auto");
```

### Python
```python
from cosmospy import BroadcastMode, Transaction

# Create swap message
swap_msg = {
    "type": "paw/dex/MsgSwap",
    "value": {
        "creator": "paw1abc...xyz",
        "pool_id": "1",
        "token_in": {"denom": "upaw", "amount": "100000"},
        "min_token_out": "95000"
    }
}

# Sign and broadcast
tx = Transaction(...)
result = tx.broadcast(mode=BroadcastMode.SYNC)
```

### Go
```go
import (
    dextypes "github.com/paw-chain/paw/x/dex/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// Create swap message
msg := &dextypes.MsgSwap{
    Creator:     "paw1abc...xyz",
    PoolId:      1,
    TokenIn:     sdk.NewCoin("upaw", sdk.NewInt(100000)),
    MinTokenOut: sdk.NewInt(95000),
}

// Broadcast via client...
```

## Testing

### Unit Tests
```bash
# Run all DEX module tests
go test ./x/dex/...

# Run with coverage
go test -cover ./x/dex/...

# Run specific test
go test ./x/dex/keeper -run TestSwap
```

### Integration Tests
```bash
# Run integration test suite
go test ./x/dex/keeper -run TestIntegration -v
```

### Simulation Tests
```bash
# Run simulation tests
go test ./x/dex/simulation -run TestSimulation -v
```

## Monitoring

### Key Metrics
- Total Value Locked (TVL) per pool
- 24h trading volume
- Number of active pools
- Number of active limit orders
- Average swap slippage
- Fee revenue (LP + protocol)

### Prometheus Metrics
The module exposes Prometheus metrics:
- `dex_pool_count`: Total number of pools
- `dex_swap_count`: Total swaps executed
- `dex_total_volume`: Cumulative trading volume
- `dex_pool_liquidity{pool_id}`: Liquidity per pool
- `dex_order_count`: Active limit orders

## Future Enhancements

### Planned Features
- Concentrated liquidity (Uniswap v3 style)
- Multi-hop routing for optimal swap paths
- Liquidity mining rewards
- Governance token integration
- Advanced order types (stop-loss, take-profit)
- Cross-chain liquidity aggregation

### Research Areas
- Just-in-time liquidity
- Dynamic fee adjustment based on volatility
- MEV protection mechanisms
- Impermanent loss hedging

## References

- [Uniswap V2 Whitepaper](https://uniswap.org/whitepaper.pdf)
- [Constant Product Market Maker](https://github.com/runtimeverification/verified-smart-contracts/blob/uniswap/uniswap/x-y-k.pdf)
- [Cosmos SDK Bank Module](https://docs.cosmos.network/main/modules/bank)
- [IBC Protocol](https://github.com/cosmos/ibc)

---

**Module Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-06
