/**
 * Keystore management for secure private key storage
 * Implements secure keystore with Web Crypto API
 */
type SecureKeystore = import('./types').SecureKeystore;
/**
 * Encrypt private key into secure keystore format using Web Crypto API
 * @param privateKey - Private key to encrypt
 * @param password - Encryption password
 * @param address - Associated address
 * @param name - Optional wallet name
 * @returns Secure keystore object
 */
export declare function encryptKeystore(privateKey: Uint8Array, password: string, address: string, name?: string): Promise<SecureKeystore>;
/**
 * Decrypt secure keystore to retrieve private key using Web Crypto API
 * @param keystore - Secure keystore object
 * @param password - Decryption password
 * @returns Decrypted private key
 */
export declare function decryptKeystore(keystore: SecureKeystore, password: string): Promise<Uint8Array>;
/**
 * Export secure keystore to JSON string
 * @param keystore - Secure keystore object
 * @param pretty - Pretty print JSON
 * @returns JSON string
 */
export declare function exportKeystore(keystore: SecureKeystore, pretty?: boolean): string;
/**
 * Import secure keystore from JSON string
 * @param json - JSON string
 * @returns Secure keystore object
 */
export declare function importKeystore(json: string): SecureKeystore;
/**
 * Validate secure keystore structure
 * @param keystore - Secure keystore object
 * @returns true if valid
 */
export declare function validateKeystore(keystore: SecureKeystore): boolean;
/**
 * Change keystore password
 * @param keystore - Original secure keystore
 * @param oldPassword - Current password
 * @param newPassword - New password
 * @returns New keystore with updated password
 */
export declare function changeKeystorePassword(keystore: SecureKeystore, oldPassword: string, newPassword: string): Promise<SecureKeystore>;
/**
 * Generate keystore filename
 * @param address - Wallet address
 * @param timestamp - Optional timestamp
 * @returns Filename string
 */
export declare function generateKeystoreFilename(address: string, timestamp?: number): string;
/**
 * Verify keystore password without decrypting
 * @param keystore - Secure keystore object
 * @param password - Password to verify
 * @returns true if password is correct
 */
export declare function verifyKeystorePassword(keystore: SecureKeystore, password: string): Promise<boolean>;
/**
 * Estimate keystore decryption time
 * @param iterations - Number of PBKDF2 iterations
 * @returns Estimated time in milliseconds
 */
export declare function estimateDecryptionTime(iterations: number): number;
/**
 * Get keystore security level
 * @param keystore - Secure keystore object
 * @returns Security level (low, medium, high, excellent)
 */
export declare function getKeystoreSecurityLevel(keystore: SecureKeystore): 'low' | 'medium' | 'high' | 'excellent';
/**
 * Sanitize keystore for logging (removes sensitive data)
 * @param keystore - Secure keystore object
 * @returns Sanitized keystore (safe to log)
 */
export declare function sanitizeKeystore(keystore: SecureKeystore): Partial<SecureKeystore>;
export {};
//# sourceMappingURL=keystore.d.ts.map