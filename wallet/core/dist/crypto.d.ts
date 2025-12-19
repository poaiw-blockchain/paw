/**
 * Cryptographic utilities for PAW Wallet
 * Implements BIP39/BIP32/BIP44 standards for HD wallets
 */
import { HDPath, KeyPair, WalletAccount, EncryptedData } from './types';
export declare const DEFAULT_COIN_TYPE = 118;
export declare const DEFAULT_HD_PATH = "m/44'/118'/0'/0/0";
export declare const PAW_BECH32_PREFIX = "paw";
/**
 * Generate a new mnemonic phrase using secure entropy
 * Uses Web Crypto API for cryptographically secure randomness
 * @param strength - Entropy strength in bits (128, 160, 192, 224, 256)
 * @returns BIP39 mnemonic phrase
 */
export declare function generateMnemonic(strength?: 128 | 160 | 192 | 224 | 256): string;
/**
 * Validate a mnemonic phrase
 * @param mnemonic - BIP39 mnemonic phrase
 * @returns true if valid, false otherwise
 */
export declare function validateMnemonic(mnemonic: string): boolean;
/**
 * Convert mnemonic to seed
 * @param mnemonic - BIP39 mnemonic phrase
 * @param password - Optional passphrase for additional security
 * @returns Seed buffer
 */
export declare function mnemonicToSeed(mnemonic: string, password?: string): Promise<Buffer>;
/**
 * Derive HD path string from components
 * @param path - HD path components
 * @returns HD path string (e.g., "m/44'/118'/0'/0/0")
 */
export declare function deriveHDPath(path?: Partial<HDPath>): string;
/**
 * Parse HD path string into components
 * @param pathString - HD path string
 * @returns HD path components
 */
export declare function parseHDPath(pathString: string): HDPath;
/**
 * Derive private key from mnemonic and HD path
 * @param mnemonic - BIP39 mnemonic phrase
 * @param hdPath - HD derivation path
 * @param password - Optional mnemonic passphrase
 * @returns Private key as Uint8Array
 */
export declare function derivePrivateKey(mnemonic: string, hdPath?: string, password?: string): Promise<Uint8Array>;
/**
 * Derive public key from private key
 * @param privateKey - Private key
 * @returns Public key (compressed, 33 bytes)
 */
export declare function derivePublicKey(privateKey: Uint8Array): Uint8Array;
/**
 * Derive key pair from private key
 * @param privateKey - Private key
 * @returns Key pair object
 */
export declare function deriveKeyPair(privateKey: Uint8Array): KeyPair;
/**
 * Convert public key to address
 * @param publicKey - Public key (compressed)
 * @param prefix - Bech32 prefix (default: 'paw')
 * @returns Bech32 address
 */
export declare function publicKeyToAddress(publicKey: Uint8Array, prefix?: string): string;
/**
 * Validate address format
 * @param address - Bech32 address
 * @param prefix - Expected prefix
 * @returns true if valid, false otherwise
 */
export declare function validateAddress(address: string, prefix?: string): boolean;
/**
 * Create wallet account from mnemonic
 * @param mnemonic - BIP39 mnemonic phrase
 * @param hdPath - HD derivation path
 * @param password - Optional mnemonic passphrase
 * @param prefix - Bech32 prefix
 * @returns Wallet account
 */
export declare function createWalletAccount(mnemonic: string, hdPath?: string, password?: string, prefix?: string): Promise<WalletAccount>;
/**
 * Sign data with private key
 * @param data - Data to sign
 * @param privateKey - Private key
 * @returns Signature
 */
export declare function signData(data: Uint8Array, privateKey: Uint8Array): Uint8Array;
/**
 * Verify signature
 * @param data - Original data
 * @param signature - Signature to verify
 * @param publicKey - Public key
 * @returns true if signature is valid
 */
export declare function verifySignature(data: Uint8Array, signature: Uint8Array, publicKey: Uint8Array): boolean;
/**
 * Generate cryptographically secure random bytes
 * Uses Web Crypto API for secure randomness
 * @param length - Number of bytes to generate
 * @returns Random bytes
 */
export declare function randomBytes(length: number): Uint8Array;
/**
 * Hash data using SHA-256
 * @param data - Data to hash
 * @returns Hash
 */
export declare function hash256(data: Uint8Array): Uint8Array;
/**
 * Encode data to hex string
 * @param data - Data to encode
 * @returns Hex string
 */
export declare function toHexString(data: Uint8Array): string;
/**
 * Decode hex string to bytes
 * @param hex - Hex string
 * @returns Bytes
 */
export declare function fromHexString(hex: string): Uint8Array;
/**
 * Encrypt data with AES-256-GCM using Web Crypto API
 * SECURE: Uses proper KDF (PBKDF2), unique IV, and authenticated encryption
 * @param data - Data to encrypt
 * @param password - Encryption password
 * @returns Encrypted data with metadata
 */
export declare function encryptAES(data: string, password: string): Promise<EncryptedData>;
/**
 * Decrypt data with AES-256-GCM using Web Crypto API
 * @param encryptedData - Encrypted data object
 * @param password - Decryption password
 * @returns Decrypted data
 */
export declare function decryptAES(encryptedData: EncryptedData, password: string): Promise<string>;
/**
 * Encrypt mnemonic phrase securely
 * @param mnemonic - Mnemonic phrase to encrypt
 * @param password - Encryption password
 * @returns Encrypted mnemonic
 */
export declare function encryptMnemonic(mnemonic: string, password: string): Promise<EncryptedData>;
/**
 * Decrypt mnemonic phrase
 * @param encryptedData - Encrypted mnemonic
 * @param password - Decryption password
 * @returns Decrypted mnemonic
 */
export declare function decryptMnemonic(encryptedData: EncryptedData, password: string): Promise<string>;
//# sourceMappingURL=crypto.d.ts.map