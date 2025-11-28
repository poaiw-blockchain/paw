# Developer Quick Start

Get started building on PAW Blockchain in minutes.

## Prerequisites

- Node.js 18+ or Python 3.9+ or Go 1.23+
- Basic blockchain knowledge
- PAW testnet account with tokens

## Installation

::: code-group

```bash [JavaScript/TypeScript]
npm install @paw/sdk
# or
yarn add @paw/sdk
```

```bash [Python]
pip install paw-sdk
```

```bash [Go]
go get <MODULE_PATH>/sdk
```

:::

## Quick Examples

### JavaScript SDK

```javascript
import { PAWClient } from '@paw/sdk';

// Initialize client
const client = new PAWClient({
  rpc: 'https://rpc.paw.network',
  chainId: 'paw-testnet-1'
});

// Get account balance
const balance = await client.getBalance('paw1xxxxx...');
console.log(`Balance: ${balance.amount} ${balance.denom}`);

// Send transaction
const result = await client.sendTokens(
  'paw1from...',
  'paw1to...',
  '1000000upaw',
  { fees: '500upaw' }
);
console.log(`TX Hash: ${result.transactionHash}`);

// Query DEX
const pools = await client.dex.getPools();
console.log(`Available pools: ${pools.length}`);

// Swap tokens
const swap = await client.dex.swap({
  poolId: 1,
  tokenIn: '1000000upaw',
  minTokenOut: '950000uusdc',
  slippage: 0.01
});
```

### Python SDK

```python
from paw_sdk import PAWClient, Wallet

# Initialize client
client = PAWClient(
    rpc_url='https://rpc.paw.network',
    chain_id='paw-testnet-1'
)

# Load wallet
wallet = Wallet.from_mnemonic('your 24 words here...')

# Get balance
balance = client.get_balance(wallet.address)
print(f"Balance: {balance['amount']} {balance['denom']}")

# Send transaction
tx = client.send_tokens(
    wallet,
    to_address='paw1to...',
    amount='1000000upaw',
    fees='500upaw'
)
print(f"TX Hash: {tx['txhash']}")

# Stake tokens
stake = client.staking.delegate(
    wallet,
    validator='pawvaloper1...',
    amount='1000000upaw'
)

# Vote on proposal
vote = client.governance.vote(
    wallet,
    proposal_id=1,
    option='yes'
)
```

### Go SDK

```go
package main

import (
    "<MODULE_PATH>/sdk"
    "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func main() {
    // Initialize client
    client := sdk.NewClient(&sdk.Config{
        RPC:     "https://rpc.paw.network",
        ChainID: "paw-testnet-1",
    })

    // Create account
    privKey := secp256k1.GenPrivKey()
    address := sdk.AccAddressFromHex(privKey.PubKey().Address())

    // Get balance
    balance, _ := client.GetBalance(address.String())
    fmt.Printf("Balance: %s\n", balance.String())

    // Send transaction
    msg := sdk.NewMsgSend(
        address,
        toAddress,
        sdk.NewCoins(sdk.NewCoin("upaw", 1000000)),
    )

    tx, _ := client.BroadcastTx(msg, privKey)
    fmt.Printf("TX Hash: %s\n", tx.TxHash)
}
```

## Common Tasks

### Query Blockchain Data

```javascript
// Get latest block
const block = await client.getLatestBlock();

// Get transaction
const tx = await client.getTx('TX_HASH');

// Query account
const account = await client.getAccount('paw1xxxxx...');

// Get validator info
const validator = await client.staking.getValidator('pawvaloper1...');
```

### Build Transactions

```javascript
// Compose transaction
const tx = await client.buildTx({
  messages: [
    {
      type: '/cosmos.bank.v1beta1.MsgSend',
      value: {
        fromAddress: 'paw1from...',
        toAddress: 'paw1to...',
        amount: [{ denom: 'upaw', amount: '1000000' }]
      }
    }
  ],
  fees: { amount: '500upaw', gas: '200000' },
  memo: 'Payment for services'
});

// Sign and broadcast
const result = await client.signAndBroadcast(tx, wallet);
```

### Work with DEX

```javascript
// Get pool information
const pool = await client.dex.getPool(1);

// Calculate swap output
const output = await client.dex.calculateSwapOutput(
  poolId: 1,
  tokenIn: '1000000upaw'
);

// Add liquidity
const addLiquidity = await client.dex.addLiquidity({
  poolId: 1,
  tokenA: '1000000upaw',
  tokenB: '1000000uusdc'
}, wallet);

// Remove liquidity
const removeLiquidity = await client.dex.removeLiquidity({
  poolId: 1,
  lpTokens: '500000000'
}, wallet);
```

## Testing

### Local Testnet

```bash
# Start local node
 clone <REPO_URL>
cd paw
make install
pawd init test-node --chain-id test-1
pawd start
```

### Unit Tests

::: code-group

```javascript [JavaScript]
import { PAWClient } from '@paw/sdk';
import { expect } from 'chai';

describe('PAW SDK Tests', () => {
  let client;

  before(() => {
    client = new PAWClient({
      rpc: 'http://localhost:26657',
      chainId: 'test-1'
    });
  });

  it('should get balance', async () => {
    const balance = await client.getBalance('paw1test...');
    expect(balance).to.have.property('amount');
  });
});
```

```python [Python]
import unittest
from paw_sdk import PAWClient

class TestPAWSDK(unittest.TestCase):
    def setUp(self):
        self.client = PAWClient(
            rpc_url='http://localhost:26657',
            chain_id='test-1'
        )

    def test_get_balance(self):
        balance = self.client.get_balance('paw1test...')
        self.assertIn('amount', balance)
```

:::

## Resources

- **[JavaScript SDK](/developer/javascript-sdk)** - Complete JS/TS reference
- **[Python SDK](/developer/python-sdk)** - Python documentation
- **[Go Development](/developer/go-development)** - Go integration
- **[API Reference](/developer/api)** - REST and gRPC APIs
- **[Examples Repo](<REPO_URL>-examples)** - Code samples

## Next Steps

1. **[Explore SDKs](/developer/javascript-sdk)** - Deep dive into SDK features
2. **[Build Smart Contracts](/developer/smart-contracts)** - Create dApps
3. **[Develop Modules](/developer/module-development)** - Extend PAW

---

**Next:** [JavaScript SDK](/developer/javascript-sdk) →
