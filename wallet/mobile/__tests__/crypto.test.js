/**
 * Crypto utility tests
 */

import CryptoJS from 'crypto-js';
import {
  generateMnemonic,
  validateMnemonic,
  derivePrivateKeyFromMnemonic,
  getPublicKey,
  deriveAddress,
  generateWallet,
  importWalletFromMnemonic,
  importWalletFromPrivateKey,
  signMessage,
  verifySignature,
  encrypt,
  decrypt,
  decryptWithMigration,
  isLegacyCiphertext,
} from '../src/utils/crypto';

describe('Crypto Utils', () => {
  describe('Mnemonic Generation', () => {
    test('should generate a valid 24-word mnemonic', () => {
      const mnemonic = generateMnemonic();
      expect(mnemonic).toBeTruthy();
      expect(mnemonic.split(' ').length).toBe(24);
      expect(validateMnemonic(mnemonic)).toBe(true);
    });

    test('should generate a valid 12-word mnemonic', () => {
      const mnemonic = generateMnemonic(128);
      expect(mnemonic).toBeTruthy();
      expect(mnemonic.split(' ').length).toBe(12);
      expect(validateMnemonic(mnemonic)).toBe(true);
    });

    test('should validate correct mnemonic', () => {
      const validMnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
      expect(validateMnemonic(validMnemonic)).toBe(true);
    });

    test('should reject invalid mnemonic', () => {
      const invalidMnemonic = 'invalid mnemonic phrase test';
      expect(validateMnemonic(invalidMnemonic)).toBe(false);
    });
  });

  describe('Key Derivation', () => {
    test('should derive private key from mnemonic', () => {
      const mnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
      const privateKey = derivePrivateKeyFromMnemonic(mnemonic);
      expect(privateKey).toBeTruthy();
      expect(privateKey.length).toBe(64); // 32 bytes hex
    });

    test('should derive public key from private key', () => {
      const privateKey =
        'a'.repeat(64); // Mock private key
      const publicKey = getPublicKey(privateKey);
      expect(publicKey).toBeTruthy();
      expect(publicKey.length).toBe(66); // Compressed public key
    });

    test('should derive PAW address from public key', () => {
      const publicKey = '02' + 'a'.repeat(64);
      const address = deriveAddress(publicKey, 'paw');
      expect(address).toBeTruthy();
      expect(address.startsWith('paw')).toBe(true);
    });

    test('should generate consistent address from same mnemonic', () => {
      const mnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
      const wallet1 = importWalletFromMnemonic(mnemonic);
      const wallet2 = importWalletFromMnemonic(mnemonic);
      expect(wallet1.address).toBe(wallet2.address);
    });
  });

  describe('Wallet Generation', () => {
    test('should generate a complete wallet', () => {
      const wallet = generateWallet();
      expect(wallet.mnemonic).toBeTruthy();
      expect(wallet.privateKey).toBeTruthy();
      expect(wallet.publicKey).toBeTruthy();
      expect(wallet.address).toBeTruthy();
      expect(wallet.address.startsWith('paw')).toBe(true);
    });

    test('should import wallet from mnemonic', () => {
      const mnemonic = generateMnemonic();
      const wallet = importWalletFromMnemonic(mnemonic);
      expect(wallet.privateKey).toBeTruthy();
      expect(wallet.publicKey).toBeTruthy();
      expect(wallet.address).toBeTruthy();
      expect(wallet.mnemonic).toBe(mnemonic);
    });

    test('should import wallet from private key', () => {
      const privateKey =
        'a'.repeat(64);
      const wallet = importWalletFromPrivateKey(privateKey);
      expect(wallet.privateKey).toBe(privateKey);
      expect(wallet.publicKey).toBeTruthy();
      expect(wallet.address).toBeTruthy();
    });

    test('should throw error for invalid mnemonic import', () => {
      expect(() => {
        importWalletFromMnemonic('invalid mnemonic');
      }).toThrow('Invalid mnemonic phrase');
    });
  });

  describe('Message Signing', () => {
    test('should sign a message', () => {
      const wallet = generateWallet();
      const message = 'Hello, PAW!';
      const signature = signMessage(message, wallet.privateKey);
      expect(signature).toBeTruthy();
      expect(signature.r).toBeTruthy();
      expect(signature.s).toBeTruthy();
    });

    test('should verify a valid signature', () => {
      const wallet = generateWallet();
      const message = 'Hello, PAW!';
      const signature = signMessage(message, wallet.privateKey);
      const isValid = verifySignature(message, signature, wallet.publicKey);
      expect(isValid).toBe(true);
    });

    test('should reject invalid signature', () => {
      const wallet1 = generateWallet();
      const wallet2 = generateWallet();
      const message = 'Hello, PAW!';
      const signature = signMessage(message, wallet1.privateKey);
      const isValid = verifySignature(message, signature, wallet2.publicKey);
      expect(isValid).toBe(false);
    });
  });

  describe('Encryption', () => {
    test('should encrypt and decrypt data', () => {
      const data = 'sensitive data';
      const password = 'strong_password';
      const encrypted = encrypt(data, password);
      const decrypted = decrypt(encrypted, password);
      expect(decrypted).toBe(data);
    });

    test('includes metadata for new encryption format', () => {
      const encrypted = encrypt('payload', 'password');
      const parsed = JSON.parse(encrypted);
      expect(parsed.v).toBe(1);
      expect(parsed.ct).toBeTruthy();
      expect(parsed.salt).toBeTruthy();
      expect(parsed.iv).toBeTruthy();
      expect(parsed.iter).toBeGreaterThan(0);
    });

    test('can decrypt legacy ciphertext for backward compatibility', () => {
      const data = 'legacy data';
      const password = 'legacy';
      const legacyCipher = CryptoJS.AES.encrypt(data, password).toString();
      const decrypted = decrypt(legacyCipher, password);
      expect(decrypted).toBe(data);
    });

    test('detects legacy payloads and produces migrated ciphertext', () => {
      const data = 'to migrate';
      const password = 'legacy';
      const legacyCipher = CryptoJS.AES.encrypt(data, password).toString();
      expect(isLegacyCiphertext(legacyCipher)).toBe(true);
      const result = decryptWithMigration(legacyCipher, password);
      expect(result.plaintext).toBe(data);
      expect(result.migratedCiphertext).toBeTruthy();
      expect(isLegacyCiphertext(result.migratedCiphertext)).toBe(false);
      expect(decrypt(result.migratedCiphertext, password)).toBe(data);
    });

    test('should fail to decrypt with wrong password', () => {
      const data = 'sensitive data';
      const password = 'correct_password';
      const wrongPassword = 'wrong_password';
      const encrypted = encrypt(data, password);
      const decrypted = decrypt(encrypted, wrongPassword);
      expect(decrypted).not.toBe(data);
    });

    test('encrypted data should be different from original', () => {
      const data = 'sensitive data';
      const password = 'password';
      const encrypted = encrypt(data, password);
      expect(encrypted).not.toBe(data);
    });
  });
});
