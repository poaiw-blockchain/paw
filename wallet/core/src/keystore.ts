/**
 * Keystore management for secure private key storage
 * Implements secure keystore with Web Crypto API
 */

import { webcrypto } from 'crypto';
import {
  secureRandom,
  constantTimeCompare,
  generateUUID,
  hmacSha256,
} from './security';
import { deriveEncryptionAndMacKeys, RECOMMENDED_PBKDF2_ITERATIONS } from './keyDerivation';

const KEYSTORE_VERSION = 4; // Updated version for new secure format
const DEFAULT_ITERATIONS = RECOMMENDED_PBKDF2_ITERATIONS;
const DEFAULT_CIPHER = 'AES-256-GCM';
const DEFAULT_KDF = 'PBKDF2';
type SecureKeystore = import('./types').SecureKeystore;

/**
 * Encrypt private key into secure keystore format using Web Crypto API
 * @param privateKey - Private key to encrypt
 * @param password - Encryption password
 * @param address - Associated address
 * @param name - Optional wallet name
 * @returns Secure keystore object
 */
export async function encryptKeystore(
  privateKey: Uint8Array,
  password: string,
  address: string,
  name?: string
): Promise<SecureKeystore> {
  if (password.length < 12) {
    throw new Error('Password must be at least 12 characters for security');
  }

  // Generate random salt and IV
  const salt = secureRandom(32);
  const iv = secureRandom(12); // 12 bytes for GCM

  // Derive encryption and MAC keys using PBKDF2
  const { encryptionKey, macKey } = await deriveEncryptionAndMacKeys(
    password,
    salt,
    DEFAULT_ITERATIONS
  );

  // Import encryption key
  const cryptoKey = await webcrypto.subtle.importKey(
    'raw',
    encryptionKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt']
  );

  // Encrypt private key
  const encrypted = await webcrypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: iv,
    },
    cryptoKey,
    privateKey
  );

  const ciphertext = Buffer.from(encrypted).toString('base64');

  // Calculate HMAC for integrity (encrypt-then-MAC pattern)
  const mac = await hmacSha256(macKey, new Uint8Array(encrypted));

  // Generate unique ID
  const id = generateUUID();

  return {
    version: KEYSTORE_VERSION,
    crypto: {
      cipher: DEFAULT_CIPHER,
      ciphertext,
      kdf: DEFAULT_KDF,
      kdfparams: {
        salt: Buffer.from(salt).toString('base64'),
        iterations: DEFAULT_ITERATIONS,
        dklen: 64, // 32 for encryption + 32 for MAC
      },
      mac: Buffer.from(mac).toString('hex'),
      iv: Buffer.from(iv).toString('base64'),
    },
    id,
    address,
    meta: {
      name,
      timestamp: Date.now(),
    },
  };
}

/**
 * Decrypt secure keystore to retrieve private key using Web Crypto API
 * @param keystore - Secure keystore object
 * @param password - Decryption password
 * @returns Decrypted private key
 */
export async function decryptKeystore(
  keystore: SecureKeystore,
  password: string
): Promise<Uint8Array> {
  // Validate keystore format
  if (keystore.version !== KEYSTORE_VERSION) {
    // Support legacy keystores (version 3)
    if (keystore.version === 3) {
      throw new Error(
        'Legacy keystore detected. Please migrate using the migration tool.'
      );
    }
    throw new Error(`Unsupported keystore version: ${keystore.version}`);
  }

  const { crypto } = keystore;

  if (crypto.kdf !== DEFAULT_KDF && crypto.kdf !== 'Argon2id') {
    throw new Error(`Unsupported KDF: ${crypto.kdf}`);
  }

  // Decode parameters
  const salt = Buffer.from(crypto.kdfparams.salt, 'base64');
  const iv = Buffer.from(crypto.iv, 'base64');
  const ciphertext = Buffer.from(crypto.ciphertext, 'base64');
  const expectedMac = Buffer.from(crypto.mac, 'hex');

  // Derive encryption and MAC keys
  const { encryptionKey, macKey } = await deriveEncryptionAndMacKeys(
    password,
    salt,
    crypto.kdfparams.iterations
  );

  // Verify HMAC (constant-time comparison to prevent timing attacks)
  const calculatedMac = await hmacSha256(macKey, ciphertext);

  if (!constantTimeCompare(calculatedMac, expectedMac)) {
    throw new Error('Invalid password or corrupted keystore');
  }

  // Import decryption key
  const cryptoKey = await webcrypto.subtle.importKey(
    'raw',
    encryptionKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['decrypt']
  );

  // Decrypt private key
  try {
    const decrypted = await webcrypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: iv,
      },
      cryptoKey,
      ciphertext
    );

    return new Uint8Array(decrypted);
  } catch (error) {
    throw new Error('Failed to decrypt keystore: invalid password or corrupted data');
  }
}


/**
 * Export secure keystore to JSON string
 * @param keystore - Secure keystore object
 * @param pretty - Pretty print JSON
 * @returns JSON string
 */
export function exportKeystore(keystore: SecureKeystore, pretty: boolean = false): string {
  return JSON.stringify(keystore, null, pretty ? 2 : 0);
}

/**
 * Import secure keystore from JSON string
 * @param json - JSON string
 * @returns Secure keystore object
 */
export function importKeystore(json: string): SecureKeystore {
  try {
    const keystore = JSON.parse(json);

    // Validate required fields
    if (!keystore.version || !keystore.crypto || !keystore.id || !keystore.address) {
      throw new Error('Invalid keystore format');
    }

    return keystore as SecureKeystore;
  } catch (error) {
    throw new Error('Failed to parse keystore JSON');
  }
}

/**
 * Validate secure keystore structure
 * @param keystore - Secure keystore object
 * @returns true if valid
 */
export function validateKeystore(keystore: SecureKeystore): boolean {
  try {
    // Check version
    if (keystore.version !== KEYSTORE_VERSION && keystore.version !== 3) {
      return false;
    }

    // Check crypto object
    const { crypto } = keystore;
    if (
      !crypto.cipher ||
      !crypto.ciphertext ||
      !crypto.kdf ||
      !crypto.kdfparams ||
      !crypto.mac ||
      !crypto.iv
    ) {
      return false;
    }

    // Check KDF params
    if (
      !crypto.kdfparams.salt ||
      !crypto.kdfparams.iterations ||
      !crypto.kdfparams.dklen
    ) {
      return false;
    }

    // Check ID and address
    if (!keystore.id || !keystore.address) {
      return false;
    }

    return true;
  } catch {
    return false;
  }
}

/**
 * Change keystore password
 * @param keystore - Original secure keystore
 * @param oldPassword - Current password
 * @param newPassword - New password
 * @returns New keystore with updated password
 */
export async function changeKeystorePassword(
  keystore: SecureKeystore,
  oldPassword: string,
  newPassword: string
): Promise<SecureKeystore> {
  if (newPassword.length < 12) {
    throw new Error('New password must be at least 12 characters for security');
  }

  // Decrypt with old password
  const privateKey = await decryptKeystore(keystore, oldPassword);

  // Re-encrypt with new password
  return encryptKeystore(privateKey, newPassword, keystore.address, keystore.meta?.name);
}

/**
 * Generate keystore filename
 * @param address - Wallet address
 * @param timestamp - Optional timestamp
 * @returns Filename string
 */
export function generateKeystoreFilename(address: string, timestamp?: number): string {
  const ts = timestamp || Date.now();
  const date = new Date(ts).toISOString().replace(/:/g, '-').split('.')[0];
  return `UTC--${date}--${address}.json`;
}

/**
 * Verify keystore password without decrypting
 * @param keystore - Secure keystore object
 * @param password - Password to verify
 * @returns true if password is correct
 */
export async function verifyKeystorePassword(
  keystore: SecureKeystore,
  password: string
): Promise<boolean> {
  try {
    await decryptKeystore(keystore, password);
    return true;
  } catch {
    return false;
  }
}

/**
 * Estimate keystore decryption time
 * @param iterations - Number of PBKDF2 iterations
 * @returns Estimated time in milliseconds
 */
export function estimateDecryptionTime(iterations: number): number {
  // Rough estimate: ~1ms per 1000 iterations on average hardware
  return (iterations / 1000) * 1;
}

/**
 * Get keystore security level
 * @param keystore - Secure keystore object
 * @returns Security level (low, medium, high, excellent)
 */
export function getKeystoreSecurityLevel(
  keystore: SecureKeystore
): 'low' | 'medium' | 'high' | 'excellent' {
  const iterations = keystore.crypto.kdfparams.iterations;

  if (iterations < 50000) {
    return 'low';
  } else if (iterations < 100000) {
    return 'medium';
  } else if (iterations < RECOMMENDED_PBKDF2_ITERATIONS) {
    return 'high';
  } else {
    return 'excellent';
  }
}

/**
 * Sanitize keystore for logging (removes sensitive data)
 * @param keystore - Secure keystore object
 * @returns Sanitized keystore (safe to log)
 */
export function sanitizeKeystore(keystore: SecureKeystore): Partial<SecureKeystore> {
  return {
    version: keystore.version,
    id: keystore.id,
    address: keystore.address,
    meta: keystore.meta,
    crypto: {
      cipher: keystore.crypto.cipher,
      kdf: keystore.crypto.kdf,
      kdfparams: keystore.crypto.kdfparams,
      iv: '[REDACTED]',
      ciphertext: '[REDACTED]',
      mac: '[REDACTED]',
    },
  };
}
