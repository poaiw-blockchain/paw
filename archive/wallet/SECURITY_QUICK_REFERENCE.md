# PAW Wallet Security - Quick Reference

## Core Security Functions

### Encryption/Decryption

```typescript
import { encryptAES, decryptAES, encryptMnemonic, decryptMnemonic } from '@paw-chain/wallet-core';

// Encrypt any string data
const encrypted = await encryptAES('sensitive data', 'StrongPassword123!');
// Returns: { ciphertext, salt, iv, algorithm, kdf, iterations }

// Decrypt data
const decrypted = await decryptAES(encrypted, 'StrongPassword123!');

// Encrypt mnemonic (validates mnemonic first)
const encryptedMnemonic = await encryptMnemonic(mnemonic, password);
const decryptedMnemonic = await decryptMnemonic(encryptedMnemonic, password);
```

### Keystore Management

```typescript
import { encryptKeystore, decryptKeystore } from '@paw-chain/wallet-core';

// Create secure keystore
const keystore = await encryptKeystore(
  privateKey,      // Uint8Array (32 bytes)
  password,        // String (12+ characters)
  address,         // Bech32 address
  'My Wallet'      // Optional name
);

// Decrypt keystore
const privateKey = await decryptKeystore(keystore, password);
```

### Password Validation

```typescript
import { validatePasswordStrength } from '@paw-chain/wallet-core';

const result = validatePasswordStrength(password);
// {
//   valid: boolean,
//   strength: 'weak' | 'medium' | 'strong',
//   errors: string[],
//   score: number
// }

if (!result.valid) {
  console.error('Password errors:', result.errors);
}
```

### Secure Random Generation

```typescript
import { secureRandom, secureRandomHex, generateUUID } from '@paw-chain/wallet-core';

// Generate random bytes
const bytes = secureRandom(32);           // Uint8Array (32 bytes)
const hex = secureRandomHex(32);          // Hex string (64 chars)
const uuid = generateUUID();              // UUID v4 string
```

### Key Derivation

```typescript
import { deriveKeyBytes, RECOMMENDED_PBKDF2_ITERATIONS } from '@paw-chain/wallet-core';

// Derive key from password (PBKDF2)
const key = await deriveKeyBytes(
  password,                          // String
  salt,                              // Uint8Array (32 bytes)
  RECOMMENDED_PBKDF2_ITERATIONS,     // 210,000
  32                                 // Key length in bytes
);

// Using Argon2id (more secure, optional)
import { deriveKeyArgon2 } from '@paw-chain/wallet-core';
const argonKey = await deriveKeyArgon2(password, salt);
```

### Rate Limiting

```typescript
import { RateLimiter } from '@paw-chain/wallet-core';

const limiter = new RateLimiter(5, 300000); // 5 attempts per 5 minutes

if (!limiter.isAllowed(address)) {
  const remaining = limiter.getRemainingAttempts(address);
  throw new Error(`Too many attempts. ${remaining} remaining.`);
}

// On successful authentication
limiter.reset(address);
```

### Secure Memory Handling

```typescript
import { secureWipe } from '@paw-chain/wallet-core';

const privateKey = await decryptKeystore(keystore, password);
try {
  // Use private key for signing
  const signature = sign(data, privateKey);
  return signature;
} finally {
  // Always wipe sensitive data
  secureWipe(privateKey);
}
```

## Security Best Practices

### 1. Always Validate Passwords

```typescript
// ❌ BAD
const keystore = await encryptKeystore(privateKey, userPassword, address);

// ✅ GOOD
const validation = validatePasswordStrength(userPassword);
if (!validation.valid) {
  throw new Error(`Weak password: ${validation.errors.join(', ')}`);
}
const keystore = await encryptKeystore(privateKey, userPassword, address);
```

### 2. Never Store Unencrypted Keys

```typescript
// ❌ BAD
localStorage.setItem('privateKey', privateKeyHex);

// ✅ GOOD
const encrypted = await encryptMnemonic(mnemonic, password);
localStorage.setItem('wallet', JSON.stringify(encrypted));
```

### 3. Use Constant-Time Comparisons

```typescript
import { constantTimeCompare, constantTimeCompareString } from '@paw-chain/wallet-core';

// ❌ BAD (timing attack vulnerable)
if (hash1 === hash2) { /* ... */ }

// ✅ GOOD (constant-time)
if (constantTimeCompareString(hash1, hash2)) { /* ... */ }
```

### 4. Implement Rate Limiting

```typescript
// ❌ BAD
async function login(address: string, password: string) {
  const keystore = getKeystore(address);
  return await decryptKeystore(keystore, password);
}

// ✅ GOOD
const loginLimiter = new RateLimiter(5, 300000);

async function login(address: string, password: string) {
  if (!loginLimiter.isAllowed(address)) {
    throw new Error('Too many login attempts');
  }

  try {
    const keystore = getKeystore(address);
    const result = await decryptKeystore(keystore, password);
    loginLimiter.reset(address);
    return result;
  } catch (error) {
    throw new Error('Invalid password');
  }
}
```

### 5. Wipe Sensitive Data

```typescript
// ❌ BAD
const mnemonic = await decryptMnemonic(encrypted, password);
const privateKey = await derivePrivateKey(mnemonic);
const signature = sign(data, privateKey);
return signature;

// ✅ GOOD
const mnemonic = await decryptMnemonic(encrypted, password);
try {
  const privateKey = await derivePrivateKey(mnemonic);
  try {
    const signature = sign(data, privateKey);
    return signature;
  } finally {
    secureWipe(privateKey);
  }
} finally {
  // Can't truly wipe strings in JS, but best effort
  secureWipeString(mnemonic);
}
```

## Common Patterns

### Wallet Creation

```typescript
import {
  generateMnemonic,
  validatePasswordStrength,
  encryptMnemonic,
  derivePrivateKey,
  encryptKeystore,
  publicKeyToAddress,
  derivePublicKey,
} from '@paw-chain/wallet-core';

async function createWallet(password: string) {
  // 1. Validate password
  const validation = validatePasswordStrength(password);
  if (!validation.valid) {
    throw new Error(validation.errors.join(', '));
  }

  // 2. Generate mnemonic
  const mnemonic = generateMnemonic(256); // 24 words

  // 3. Encrypt mnemonic
  const encryptedMnemonic = await encryptMnemonic(mnemonic, password);

  // 4. Derive keys
  const privateKey = await derivePrivateKey(mnemonic);
  const publicKey = derivePublicKey(privateKey);
  const address = publicKeyToAddress(publicKey);

  // 5. Create keystore
  const keystore = await encryptKeystore(privateKey, password, address);

  // 6. Wipe sensitive data
  secureWipe(privateKey);

  return {
    mnemonic: encryptedMnemonic,
    keystore,
    address,
  };
}
```

### Wallet Recovery

```typescript
async function recoverWallet(mnemonic: string, password: string) {
  // 1. Validate mnemonic
  if (!validateMnemonic(mnemonic)) {
    throw new Error('Invalid mnemonic phrase');
  }

  // 2. Validate password
  const validation = validatePasswordStrength(password);
  if (!validation.valid) {
    throw new Error(validation.errors.join(', '));
  }

  // 3. Derive keys
  const privateKey = await derivePrivateKey(mnemonic);
  const publicKey = derivePublicKey(privateKey);
  const address = publicKeyToAddress(publicKey);

  // 4. Encrypt mnemonic
  const encryptedMnemonic = await encryptMnemonic(mnemonic, password);

  // 5. Create keystore
  const keystore = await encryptKeystore(privateKey, password, address);

  // 6. Wipe sensitive data
  secureWipe(privateKey);

  return {
    mnemonic: encryptedMnemonic,
    keystore,
    address,
  };
}
```

### Transaction Signing

```typescript
async function signTransaction(keystore: SecureKeystore, password: string, tx: Transaction) {
  // 1. Decrypt private key
  const privateKey = await decryptKeystore(keystore, password);

  try {
    // 2. Sign transaction
    const signature = signData(serializeTx(tx), privateKey);
    return signature;
  } finally {
    // 3. Always wipe private key
    secureWipe(privateKey);
  }
}
```

## Migration Checklist

- [ ] Remove crypto-js dependency
- [ ] Install @noble packages
- [ ] Update all encryption calls to async/await
- [ ] Handle new EncryptedData format
- [ ] Update password validation (12 char min)
- [ ] Implement password strength checking
- [ ] Add rate limiting for password attempts
- [ ] Wipe sensitive data after use
- [ ] Test all encryption/decryption flows
- [ ] Migrate existing keystores
- [ ] Update documentation

## Security Constants

```typescript
// From keyDerivation.ts
MIN_PBKDF2_ITERATIONS = 100,000        // Absolute minimum
RECOMMENDED_PBKDF2_ITERATIONS = 210,000 // OWASP 2023 standard

// Password requirements
MIN_PASSWORD_LENGTH = 12
REQUIRED_CHAR_TYPES = ['lowercase', 'uppercase', 'numbers', 'special']

// Rate limiting defaults
DEFAULT_MAX_ATTEMPTS = 5
DEFAULT_WINDOW_MS = 300000 // 5 minutes
```

## Error Handling

```typescript
try {
  const decrypted = await decryptAES(encrypted, password);
} catch (error) {
  if (error.message.includes('invalid password')) {
    // Wrong password
  } else if (error.message.includes('corrupted')) {
    // Tampered data
  } else {
    // Other error
  }
}
```

## Performance Tips

1. **Cache derived keys during session** (don't re-derive for each operation)
2. **Use Web Workers** for key derivation in browser
3. **Show progress indicators** (KDF takes ~210ms)
4. **Consider Argon2id** for better performance/security ratio
5. **Benchmark on target hardware** using `calculateOptimalIterations()`

## Support

- **Documentation:** See MIGRATION_GUIDE.md
- **Security Issues:** security@paw-chain.io
- **Bug Reports:**  Issues

---

**Last Updated:** 2025-11-25
**Version:** 2.0.0
