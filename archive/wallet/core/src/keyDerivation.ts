/**
 * Secure key derivation functions for PAW Wallet
 * Implements PBKDF2 and Argon2id for password-based key derivation
 */

import { webcrypto } from 'crypto';
import { KeyDerivationParams } from './types';
import { secureRandom } from './security';

// OWASP recommended minimum iterations for PBKDF2-SHA256
export const MIN_PBKDF2_ITERATIONS = 100000;
export const RECOMMENDED_PBKDF2_ITERATIONS = 210000; // OWASP 2023 recommendation

/**
 * Derive cryptographic key from password using PBKDF2
 * @param params - Key derivation parameters
 * @returns Derived CryptoKey
 */
export async function deriveKey(params: KeyDerivationParams): Promise<CryptoKey> {
  const {
    password,
    salt = secureRandom(32),
    iterations = RECOMMENDED_PBKDF2_ITERATIONS,
    keyLength = 256,
  } = params;

  if (iterations < MIN_PBKDF2_ITERATIONS) {
    throw new Error(
      `Iterations must be at least ${MIN_PBKDF2_ITERATIONS} for security. Recommended: ${RECOMMENDED_PBKDF2_ITERATIONS}`
    );
  }

  // Import password as key material
  const keyMaterial = await webcrypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  // Derive key using PBKDF2
  const derivedKey = await webcrypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: iterations,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: keyLength },
    false,
    ['encrypt', 'decrypt']
  );

  return derivedKey;
}

/**
 * Derive raw key bytes from password using PBKDF2
 * Useful when you need the raw key material instead of CryptoKey
 * @param password - Password
 * @param salt - Salt (32 bytes recommended)
 * @param iterations - Number of iterations
 * @param keyLength - Key length in bytes
 * @returns Derived key bytes
 */
export async function deriveKeyBytes(
  password: string,
  salt: Uint8Array,
  iterations: number = RECOMMENDED_PBKDF2_ITERATIONS,
  keyLength: number = 32
): Promise<Uint8Array> {
  if (iterations < MIN_PBKDF2_ITERATIONS) {
    throw new Error(
      `Iterations must be at least ${MIN_PBKDF2_ITERATIONS} for security`
    );
  }

  // Import password as key material
  const keyMaterial = await webcrypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveBits']
  );

  // Derive raw bits
  const derivedBits = await webcrypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: iterations,
      hash: 'SHA-256',
    },
    keyMaterial,
    keyLength * 8 // Convert bytes to bits
  );

  return new Uint8Array(derivedBits);
}

/**
 * Derive key from password using Argon2id (more secure than PBKDF2)
 * Requires @noble/hashes package
 * @param password - Password
 * @param salt - Salt (16 bytes minimum)
 * @param memorySize - Memory size in KB (default: 65536 = 64MB)
 * @param iterations - Number of iterations (default: 3)
 * @param parallelism - Degree of parallelism (default: 4)
 * @param keyLength - Output key length in bytes (default: 32)
 * @returns Derived key
 */
export async function deriveKeyArgon2(
  password: string,
  salt?: Uint8Array,
  memorySize: number = 65536,
  iterations: number = 3,
  parallelism: number = 4,
  keyLength: number = 32
): Promise<Uint8Array> {
  try {
    // Dynamic import to avoid bundling if not used
    const { argon2id } = await import('@noble/hashes/argon2');

    const actualSalt = salt || secureRandom(16);

    return argon2id(new TextEncoder().encode(password), actualSalt, {
      m: memorySize,
      t: iterations,
      p: parallelism,
      dkLen: keyLength,
    });
  } catch (error) {
    throw new Error(
      'Argon2 not available. Install @noble/hashes: npm install @noble/hashes'
    );
  }
}

/**
 * Derive encryption and MAC keys from password
 * Returns two separate keys for encrypt-then-MAC pattern
 * @param password - Password
 * @param salt - Salt
 * @param iterations - PBKDF2 iterations
 * @returns Object with encryption key and MAC key
 */
export async function deriveEncryptionAndMacKeys(
  password: string,
  salt: Uint8Array,
  iterations: number = RECOMMENDED_PBKDF2_ITERATIONS
): Promise<{ encryptionKey: Uint8Array; macKey: Uint8Array }> {
  // Derive 64 bytes: 32 for encryption, 32 for MAC
  const derivedKey = await deriveKeyBytes(password, salt, iterations, 64);

  return {
    encryptionKey: derivedKey.slice(0, 32),
    macKey: derivedKey.slice(32, 64),
  };
}

/**
 * Generate salt for key derivation
 * @param bytes - Number of bytes (default: 32)
 * @returns Random salt
 */
export function generateSalt(bytes: number = 32): Uint8Array {
  return secureRandom(bytes);
}

/**
 * Calculate optimal PBKDF2 iterations for target delay
 * Benchmarks the system and calculates iterations needed
 * @param targetDelayMs - Target delay in milliseconds (default: 500ms)
 * @returns Recommended number of iterations
 */
export async function calculateOptimalIterations(
  targetDelayMs: number = 500
): Promise<number> {
  const testPassword = 'test-password-for-benchmarking';
  const testSalt = secureRandom(32);
  const testIterations = 10000;

  const start = performance.now();
  await deriveKeyBytes(testPassword, testSalt, testIterations, 32);
  const end = performance.now();

  const timeFor10k = end - start;
  const iterationsPerMs = testIterations / timeFor10k;
  const optimalIterations = Math.floor(iterationsPerMs * targetDelayMs);

  // Ensure we meet minimum requirements
  return Math.max(optimalIterations, MIN_PBKDF2_ITERATIONS);
}

/**
 * Verify derived key matches expected value (constant-time)
 * @param password - Password to test
 * @param salt - Salt used for derivation
 * @param iterations - Number of iterations
 * @param expectedKey - Expected derived key
 * @returns true if match
 */
export async function verifyDerivedKey(
  password: string,
  salt: Uint8Array,
  iterations: number,
  expectedKey: Uint8Array
): Promise<boolean> {
  const derivedKey = await deriveKeyBytes(password, salt, iterations, expectedKey.length);

  // Constant-time comparison to prevent timing attacks
  if (derivedKey.length !== expectedKey.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < derivedKey.length; i++) {
    result |= derivedKey[i] ^ expectedKey[i];
  }

  return result === 0;
}

/**
 * Key derivation info for debugging/auditing
 * @param iterations - Number of iterations
 * @returns Info object
 */
export function getKeyDerivationInfo(iterations: number): {
  iterations: number;
  estimatedTimeMs: number;
  securityLevel: 'weak' | 'acceptable' | 'good' | 'excellent';
  meetsOWASP: boolean;
} {
  // Rough estimate: 1ms per 1000 iterations on average hardware
  const estimatedTimeMs = (iterations / 1000) * 1;

  let securityLevel: 'weak' | 'acceptable' | 'good' | 'excellent';
  if (iterations < 50000) {
    securityLevel = 'weak';
  } else if (iterations < MIN_PBKDF2_ITERATIONS) {
    securityLevel = 'acceptable';
  } else if (iterations < RECOMMENDED_PBKDF2_ITERATIONS) {
    securityLevel = 'good';
  } else {
    securityLevel = 'excellent';
  }

  return {
    iterations,
    estimatedTimeMs,
    securityLevel,
    meetsOWASP: iterations >= RECOMMENDED_PBKDF2_ITERATIONS,
  };
}

/**
 * Compare two key derivation configurations
 * @param config1 - First configuration
 * @param config2 - Second configuration
 * @returns Comparison result
 */
export function compareKeyDerivationSecurity(
  config1: { kdf: string; iterations: number },
  config2: { kdf: string; iterations: number }
): 'stronger' | 'weaker' | 'equal' {
  // Argon2 is generally stronger than PBKDF2
  if (config1.kdf === 'Argon2id' && config2.kdf === 'PBKDF2') {
    return 'stronger';
  }
  if (config1.kdf === 'PBKDF2' && config2.kdf === 'Argon2id') {
    return 'weaker';
  }

  // Same KDF, compare iterations
  if (config1.iterations > config2.iterations) {
    return 'stronger';
  } else if (config1.iterations < config2.iterations) {
    return 'weaker';
  }

  return 'equal';
}
