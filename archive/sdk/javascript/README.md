# PAW JavaScript/TypeScript SDK

Official JavaScript/TypeScript SDK for interacting with the PAW blockchain.

## Features

- **Full TypeScript Support**: Complete type definitions for all APIs
- **Wallet Management**: Create, import, and manage wallets with BIP39 mnemonics
- **Transaction Building**: Easy-to-use transaction builders with automatic fee estimation
- **Module Support**:
  - **Bank**: Send tokens, query balances
  - **DEX**: Create pools, swap tokens, add/remove liquidity
  - **Staking**: Delegate, undelegate, redelegate, withdraw rewards
  - **Governance**: Submit proposals, vote, deposit
- **Dual Build**: ESM and CommonJS support
- **Browser Compatible**: Works in both Node.js and browser environments

## Installation

```bash
npm install @paw-chain/sdk
```

## Quick Start

### Create a Wallet

```typescript
import { PawWallet } from '@paw-chain/sdk';

// Generate a new mnemonic
const mnemonic = PawWallet.generateMnemonic();
console.log('Save this mnemonic:', mnemonic);

// Create wallet from mnemonic
const wallet = new PawWallet('paw');
await wallet.fromMnemonic(mnemonic);

// Get address
const address = await wallet.getAddress();
console.log('Address:', address);
```

### Connect to PAW Blockchain

```typescript
import { PawClient, PawWallet } from '@paw-chain/sdk';

// Initialize client
const client = new PawClient({
  rpcEndpoint: 'http://localhost:26657',
  restEndpoint: 'http://localhost:1317',
  chainId: 'paw-testnet-1',
  gasPrice: '0.025upaw'
});

// Connect with wallet for signing
await client.connectWithWallet(wallet);
```

### Send Tokens

```typescript
const result = await client.bank.send(
  senderAddress,
  {
    recipient: 'paw1...',
    amount: '1000000', // 1 PAW (6 decimals)
    denom: 'upaw'
  }
);

console.log('Transaction hash:', result.transactionHash);
```

### Query Balance

```typescript
const balance = await client.bank.getBalance(address, 'upaw');
console.log('Balance:', balance);
```

## Module Examples

### Bank Module

```typescript
// Get all balances
const balances = await client.bank.getAllBalances(address);

// Send tokens
await client.bank.send(sender, {
  recipient: 'paw1...',
  amount: '1000000',
  denom: 'upaw',
  memo: 'Payment for services'
});

// Multi-send
await client.bank.multiSend(sender, [
  { address: 'paw1...', amount: '500000' },
  { address: 'paw1...', amount: '300000' }
]);
```

### DEX Module

```typescript
// Get all pools
const pools = await client.dex.getAllPools();

// Get specific pool
const pool = await client.dex.getPoolByTokens('upaw', 'uatom');

// Calculate swap output
const amountOut = client.dex.calculateSwapOutput(
  '1000000',
  pool.reserveA,
  pool.reserveB,
  pool.swapFee
);

// Execute swap
await client.dex.swap(sender, {
  poolId: pool.id,
  tokenIn: 'upaw',
  amountIn: '1000000',
  minAmountOut: '900000'
});

// Create pool
await client.dex.createPool(creator, {
  tokenA: 'upaw',
  tokenB: 'uatom',
  amountA: '10000000',
  amountB: '5000000'
});

// Add liquidity
await client.dex.addLiquidity(sender, {
  poolId: pool.id,
  amountA: '1000000',
  amountB: '500000',
  minShares: '0'
});

// Remove liquidity
await client.dex.removeLiquidity(sender, {
  poolId: pool.id,
  shares: '1000000',
  minAmountA: '900000',
  minAmountB: '450000'
});
```

### Staking Module

```typescript
// Get all validators
const validators = await client.staking.getValidators();

// Delegate
await client.staking.delegate(delegator, {
  validatorAddress: 'pawvaloper1...',
  amount: '1000000'
});

// Undelegate
await client.staking.undelegate(delegator, {
  validatorAddress: 'pawvaloper1...',
  amount: '500000'
});

// Redelegate
await client.staking.redelegate(delegator, {
  srcValidatorAddress: 'pawvaloper1...',
  dstValidatorAddress: 'pawvaloper1...',
  amount: '500000'
});

// Withdraw rewards
await client.staking.withdrawRewards(delegator, 'pawvaloper1...');

// Withdraw all rewards
await client.staking.withdrawAllRewards(delegator);

// Get delegations
const delegations = await client.staking.getDelegations(delegator);

// Get rewards
const rewards = await client.staking.getRewards(delegator);
```

### Governance Module

```typescript
import { VoteOption } from '@paw-chain/sdk';

// Get all proposals
const proposals = await client.governance.getProposals();

// Get active proposals
const activeProposals = await client.governance.getProposals(2);

// Submit text proposal
await client.governance.submitTextProposal(
  proposer,
  'Proposal Title',
  'Proposal description',
  '10000000', // Initial deposit
  'upaw'
);

// Vote on proposal
await client.governance.vote(voter, {
  proposalId: '1',
  option: VoteOption.YES,
  metadata: 'I support this'
});

// Deposit to proposal
await client.governance.deposit(depositor, {
  proposalId: '1',
  amount: '1000000'
});

// Get tally
const tally = await client.governance.getTally('1');

// Get governance parameters
const votingParams = await client.governance.getParams('voting');
const depositParams = await client.governance.getParams('deposit');
const tallyParams = await client.governance.getParams('tallying');
```

## Advanced Usage

### Custom Gas Configuration

```typescript
const result = await client.bank.send(
  sender,
  { recipient: 'paw1...', amount: '1000000' },
  {
    gasLimit: 200000,
    gasPrice: '0.05upaw',
    memo: 'Custom gas settings'
  }
);
```

### Transaction Simulation

```typescript
const message = {
  typeUrl: '/cosmos.bank.v1beta1.MsgSend',
  value: {
    fromAddress: sender,
    toAddress: 'paw1...',
    amount: [{ denom: 'upaw', amount: '1000000' }]
  }
};

const txBuilder = client.getTxBuilder();
const gasEstimate = await txBuilder.simulate(sender, [message]);
console.log('Estimated gas:', gasEstimate);
```

### Custom HD Path

```typescript
const wallet = new PawWallet('paw');
await wallet.fromMnemonic(mnemonic, "m/44'/118'/0'/0/0");
```

## API Reference

### PawClient

Main client for interacting with PAW blockchain.

**Constructor Options:**
- `rpcEndpoint`: RPC endpoint URL
- `restEndpoint`: REST API endpoint URL (optional)
- `chainId`: Chain ID
- `prefix`: Address prefix (default: 'paw')
- `gasPrice`: Default gas price (default: '0.025upaw')
- `gasAdjustment`: Gas adjustment multiplier (default: 1.5)

**Methods:**
- `connect()`: Connect to blockchain (read-only)
- `connectWithWallet(wallet)`: Connect with wallet for signing
- `disconnect()`: Disconnect from blockchain
- `getHeight()`: Get current block height
- `getChainId()`: Get chain ID
- `isConnected()`: Check connection status
- `canSign()`: Check if wallet is connected

### PawWallet

Wallet management with BIP39 support.

**Static Methods:**
- `generateMnemonic()`: Generate 24-word mnemonic
- `validateMnemonic(mnemonic)`: Validate mnemonic

**Methods:**
- `fromMnemonic(mnemonic, hdPath?)`: Initialize from mnemonic
- `getAccounts()`: Get all accounts
- `getAddress()`: Get first account address
- `getSigner()`: Get offline signer
- `exportMnemonic()`: Export mnemonic (use with caution!)

## Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

## Building

```bash
# Build for production
npm run build

# Development mode (watch)
npm run dev
```

## Examples

See the `examples/` directory for complete working examples:

- `basic-usage.ts`: Wallet creation and basic operations
- `dex-trading.ts`: DEX trading and liquidity management
- `staking.ts`: Staking operations
- `governance.ts`: Governance participation

Run examples:

```bash
# Set environment variables
export MNEMONIC="your mnemonic here"
export RPC_ENDPOINT="http://localhost:26657"
export CHAIN_ID="paw-testnet-1"

# Run example
npx ts-node examples/basic-usage.ts
```

## TypeScript Support

The SDK is written in TypeScript and includes complete type definitions. No additional `@types` packages are needed.

```typescript
import type { Pool, Validator, Proposal, TxResult } from '@paw-chain/sdk';
```

## Browser Support

The SDK works in modern browsers. Use a bundler like Vite, Webpack, or Rollup:

```typescript
import { PawClient } from '@paw-chain/sdk';

// Works in browser!
const client = new PawClient({
  rpcEndpoint: 'https://rpc.paw.network',
  chainId: 'paw-1'
});
```

## Error Handling

```typescript
try {
  const result = await client.bank.send(sender, {
    recipient: 'paw1...',
    amount: '1000000'
  });
  console.log('Success:', result.transactionHash);
} catch (error) {
  if (error.message.includes('insufficient funds')) {
    console.error('Not enough balance');
  } else {
    console.error('Transaction failed:', error);
  }
}
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- Documentation: https://docs.paw.network
- : https://github.com/paw-chain/paw
- Discord: https://discord.gg/DBHTc2QV

## Changelog

### 1.0.0
- Initial release
- Full support for bank, DEX, staking, and governance modules
- TypeScript support
- Comprehensive test suite
- Complete documentation and examples
