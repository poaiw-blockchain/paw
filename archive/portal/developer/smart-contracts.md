# Smart Contracts

Build and deploy smart contracts on PAW using CosmWasm.

## Overview

PAW supports CosmWasm smart contracts written in Rust.

## Prerequisites

- Rust 1.70+
- CosmWasm toolchain
- PAW testnet account

## Setup

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Add wasm target
rustup target add wasm32-unknown-unknown

# Install cargo-generate
cargo install cargo-generate

# Create contract
cargo generate -- https://github.com/CosmWasm/cw-template --name my-contract
```

## Build & Deploy

```bash
# Build contract
cd my-contract
cargo wasm

# Optimize
docker run --rm -v "$(pwd)":/code \
  cosmwasm/rust-optimizer:0.15.0

# Deploy
pawd tx wasm store artifacts/my_contract.wasm \
  --from my-wallet \
  --gas auto \
  --fees 500upaw

# Instantiate
pawd tx wasm instantiate CODE_ID '{"count":0}' \
  --from my-wallet \
  --label "my-contract" \
  --admin my-wallet
```

## Examples

See [CosmWasm Examples](<REPO_URL>-examples/tree/main/contracts)

---

**Previous:** [Go Development](/developer/go-development) | **Next:** [Module Development](/developer/module-development) →
