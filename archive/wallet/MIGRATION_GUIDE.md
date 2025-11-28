# PAW Wallet Cryptography Migration Guide

## Overview

This guide covers the migration from **CryptoJS** to **Web Crypto API** for improved security and performance. This is a **CRITICAL SECURITY UPDATE** that addresses known vulnerabilities in the previous cryptographic implementation.

## Security Improvements

### What Changed

| Aspect | Before (CryptoJS) | After (Web Crypto API) |
|--------|------------------|----------------------|
| **Encryption** | CryptoJS AES-GCM (limited implementation) | Native Web Crypto AES-256-GCM |
| **KDF** | PBKDF2 with variable iterations | PBKDF2 with 210,000 iterations (OWASP 2023) |
| **Random** | Math.random() in some cases | webcrypto.getRandomValues() |
| **IV Generation** | Sometimes reused | Always unique per encryption |
| **MAC** | SHA-256 (non-constant time) | HMAC-SHA256 (constant-time comparison) |
| **Password Min** | 8 characters | 12 characters |
| **Type Safety** | Loose typing | Full TypeScript interfaces |

### Why This Matters

1. **CryptoJS GCM Mode**: Has known limitations and is not recommended for production use
2. **Timing Attacks**: Old implementation vulnerable to timing-based password guessing
3. **IV Reuse**: Potential for IV reuse in high-throughput scenarios
4. **Low Iteration Count**: Previous default (100k) below OWASP 2023 recommendations
5. **Weak Random**: Non-cryptographic random in some code paths

## Breaking Changes

### 1. Encryption Functions Are Now Async

**Before:**
```typescript
import { encryptAES, decryptAES } from '@paw-chain/wallet-core';

// Synchronous
const encrypted = encryptAES(data, password);
const decrypted = decryptAES(encrypted, password);
```

**After:**
```typescript
import { encryptAES, decryptAES } from '@paw-chain/wallet-core';

// Asynchronous
const encrypted = await encryptAES(data, password);
const decrypted = await decryptAES(encrypted, password);
```

### 2. Encrypted Data Format Changed

**Before:**
```typescript
// Returns a string
const encrypted: string = encryptAES(data, password);
```

**After:**
```typescript
// Returns an object with metadata
const encrypted: EncryptedData = await encryptAES(data, password);
// {
//   ciphertext: "base64...",
//   salt: "base64...",
//   iv: "base64...",
//   algorithm: "AES-256-GCM",
//   kdf: "PBKDF2",
//   iterations: 210000
// }
```

### 3. Keystore Version Updated

**Before:**
```typescript
// Version 3 keystore
{
  "version": 3,
  "crypto": {
    "cipher": "aes-256-gcm",
    "cipherparams": { "iv": "..." },
    // ...
  }
}
```

**After:**
```typescript
// Version 4 keystore (secure)
{
  "version": 4,
  "crypto": {
    "cipher": "AES-256-GCM",
    "kdf": "PBKDF2",
    "kdfparams": {
      "iterations": 210000,
      "dklen": 64
    },
    "iv": "...",
    "mac": "...",
    // ...
  }
}
```

### 4. Minimum Password Length Increased

**Before:**
```typescript
// Minimum 8 characters
const keystore = await encryptKeystore(privateKey, "short123", address);
```

**After:**
```typescript
// Minimum 12 characters (throws error if shorter)
const keystore = await encryptKeystore(privateKey, "LongerPassword123!", address);
```

## Migration Steps

### Step 1: Update Dependencies

Update your `package.json`:

```bash
# Remove old dependency
npm uninstall crypto-js

# Install new dependencies
npm install @noble/hashes@^1.3.3 @noble/secp256k1@^2.0.0 @noble/ciphers@^0.4.0
```

### Step 2: Update Import Statements

**Before:**
```typescript
import * as CryptoJS from 'crypto-js';
```

**After:**
```typescript
import { encryptAES, decryptAES, encryptMnemonic, decryptMnemonic } from '@paw-chain/wallet-core';
import { secureRandom, validatePasswordStrength } from '@paw-chain/wallet-core/security';
```

### Step 3: Update Encryption Code

**Before:**
```typescript
function saveWallet(mnemonic: string, password: string) {
  const encrypted = encryptAES(mnemonic, password);
  localStorage.setItem('wallet', encrypted);
}

function loadWallet(password: string) {
  const encrypted = localStorage.getItem('wallet');
  return decryptAES(encrypted, password);
}
```

**After:**
```typescript
async function saveWallet(mnemonic: string, password: string) {
  const encrypted = await encryptMnemonic(mnemonic, password);
  localStorage.setItem('wallet', JSON.stringify(encrypted));
}

async function loadWallet(password: string) {
  const encrypted = JSON.parse(localStorage.getItem('wallet')!);
  return await decryptMnemonic(encrypted, password);
}
```

### Step 4: Update Keystore Creation

**Before:**
```typescript
const keystore = await encryptKeystore(
  privateKey,
  "password", // 8 chars OK
  address
);
```

**After:**
```typescript
// Validate password first
const validation = validatePasswordStrength("MySecurePass123!");
if (!validation.valid) {
  throw new Error(`Weak password: ${validation.errors.join(', ')}`);
}

const keystore = await encryptKeystore(
  privateKey,
  "MySecurePass123!", // 12+ chars required
  address
);
```

### Step 5: Migrate Existing Keystores

For existing v3 keystores, you need to decrypt with old code and re-encrypt with new code:

```typescript
import { decryptKeystore as decryptV3, encryptKeystore as encryptV4 } from '@paw-chain/wallet-core';

async function migrateKeystore(v3Keystore: any, password: string) {
  // This will throw error directing to migration
  // You need to keep old wallet-core version temporarily

  // 1. Decrypt with old version
  const privateKey = await decryptV3(v3Keystore, password);

  // 2. Re-encrypt with new version
  const v4Keystore = await encryptV4(
    privateKey,
    password,
    v3Keystore.address,
    v3Keystore.meta?.name
  );

  return v4Keystore;
}
```

### Step 6: Update Tests

All tests must now handle async encryption:

**Before:**
```typescript
test('encrypts data', () => {
  const encrypted = encryptAES('data', 'password');
  expect(encrypted).toBeDefined();
});
```

**After:**
```typescript
test('encrypts data', async () => {
  const encrypted = await encryptAES('data', 'StrongPassword123!');
  expect(encrypted.ciphertext).toBeDefined();
  expect(encrypted.algorithm).toBe('AES-256-GCM');
  expect(encrypted.iterations).toBeGreaterThanOrEqual(210000);
});
```

## Password Strength Validation

Always validate passwords before using them:

```typescript
import { validatePasswordStrength } from '@paw-chain/wallet-core/security';

function createWallet(password: string) {
  const strength = validatePasswordStrength(password);

  if (!strength.valid) {
    throw new Error(`Password requirements not met:\n${strength.errors.join('\n')}`);
  }

  if (strength.strength === 'weak' || strength.strength === 'medium') {
    console.warn('Password could be stronger');
  }

  // Proceed with wallet creation
}
```

## Security Best Practices

### 1. Never Store Unencrypted Keys

```typescript
// ❌ BAD
localStorage.setItem('privateKey', privateKeyHex);

// ✅ GOOD
const encrypted = await encryptAES(privateKeyHex, password);
localStorage.setItem('privateKey', JSON.stringify(encrypted));
```

### 2. Use Secure Random for Everything

```typescript
import { secureRandom } from '@paw-chain/wallet-core/security';

// ❌ BAD
const id = Math.random().toString(36);

// ✅ GOOD
const id = Buffer.from(secureRandom(16)).toString('hex');
```

### 3. Wipe Sensitive Data When Done

```typescript
import { secureWipe } from '@paw-chain/wallet-core/security';

async function signTransaction(keystore, password) {
  const privateKey = await decryptKeystore(keystore, password);

  try {
    // Use private key
    const signature = sign(tx, privateKey);
    return signature;
  } finally {
    // Always wipe sensitive data
    secureWipe(privateKey);
  }
}
```

### 4. Implement Rate Limiting

```typescript
import { RateLimiter } from '@paw-chain/wallet-core/security';

const limiter = new RateLimiter(5, 300000); // 5 attempts per 5 minutes

async function unlockWallet(address: string, password: string) {
  if (!limiter.isAllowed(address)) {
    const remaining = limiter.getRemainingAttempts(address);
    throw new Error(`Too many attempts. Try again later. Remaining: ${remaining}`);
  }

  try {
    const keystore = loadKeystore(address);
    const privateKey = await decryptKeystore(keystore, password);
    limiter.reset(address); // Success, reset counter
    return privateKey;
  } catch (error) {
    throw new Error('Invalid password');
  }
}
```

## Performance Considerations

### KDF Iterations Impact

The new default (210,000 iterations) provides excellent security but takes ~200-500ms to derive keys. This is **intentional** for password-based protection.

```typescript
import { calculateOptimalIterations } from '@paw-chain/wallet-core/keyDerivation';

// Benchmark your system
const optimal = await calculateOptimalIterations(500); // Target 500ms
console.log(`Optimal iterations: ${optimal}`);
```

### Caching Derived Keys

For better UX, cache derived keys during a session:

```typescript
class WalletSession {
  private derivedKey: CryptoKey | null = null;

  async unlock(password: string) {
    // Derive once
    this.derivedKey = await deriveKey({ password });
  }

  async sign(data: Uint8Array) {
    if (!this.derivedKey) throw new Error('Wallet locked');
    // Use cached key
    return await encrypt(data, this.derivedKey);
  }

  lock() {
    this.derivedKey = null;
  }
}
```

## Troubleshooting

### Error: "Legacy keystore detected"

You're trying to decrypt a v3 keystore with v4 code. Follow Step 5 above to migrate.

### Error: "Password must be at least 12 characters"

The minimum password length is now 12 characters for security. Update your UI validation.

### Error: "Decryption failed: invalid password or corrupted data"

1. Check password is correct
2. Verify keystore JSON is valid
3. Ensure keystore hasn't been modified
4. Check for version mismatch

### Performance Issues

If key derivation is too slow:

1. Consider using Argon2id (more secure, better performance)
2. Cache derived keys during session
3. Use web workers for background derivation
4. Show progress indicator to users

## Migration Checklist

- [ ] Update dependencies (remove crypto-js, add @noble packages)
- [ ] Update all encryption calls to use async/await
- [ ] Change encrypted data storage to use JSON.stringify
- [ ] Update password validation (12 char minimum)
- [ ] Migrate existing v3 keystores to v4
- [ ] Update all tests for async functions
- [ ] Implement password strength validation
- [ ] Add rate limiting for password attempts
- [ ] Wipe sensitive data after use
- [ ] Update UI for longer key derivation time
- [ ] Test thoroughly before deployment

## Support

For questions or issues:
-  Issues: https://github.com/paw-chain/paw/issues
- Security Contact: security@paw-chain.io

## Security Disclosure

If you discover a security vulnerability, please email security@paw-chain.io with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (optional)

**DO NOT** create public  issues for security vulnerabilities.
