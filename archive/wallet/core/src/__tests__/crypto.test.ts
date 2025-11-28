/**
 * Comprehensive security tests for PAW Wallet cryptography
 * Tests Web Crypto API implementation for encryption, key derivation, and security
 */

import {
  generateMnemonic,
  validateMnemonic,
  encryptAES,
  decryptAES,
  encryptMnemonic,
  decryptMnemonic,
  randomBytes,
  derivePrivateKey,
  publicKeyToAddress,
  derivePublicKey,
} from '../crypto';
import { encryptKeystore, decryptKeystore, validateKeystore } from '../keystore';
import {
  secureRandom,
  constantTimeCompare,
  constantTimeCompareString,
  validatePasswordStrength,
  secureWipe,
  generateUUID,
} from '../security';
import {
  deriveKeyBytes,
  calculateOptimalIterations,
  RECOMMENDED_PBKDF2_ITERATIONS,
  MIN_PBKDF2_ITERATIONS,
} from '../keyDerivation';

describe('Crypto Security Tests', () => {
  describe('Secure Random Generation', () => {
    it('should generate cryptographically secure random bytes', () => {
      const bytes1 = secureRandom(32);
      const bytes2 = secureRandom(32);

      expect(bytes1.length).toBe(32);
      expect(bytes2.length).toBe(32);
      // Should be different
      expect(Buffer.from(bytes1).toString('hex')).not.toBe(
        Buffer.from(bytes2).toString('hex')
      );
    });

    it('should generate unique UUIDs', () => {
      const uuid1 = generateUUID();
      const uuid2 = generateUUID();

      expect(uuid1).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
      );
      expect(uuid1).not.toBe(uuid2);
    });
  });

  describe('AES Encryption/Decryption', () => {
    const testData = 'sensitive wallet data';
    const password = 'SecurePassword123!@#';

    it('should encrypt and decrypt correctly', async () => {
      const encrypted = await encryptAES(testData, password);
      const decrypted = await decryptAES(encrypted, password);

      expect(decrypted).toBe(testData);
    });

    it('should use unique IVs for each encryption', async () => {
      const encrypted1 = await encryptAES(testData, password);
      const encrypted2 = await encryptAES(testData, password);

      // IVs should be different
      expect(encrypted1.iv).not.toBe(encrypted2.iv);
      // Ciphertexts should be different due to different IVs
      expect(encrypted1.ciphertext).not.toBe(encrypted2.ciphertext);
      // Salts should be different
      expect(encrypted1.salt).not.toBe(encrypted2.salt);
    });

    it('should fail with wrong password', async () => {
      const encrypted = await encryptAES(testData, 'correct-password');

      await expect(decryptAES(encrypted, 'wrong-password')).rejects.toThrow();
    });

    it('should use secure KDF (PBKDF2 with recommended iterations)', async () => {
      const encrypted = await encryptAES(testData, password);

      expect(encrypted.kdf).toBe('PBKDF2');
      expect(encrypted.iterations).toBeGreaterThanOrEqual(RECOMMENDED_PBKDF2_ITERATIONS);
      expect(encrypted.algorithm).toBe('AES-256-GCM');
    });

    it('should fail on tampered ciphertext', async () => {
      const encrypted = await encryptAES(testData, password);

      // Tamper with ciphertext
      const tampered = {
        ...encrypted,
        ciphertext: Buffer.from('tampered', 'utf8').toString('base64'),
      };

      await expect(decryptAES(tampered, password)).rejects.toThrow();
    });

    it('should fail on tampered IV', async () => {
      const encrypted = await encryptAES(testData, password);

      // Tamper with IV
      const tampered = {
        ...encrypted,
        iv: Buffer.from(secureRandom(12)).toString('base64'),
      };

      await expect(decryptAES(tampered, password)).rejects.toThrow();
    });
  });

  describe('Mnemonic Generation and Encryption', () => {
    it('should generate valid 24-word mnemonic with secure entropy', () => {
      const mnemonic = generateMnemonic(256);

      expect(validateMnemonic(mnemonic)).toBe(true);
      expect(mnemonic.split(' ').length).toBe(24);
    });

    it('should generate valid 12-word mnemonic', () => {
      const mnemonic = generateMnemonic(128);

      expect(validateMnemonic(mnemonic)).toBe(true);
      expect(mnemonic.split(' ').length).toBe(12);
    });

    it('should encrypt and decrypt mnemonic correctly', async () => {
      const mnemonic = generateMnemonic(256);
      const password = 'StrongPassword123!@#';

      const encrypted = await encryptMnemonic(mnemonic, password);
      const decrypted = await decryptMnemonic(encrypted, password);

      expect(decrypted).toBe(mnemonic);
      expect(validateMnemonic(decrypted)).toBe(true);
    });

    it('should reject invalid mnemonic encryption', async () => {
      const invalidMnemonic = 'invalid mnemonic phrase that is not valid';
      const password = 'password123';

      await expect(encryptMnemonic(invalidMnemonic, password)).rejects.toThrow(
        'Invalid mnemonic phrase'
      );
    });
  });

  describe('Keystore Security', () => {
    const testPrivateKey = secureRandom(32);
    const testAddress = 'paw1test...';
    const password = 'SecurePassword123!@#';

    it('should create secure keystore with proper encryption', async () => {
      const keystore = await encryptKeystore(testPrivateKey, password, testAddress);

      expect(keystore.version).toBe(4); // New secure version
      expect(keystore.crypto.cipher).toBe('AES-256-GCM');
      expect(keystore.crypto.kdf).toBe('PBKDF2');
      expect(keystore.crypto.kdfparams.iterations).toBeGreaterThanOrEqual(
        RECOMMENDED_PBKDF2_ITERATIONS
      );
      expect(keystore.address).toBe(testAddress);
      expect(validateKeystore(keystore)).toBe(true);
    });

    it('should decrypt keystore correctly', async () => {
      const keystore = await encryptKeystore(testPrivateKey, password, testAddress);
      const decrypted = await decryptKeystore(keystore, password);

      expect(Buffer.from(decrypted).toString('hex')).toBe(
        Buffer.from(testPrivateKey).toString('hex')
      );
    });

    it('should fail with wrong password', async () => {
      const keystore = await encryptKeystore(testPrivateKey, password, testAddress);

      await expect(decryptKeystore(keystore, 'wrong-password')).rejects.toThrow();
    });

    it('should enforce minimum password length', async () => {
      const weakPassword = 'short';

      await expect(
        encryptKeystore(testPrivateKey, weakPassword, testAddress)
      ).rejects.toThrow('Password must be at least 12 characters');
    });

    it('should detect tampered MAC', async () => {
      const keystore = await encryptKeystore(testPrivateKey, password, testAddress);

      // Tamper with MAC
      const tampered = {
        ...keystore,
        crypto: {
          ...keystore.crypto,
          mac: '0'.repeat(64), // Invalid MAC
        },
      };

      await expect(decryptKeystore(tampered, password)).rejects.toThrow(
        'Invalid password or corrupted keystore'
      );
    });
  });

  describe('Key Derivation', () => {
    const password = 'test-password';
    const salt = secureRandom(32);

    it('should derive key with PBKDF2', async () => {
      const key = await deriveKeyBytes(password, salt, RECOMMENDED_PBKDF2_ITERATIONS, 32);

      expect(key.length).toBe(32);
    });

    it('should enforce minimum iterations', async () => {
      await expect(
        deriveKeyBytes(password, salt, MIN_PBKDF2_ITERATIONS - 1, 32)
      ).rejects.toThrow();
    });

    it('should produce consistent keys with same inputs', async () => {
      const key1 = await deriveKeyBytes(password, salt, RECOMMENDED_PBKDF2_ITERATIONS, 32);
      const key2 = await deriveKeyBytes(password, salt, RECOMMENDED_PBKDF2_ITERATIONS, 32);

      expect(Buffer.from(key1).toString('hex')).toBe(
        Buffer.from(key2).toString('hex')
      );
    });

    it('should produce different keys with different salts', async () => {
      const salt2 = secureRandom(32);
      const key1 = await deriveKeyBytes(password, salt, RECOMMENDED_PBKDF2_ITERATIONS, 32);
      const key2 = await deriveKeyBytes(password, salt2, RECOMMENDED_PBKDF2_ITERATIONS, 32);

      expect(Buffer.from(key1).toString('hex')).not.toBe(
        Buffer.from(key2).toString('hex')
      );
    });

    it('should calculate optimal iterations based on performance', async () => {
      const iterations = await calculateOptimalIterations(100);

      expect(iterations).toBeGreaterThanOrEqual(MIN_PBKDF2_ITERATIONS);
    }, 10000); // Longer timeout for benchmarking
  });

  describe('Password Strength Validation', () => {
    it('should reject weak passwords', () => {
      const weak = validatePasswordStrength('password');

      expect(weak.valid).toBe(false);
      expect(weak.strength).toBe('weak');
      expect(weak.errors.length).toBeGreaterThan(0);
    });

    it('should accept strong passwords', () => {
      const strong = validatePasswordStrength('MyStr0ng!P@ssw0rd2024');

      expect(strong.valid).toBe(true);
      expect(strong.strength).toBe('strong');
      expect(strong.errors.length).toBe(0);
    });

    it('should require minimum 12 characters', () => {
      const short = validatePasswordStrength('Short1!');

      expect(short.valid).toBe(false);
      expect(short.errors).toContain('Password must be at least 12 characters');
    });

    it('should require mixed case, numbers, and special characters', () => {
      const result = validatePasswordStrength('alllowercase');

      expect(result.errors).toContain('Password must contain uppercase letters');
      expect(result.errors).toContain('Password must contain numbers');
      expect(result.errors).toContain('Password must contain special characters');
    });
  });

  describe('Constant-Time Comparison', () => {
    it('should compare arrays in constant time', () => {
      const array1 = new Uint8Array([1, 2, 3, 4, 5]);
      const array2 = new Uint8Array([1, 2, 3, 4, 5]);
      const array3 = new Uint8Array([1, 2, 3, 4, 6]);

      expect(constantTimeCompare(array1, array2)).toBe(true);
      expect(constantTimeCompare(array1, array3)).toBe(false);
    });

    it('should compare strings in constant time', () => {
      const str1 = 'test-string-123';
      const str2 = 'test-string-123';
      const str3 = 'test-string-456';

      expect(constantTimeCompareString(str1, str2)).toBe(true);
      expect(constantTimeCompareString(str1, str3)).toBe(false);
    });

    it('should return false for different lengths', () => {
      const array1 = new Uint8Array([1, 2, 3]);
      const array2 = new Uint8Array([1, 2, 3, 4]);

      expect(constantTimeCompare(array1, array2)).toBe(false);
    });
  });

  describe('Secure Memory Wiping', () => {
    it('should wipe sensitive data from memory', () => {
      const sensitiveData = secureRandom(32);
      const original = Buffer.from(sensitiveData).toString('hex');

      secureWipe(sensitiveData);

      // Data should be different after wiping
      expect(Buffer.from(sensitiveData).toString('hex')).not.toBe(original);
    });
  });

  describe('HD Wallet Key Derivation', () => {
    it('should derive deterministic keys from mnemonic', async () => {
      const mnemonic = generateMnemonic(256);
      const path = "m/44'/118'/0'/0/0";

      const privateKey1 = await derivePrivateKey(mnemonic, path);
      const privateKey2 = await derivePrivateKey(mnemonic, path);

      // Should be deterministic
      expect(Buffer.from(privateKey1).toString('hex')).toBe(
        Buffer.from(privateKey2).toString('hex')
      );
    });

    it('should derive different keys for different paths', async () => {
      const mnemonic = generateMnemonic(256);

      const key1 = await derivePrivateKey(mnemonic, "m/44'/118'/0'/0/0");
      const key2 = await derivePrivateKey(mnemonic, "m/44'/118'/0'/0/1");

      expect(Buffer.from(key1).toString('hex')).not.toBe(
        Buffer.from(key2).toString('hex')
      );
    });

    it('should derive correct public key and address', async () => {
      const mnemonic = generateMnemonic(256);
      const privateKey = await derivePrivateKey(mnemonic);
      const publicKey = derivePublicKey(privateKey);
      const address = publicKeyToAddress(publicKey);

      expect(publicKey.length).toBe(33); // Compressed secp256k1
      expect(address).toMatch(/^paw1[a-z0-9]+$/);
    });
  });

  describe('Security Edge Cases', () => {
    it('should handle empty password gracefully', async () => {
      await expect(encryptAES('data', '')).rejects.toThrow();
    });

    it('should handle very long passwords', async () => {
      const longPassword = 'a'.repeat(1000);
      const data = 'test data';

      const encrypted = await encryptAES(data, longPassword);
      const decrypted = await decryptAES(encrypted, longPassword);

      expect(decrypted).toBe(data);
    });

    it('should handle unicode in passwords', async () => {
      const unicodePassword = '密码🔐パスワード';
      const data = 'test data';

      const encrypted = await encryptAES(data, unicodePassword);
      const decrypted = await decryptAES(encrypted, unicodePassword);

      expect(decrypted).toBe(data);
    });

    it('should handle unicode in encrypted data', async () => {
      const password = 'SecurePassword123!';
      const unicodeData = 'Test with emojis 🔐💎🚀 and 中文';

      const encrypted = await encryptAES(unicodeData, password);
      const decrypted = await decryptAES(encrypted, password);

      expect(decrypted).toBe(unicodeData);
    });
  });
});
