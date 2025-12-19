"use strict";
/**
 * Keystore management for secure private key storage
 * Implements secure keystore with Web Crypto API
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.encryptKeystore = encryptKeystore;
exports.decryptKeystore = decryptKeystore;
exports.exportKeystore = exportKeystore;
exports.importKeystore = importKeystore;
exports.validateKeystore = validateKeystore;
exports.changeKeystorePassword = changeKeystorePassword;
exports.generateKeystoreFilename = generateKeystoreFilename;
exports.verifyKeystorePassword = verifyKeystorePassword;
exports.estimateDecryptionTime = estimateDecryptionTime;
exports.getKeystoreSecurityLevel = getKeystoreSecurityLevel;
exports.sanitizeKeystore = sanitizeKeystore;
const crypto_1 = require("crypto");
const security_1 = require("./security");
const keyDerivation_1 = require("./keyDerivation");
const KEYSTORE_VERSION = 4; // Updated version for new secure format
const DEFAULT_ITERATIONS = keyDerivation_1.RECOMMENDED_PBKDF2_ITERATIONS;
const DEFAULT_CIPHER = 'AES-256-GCM';
const DEFAULT_KDF = 'PBKDF2';
/**
 * Encrypt private key into secure keystore format using Web Crypto API
 * @param privateKey - Private key to encrypt
 * @param password - Encryption password
 * @param address - Associated address
 * @param name - Optional wallet name
 * @returns Secure keystore object
 */
async function encryptKeystore(privateKey, password, address, name) {
    if (password.length < 12) {
        throw new Error('Password must be at least 12 characters for security');
    }
    // Generate random salt and IV
    const salt = (0, security_1.secureRandom)(32);
    const iv = (0, security_1.secureRandom)(12); // 12 bytes for GCM
    // Derive encryption and MAC keys using PBKDF2
    const { encryptionKey, macKey } = await (0, keyDerivation_1.deriveEncryptionAndMacKeys)(password, salt, DEFAULT_ITERATIONS);
    // Import encryption key
    const cryptoKey = await crypto_1.webcrypto.subtle.importKey('raw', encryptionKey, { name: 'AES-GCM', length: 256 }, false, ['encrypt']);
    // Encrypt private key
    const encrypted = await crypto_1.webcrypto.subtle.encrypt({
        name: 'AES-GCM',
        iv: iv,
    }, cryptoKey, privateKey);
    const ciphertext = Buffer.from(encrypted).toString('base64');
    // Calculate HMAC for integrity (encrypt-then-MAC pattern)
    const mac = await (0, security_1.hmacSha256)(macKey, new Uint8Array(encrypted));
    // Generate unique ID
    const id = (0, security_1.generateUUID)();
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
async function decryptKeystore(keystore, password) {
    // Validate keystore format
    if (keystore.version !== KEYSTORE_VERSION) {
        // Support legacy keystores (version 3)
        if (keystore.version === 3) {
            throw new Error('Legacy keystore detected. Please migrate using the migration tool.');
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
    const { encryptionKey, macKey } = await (0, keyDerivation_1.deriveEncryptionAndMacKeys)(password, salt, crypto.kdfparams.iterations);
    // Verify HMAC (constant-time comparison to prevent timing attacks)
    const calculatedMac = await (0, security_1.hmacSha256)(macKey, ciphertext);
    if (!(0, security_1.constantTimeCompare)(calculatedMac, expectedMac)) {
        throw new Error('Invalid password or corrupted keystore');
    }
    // Import decryption key
    const cryptoKey = await crypto_1.webcrypto.subtle.importKey('raw', encryptionKey, { name: 'AES-GCM', length: 256 }, false, ['decrypt']);
    // Decrypt private key
    try {
        const decrypted = await crypto_1.webcrypto.subtle.decrypt({
            name: 'AES-GCM',
            iv: iv,
        }, cryptoKey, ciphertext);
        return new Uint8Array(decrypted);
    }
    catch (error) {
        throw new Error('Failed to decrypt keystore: invalid password or corrupted data');
    }
}
/**
 * Export secure keystore to JSON string
 * @param keystore - Secure keystore object
 * @param pretty - Pretty print JSON
 * @returns JSON string
 */
function exportKeystore(keystore, pretty = false) {
    return JSON.stringify(keystore, null, pretty ? 2 : 0);
}
/**
 * Import secure keystore from JSON string
 * @param json - JSON string
 * @returns Secure keystore object
 */
function importKeystore(json) {
    try {
        const keystore = JSON.parse(json);
        // Validate required fields
        if (!keystore.version || !keystore.crypto || !keystore.id || !keystore.address) {
            throw new Error('Invalid keystore format');
        }
        return keystore;
    }
    catch (error) {
        throw new Error('Failed to parse keystore JSON');
    }
}
/**
 * Validate secure keystore structure
 * @param keystore - Secure keystore object
 * @returns true if valid
 */
function validateKeystore(keystore) {
    try {
        // Check version
        if (keystore.version !== KEYSTORE_VERSION && keystore.version !== 3) {
            return false;
        }
        // Check crypto object
        const { crypto } = keystore;
        if (!crypto.cipher ||
            !crypto.ciphertext ||
            !crypto.kdf ||
            !crypto.kdfparams ||
            !crypto.mac ||
            !crypto.iv) {
            return false;
        }
        // Check KDF params
        if (!crypto.kdfparams.salt ||
            !crypto.kdfparams.iterations ||
            !crypto.kdfparams.dklen) {
            return false;
        }
        // Check ID and address
        if (!keystore.id || !keystore.address) {
            return false;
        }
        return true;
    }
    catch {
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
async function changeKeystorePassword(keystore, oldPassword, newPassword) {
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
function generateKeystoreFilename(address, timestamp) {
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
async function verifyKeystorePassword(keystore, password) {
    try {
        await decryptKeystore(keystore, password);
        return true;
    }
    catch {
        return false;
    }
}
/**
 * Estimate keystore decryption time
 * @param iterations - Number of PBKDF2 iterations
 * @returns Estimated time in milliseconds
 */
function estimateDecryptionTime(iterations) {
    // Rough estimate: ~1ms per 1000 iterations on average hardware
    return (iterations / 1000) * 1;
}
/**
 * Get keystore security level
 * @param keystore - Secure keystore object
 * @returns Security level (low, medium, high, excellent)
 */
function getKeystoreSecurityLevel(keystore) {
    const iterations = keystore.crypto.kdfparams.iterations;
    if (iterations < 50000) {
        return 'low';
    }
    else if (iterations < 100000) {
        return 'medium';
    }
    else if (iterations < keyDerivation_1.RECOMMENDED_PBKDF2_ITERATIONS) {
        return 'high';
    }
    else {
        return 'excellent';
    }
}
/**
 * Sanitize keystore for logging (removes sensitive data)
 * @param keystore - Secure keystore object
 * @returns Sanitized keystore (safe to log)
 */
function sanitizeKeystore(keystore) {
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
//# sourceMappingURL=keystore.js.map