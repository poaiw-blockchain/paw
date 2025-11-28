# PAW Wallet SDK Documentation

Complete TypeScript SDK for building PAW blockchain wallets with HD wallet support, hardware wallet integration, and comprehensive transaction capabilities.

## Installation

```bash
npm install @paw-chain/sdk
# or
yarn add @paw-chain/sdk
```

## Features

- **HD Wallet Support**: BIP39/BIP44 compliant HD wallet generation
- **Hardware Wallets**: Ledger and Trezor integration interfaces
- **Transaction Building**: Complete transaction builder with all PAW modules
- **Address Book**: Built-in address book management
- **Keystore**: Encrypted keystore for secure storage
- **Multi-Account**: Support for multiple accounts from single mnemonic
- **Signing**: Message and transaction signing capabilities

## Quick Start

### Basic Wallet

```typescript
import { PawWalletEnhanced, PawClient } from '@paw-chain/sdk';

// Create new wallet
const wallet = new PawWalletEnhanced('paw');

// Generate mnemonic
const mnemonic = PawWalletEnhanced.generateMnemonic();
console.log('Save this mnemonic:', mnemonic);

// Initialize wallet
await wallet.fromMnemonic(mnemonic);

// Get address
const address = await wallet.getAddress();
console.log('Address:', address);

// Create client
const client = new PawClient('http://localhost:1317', wallet.getSigner());
await client.connect();

// Send tokens
const result = await client.bank.send(
  address,
  'paw1recipient...',
  [{ denom: 'upaw', amount: '1000000' }]
);
```

### HD Wallet with Multiple Accounts

```typescript
import { PawWalletEnhanced } from '@paw-chain/sdk';

const wallet = new PawWalletEnhanced('paw');

// Create 5 accounts from same mnemonic
await wallet.fromMnemonicMultiAccount(mnemonic, 5);

// Get all addresses
const addresses = await wallet.getAddresses();
console.log('Accounts:', addresses);

// Get specific account
const account2 = await wallet.getAccount(1); // Zero-indexed
console.log('Account 2:', account2.address);
```

### Custom HD Path

```typescript
import { PawWalletEnhanced, HDPath } from '@paw-chain/sdk';

const wallet = new PawWalletEnhanced('paw', 118); // 118 is Cosmos coin type

// Custom HD path
const customPath: Partial<HDPath> = {
  account: 0,
  change: 0,
  addressIndex: 5 // Derive 6th address
};

await wallet.fromMnemonic(mnemonic, customPath);
```

### Hardware Wallet (Ledger/Trezor)

```typescript
import {
  createHardwareWallet,
  HardwareWalletType,
  isHardwareWalletSupported,
  PawClient
} from '@paw-chain/sdk';

// Check support
if (isHardwareWalletSupported(HardwareWalletType.LEDGER)) {
  console.log('Ledger is supported!');
}

// Create Ledger wallet
const ledger = createHardwareWallet({
  type: HardwareWalletType.LEDGER,
  hdPath: "m/44'/118'/0'/0/0",
  prefix: 'paw'
});

// Connect to device
await ledger.connect();

// Get accounts
const accounts = await ledger.getAccounts();
console.log('Ledger address:', accounts[0].address);

// Use with client
const client = new PawClient('http://localhost:1317', ledger);
await client.connect();
```

### Address Management

```typescript
import { PawWalletEnhanced } from '@paw-chain/sdk';

// Validate address
const isValid = PawWalletEnhanced.isValidAddress('paw1...');

// Convert address between prefixes
const cosmosAddress = PawWalletEnhanced.convertAddress(
  'paw1abc...',
  'cosmos'
);

// Derive address from public key
const address = PawWalletEnhanced.deriveAddress(pubkey, 'paw');
```

### Address Book

```typescript
import { AddressBook } from '@paw-chain/sdk';

const addressBook = new AddressBook();

// Add contact
addressBook.addAddress({
  name: 'Alice',
  address: 'paw1alice...',
  memo: 'Friend from college',
  tags: ['friend', 'trusted']
});

// Get contact
const alice = addressBook.getAddress('Alice');
console.log('Alice address:', alice?.address);

// Search by tag
const friends = addressBook.searchByTag('friend');

// Export/Import
const json = addressBook.export();
localStorage.setItem('addressBook', json);

const newBook = new AddressBook();
newBook.import(localStorage.getItem('addressBook')!);
```

## Transaction Examples

### Bank Transfer

```typescript
// Send tokens
await client.bank.send(
  fromAddress,
  toAddress,
  [{ denom: 'upaw', amount: '1000000' }], // 1 PAW
  { memo: 'Payment for services' }
);

// Multi-send
await client.bank.multiSend(
  fromAddress,
  [
    { address: 'paw1...', coins: [{ denom: 'upaw', amount: '500000' }] },
    { address: 'paw2...', coins: [{ denom: 'upaw', amount: '500000' }] }
  ]
);
```

### DEX Operations

```typescript
// Create liquidity pool
await client.dex.createPool(
  address,
  { denom: 'upaw', amount: '1000000000' },
  { denom: 'uusdc', amount: '1000000000' },
  { swapFee: '0.003' } // 0.3% fee
);

// Swap tokens
await client.dex.swap(
  address,
  1, // pool ID
  { denom: 'upaw', amount: '1000000' },
  { denom: 'uusdc', amount: '950000' }, // minimum output
  { slippage: '0.05' } // 5% slippage tolerance
);

// Add liquidity
await client.dex.addLiquidity(
  address,
  1, // pool ID
  { denom: 'upaw', amount: '1000000' },
  { denom: 'uusdc', amount: '1000000' }
);

// Remove liquidity
await client.dex.removeLiquidity(
  address,
  1, // pool ID
  '1000000' // LP token amount
);
```

### Staking

```typescript
// Delegate tokens
await client.staking.delegate(
  address,
  'pawvaloper1...',
  { denom: 'upaw', amount: '1000000000' }
);

// Undelegate tokens
await client.staking.undelegate(
  address,
  'pawvaloper1...',
  { denom: 'upaw', amount: '1000000000' }
);

// Redelegate to different validator
await client.staking.redelegate(
  address,
  'pawvaloper1...', // source validator
  'pawvaloper2...', // destination validator
  { denom: 'upaw', amount: '1000000000' }
);

// Claim rewards
await client.staking.claimRewards(
  address,
  ['pawvaloper1...', 'pawvaloper2...'] // validators to claim from
);
```

### Governance

```typescript
// Submit proposal
await client.governance.submitProposal(
  address,
  {
    title: 'Increase DEX fee',
    description: 'Proposal to increase DEX fee to 0.4%',
    type: 'ParameterChangeProposal',
    changes: [
      {
        subspace: 'dex',
        key: 'SwapFee',
        value: '0.004'
      }
    ]
  },
  [{ denom: 'upaw', amount: '10000000000' }] // 10,000 PAW deposit
);

// Vote on proposal
await client.governance.vote(
  address,
  1, // proposal ID
  'yes' // vote option: yes, no, abstain, no_with_veto
);

// Deposit to proposal
await client.governance.deposit(
  address,
  1,
  [{ denom: 'upaw', amount: '1000000000' }]
);
```

## Message Signing

```typescript
// Sign arbitrary message
const signature = await wallet.signArbitrary(
  address,
  'Hello, PAW blockchain!'
);

console.log('Signature:', signature.signature);
console.log('Public key:', signature.pub_key.value);

// Verify signature
const isValid = await PawWalletEnhanced.verifyArbitrary(
  address,
  'Hello, PAW blockchain!',
  signature.signature,
  pubkey
);
```

## Security Best Practices

### 1. Mnemonic Storage

```typescript
// NEVER store mnemonic in plain text
// NEVER log mnemonic to console in production
// ALWAYS use encrypted keystore

// Generate mnemonic
const mnemonic = PawWalletEnhanced.generateMnemonic();

// Encrypt with keystore (requires implementation)
// const keystore = await KeystoreManager.encrypt(mnemonic, password);
// localStorage.setItem('keystore', JSON.stringify(keystore));

// Decrypt when needed
// const decrypted = await KeystoreManager.decrypt(keystore, password);
// await wallet.fromMnemonic(decrypted);
```

### 2. Clear Sensitive Data

```typescript
// Clear wallet from memory when done
wallet.clear();

// Clear variables containing sensitive data
let mnemonic = PawWalletEnhanced.generateMnemonic();
// ... use mnemonic ...
mnemonic = null; // Clear reference
```

### 3. Validate Input

```typescript
// Always validate addresses before sending
if (!PawWalletEnhanced.isValidAddress(recipientAddress)) {
  throw new Error('Invalid recipient address');
}

// Validate mnemonic before import
if (!PawWalletEnhanced.validateMnemonic(userInputMnemonic)) {
  throw new Error('Invalid mnemonic phrase');
}
```

### 4. Transaction Confirmation

```typescript
// Always show transaction details to user before signing
const txDetails = {
  from: fromAddress,
  to: toAddress,
  amount: '1.0 PAW',
  fee: '0.001 PAW',
  memo: 'Payment'
};

// Wait for user confirmation before broadcasting
if (await confirmTransaction(txDetails)) {
  await client.bank.send(...);
}
```

## Advanced Features

### Custom Transaction Builder

```typescript
import { TxBuilder } from '@paw-chain/sdk';

const txBuilder = new TxBuilder(client);

// Build custom transaction
const tx = await txBuilder
  .addMessage({
    typeUrl: '/cosmos.bank.v1beta1.MsgSend',
    value: {
      fromAddress: address,
      toAddress: 'paw1...',
      amount: [{ denom: 'upaw', amount: '1000000' }]
    }
  })
  .setMemo('Custom memo')
  .setGas(200000)
  .setFee([{ denom: 'upaw', amount: '5000' }])
  .build();

// Sign and broadcast
const result = await client.signAndBroadcast(address, [tx.message], tx.fee);
```

### Query Account Info

```typescript
// Get account balance
const balance = await client.bank.getBalance(address, 'upaw');
console.log('Balance:', balance.amount);

// Get all balances
const allBalances = await client.bank.getAllBalances(address);

// Get account info
const account = await client.getAccount(address);
console.log('Account:', {
  address: account.address,
  accountNumber: account.accountNumber,
  sequence: account.sequence
});
```

### Network Configuration

```typescript
// Mainnet
const mainnetClient = new PawClient(
  'https://rpc.paw-chain.com:1317',
  wallet.getSigner()
);

// Testnet
const testnetClient = new PawClient(
  'https://testnet-rpc.paw-chain.com:1317',
  wallet.getSigner()
);

// Local devnet
const devnetClient = new PawClient(
  'http://localhost:1317',
  wallet.getSigner()
);
```

## TypeScript Types

All types are fully typed for TypeScript:

```typescript
import type {
  Coin,
  HDPath,
  AddressBookEntry,
  HardwareWalletOptions,
  WalletAccount
} from '@paw-chain/sdk';
```

## Error Handling

```typescript
try {
  await wallet.fromMnemonic(mnemonic);
  const result = await client.bank.send(...);
  console.log('Transaction hash:', result.transactionHash);
} catch (error) {
  if (error.message.includes('insufficient funds')) {
    console.error('Not enough tokens');
  } else if (error.message.includes('invalid mnemonic')) {
    console.error('Invalid recovery phrase');
  } else {
    console.error('Transaction failed:', error.message);
  }
}
```

## Browser vs Node.js

### Browser

```typescript
// Works in browser with bundler (Webpack, Vite, etc.)
import { PawWalletEnhanced } from '@paw-chain/sdk';
```

### Node.js

```typescript
// Works in Node.js
const { PawWalletEnhanced } = require('@paw-chain/sdk');
```

### React Native

```typescript
// Requires polyfills for crypto APIs
import { PawWalletEnhanced } from '@paw-chain/sdk';

// Add to your app:
// npm install react-native-get-random-values
// import 'react-native-get-random-values';
```

## Testing

```typescript
import { PawWalletEnhanced } from '@paw-chain/sdk';

describe('Wallet', () => {
  it('should generate valid mnemonic', () => {
    const mnemonic = PawWalletEnhanced.generateMnemonic();
    expect(PawWalletEnhanced.validateMnemonic(mnemonic)).toBe(true);
  });

  it('should create wallet from mnemonic', async () => {
    const wallet = new PawWalletEnhanced();
    const mnemonic = PawWalletEnhanced.generateMnemonic();
    await wallet.fromMnemonic(mnemonic);

    const address = await wallet.getAddress();
    expect(address).toMatch(/^paw1[a-z0-9]{38}$/);
  });
});
```

## Resources

- [CosmJS Documentation](https://cosmoshub.io/cosmjs/)
- [BIP39 Specification](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
- [BIP44 Specification](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki)
- [Ledger Cosmos App](https://github.com/cosmos/ledger-cosmos)
- [PAW Blockchain Documentation](https://docs.paw-chain.com)

## License

MIT
