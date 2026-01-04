# Hardware Wallet Integration Guide

Complete guide for using Ledger and Trezor hardware wallets with PAW Chain.

## Table of Contents

1. [Overview](#overview)
2. [Supported Devices](#supported-devices)
3. [Quick Start](#quick-start)
4. [Ledger Integration](#ledger-integration)
5. [Trezor Integration](#trezor-integration)
6. [API Reference](#api-reference)
7. [Examples](#examples)
8. [Security Considerations](#security-considerations)
9. [Troubleshooting](#troubleshooting)

## Overview

PAW Chain provides first-class hardware wallet support through the `@paw-chain/wallet-core` SDK. This integration allows users to:

- **Securely manage** private keys on dedicated hardware
- **Sign transactions** without exposing keys to the computer
- **Verify addresses** on device screen
- **Multi-account support** via BIP44 derivation
- **Browser integration** using WebUSB and WebHID

### Architecture

```
┌─────────────────┐
│   Web App       │
│  (React/Vue)    │
└────────┬────────┘
         │
         ├─────────────────┐
         ▼                 ▼
┌─────────────────┐ ┌──────────────┐
│  Ledger API     │ │  Trezor API  │
│  (WebUSB)       │ │ (WebUSB/HID) │
└────────┬────────┘ └──────┬───────┘
         │                 │
         ▼                 ▼
┌─────────────────┐ ┌──────────────┐
│ Ledger Device   │ │Trezor Device │
│ (Cosmos App)    │ │ (Firmware)   │
└─────────────────┘ └──────────────┘
```

## Supported Devices

### Ledger

| Device | Status | Firmware Required |
|--------|--------|-------------------|
| Ledger Nano S | ✅ Supported | 2.1.0+ |
| Ledger Nano S Plus | ✅ Supported | 1.0.0+ |
| Ledger Nano X | ✅ Supported | 2.0.0+ |

**Requirements:**
- Cosmos app installed (from Ledger Live)
- USB connection
- Browser with WebUSB support

### Trezor

| Device | Status | Firmware Required |
|--------|--------|-------------------|
| Trezor One | ✅ Supported | 1.10.0+ |
| Trezor Model T | ✅ Supported | 2.5.0+ |

**Requirements:**
- Latest firmware
- USB connection
- Browser with WebUSB/WebHID support

### Browser Support

| Browser | Ledger | Trezor | Notes |
|---------|--------|--------|-------|
| Chrome 89+ | ✅ | ✅ | Full support |
| Edge 89+ | ✅ | ✅ | Full support |
| Opera 76+ | ✅ | ✅ | Full support |
| Firefox | ❌ | ⚠️ | Limited (via Trezor Bridge) |
| Safari | ❌ | ❌ | Not supported |

## Quick Start

### Installation

```bash
npm install @paw-chain/wallet-core
```

### Basic Usage

```typescript
import { connectLedger, connectTrezor } from '@paw-chain/wallet-core';

// Connect to Ledger
const ledger = await connectLedger();
const address = await ledger.getAddress("m/44'/118'/0'/0/0", true);

// Connect to Trezor
const trezor = await connectTrezor();
const address = await trezor.getAddress("m/44'/118'/0'/0/0", true);
```

## Ledger Integration

### Setup

1. **Install Cosmos App**
   - Open Ledger Live
   - Go to Manager
   - Search for "Cosmos"
   - Install the app

2. **Connect Device**
   ```typescript
   import { LedgerWallet } from '@paw-chain/wallet-core';

   const wallet = new LedgerWallet({
     timeout: 60000,
     coinType: 118,
     prefix: 'paw',
   });

   await wallet.connect();
   ```

### Get Address

```typescript
// Get address without device confirmation
const address = await wallet.getAddress("m/44'/118'/0'/0/0", false);

// Get address with device confirmation (recommended)
const verifiedAddress = await wallet.getAddress("m/44'/118'/0'/0/0", true);
```

### Sign Transaction

```typescript
import { buildTxBody, buildAuthInfo } from '@paw-chain/wallet-core';

// Build transaction
const txBody = buildTxBody(messages, memo);
const authInfo = buildAuthInfo(publicKey, sequence, gasLimit, fee, feeDenom);

// Prepare sign doc
const signDoc = {
  chainId: 'paw-1',
  accountNumber: '123',
  sequence: '0',
  fee: {
    amount: [{ denom: 'upaw', amount: '5000' }],
    gas: '200000'
  },
  msgs: messages,
  memo: 'Hello PAW Chain'
};

// Convert to bytes
const txBytes = Buffer.from(JSON.stringify(signDoc), 'utf8');

// Sign with Ledger
const { signature, publicKey } = await wallet.signTransaction(
  "m/44'/118'/0'/0/0",
  txBytes,
  true // Show on device
);
```

### Account Discovery

```typescript
import { HardwareWalletUtils } from '@paw-chain/wallet-core';

// Generate paths for first 10 accounts
const paths = HardwareWalletUtils.generatePaths(118, 10, 0);

// Get addresses for all paths
const accounts = await wallet.getAddresses(paths);

accounts.forEach((account, index) => {
  console.log(`Account ${index}: ${account.address}`);
});
```

### Error Handling

```typescript
import { HardwareWalletUtils, HardwareWalletError } from '@paw-chain/wallet-core';

try {
  await wallet.signTransaction(path, txBytes);
} catch (error) {
  if (error instanceof HardwareWalletError) {
    const message = HardwareWalletUtils.getErrorMessage(error);

    switch (error.code) {
      case 'USER_REJECTED':
        console.log('User cancelled on device');
        break;
      case 'DEVICE_LOCKED':
        console.log('Please unlock your Ledger');
        break;
      case 'APP_NOT_OPEN':
        console.log('Please open Cosmos app');
        break;
      default:
        console.error(message);
    }
  }
}
```

## Trezor Integration

### Setup

1. **Update Firmware**
   - Visit https://trezor.io/start
   - Update to latest firmware

2. **Connect Device**
   ```typescript
   import { TrezorWallet } from '@paw-chain/wallet-core';

   const wallet = new TrezorWallet({
     timeout: 60000,
     coinType: 118,
     prefix: 'paw',
   });

   await wallet.connect();
   ```

### Get Address

```typescript
// Get address with device confirmation
const address = await wallet.getAddress("m/44'/118'/0'/0/0", true);
```

### Sign Transaction

```typescript
// Prepare transaction
const signDoc = {
  chain_id: 'paw-1',
  account_number: '123',
  sequence: '0',
  fee: {
    amount: [{ denom: 'upaw', amount: '5000' }],
    gas: '200000'
  },
  msgs: [
    {
      type: 'cosmos-sdk/MsgSend',
      value: {
        from_address: fromAddress,
        to_address: toAddress,
        amount: [{ denom: 'upaw', amount: '1000000' }]
      }
    }
  ],
  memo: ''
};

const txBytes = Buffer.from(JSON.stringify(signDoc), 'utf8');

// Sign with Trezor
const { signature, publicKey } = await wallet.signTransaction(
  "m/44'/118'/0'/0/0",
  txBytes,
  true
);
```

## API Reference

### HardwareWalletFactory

Factory for creating hardware wallet instances.

```typescript
class HardwareWalletFactory {
  static create(type: HardwareWalletType, config?: HardwareWalletConfig): IHardwareWallet;
  static getSupportedWallets(): Promise<HardwareWalletType[]>;
  static detectWallets(): Promise<HardwareWalletInfo[]>;
}
```

### IHardwareWallet Interface

```typescript
interface IHardwareWallet {
  readonly type: HardwareWalletType;

  isConnected(): Promise<boolean>;
  connect(): Promise<HardwareWalletInfo>;
  disconnect(): Promise<void>;
  getDeviceInfo(): Promise<HardwareWalletInfo>;

  getPublicKey(path: string, showOnDevice?: boolean): Promise<Uint8Array>;
  getAddress(path: string, showOnDevice?: boolean): Promise<string>;
  getAddresses(paths: string[]): Promise<HardwareWalletAccount[]>;

  signTransaction(
    path: string,
    txBytes: Uint8Array,
    showOnDevice?: boolean
  ): Promise<SignatureResult>;

  signMessage(
    path: string,
    message: string | Uint8Array,
    showOnDevice?: boolean
  ): Promise<SignatureResult>;
}
```

### HardwareWalletUtils

Utility functions for hardware wallets.

```typescript
class HardwareWalletUtils {
  // Generate BIP44 derivation paths
  static generatePaths(
    coinType: number = 118,
    accountCount: number = 10,
    account: number = 0
  ): string[];

  // Parse derivation path
  static parsePath(path: string): {
    coinType: number;
    account: number;
    change: number;
    index: number;
  };

  // Validate derivation path
  static isValidPath(path: string): boolean;

  // Get user-friendly error message
  static getErrorMessage(error: HardwareWalletError): string;
}
```

### HardwareWalletManager

Manage multiple hardware wallets.

```typescript
class HardwareWalletManager {
  addWallet(type: HardwareWalletType, config?: HardwareWalletConfig): Promise<string>;
  getWallet(id: string): IHardwareWallet | undefined;
  removeWallet(id: string): Promise<void>;
  getAllWallets(): Map<string, IHardwareWallet>;
  disconnectAll(): Promise<void>;
}
```

## Examples

### Complete React Example

```typescript
import React, { useState } from 'react';
import {
  connectLedger,
  connectTrezor,
  HardwareWalletType,
  HardwareWalletError,
  HardwareWalletUtils
} from '@paw-chain/wallet-core';

function HardwareWalletConnect() {
  const [wallet, setWallet] = useState(null);
  const [address, setAddress] = useState('');
  const [error, setError] = useState('');

  const connectDevice = async (type: HardwareWalletType) => {
    try {
      setError('');

      const device = type === HardwareWalletType.LEDGER
        ? await connectLedger()
        : await connectTrezor();

      setWallet(device);

      // Get address
      const addr = await device.getAddress("m/44'/118'/0'/0/0", true);
      setAddress(addr);

    } catch (err) {
      if (err instanceof HardwareWalletError) {
        setError(HardwareWalletUtils.getErrorMessage(err));
      } else {
        setError(err.message);
      }
    }
  };

  return (
    <div>
      <button onClick={() => connectDevice(HardwareWalletType.LEDGER)}>
        Connect Ledger
      </button>
      <button onClick={() => connectDevice(HardwareWalletType.TREZOR)}>
        Connect Trezor
      </button>

      {address && <p>Address: {address}</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
}
```

### Send Transaction Example

```typescript
async function sendTokens(
  wallet: IHardwareWallet,
  fromAddress: string,
  toAddress: string,
  amount: string
) {
  // Create message
  const message = {
    typeUrl: '/cosmos.bank.v1beta1.MsgSend',
    value: {
      fromAddress,
      toAddress,
      amount: [{ denom: 'upaw', amount }]
    }
  };

  // Build transaction
  const signDoc = {
    chain_id: 'paw-1',
    account_number: '123',
    sequence: '0',
    fee: {
      amount: [{ denom: 'upaw', amount: '5000' }],
      gas: '200000'
    },
    msgs: [message],
    memo: ''
  };

  // Sign
  const txBytes = Buffer.from(JSON.stringify(signDoc), 'utf8');
  const { signature } = await wallet.signTransaction(
    "m/44'/118'/0'/0/0",
    txBytes,
    true
  );

  // Broadcast
  // ... broadcast transaction to network
}
```

### Account Discovery Example

```typescript
async function discoverAccounts(wallet: IHardwareWallet) {
  const paths = HardwareWalletUtils.generatePaths(118, 5, 0);
  const accounts = await wallet.getAddresses(paths);

  // Filter accounts with balance
  const accountsWithBalance = await Promise.all(
    accounts.map(async (account) => {
      const balance = await checkBalance(account.address);
      return { ...account, balance };
    })
  );

  return accountsWithBalance.filter(acc => acc.balance > 0);
}
```

## Security Considerations

### Best Practices

1. **Always Verify on Device**
   ```typescript
   // Always use showOnDevice=true for addresses
   const address = await wallet.getAddress(path, true);
   ```

2. **Validate Transactions**
   ```typescript
   // Review transaction details on device screen
   // Check amount, recipient, fees
   ```

3. **Secure Connection**
   ```typescript
   // Use HTTPS in production
   // Verify SSL certificates
   ```

4. **Handle Disconnection**
   ```typescript
   // Implement reconnection logic
   try {
     if (!await wallet.isConnected()) {
       await wallet.connect();
     }
   } catch (error) {
     // Handle connection error
   }
   ```

### Security Checklist

- [ ] Use latest firmware
- [ ] Verify addresses on device screen
- [ ] Check transaction details before signing
- [ ] Use HTTPS in production
- [ ] Implement proper error handling
- [ ] Clear sensitive data after use
- [ ] Use timeout protection
- [ ] Implement rate limiting
- [ ] Log security events
- [ ] Regular security audits

## Troubleshooting

### Ledger Issues

#### Device Not Found

**Problem:** Cannot connect to Ledger device

**Solutions:**
1. Check USB connection
2. Unlock device with PIN
3. Open Cosmos app on device
4. Enable browser support in Ledger Live
5. Try different USB port/cable

```typescript
// Check if Ledger is supported
import { isLedgerSupported } from '@paw-chain/wallet-core';

if (!await isLedgerSupported()) {
  console.error('Ledger not supported in this browser');
}
```

#### App Not Open

**Problem:** `APP_NOT_OPEN` error

**Solution:**
1. Open Cosmos app on Ledger
2. Wait for "Application is ready" message

#### Permission Denied

**Problem:** Browser blocks USB access

**Solution:**
1. Use supported browser (Chrome/Edge/Opera)
2. Allow USB permissions when prompted
3. Check browser security settings

### Trezor Issues

#### Connection Failed

**Problem:** Cannot connect to Trezor

**Solutions:**
1. Update Trezor firmware
2. Install Trezor Bridge (Firefox)
3. Check USB connection
4. Disable conflicting extensions

#### PIN Required

**Problem:** Device requires PIN

**Solution:**
1. Enter PIN on device screen
2. Follow on-screen instructions

### Common Errors

| Error Code | Description | Solution |
|------------|-------------|----------|
| `USER_REJECTED` | User cancelled on device | Retry transaction |
| `NOT_CONNECTED` | Device not connected | Connect device |
| `DEVICE_LOCKED` | Device requires PIN | Unlock device |
| `APP_NOT_OPEN` | App not open (Ledger) | Open Cosmos app |
| `INVALID_PATH` | Invalid derivation path | Check path format |
| `TIMEOUT` | Operation timeout | Increase timeout |

### Debug Mode

Enable debug logging:

```typescript
const wallet = new LedgerWallet({
  debug: true  // Enable debug mode
});
```

## Testing

### Unit Tests

```bash
cd wallet/core
npm test -- hardware
```

### Manual Testing

1. **Connect Device**
   - Verify connection
   - Check device info

2. **Get Address**
   - Verify address on screen
   - Compare with expected format

3. **Sign Transaction**
   - Review transaction details
   - Verify signature

4. **Error Scenarios**
   - Test disconnection
   - Test user rejection
   - Test timeout

## Resources

### Official Documentation

- **Ledger:**
  - [Developer Portal](https://developers.ledger.com/)
  - [Cosmos App](https://github.com/Zondax/ledger-cosmos)

- **Trezor:**
  - [Developer Docs](https://docs.trezor.io/)
  - [Connect API](https://github.com/trezor/trezor-suite)

### PAW Chain Resources

- [ Repository](https://github.com/paw-chain/paw)
- [Documentation](https://docs.pawchain.network)
- [Support](mailto:support@pawchain.network)

### Community

- [Discord](https://discord.gg/DBHTc2QV)
- [Twitter](https://twitter.com/pawchain)
- [Forum](https://forum.pawchain.network)

## License

This documentation is part of the PAW Chain project and is licensed under MIT License.
