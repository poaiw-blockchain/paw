"use strict";
/**
 * Secure key derivation functions for PAW Wallet
 * Implements PBKDF2 and Argon2id for password-based key derivation
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.RECOMMENDED_PBKDF2_ITERATIONS = exports.MIN_PBKDF2_ITERATIONS = void 0;
exports.deriveKey = deriveKey;
exports.deriveKeyBytes = deriveKeyBytes;
exports.deriveKeyArgon2 = deriveKeyArgon2;
exports.deriveEncryptionAndMacKeys = deriveEncryptionAndMacKeys;
exports.generateSalt = generateSalt;
exports.calculateOptimalIterations = calculateOptimalIterations;
exports.verifyDerivedKey = verifyDerivedKey;
exports.getKeyDerivationInfo = getKeyDerivationInfo;
exports.compareKeyDerivationSecurity = compareKeyDerivationSecurity;
const crypto_1 = require("crypto");
const security_1 = require("./security");
// OWASP recommended minimum iterations for PBKDF2-SHA256
exports.MIN_PBKDF2_ITERATIONS = 100000;
exports.RECOMMENDED_PBKDF2_ITERATIONS = 210000; // OWASP 2023 recommendation
/**
 * Derive cryptographic key from password using PBKDF2
 * @param params - Key derivation parameters
 * @returns Derived CryptoKey
 */
async function deriveKey(params) {
    const { password, salt = (0, security_1.secureRandom)(32), iterations = exports.RECOMMENDED_PBKDF2_ITERATIONS, keyLength = 256, } = params;
    if (iterations < exports.MIN_PBKDF2_ITERATIONS) {
        throw new Error(`Iterations must be at least ${exports.MIN_PBKDF2_ITERATIONS} for security. Recommended: ${exports.RECOMMENDED_PBKDF2_ITERATIONS}`);
    }
    // Import password as key material
    const keyMaterial = await crypto_1.webcrypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveBits', 'deriveKey']);
    // Derive key using PBKDF2
    const derivedKey = await crypto_1.webcrypto.subtle.deriveKey({
        name: 'PBKDF2',
        salt: salt,
        iterations: iterations,
        hash: 'SHA-256',
    }, keyMaterial, { name: 'AES-GCM', length: keyLength }, false, ['encrypt', 'decrypt']);
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
async function deriveKeyBytes(password, salt, iterations = exports.RECOMMENDED_PBKDF2_ITERATIONS, keyLength = 32) {
    if (iterations < exports.MIN_PBKDF2_ITERATIONS) {
        throw new Error(`Iterations must be at least ${exports.MIN_PBKDF2_ITERATIONS} for security`);
    }
    // Import password as key material
    const keyMaterial = await crypto_1.webcrypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveBits']);
    // Derive raw bits
    const derivedBits = await crypto_1.webcrypto.subtle.deriveBits({
        name: 'PBKDF2',
        salt: salt,
        iterations: iterations,
        hash: 'SHA-256',
    }, keyMaterial, keyLength * 8 // Convert bytes to bits
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
async function deriveKeyArgon2(password, salt, memorySize = 65536, iterations = 3, parallelism = 4, keyLength = 32) {
    try {
        // Dynamic import to avoid bundling if not used
        const { argon2id } = await Promise.resolve().then(() => __importStar(require('@noble/hashes/argon2')));
        const actualSalt = salt || (0, security_1.secureRandom)(16);
        return argon2id(new TextEncoder().encode(password), actualSalt, {
            m: memorySize,
            t: iterations,
            p: parallelism,
            dkLen: keyLength,
        });
    }
    catch (error) {
        throw new Error('Argon2 not available. Install @noble/hashes: npm install @noble/hashes');
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
async function deriveEncryptionAndMacKeys(password, salt, iterations = exports.RECOMMENDED_PBKDF2_ITERATIONS) {
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
function generateSalt(bytes = 32) {
    return (0, security_1.secureRandom)(bytes);
}
/**
 * Calculate optimal PBKDF2 iterations for target delay
 * Benchmarks the system and calculates iterations needed
 * @param targetDelayMs - Target delay in milliseconds (default: 500ms)
 * @returns Recommended number of iterations
 */
async function calculateOptimalIterations(targetDelayMs = 500) {
    const testPassword = 'test-password-for-benchmarking';
    const testSalt = (0, security_1.secureRandom)(32);
    const testIterations = 10000;
    const start = performance.now();
    await deriveKeyBytes(testPassword, testSalt, testIterations, 32);
    const end = performance.now();
    const timeFor10k = end - start;
    const iterationsPerMs = testIterations / timeFor10k;
    const optimalIterations = Math.floor(iterationsPerMs * targetDelayMs);
    // Ensure we meet minimum requirements
    return Math.max(optimalIterations, exports.MIN_PBKDF2_ITERATIONS);
}
/**
 * Verify derived key matches expected value (constant-time)
 * @param password - Password to test
 * @param salt - Salt used for derivation
 * @param iterations - Number of iterations
 * @param expectedKey - Expected derived key
 * @returns true if match
 */
async function verifyDerivedKey(password, salt, iterations, expectedKey) {
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
function getKeyDerivationInfo(iterations) {
    // Rough estimate: 1ms per 1000 iterations on average hardware
    const estimatedTimeMs = (iterations / 1000) * 1;
    let securityLevel;
    if (iterations < 50000) {
        securityLevel = 'weak';
    }
    else if (iterations < exports.MIN_PBKDF2_ITERATIONS) {
        securityLevel = 'acceptable';
    }
    else if (iterations < exports.RECOMMENDED_PBKDF2_ITERATIONS) {
        securityLevel = 'good';
    }
    else {
        securityLevel = 'excellent';
    }
    return {
        iterations,
        estimatedTimeMs,
        securityLevel,
        meetsOWASP: iterations >= exports.RECOMMENDED_PBKDF2_ITERATIONS,
    };
}
/**
 * Compare two key derivation configurations
 * @param config1 - First configuration
 * @param config2 - Second configuration
 * @returns Comparison result
 */
function compareKeyDerivationSecurity(config1, config2) {
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
    }
    else if (config1.iterations < config2.iterations) {
        return 'weaker';
    }
    return 'equal';
}
//# sourceMappingURL=keyDerivation.js.map