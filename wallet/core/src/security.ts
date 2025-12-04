/**
 * Security utilities for PAW Wallet
 * Provides secure random generation, timing-attack prevention, and password validation
 */

import { webcrypto } from 'crypto';
import { PasswordStrength } from './types';

/**
 * Generate cryptographically secure random bytes
 * Uses Web Crypto API for maximum security
 * @param bytes - Number of bytes to generate
 * @returns Secure random bytes
 */
export function secureRandom(bytes: number): Uint8Array {
  if (bytes <= 0 || !Number.isInteger(bytes)) {
    throw new Error('Bytes must be a positive integer');
  }
  return webcrypto.getRandomValues(new Uint8Array(bytes));
}

/**
 * Generate secure random hex string
 * @param bytes - Number of bytes
 * @returns Hex string
 */
export function secureRandomHex(bytes: number): string {
  const randomBytes = secureRandom(bytes);
  return Buffer.from(randomBytes).toString('hex');
}

/**
 * Constant-time comparison for Uint8Array (timing attack prevention)
 * CRITICAL: Never use === for comparing sensitive data (keys, MACs, etc.)
 * @param a - First array
 * @param b - Second array
 * @returns true if arrays are equal
 */
export function constantTimeCompare(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a[i] ^ b[i];
  }
  return result === 0;
}

/**
 * Constant-time string comparison (timing attack prevention)
 * @param a - First string
 * @param b - Second string
 * @returns true if strings are equal
 */
export function constantTimeCompareString(a: string, b: string): boolean {
  if (a.length !== b.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }

  return result === 0;
}

/**
 * Validate password strength according to OWASP guidelines
 * @param password - Password to validate
 * @returns Password strength analysis
 */
export function validatePasswordStrength(password: string): PasswordStrength {
  const errors: string[] = [];
  let score = 0;

  // Minimum length check (OWASP: 12+ characters)
  if (password.length < 12) {
    errors.push('Password must be at least 12 characters');
  } else {
    score += 1;
    if (password.length >= 16) {
      score += 1;
    }
    if (password.length >= 20) {
      score += 1;
    }
  }

  // Lowercase letters
  if (!/[a-z]/.test(password)) {
    errors.push('Password must contain lowercase letters');
  } else {
    score += 1;
  }

  // Uppercase letters
  if (!/[A-Z]/.test(password)) {
    errors.push('Password must contain uppercase letters');
  } else {
    score += 1;
  }

  // Numbers
  if (!/[0-9]/.test(password)) {
    errors.push('Password must contain numbers');
  } else {
    score += 1;
  }

  // Special characters
  if (!/[^a-zA-Z0-9]/.test(password)) {
    errors.push('Password must contain special characters');
  } else {
    score += 1;
  }

  // Check for common patterns
  const commonPatterns = [
    /(.)\1{2,}/, // Repeated characters (aaa, 111)
    /^(123|abc|qwerty|password)/i, // Common sequences
    /^[0-9]+$/, // Only numbers
    /^[a-zA-Z]+$/, // Only letters
  ];

  for (const pattern of commonPatterns) {
    if (pattern.test(password)) {
      errors.push('Password contains common patterns');
      score -= 1;
      break;
    }
  }

  // Calculate strength
  let strength: 'weak' | 'medium' | 'strong';
  if (errors.length > 0 || score < 4) {
    strength = 'weak';
  } else if (score < 6) {
    strength = 'medium';
  } else {
    strength = 'strong';
  }

  return {
    valid: errors.length === 0,
    strength,
    errors,
    score: Math.max(0, score),
  };
}

/**
 * Secure memory wiping (best effort in JavaScript)
 * Note: JavaScript doesn't allow true memory wiping, but we do our best
 * @param data - Data to wipe (will be overwritten with random data)
 */
export function secureWipe(data: Uint8Array): void {
  if (data && data.length > 0) {
    // Overwrite with random data
    webcrypto.getRandomValues(data);
    // Then zero it out
    data.fill(0);
  }
}

/**
 * Secure string wiping (limited effectiveness in JS)
 * @param str - String to wipe
 * @returns Zeroed string (caller should not use original reference)
 */
export function secureWipeString(str: string): string {
  // In JavaScript, we can't truly wipe strings from memory
  // But we can create a new reference filled with zeros
  return '\0'.repeat(str.length);
}

/**
 * Generate UUID v4 using secure random
 * @returns UUID v4 string
 */
export function generateUUID(): string {
  const bytes = secureRandom(16);

  // Set version (4) and variant bits
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;

  const hex = Buffer.from(bytes).toString('hex');
  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20, 32),
  ].join('-');
}

/**
 * Hash data using SHA-256
 * @param data - Data to hash
 * @returns Hash digest
 */
export async function sha256(data: Uint8Array): Promise<Uint8Array> {
  const hash = await webcrypto.subtle.digest('SHA-256', data);
  return new Uint8Array(hash);
}

/**
 * Hash data using SHA-512
 * @param data - Data to hash
 * @returns Hash digest
 */
export async function sha512(data: Uint8Array): Promise<Uint8Array> {
  const hash = await webcrypto.subtle.digest('SHA-512', data);
  return new Uint8Array(hash);
}

/**
 * HMAC-SHA256
 * @param key - HMAC key
 * @param data - Data to authenticate
 * @returns HMAC digest
 */
export async function hmacSha256(key: Uint8Array, data: Uint8Array): Promise<Uint8Array> {
  const cryptoKey = await webcrypto.subtle.importKey(
    'raw',
    key,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );

  const signature = await webcrypto.subtle.sign('HMAC', cryptoKey, data);
  return new Uint8Array(signature);
}

/**
 * HMAC-SHA512
 * @param key - HMAC key
 * @param data - Data to authenticate
 * @returns HMAC digest
 */
export async function hmacSha512(key: Uint8Array, data: Uint8Array): Promise<Uint8Array> {
  const cryptoKey = await webcrypto.subtle.importKey(
    'raw',
    key,
    { name: 'HMAC', hash: 'SHA-512' },
    false,
    ['sign']
  );

  const signature = await webcrypto.subtle.sign('HMAC', cryptoKey, data);
  return new Uint8Array(signature);
}

/**
 * Verify HMAC-SHA256
 * @param key - HMAC key
 * @param data - Data to verify
 * @param signature - HMAC signature to check
 * @returns true if valid
 */
export async function verifyHmacSha256(
  key: Uint8Array,
  data: Uint8Array,
  signature: Uint8Array
): Promise<boolean> {
  const calculated = await hmacSha256(key, data);
  return constantTimeCompare(calculated, signature);
}

/**
 * Check if password has been compromised (using local checks only)
 * For production, integrate with Have I Been Pwned API
 * @param password - Password to check
 * @returns Warning if password appears weak
 */
export function checkPasswordCompromise(password: string): {
  compromised: boolean;
  reason?: string;
} {
  // Common passwords list (top 100)
  const commonPasswords = [
    'password', 'Password123', '123456', '12345678', 'qwerty', 'abc123',
    'monkey', '1234567', 'letmein', 'trustno1', 'dragon', 'baseball',
    'iloveyou', 'master', 'sunshine', 'ashley', 'bailey', 'passw0rd',
    'shadow', '123123', '654321', 'superman', 'qazwsx', 'michael',
  ];

  const lowerPassword = password.toLowerCase();

  for (const common of commonPasswords) {
    if (lowerPassword.includes(common.toLowerCase())) {
      return {
        compromised: true,
        reason: 'Password contains common password pattern',
      };
    }
  }

  // Check for keyboard patterns
  const keyboardPatterns = ['qwerty', 'asdfgh', 'zxcvbn', '123456', '0987654'];
  for (const pattern of keyboardPatterns) {
    if (lowerPassword.includes(pattern)) {
      return {
        compromised: true,
        reason: 'Password contains keyboard pattern',
      };
    }
  }

  return { compromised: false };
}

/**
 * Rate limiting helper for password attempts
 */
export class RateLimiter {
  private attempts: Map<string, number[]> = new Map();
  private readonly maxAttempts: number;
  private readonly windowMs: number;

  constructor(maxAttempts: number = 5, windowMs: number = 300000) {
    this.maxAttempts = maxAttempts;
    this.windowMs = windowMs;
  }

  /**
   * Check if action is allowed
   * @param identifier - Unique identifier (e.g., address)
   * @returns true if allowed
   */
  isAllowed(identifier: string): boolean {
    const now = Date.now();
    const attempts = this.attempts.get(identifier) || [];

    // Remove old attempts outside the window
    const recentAttempts = attempts.filter(time => now - time < this.windowMs);

    if (recentAttempts.length >= this.maxAttempts) {
      return false;
    }

    recentAttempts.push(now);
    this.attempts.set(identifier, recentAttempts);
    return true;
  }

  /**
   * Reset attempts for identifier
   * @param identifier - Unique identifier
   */
  reset(identifier: string): void {
    this.attempts.delete(identifier);
  }

  /**
   * Get remaining attempts
   * @param identifier - Unique identifier
   * @returns Number of remaining attempts
   */
  getRemainingAttempts(identifier: string): number {
    const now = Date.now();
    const attempts = this.attempts.get(identifier) || [];
    const recentAttempts = attempts.filter(time => now - time < this.windowMs);
    return Math.max(0, this.maxAttempts - recentAttempts.length);
  }
}

/**
 * Sanitize error messages to prevent information leakage
 * @param error - Original error
 * @returns Safe error message
 */
export function sanitizeError(error: any): string {
  // Never expose internal details in production
  if (process.env.NODE_ENV === 'production') {
    return 'An error occurred. Please try again.';
  }

  // In development, provide more context
  if (error instanceof Error) {
    return error.message;
  }

  return String(error);
}
