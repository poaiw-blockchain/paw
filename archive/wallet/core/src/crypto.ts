/**
 * Cryptographic utilities for PAW Wallet
 * Implements BIP39/BIP32/BIP44 standards for HD wallets
 */

import * as bip39 from 'bip39';
import * as bip32 from 'bip32';
import { sha256, ripemd160 } from '@cosmjs/crypto';
import { fromHex, toHex, toBech32, fromBech32 } from '@cosmjs/encoding';
import { webcrypto } from 'crypto';
import { HDPath, KeyPair, WalletAccount, EncryptedData } from './types';
import { ec as EC } from 'elliptic';
import { secureRandom } from './security';
import { deriveKeyBytes, RECOMMENDED_PBKDF2_ITERATIONS } from './keyDerivation';

const secp256k1 = new EC('secp256k1');

// PAW uses Cosmos coin type 118 for compatibility
export const DEFAULT_COIN_TYPE = 118;
export const DEFAULT_HD_PATH = "m/44'/118'/0'/0/0";
export const PAW_BECH32_PREFIX = 'paw';

/**
 * Generate a new mnemonic phrase using secure entropy
 * Uses Web Crypto API for cryptographically secure randomness
 * @param strength - Entropy strength in bits (128, 160, 192, 224, 256)
 * @returns BIP39 mnemonic phrase
 */
export function generateMnemonic(strength: 128 | 160 | 192 | 224 | 256 = 256): string {
  // Use Web Crypto API for secure entropy generation
  const entropy = secureRandom(strength / 8);
  return bip39.entropyToMnemonic(Buffer.from(entropy));
}

/**
 * Validate a mnemonic phrase
 * @param mnemonic - BIP39 mnemonic phrase
 * @returns true if valid, false otherwise
 */
export function validateMnemonic(mnemonic: string): boolean {
  return bip39.validateMnemonic(mnemonic);
}

/**
 * Convert mnemonic to seed
 * @param mnemonic - BIP39 mnemonic phrase
 * @param password - Optional passphrase for additional security
 * @returns Seed buffer
 */
export async function mnemonicToSeed(mnemonic: string, password: string = ''): Promise<Buffer> {
  if (!validateMnemonic(mnemonic)) {
    throw new Error('Invalid mnemonic phrase');
  }
  return bip39.mnemonicToSeed(mnemonic, password);
}

/**
 * Derive HD path string from components
 * @param path - HD path components
 * @returns HD path string (e.g., "m/44'/118'/0'/0/0")
 */
export function deriveHDPath(path: Partial<HDPath> = {}): string {
  const {
    coinType = DEFAULT_COIN_TYPE,
    account = 0,
    change = 0,
    addressIndex = 0,
  } = path;

  return `m/44'/${coinType}'/${account}'/${change}/${addressIndex}`;
}

/**
 * Parse HD path string into components
 * @param pathString - HD path string
 * @returns HD path components
 */
export function parseHDPath(pathString: string): HDPath {
  const parts = pathString.split('/').slice(1); // Remove 'm'

  if (parts.length !== 5) {
    throw new Error('Invalid HD path format');
  }

  return {
    coinType: parseInt(parts[1].replace("'", '')),
    account: parseInt(parts[2].replace("'", '')),
    change: parseInt(parts[3]),
    addressIndex: parseInt(parts[4]),
  };
}

/**
 * Derive private key from mnemonic and HD path
 * @param mnemonic - BIP39 mnemonic phrase
 * @param hdPath - HD derivation path
 * @param password - Optional mnemonic passphrase
 * @returns Private key as Uint8Array
 */
export async function derivePrivateKey(
  mnemonic: string,
  hdPath: string = DEFAULT_HD_PATH,
  password: string = ''
): Promise<Uint8Array> {
  const seed = await mnemonicToSeed(mnemonic, password);
  const masterKey = bip32.fromSeed(seed);
  const childKey = masterKey.derivePath(hdPath);

  if (!childKey.privateKey) {
    throw new Error('Failed to derive private key');
  }

  return new Uint8Array(childKey.privateKey);
}

/**
 * Derive public key from private key
 * @param privateKey - Private key
 * @returns Public key (compressed, 33 bytes)
 */
export function derivePublicKey(privateKey: Uint8Array): Uint8Array {
  const keyPair = secp256k1.keyFromPrivate(Buffer.from(privateKey));
  const publicKey = keyPair.getPublic(true, 'array');
  return new Uint8Array(publicKey);
}

/**
 * Derive key pair from private key
 * @param privateKey - Private key
 * @returns Key pair object
 */
export function deriveKeyPair(privateKey: Uint8Array): KeyPair {
  return {
    privateKey,
    publicKey: derivePublicKey(privateKey),
  };
}

/**
 * Convert public key to address
 * @param publicKey - Public key (compressed)
 * @param prefix - Bech32 prefix (default: 'paw')
 * @returns Bech32 address
 */
export function publicKeyToAddress(publicKey: Uint8Array, prefix: string = PAW_BECH32_PREFIX): string {
  const hash1 = sha256(publicKey);
  const hash2 = ripemd160(hash1);
  return toBech32(prefix, hash2);
}

/**
 * Validate address format
 * @param address - Bech32 address
 * @param prefix - Expected prefix
 * @returns true if valid, false otherwise
 */
export function validateAddress(address: string, prefix: string = PAW_BECH32_PREFIX): boolean {
  try {
    const { prefix: actualPrefix, data } = fromBech32(address);
    return actualPrefix === prefix && data.length === 20;
  } catch {
    return false;
  }
}

/**
 * Create wallet account from mnemonic
 * @param mnemonic - BIP39 mnemonic phrase
 * @param hdPath - HD derivation path
 * @param password - Optional mnemonic passphrase
 * @param prefix - Bech32 prefix
 * @returns Wallet account
 */
export async function createWalletAccount(
  mnemonic: string,
  hdPath: string = DEFAULT_HD_PATH,
  password: string = '',
  prefix: string = PAW_BECH32_PREFIX
): Promise<WalletAccount> {
  const privateKey = await derivePrivateKey(mnemonic, hdPath, password);
  const publicKey = derivePublicKey(privateKey);
  const address = publicKeyToAddress(publicKey, prefix);

  return {
    address,
    pubkey: publicKey,
    algo: 'secp256k1',
  };
}

/**
 * Sign data with private key
 * @param data - Data to sign
 * @param privateKey - Private key
 * @returns Signature
 */
export function signData(data: Uint8Array, privateKey: Uint8Array): Uint8Array {
  const messageHash = sha256(data);
  const keyPair = secp256k1.keyFromPrivate(Buffer.from(privateKey));
  const signature = keyPair.sign(messageHash, { canonical: true });

  // Convert to compact format (r + s)
  const r = signature.r.toArrayLike(Buffer, 'be', 32);
  const s = signature.s.toArrayLike(Buffer, 'be', 32);

  return new Uint8Array([...r, ...s]);
}

/**
 * Verify signature
 * @param data - Original data
 * @param signature - Signature to verify
 * @param publicKey - Public key
 * @returns true if signature is valid
 */
export function verifySignature(data: Uint8Array, signature: Uint8Array, publicKey: Uint8Array): boolean {
  try {
    const messageHash = sha256(data);
    const keyPair = secp256k1.keyFromPublic(Buffer.from(publicKey));

    const r = signature.slice(0, 32);
    const s = signature.slice(32, 64);

    return keyPair.verify(messageHash, { r: Buffer.from(r), s: Buffer.from(s) });
  } catch {
    return false;
  }
}

/**
 * Generate cryptographically secure random bytes
 * Uses Web Crypto API for secure randomness
 * @param length - Number of bytes to generate
 * @returns Random bytes
 */
export function randomBytes(length: number): Uint8Array {
  return secureRandom(length);
}

/**
 * Hash data using SHA-256
 * @param data - Data to hash
 * @returns Hash
 */
export function hash256(data: Uint8Array): Uint8Array {
  return sha256(data);
}

/**
 * Encode data to hex string
 * @param data - Data to encode
 * @returns Hex string
 */
export function toHexString(data: Uint8Array): string {
  return toHex(data);
}

/**
 * Decode hex string to bytes
 * @param hex - Hex string
 * @returns Bytes
 */
export function fromHexString(hex: string): Uint8Array {
  return fromHex(hex);
}

/**
 * Encrypt data with AES-256-GCM using Web Crypto API
 * SECURE: Uses proper KDF (PBKDF2), unique IV, and authenticated encryption
 * @param data - Data to encrypt
 * @param password - Encryption password
 * @returns Encrypted data with metadata
 */
export async function encryptAES(data: string, password: string): Promise<EncryptedData> {
  // Generate random salt and IV
  const salt = secureRandom(32);
  const iv = secureRandom(12); // 12 bytes for GCM

  // Derive key from password using PBKDF2
  const keyMaterial = await webcrypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  const key = await webcrypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: RECOMMENDED_PBKDF2_ITERATIONS,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );

  // Encrypt data
  const encrypted = await webcrypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: iv,
    },
    key,
    new TextEncoder().encode(data)
  );

  // Return encrypted data with all necessary parameters
  return {
    ciphertext: Buffer.from(encrypted).toString('base64'),
    salt: Buffer.from(salt).toString('base64'),
    iv: Buffer.from(iv).toString('base64'),
    algorithm: 'AES-256-GCM',
    kdf: 'PBKDF2',
    iterations: RECOMMENDED_PBKDF2_ITERATIONS,
  };
}

/**
 * Decrypt data with AES-256-GCM using Web Crypto API
 * @param encryptedData - Encrypted data object
 * @param password - Decryption password
 * @returns Decrypted data
 */
export async function decryptAES(encryptedData: EncryptedData, password: string): Promise<string> {
  // Decode salt, IV, and ciphertext from base64
  const salt = Buffer.from(encryptedData.salt, 'base64');
  const iv = Buffer.from(encryptedData.iv, 'base64');
  const ciphertext = Buffer.from(encryptedData.ciphertext, 'base64');

  // Derive same key from password
  const keyMaterial = await webcrypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  const key = await webcrypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: encryptedData.iterations,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );

  // Decrypt
  try {
    const decrypted = await webcrypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: iv,
      },
      key,
      ciphertext
    );

    return new TextDecoder().decode(decrypted);
  } catch (error) {
    throw new Error('Decryption failed: invalid password or corrupted data');
  }
}

/**
 * Encrypt mnemonic phrase securely
 * @param mnemonic - Mnemonic phrase to encrypt
 * @param password - Encryption password
 * @returns Encrypted mnemonic
 */
export async function encryptMnemonic(mnemonic: string, password: string): Promise<EncryptedData> {
  if (!validateMnemonic(mnemonic)) {
    throw new Error('Invalid mnemonic phrase');
  }
  return await encryptAES(mnemonic, password);
}

/**
 * Decrypt mnemonic phrase
 * @param encryptedData - Encrypted mnemonic
 * @param password - Decryption password
 * @returns Decrypted mnemonic
 */
export async function decryptMnemonic(
  encryptedData: EncryptedData,
  password: string
): Promise<string> {
  const mnemonic = await decryptAES(encryptedData, password);
  if (!validateMnemonic(mnemonic)) {
    throw new Error('Decrypted data is not a valid mnemonic');
  }
  return mnemonic;
}
