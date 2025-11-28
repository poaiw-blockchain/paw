# PAW Wallet Core SDK

Production-ready TypeScript SDK for building PAW blockchain wallets. Provides comprehensive functionality for key management, transaction signing, and blockchain interaction.

## Features

- **HD Wallet Support**: BIP39/BIP32/BIP44 compliant hierarchical deterministic wallets
- **Secure Key Management**: AES-256-GCM encryption, PBKDF2 key derivation
- **Transaction Signing**: Full Cosmos SDK message support + PAW custom modules
- **RPC Client**: REST API and WebSocket connectivity
- **TypeScript**: Fully typed for excellent IDE support
- **Cross-Platform**: Works in Node.js and browsers

## Installation

```bash
npm install @paw-chain/wallet-core
```

## Quick Start

### Create a New Wallet

```typescript
import { createWallet } from '@paw-chain/wallet-core';

const wallet = createWallet({
  rpcConfig: {
    restUrl: 'http://localhost:1317',
    rpcUrl: 'http://localhost:26657',
  },
});

// Generate new wallet
const { mnemonic, account } = await wallet.generate();
console.log('Mnemonic:', mnemonic);
console.log('Address:', account.address);

// Get balance
const balance = await wallet.getBalance('upaw');
console.log('Balance:', balance);
```

### Import Existing Wallet

```typescript
// From mnemonic
await wallet.createFromMnemonic('your mnemonic phrase here');

// From keystore
await wallet.importKeystore(keystoreJson, 'password');
```

### Send Tokens

```typescript
const result = await wallet.send(
  'paw1recipient...', // recipient address
  '1000000',          // amount (1 PAW = 1,000,000 upaw)
  'upaw',             // denomination
  {
    memo: 'Payment for services',
  }
);

console.log('Transaction hash:', result.transactionHash);
```

### DEX Trading

```typescript
// Swap tokens
const swapResult = await wallet.swap(
  1,           // pool ID
  'upaw',      // token in
  'uusdc',     // token out
  '1000000',   // amount in
  '990000',    // minimum amount out (slippage protection)
);

// Add liquidity
const liquidityResult = await wallet.addLiquidity(
  1,           // pool ID
  '1000000',   // amount A
  '1000000',   // amount B
);
```

### Staking

```typescript
// Delegate tokens
await wallet.delegate(
  'pawvaloper1...', // validator address
  '1000000',        // amount
  'upaw'
);

// Withdraw rewards
await wallet.withdrawRewards('pawvaloper1...');

// Undelegate
await wallet.undelegate('pawvaloper1...', '1000000', 'upaw');
```

### Governance

```typescript
// Vote on proposal
await wallet.vote(
  '1',  // proposal ID
  1,    // option: 1=Yes, 2=Abstain, 3=No, 4=NoWithVeto
);
```

## API Reference

### PAWWallet

Main wallet class providing all functionality.

#### Methods

**Wallet Creation**
- `generate(strength?: 128|160|192|224|256)` - Generate new wallet
- `createFromMnemonic(mnemonic: string, password?: string)` - Import from mnemonic
- `importPrivateKey(privateKey: Uint8Array)` - Import from private key
- `importKeystore(keystore: Keystore, password: string)` - Import from keystore
- `exportKeystore(password: string, name?: string)` - Export to keystore

**Account Info**
- `getAddress()` - Get wallet address
- `getPublicKey()` - Get public key
- `getMnemonic()` - Get mnemonic (if available)
- `getBalance(denom?: string)` - Get balance
- `getAccountInfo()` - Get account number and sequence

**Transactions**
- `send(to, amount, denom, options?)` - Send tokens
- `delegate(validator, amount, denom, options?)` - Delegate tokens
- `undelegate(validator, amount, denom, options?)` - Undelegate tokens
- `redelegate(srcValidator, dstValidator, amount, denom, options?)` - Redelegate tokens
- `withdrawRewards(validator, options?)` - Withdraw staking rewards
- `vote(proposalId, option, options?)` - Vote on proposal

**DEX Operations**
- `swap(poolId, tokenIn, tokenOut, amountIn, minAmountOut, options?)` - Swap tokens
- `createPool(tokenA, tokenB, amountA, amountB, options?)` - Create liquidity pool
- `addLiquidity(poolId, amountA, amountB, options?)` - Add liquidity
- `removeLiquidity(poolId, shares, options?)` - Remove liquidity

**Queries**
- `getValidators(status?)` - Get validators
- `getDelegations()` - Get delegations
- `getRewards()` - Get staking rewards
- `getPools()` - Get DEX pools
- `getPool(poolId)` - Get specific pool
- `simulateSwap(poolId, tokenIn, tokenOut, amountIn)` - Simulate swap
- `getTransactions(page?, limit?)` - Get transaction history

### Crypto Functions

```typescript
import {
  generateMnemonic,
  validateMnemonic,
  derivePrivateKey,
  publicKeyToAddress,
  validateAddress,
} from '@paw-chain/wallet-core';

// Generate mnemonic
const mnemonic = generateMnemonic(256); // 24 words

// Validate mnemonic
const isValid = validateMnemonic(mnemonic);

// Derive private key
const privateKey = await derivePrivateKey(mnemonic, "m/44'/118'/0'/0/0");

// Convert public key to address
const address = publicKeyToAddress(publicKey, 'paw');

// Validate address
const isValidAddress = validateAddress('paw1...', 'paw');
```

### Keystore Functions

```typescript
import {
  encryptKeystore,
  decryptKeystore,
  exportKeystore,
} from '@paw-chain/wallet-core';

// Encrypt private key
const keystore = await encryptKeystore(
  privateKey,
  'strong-password',
  address,
  'My Wallet'
);

// Export to JSON
const json = exportKeystore(keystore, true); // pretty print

// Decrypt keystore
const privateKey = await decryptKeystore(keystore, 'strong-password');
```

### RPC Client

```typescript
import { createRPCClient } from '@paw-chain/wallet-core';

const client = createRPCClient({
  restUrl: 'http://localhost:1317',
  rpcUrl: 'http://localhost:26657',
  wsUrl: 'ws://localhost:26657/websocket',
});

// Get balance
const balances = await client.getBalance('paw1...');

// Get validators
const validators = await client.getValidators('BOND_STATUS_BONDED');

// WebSocket subscriptions
client.connectWebSocket();
client.subscribeToBlocks((block) => {
  console.log('New block:', block);
});
```

## Security Best Practices

1. **Never log or expose private keys or mnemonics**
2. **Use strong passwords for keystores** (minimum 12 characters)
3. **Store mnemonics securely** (hardware wallets, encrypted storage)
4. **Verify addresses** before sending transactions
5. **Use hardware wallets** for large amounts
6. **Enable transaction simulation** before broadcasting
7. **Validate all user input** to prevent injection attacks

## Examples

See the `/examples` directory for complete working examples:

- `basic-wallet.ts` - Create wallet and send tokens
- `dex-trading.ts` - DEX operations
- `staking.ts` - Staking and rewards
- `hd-wallet.ts` - Multi-account HD wallet
- `keystore-management.ts` - Secure key storage

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Watch mode
npm run dev

# Lint
npm run lint
```

## License

MIT

## Support

For issues or questions:
-  Issues: https://github.com/paw-chain/paw/issues
- Documentation: https://docs.paw-chain.io
