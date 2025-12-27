# CosmWasm Smart Contract Integration Guide

## Overview

PAW chain supports CosmWasm with native bindings to DEX, Oracle, and Compute modules.

## Deploying Contracts

```bash
# Build and optimize
cargo build --release --target wasm32-unknown-unknown
wasm-opt -Oz target/wasm32-unknown-unknown/release/contract.wasm -o contract.wasm

# Store and instantiate
pawd tx wasm store contract.wasm --from wallet --gas auto --gas-adjustment 1.3
pawd tx wasm instantiate CODE_ID '{"param":"value"}' --label "my-contract" --from wallet
```

## DEX Module Integration

```rust
// Query pool
pub fn query_pool(deps: Deps, pool_id: u64) -> StdResult<Pool> {
    deps.querier.query(&QueryRequest::Custom(PawQuery::Dex(DexQuery::Pool { pool_id })))
}

// Execute swap with slippage protection
pub fn execute_swap(pool_id: u64, token_in: &str, amount: Uint128, min_out: Uint128) -> CosmosMsg {
    CosmosMsg::Custom(PawMsg::Dex(DexMsg::Swap {
        pool_id, token_in: token_in.to_string(), token_out: "upaw".to_string(),
        amount_in: amount, min_amount_out: min_out,
        deadline: env.block.time.plus_seconds(300).seconds() as i64,
    }))
}

// Simulate swap before executing
pub fn simulate_swap(deps: Deps, pool_id: u64, token_in: &str, amount: Uint128) -> StdResult<Uint128> {
    let resp: SimulateSwapResponse = deps.querier.query(&QueryRequest::Custom(
        PawQuery::Dex(DexQuery::SimulateSwap { pool_id, token_in: token_in.to_string(),
            token_out: "upaw".to_string(), amount_in: amount })
    ))?;
    Ok(resp.amount_out)
}
```

## Oracle Module Integration

```rust
// Query price feed
pub fn query_price(deps: Deps, asset: &str) -> StdResult<Decimal> {
    let resp: PriceResponse = deps.querier.query(&QueryRequest::Custom(
        PawQuery::Oracle(OracleQuery::Price { asset: asset.to_string() })
    ))?;
    Ok(resp.price.value)
}

// Check price freshness (prevent stale data attacks)
pub fn check_freshness(price: &Price, max_age: u64, now: u64) -> StdResult<()> {
    if now > price.timestamp + max_age {
        return Err(StdError::generic_err("Stale price"));
    }
    Ok(())
}
```

## Compute Module Integration

```rust
// Submit compute job
pub fn request_compute(specs: ComputeSpec, max_payment: Uint128) -> CosmosMsg {
    CosmosMsg::Custom(PawMsg::Compute(ComputeMsg::SubmitRequest {
        specs, container_image: "myimage:v1".to_string(),
        command: vec!["python".to_string(), "run.py".to_string()],
        env_vars: HashMap::new(), max_payment, preferred_provider: None,
    }))
}

// Query result
pub fn query_result(deps: Deps, request_id: u64) -> StdResult<ComputeResult> {
    deps.querier.query(&QueryRequest::Custom(PawQuery::Compute(ComputeQuery::Request { request_id })))
}
```

## Gas Estimates

| Operation | Gas |
|-----------|-----|
| Swap | 150k-250k |
| Add Liquidity | 200k-300k |
| Oracle Query | 10k-20k |
| Compute Submit | 100k-150k |

Use `--gas auto --gas-adjustment 1.3` for safety margin.

## Security Best Practices

```rust
// 1. Slippage protection - never use zero min_out
let min_out = expected.multiply_ratio(99u128, 100u128); // 1% slippage

// 2. Reentrancy guard
if state.locked { return Err(StdError::generic_err("Reentrancy")); }
state.locked = true;
// ... operation ...
state.locked = false;

// 3. Input validation
if amount.is_zero() { return Err(StdError::generic_err("Zero amount")); }

// 4. Access control
if sender != config.admin { return Err(StdError::generic_err("Unauthorized")); }

// 5. Oracle staleness check (see above)
```

## Query Endpoints

| Module | Endpoint |
|--------|----------|
| DEX | `/paw/dex/v1/pools/{id}` |
| DEX | `/paw/dex/v1/simulate-swap/{id}` |
| Oracle | `/paw/oracle/v1/price/{asset}` |
| Compute | `/paw/compute/v1/request/{id}` |

## Testing

```bash
cargo test                                    # Local tests
pawd tx wasm store contract.wasm --from test  # Deploy to testnet
pawd query wasm contract-state smart ADDR '{"query":{}}'  # Query state
```
