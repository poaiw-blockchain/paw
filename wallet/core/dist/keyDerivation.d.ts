/**
 * Secure key derivation functions for PAW Wallet
 * Implements PBKDF2 and Argon2id for password-based key derivation
 */
import { KeyDerivationParams } from './types';
export declare const MIN_PBKDF2_ITERATIONS = 100000;
export declare const RECOMMENDED_PBKDF2_ITERATIONS = 210000;
/**
 * Derive cryptographic key from password using PBKDF2
 * @param params - Key derivation parameters
 * @returns Derived CryptoKey
 */
export declare function deriveKey(params: KeyDerivationParams): Promise<CryptoKey>;
/**
 * Derive raw key bytes from password using PBKDF2
 * Useful when you need the raw key material instead of CryptoKey
 * @param password - Password
 * @param salt - Salt (32 bytes recommended)
 * @param iterations - Number of iterations
 * @param keyLength - Key length in bytes
 * @returns Derived key bytes
 */
export declare function deriveKeyBytes(password: string, salt: Uint8Array, iterations?: number, keyLength?: number): Promise<Uint8Array>;
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
export declare function deriveKeyArgon2(password: string, salt?: Uint8Array, memorySize?: number, iterations?: number, parallelism?: number, keyLength?: number): Promise<Uint8Array>;
/**
 * Derive encryption and MAC keys from password
 * Returns two separate keys for encrypt-then-MAC pattern
 * @param password - Password
 * @param salt - Salt
 * @param iterations - PBKDF2 iterations
 * @returns Object with encryption key and MAC key
 */
export declare function deriveEncryptionAndMacKeys(password: string, salt: Uint8Array, iterations?: number): Promise<{
    encryptionKey: Uint8Array;
    macKey: Uint8Array;
}>;
/**
 * Generate salt for key derivation
 * @param bytes - Number of bytes (default: 32)
 * @returns Random salt
 */
export declare function generateSalt(bytes?: number): Uint8Array;
/**
 * Calculate optimal PBKDF2 iterations for target delay
 * Benchmarks the system and calculates iterations needed
 * @param targetDelayMs - Target delay in milliseconds (default: 500ms)
 * @returns Recommended number of iterations
 */
export declare function calculateOptimalIterations(targetDelayMs?: number): Promise<number>;
/**
 * Verify derived key matches expected value (constant-time)
 * @param password - Password to test
 * @param salt - Salt used for derivation
 * @param iterations - Number of iterations
 * @param expectedKey - Expected derived key
 * @returns true if match
 */
export declare function verifyDerivedKey(password: string, salt: Uint8Array, iterations: number, expectedKey: Uint8Array): Promise<boolean>;
/**
 * Key derivation info for debugging/auditing
 * @param iterations - Number of iterations
 * @returns Info object
 */
export declare function getKeyDerivationInfo(iterations: number): {
    iterations: number;
    estimatedTimeMs: number;
    securityLevel: 'weak' | 'acceptable' | 'good' | 'excellent';
    meetsOWASP: boolean;
};
/**
 * Compare two key derivation configurations
 * @param config1 - First configuration
 * @param config2 - Second configuration
 * @returns Comparison result
 */
export declare function compareKeyDerivationSecurity(config1: {
    kdf: string;
    iterations: number;
}, config2: {
    kdf: string;
    iterations: number;
}): 'stronger' | 'weaker' | 'equal';
//# sourceMappingURL=keyDerivation.d.ts.map