# Developer Guide

Complete guide for building on PAW Blockchain.

## Development Environment Setup

### Prerequisites

```bash
# Required
- Node.js 18+ and npm 9+
- Go 1.21+
-  2.40+
- Make

# Optional but recommended
- Docker 24+
- VS Code or your preferred IDE
- Postman or similar API testing tool
```

### Install PAW CLI

```bash
# Clone the repository
 clone https://github.com/paw-chain/paw
cd paw

# Install dependencies
make install-deps

# Build the binary
make install

# Verify installation
pawd version
# Output: v1.0.0

# Initialize local node (optional)
pawd init mynode --chain-id paw-local
```

### Environment Configuration

```bash
# .env file
CHAIN_ID=paw-1
RPC_ENDPOINT=https://rpc.pawchain.io
API_ENDPOINT=https://api.pawchain.io
WS_ENDPOINT=wss://ws.pawchain.io

# For testnet development
CHAIN_ID=paw-testnet-1
RPC_ENDPOINT=https://testnet-rpc.pawchain.io
API_ENDPOINT=https://testnet-api.pawchain.io
```

## SDK Usage

PAW provides official SDKs in multiple languages.

### JavaScript/TypeScript SDK

#### Installation

```bash
npm install @paw-chain/sdk
# or
yarn add @paw-chain/sdk
```

#### Basic Usage

```javascript
import { PAWClient, Wallet } from '@paw-chain/sdk';

// Connect to PAW network
const client = new PAWClient({
  rpcEndpoint: 'https://rpc.pawchain.io',
  chainId: 'paw-1',
  prefix: 'paw'
});

// Create or import wallet
const wallet = await Wallet.generate(); // Create new
// or
const wallet = await Wallet.fromMnemonic('your 24 words...'); // Import

console.log('Address:', wallet.address);
// paw1abc...xyz
```

#### Sending Transactions

```javascript
// Send tokens
const result = await client.bank.send({
  from: wallet.address,
  to: 'paw1recipient...',
  amount: '1000000upaw', // 1 PAW
  memo: 'Payment for services'
});

console.log('Transaction Hash:', result.transactionHash);
console.log('Block Height:', result.height);
```

#### Querying Data

```javascript
// Get balance
const balance = await client.bank.getBalance(
  wallet.address,
  'upaw'
);
console.log('Balance:', balance.amount);

// Get transaction
const tx = await client.tx.getTx(txHash);
console.log('Transaction:', tx);
```

#### DEX Operations

```javascript
// Swap tokens
const swap = await client.dex.swap({
  sender: wallet.address,
  poolId: '1',
  tokenIn: '100000upaw',
  tokenOutMinAmount: '95000uusdc',
  slippage: 0.5 // 0.5%
});

// Add liquidity
const liquidity = await client.dex.addLiquidity({
  sender: wallet.address,
  poolId: '1',
  assetsIn: [
    { denom: 'upaw', amount: '1000000' },
    { denom: 'uusdc', amount: '1000000' }
  ]
});

// Get pool info
const pool = await client.dex.getPool('1');
console.log('Pool:', pool);
```

#### Staking Operations

```javascript
// Delegate tokens
const delegate = await client.staking.delegate({
  delegator: wallet.address,
  validator: 'pawvaloper1...',
  amount: '1000000upaw'
});

// Get delegations
const delegations = await client.staking.getDelegations(wallet.address);

// Claim rewards
const rewards = await client.distribution.withdrawRewards({
  delegator: wallet.address,
  validator: 'pawvaloper1...'
});
```

### Python SDK

#### Installation

```bash
pip install paw-sdk
```

#### Basic Usage

```python
from paw import PAWClient, Wallet
import asyncio

async def main():
    # Connect to network
    client = PAWClient(
        rpc_endpoint='https://rpc.pawchain.io',
        chain_id='paw-1'
    )

    # Create wallet
    wallet = await Wallet.generate()
    print(f'Address: {wallet.address}')

    # Send transaction
    result = await client.bank.send(
        from_address=wallet.address,
        to_address='paw1recipient...',
        amount='1000000upaw'
    )

    print(f'TX Hash: {result.tx_hash}')

asyncio.run(main())
```

#### DEX Trading

```python
# Swap tokens
swap = await client.dex.swap(
    sender=wallet.address,
    pool_id='1',
    token_in='100000upaw',
    token_out_min='95000uusdc',
    slippage=0.5
)

# Query pool
pool = await client.dex.get_pool('1')
print(f'Pool Assets: {pool.assets}')
print(f'Pool Shares: {pool.total_shares}')
```

### Go SDK

#### Installation

```bash
go get github.com/paw-chain/paw/sdk/go
```

#### Basic Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/paw-chain/paw/sdk/client"
    "github.com/paw-chain/paw/sdk/wallet"
)

func main() {
    // Create client
    c := client.NewPAWClient("https://rpc.pawchain.io")

    // Create wallet
    w, err := wallet.Generate()
    if err != nil {
        panic(err)
    }

    fmt.Println("Address:", w.Address)

    // Send transaction
    ctx := context.Background()
    result, err := c.Bank.Send(ctx, &client.SendRequest{
        From:   w.Address,
        To:     "paw1recipient...",
        Amount: "1000000upaw",
    })

    if err != nil {
        panic(err)
    }

    fmt.Println("TX Hash:", result.TxHash)
}
```

## Smart Contract Development

PAW supports CosmWasm smart contracts (coming soon).

### Contract Structure

```rust
use cosmwasm_std::{
    entry_point, Binary, Deps, DepsMut, Env,
    MessageInfo, Response, StdResult,
};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    // Initialize contract
    Ok(Response::default())
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    // Handle messages
    match msg {
        ExecuteMsg::Transfer { recipient, amount } => {
            execute_transfer(deps, info, recipient, amount)
        }
    }
}

#[entry_point]
pub fn query(
    deps: Deps,
    env: Env,
    msg: QueryMsg,
) -> StdResult<Binary> {
    // Handle queries
    match msg {
        QueryMsg::Balance { address } => {
            query_balance(deps, address)
        }
    }
}
```

### Deploying Contracts

```bash
# Build contract
cargo wasm

# Optimize
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/rust-optimizer:0.12.13

# Upload to blockchain
pawd tx wasm store artifacts/contract.wasm \
  --from wallet \
  --gas auto \
  --gas-adjustment 1.3

# Instantiate
pawd tx wasm instantiate <code-id> '{}' \
  --from wallet \
  --label "my-contract" \
  --admin <address>
```

## API Integration

### REST API

#### Base URL
```
Mainnet: https://api.pawchain.io
Testnet: https://testnet-api.pawchain.io
```

#### Authentication
Most endpoints are public. For transactions, sign with private key.

#### Get Account Info

```bash
curl https://api.pawchain.io/cosmos/auth/v1beta1/accounts/{address}
```

```json
{
  "account": {
    "@type": "/cosmos.auth.v1beta1.BaseAccount",
    "address": "paw1abc...xyz",
    "pub_key": null,
    "account_number": "42",
    "sequence": "10"
  }
}
```

#### Get Balance

```bash
curl https://api.pawchain.io/cosmos/bank/v1beta1/balances/{address}
```

#### Get Validators

```bash
curl https://api.pawchain.io/cosmos/staking/v1beta1/validators
```

### WebSocket API

```javascript
const ws = new WebSocket('wss://ws.pawchain.io');

ws.on('open', () => {
  // Subscribe to new blocks
  ws.send(JSON.stringify({
    method: 'subscribe',
    params: ["tm.event='NewBlock'"]
  }));
});

ws.on('message', (data) => {
  const event = JSON.parse(data);
  console.log('New block:', event.result.data);
});
```

### GraphQL API (Coming Soon)

```graphql
query {
  account(address: "paw1abc...xyz") {
    balance {
      denom
      amount
    }
    delegations {
      validator
      amount
    }
  }
}
```

## Testing

### Unit Tests

```javascript
// tests/wallet.test.js
import { Wallet } from '@paw-chain/sdk';

describe('Wallet', () => {
  test('generates valid mnemonic', async () => {
    const wallet = await Wallet.generate();
    expect(wallet.mnemonic).toHaveLength(24);
  });

  test('derives correct address', async () => {
    const wallet = await Wallet.fromMnemonic('known mnemonic...');
    expect(wallet.address).toBe('paw1expected...');
  });
});
```

### Integration Tests

```javascript
// tests/integration/bank.test.js
import { PAWClient, Wallet } from '@paw-chain/sdk';

describe('Bank Module Integration', () => {
  let client, wallet;

  beforeAll(async () => {
    client = new PAWClient({
      rpcEndpoint: 'http://localhost:26657',
      chainId: 'paw-local'
    });
    wallet = await Wallet.generate();
  });

  test('sends tokens successfully', async () => {
    const result = await client.bank.send({
      from: wallet.address,
      to: 'paw1recipient...',
      amount: '1000upaw'
    });

    expect(result.code).toBe(0);
    expect(result.transactionHash).toBeDefined();
  });
});
```

### E2E Tests

```javascript
// tests/e2e/dex.test.js
describe('DEX End-to-End', () => {
  test('complete swap workflow', async () => {
    // 1. Get pool info
    const pool = await client.dex.getPool('1');

    // 2. Calculate expected output
    const output = calculateSwapOutput(pool, '100000upaw');

    // 3. Execute swap
    const result = await client.dex.swap({
      sender: wallet.address,
      poolId: '1',
      tokenIn: '100000upaw',
      tokenOutMinAmount: output * 0.99 // 1% slippage
    });

    expect(result.code).toBe(0);

    // 4. Verify balance changed
    const newBalance = await client.bank.getBalance(wallet.address);
    expect(newBalance).toBeGreaterThan(previousBalance);
  });
});
```

## Deployment

### Testnet Deployment

```bash
# Configure testnet
export CHAIN_ID=paw-testnet-1
export NODE=https://testnet-rpc.pawchain.io

# Deploy contract
pawd tx wasm store contract.wasm --from wallet

# Test functionality
pawd query wasm list-code

# Monitor logs
pawd query txs --events 'message.action=store-code'
```

### Mainnet Deployment

```bash
# Review checklist
- Security audit completed ✓
- Testnet thoroughly tested ✓
- Documentation complete ✓
- Community review done ✓

# Deploy to mainnet
export CHAIN_ID=paw-1
export NODE=https://rpc.pawchain.io

pawd tx wasm store contract.wasm \
  --from wallet \
  --gas 5000000 \
  --fees 50000upaw
```

## Best Practices

### Security
- ✅ Never commit private keys or mnemonics
- ✅ Use environment variables for sensitive data
- ✅ Implement rate limiting
- ✅ Validate all inputs
- ✅ Use prepared statements for queries
- ✅ Keep dependencies updated
- ✅ Implement proper error handling

### Performance
- ✅ Cache frequently accessed data
- ✅ Use pagination for large datasets
- ✅ Batch requests when possible
- ✅ Implement connection pooling
- ✅ Monitor gas usage
- ✅ Optimize contract storage

### Code Quality
- ✅ Write comprehensive tests (>80% coverage)
- ✅ Use TypeScript for type safety
- ✅ Follow consistent code style
- ✅ Document all public APIs
- ✅ Use semantic versioning
- ✅ Implement CI/CD pipelines

## Resources

- **API Reference**: [Full API documentation](#api-reference)
- **SDK Documentation**: [Complete SDK guides](#sdk-usage)
- **Examples**: [Code examples repository](https://github.com/paw-chain/paw/tree/master/examples)
- **Discord**: [Developer community](https://discord.gg/DBHTc2QV)
- ****: [Source code](https://github.com/paw-chain/paw)
