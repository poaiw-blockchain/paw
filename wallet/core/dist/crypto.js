"use strict";
/**
 * Cryptographic utilities for PAW Wallet
 * Implements BIP39/BIP32/BIP44 standards for HD wallets
 */
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.PAW_BECH32_PREFIX = exports.DEFAULT_HD_PATH = exports.DEFAULT_COIN_TYPE = void 0;
exports.generateMnemonic = generateMnemonic;
exports.validateMnemonic = validateMnemonic;
exports.mnemonicToSeed = mnemonicToSeed;
exports.deriveHDPath = deriveHDPath;
exports.parseHDPath = parseHDPath;
exports.derivePrivateKey = derivePrivateKey;
exports.derivePublicKey = derivePublicKey;
exports.deriveKeyPair = deriveKeyPair;
exports.publicKeyToAddress = publicKeyToAddress;
exports.validateAddress = validateAddress;
exports.createWalletAccount = createWalletAccount;
exports.signData = signData;
exports.verifySignature = verifySignature;
exports.randomBytes = randomBytes;
exports.hash256 = hash256;
exports.toHexString = toHexString;
exports.fromHexString = fromHexString;
exports.encryptAES = encryptAES;
exports.decryptAES = decryptAES;
exports.encryptMnemonic = encryptMnemonic;
exports.decryptMnemonic = decryptMnemonic;
const bip39 = __importStar(require("bip39"));
const bip32_1 = __importDefault(require("bip32"));
const crypto_1 = require("@cosmjs/crypto");
const encoding_1 = require("@cosmjs/encoding");
const crypto_2 = require("crypto");
const elliptic_1 = require("elliptic");
const security_1 = require("./security");
const keyDerivation_1 = require("./keyDerivation");
const tinySecp256k1 = __importStar(require("tiny-secp256k1"));
const bip32 = (0, bip32_1.default)(tinySecp256k1);
const secp256k1 = new elliptic_1.ec('secp256k1');
// PAW uses Cosmos coin type 118 for compatibility
exports.DEFAULT_COIN_TYPE = 118;
exports.DEFAULT_HD_PATH = "m/44'/118'/0'/0/0";
exports.PAW_BECH32_PREFIX = 'paw';
/**
 * Generate a new mnemonic phrase using secure entropy
 * Uses Web Crypto API for cryptographically secure randomness
 * @param strength - Entropy strength in bits (128, 160, 192, 224, 256)
 * @returns BIP39 mnemonic phrase
 */
function generateMnemonic(strength = 256) {
    // Use Web Crypto API for secure entropy generation
    const entropy = (0, security_1.secureRandom)(strength / 8);
    return bip39.entropyToMnemonic(Buffer.from(entropy));
}
/**
 * Validate a mnemonic phrase
 * @param mnemonic - BIP39 mnemonic phrase
 * @returns true if valid, false otherwise
 */
function validateMnemonic(mnemonic) {
    return bip39.validateMnemonic(mnemonic);
}
/**
 * Convert mnemonic to seed
 * @param mnemonic - BIP39 mnemonic phrase
 * @param password - Optional passphrase for additional security
 * @returns Seed buffer
 */
async function mnemonicToSeed(mnemonic, password = '') {
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
function deriveHDPath(path = {}) {
    const { coinType = exports.DEFAULT_COIN_TYPE, account = 0, change = 0, addressIndex = 0, } = path;
    return `m/44'/${coinType}'/${account}'/${change}/${addressIndex}`;
}
/**
 * Parse HD path string into components
 * @param pathString - HD path string
 * @returns HD path components
 */
function parseHDPath(pathString) {
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
async function derivePrivateKey(mnemonic, hdPath = exports.DEFAULT_HD_PATH, password = '') {
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
function derivePublicKey(privateKey) {
    const keyPair = secp256k1.keyFromPrivate(Buffer.from(privateKey));
    const publicKey = keyPair.getPublic(true, 'array');
    return new Uint8Array(publicKey);
}
/**
 * Derive key pair from private key
 * @param privateKey - Private key
 * @returns Key pair object
 */
function deriveKeyPair(privateKey) {
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
function publicKeyToAddress(publicKey, prefix = exports.PAW_BECH32_PREFIX) {
    const hash1 = (0, crypto_1.sha256)(publicKey);
    const hash2 = (0, crypto_1.ripemd160)(hash1);
    return (0, encoding_1.toBech32)(prefix, hash2);
}
/**
 * Validate address format
 * @param address - Bech32 address
 * @param prefix - Expected prefix
 * @returns true if valid, false otherwise
 */
function validateAddress(address, prefix = exports.PAW_BECH32_PREFIX) {
    try {
        const { prefix: actualPrefix, data } = (0, encoding_1.fromBech32)(address);
        return actualPrefix === prefix && data.length === 20;
    }
    catch {
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
async function createWalletAccount(mnemonic, hdPath = exports.DEFAULT_HD_PATH, password = '', prefix = exports.PAW_BECH32_PREFIX) {
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
function signData(data, privateKey) {
    const messageHash = (0, crypto_1.sha256)(data);
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
function verifySignature(data, signature, publicKey) {
    try {
        const messageHash = (0, crypto_1.sha256)(data);
        const keyPair = secp256k1.keyFromPublic(Buffer.from(publicKey));
        const r = signature.slice(0, 32);
        const s = signature.slice(32, 64);
        return keyPair.verify(messageHash, { r: Buffer.from(r), s: Buffer.from(s) });
    }
    catch {
        return false;
    }
}
/**
 * Generate cryptographically secure random bytes
 * Uses Web Crypto API for secure randomness
 * @param length - Number of bytes to generate
 * @returns Random bytes
 */
function randomBytes(length) {
    return (0, security_1.secureRandom)(length);
}
/**
 * Hash data using SHA-256
 * @param data - Data to hash
 * @returns Hash
 */
function hash256(data) {
    return (0, crypto_1.sha256)(data);
}
/**
 * Encode data to hex string
 * @param data - Data to encode
 * @returns Hex string
 */
function toHexString(data) {
    return (0, encoding_1.toHex)(data);
}
/**
 * Decode hex string to bytes
 * @param hex - Hex string
 * @returns Bytes
 */
function fromHexString(hex) {
    return (0, encoding_1.fromHex)(hex);
}
/**
 * Encrypt data with AES-256-GCM using Web Crypto API
 * SECURE: Uses proper KDF (PBKDF2), unique IV, and authenticated encryption
 * @param data - Data to encrypt
 * @param password - Encryption password
 * @returns Encrypted data with metadata
 */
async function encryptAES(data, password) {
    // Generate random salt and IV
    const salt = (0, security_1.secureRandom)(32);
    const iv = (0, security_1.secureRandom)(12); // 12 bytes for GCM
    // Derive key from password using PBKDF2
    const keyMaterial = await crypto_2.webcrypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveBits', 'deriveKey']);
    const key = await crypto_2.webcrypto.subtle.deriveKey({
        name: 'PBKDF2',
        salt: salt,
        iterations: keyDerivation_1.RECOMMENDED_PBKDF2_ITERATIONS,
        hash: 'SHA-256',
    }, keyMaterial, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
    // Encrypt data
    const encrypted = await crypto_2.webcrypto.subtle.encrypt({
        name: 'AES-GCM',
        iv: iv,
    }, key, new TextEncoder().encode(data));
    // Return encrypted data with all necessary parameters
    return {
        ciphertext: Buffer.from(encrypted).toString('base64'),
        salt: Buffer.from(salt).toString('base64'),
        iv: Buffer.from(iv).toString('base64'),
        algorithm: 'AES-256-GCM',
        kdf: 'PBKDF2',
        iterations: keyDerivation_1.RECOMMENDED_PBKDF2_ITERATIONS,
    };
}
/**
 * Decrypt data with AES-256-GCM using Web Crypto API
 * @param encryptedData - Encrypted data object
 * @param password - Decryption password
 * @returns Decrypted data
 */
async function decryptAES(encryptedData, password) {
    // Decode salt, IV, and ciphertext from base64
    const salt = Buffer.from(encryptedData.salt, 'base64');
    const iv = Buffer.from(encryptedData.iv, 'base64');
    const ciphertext = Buffer.from(encryptedData.ciphertext, 'base64');
    // Derive same key from password
    const keyMaterial = await crypto_2.webcrypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveBits', 'deriveKey']);
    const key = await crypto_2.webcrypto.subtle.deriveKey({
        name: 'PBKDF2',
        salt: salt,
        iterations: encryptedData.iterations,
        hash: 'SHA-256',
    }, keyMaterial, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
    // Decrypt
    try {
        const decrypted = await crypto_2.webcrypto.subtle.decrypt({
            name: 'AES-GCM',
            iv: iv,
        }, key, ciphertext);
        return new TextDecoder().decode(decrypted);
    }
    catch (error) {
        throw new Error('Decryption failed: invalid password or corrupted data');
    }
}
/**
 * Encrypt mnemonic phrase securely
 * @param mnemonic - Mnemonic phrase to encrypt
 * @param password - Encryption password
 * @returns Encrypted mnemonic
 */
async function encryptMnemonic(mnemonic, password) {
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
async function decryptMnemonic(encryptedData, password) {
    const mnemonic = await decryptAES(encryptedData, password);
    if (!validateMnemonic(mnemonic)) {
        throw new Error('Decrypted data is not a valid mnemonic');
    }
    return mnemonic;
}
//# sourceMappingURL=crypto.js.map