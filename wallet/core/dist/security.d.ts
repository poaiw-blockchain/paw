/**
 * Security utilities for PAW Wallet
 * Provides secure random generation, timing-attack prevention, and password validation
 */
import { PasswordStrength } from './types';
/**
 * Generate cryptographically secure random bytes
 * Uses Web Crypto API for maximum security
 * @param bytes - Number of bytes to generate
 * @returns Secure random bytes
 */
export declare function secureRandom(bytes: number): Uint8Array;
/**
 * Generate secure random hex string
 * @param bytes - Number of bytes
 * @returns Hex string
 */
export declare function secureRandomHex(bytes: number): string;
/**
 * Constant-time comparison for Uint8Array (timing attack prevention)
 * CRITICAL: Never use === for comparing sensitive data (keys, MACs, etc.)
 * @param a - First array
 * @param b - Second array
 * @returns true if arrays are equal
 */
export declare function constantTimeCompare(a: Uint8Array, b: Uint8Array): boolean;
/**
 * Constant-time string comparison (timing attack prevention)
 * @param a - First string
 * @param b - Second string
 * @returns true if strings are equal
 */
export declare function constantTimeCompareString(a: string, b: string): boolean;
/**
 * Validate password strength according to OWASP guidelines
 * @param password - Password to validate
 * @returns Password strength analysis
 */
export declare function validatePasswordStrength(password: string): PasswordStrength;
/**
 * Secure memory wiping (best effort in JavaScript)
 * Note: JavaScript doesn't allow true memory wiping, but we do our best
 * @param data - Data to wipe (will be overwritten with random data)
 */
export declare function secureWipe(data: Uint8Array): void;
/**
 * Secure string wiping (limited effectiveness in JS)
 * @param str - String to wipe
 * @returns Zeroed string (caller should not use original reference)
 */
export declare function secureWipeString(str: string): string;
/**
 * Generate UUID v4 using secure random
 * @returns UUID v4 string
 */
export declare function generateUUID(): string;
/**
 * Hash data using SHA-256
 * @param data - Data to hash
 * @returns Hash digest
 */
export declare function sha256(data: Uint8Array): Promise<Uint8Array>;
/**
 * Hash data using SHA-512
 * @param data - Data to hash
 * @returns Hash digest
 */
export declare function sha512(data: Uint8Array): Promise<Uint8Array>;
/**
 * HMAC-SHA256
 * @param key - HMAC key
 * @param data - Data to authenticate
 * @returns HMAC digest
 */
export declare function hmacSha256(key: Uint8Array, data: Uint8Array): Promise<Uint8Array>;
/**
 * HMAC-SHA512
 * @param key - HMAC key
 * @param data - Data to authenticate
 * @returns HMAC digest
 */
export declare function hmacSha512(key: Uint8Array, data: Uint8Array): Promise<Uint8Array>;
/**
 * Verify HMAC-SHA256
 * @param key - HMAC key
 * @param data - Data to verify
 * @param signature - HMAC signature to check
 * @returns true if valid
 */
export declare function verifyHmacSha256(key: Uint8Array, data: Uint8Array, signature: Uint8Array): Promise<boolean>;
/**
 * Check if password has been compromised (using local checks only)
 * For production, integrate with Have I Been Pwned API
 * @param password - Password to check
 * @returns Warning if password appears weak
 */
export declare function checkPasswordCompromise(password: string): {
    compromised: boolean;
    reason?: string;
};
/**
 * Rate limiting helper for password attempts
 */
export declare class RateLimiter {
    private attempts;
    private readonly maxAttempts;
    private readonly windowMs;
    constructor(maxAttempts?: number, windowMs?: number);
    /**
     * Check if action is allowed
     * @param identifier - Unique identifier (e.g., address)
     * @returns true if allowed
     */
    isAllowed(identifier: string): boolean;
    /**
     * Reset attempts for identifier
     * @param identifier - Unique identifier
     */
    reset(identifier: string): void;
    /**
     * Get remaining attempts
     * @param identifier - Unique identifier
     * @returns Number of remaining attempts
     */
    getRemainingAttempts(identifier: string): number;
}
/**
 * Sanitize error messages to prevent information leakage
 * @param error - Original error
 * @returns Safe error message
 */
export declare function sanitizeError(error: any): string;
//# sourceMappingURL=security.d.ts.map