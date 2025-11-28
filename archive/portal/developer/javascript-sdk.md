# JavaScript SDK

Complete reference for the PAW JavaScript/TypeScript SDK.

## Installation

```bash
npm install @paw/sdk
# or
yarn add @paw/sdk
# or
pnpm add @paw/sdk
```

## Initialization

```typescript
import { PAWClient, Wallet } from '@paw/sdk';

const client = new PAWClient({
  rpc: 'https://rpc.paw.network',
  rest: 'https://api.paw.network',
  chainId: 'paw-mainnet-1',
  prefix: 'paw'
});
```

## Wallet Management

```typescript
// Create new wallet
const wallet = await Wallet.generate();
console.log(wallet.mnemonic); // Save this!
console.log(wallet.address);

// Import from mnemonic
const wallet = await Wallet.fromMnemonic(
  'word1 word2 ... word24'
);

// Import from private key
const wallet = await Wallet.fromPrivateKey('0x...');

// Sign message
const signature = await wallet.signMessage('Hello PAW');
```

## Transactions

```typescript
// Send tokens
const result = await client.sendTokens(
  wallet,
  'paw1recipient...',
  [{ denom: 'upaw', amount: '1000000' }],
  { fees: '500upaw', gas: '200000' }
);

// Multi-send
const multiSend = await client.multiSend(
  wallet,
  [
    { address: 'paw1...', amount: '1000000upaw' },
    { address: 'paw2...', amount: '2000000upaw' }
  ]
);
```

## Staking Operations

```typescript
// Delegate
await client.staking.delegate(
  wallet,
  'pawvaloper1...',
  '1000000upaw'
);

// Undelegate
await client.staking.undelegate(
  wallet,
  'pawvaloper1...',
  '1000000upaw'
);

// Redelegate
await client.staking.redelegate(
  wallet,
  'pawvaloper1from...',
  'pawvaloper1to...',
  '1000000upaw'
);

// Claim rewards
await client.distribution.withdrawRewards(
  wallet,
  'pawvaloper1...'
);
```

## DEX Operations

```typescript
// Get pools
const pools = await client.dex.getPools();

// Swap tokens
const swap = await client.dex.swap(
  wallet,
  {
    poolId: 1,
    tokenIn: { denom: 'upaw', amount: '1000000' },
    minTokenOut: '950000',
    slippage: 0.01
  }
);

// Add liquidity
const add = await client.dex.addLiquidity(
  wallet,
  {
    poolId: 1,
    tokenA: { denom: 'upaw', amount: '1000000' },
    tokenB: { denom: 'uusdc', amount: '1000000' }
  }
);
```

## Governance

```typescript
// Submit proposal
const proposal = await client.gov.submitProposal(
  wallet,
  {
    title: 'Increase Block Size',
    description: 'Detailed description...',
    type: 'text',
    deposit: '1000000000upaw'
  }
);

// Vote
await client.gov.vote(
  wallet,
  1, // proposal ID
  'yes' // yes, no, abstain, no_with_veto
);

// Query proposals
const proposals = await client.gov.getProposals();
```

## Query Methods

```typescript
// Account info
const account = await client.getAccount('paw1...');

// Balance
const balance = await client.getBalance('paw1...', 'upaw');

// Validators
const validators = await client.staking.getValidators();

// Transaction
const tx = await client.getTx('TX_HASH');

// Latest block
const block = await client.getLatestBlock();
```

## WebSocket Subscriptions

```typescript
// Subscribe to new blocks
client.subscribeNewBlock((block) => {
  console.log(`New block: ${block.header.height}`);
});

// Subscribe to transactions
client.subscribeTx((tx) => {
  console.log(`New TX: ${tx.hash}`);
});

// Subscribe to events
client.subscribeEvent('transfer', (event) => {
  console.log(`Transfer: ${event.amount}`);
});
```

## Advanced Features

```typescript
// Estimate gas
const gasEstimate = await client.estimateGas(messages);

// Simulate transaction
const simulation = await client.simulate(messages);

// Build custom message
const customMsg = {
  typeUrl: '/cosmos.bank.v1beta1.MsgSend',
  value: {
    fromAddress: 'paw1...',
    toAddress: 'paw2...',
    amount: [{ denom: 'upaw', amount: '1000000' }]
  }
};

await client.signAndBroadcast([customMsg], wallet);
```

## Error Handling

```typescript
try {
  await client.sendTokens(wallet, 'paw1...', '1000000upaw');
} catch (error) {
  if (error.code === 'INSUFFICIENT_FUNDS') {
    console.log('Not enough balance');
  } else if (error.code === 'INVALID_ADDRESS') {
    console.log('Invalid recipient address');
  } else {
    console.error('Transaction failed:', error.message);
  }
}
```

## TypeScript Types

```typescript
import {
  PAWClient,
  Wallet,
  Coin,
  StdFee,
  TxResponse,
  Account,
  Validator
} from '@paw/sdk';

interface TransactionOptions {
  fees?: StdFee;
  gas?: string;
  memo?: string;
}

interface SwapParams {
  poolId: number;
  tokenIn: Coin;
  minTokenOut: string;
  slippage: number;
}
```

## Examples

See [ Examples](<REPO_URL>-examples/tree/main/javascript)

---

**Previous:** [Quick Start](/developer/quick-start) | **Next:** [Python SDK](/developer/python-sdk) →
