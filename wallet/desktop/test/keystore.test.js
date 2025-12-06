/**
 * Keystore Service Tests
 */

import { KeystoreService } from '../src/services/keystore';

describe('KeystoreService', () => {
  let keystoreService;

  beforeEach(() => {
    keystoreService = new KeystoreService();
    jest.clearAllMocks();
    window.electron.store.get.mockResolvedValue(null);
    window.electron.store.set.mockResolvedValue();
    window.electron.store.delete.mockResolvedValue();
    window.electron.store.clear.mockResolvedValue();
  });

  describe('Mnemonic Generation', () => {
    test('should generate valid 24-word mnemonic', async () => {
      const mnemonic = await keystoreService.generateMnemonic();

      expect(mnemonic).toBeDefined();
      expect(typeof mnemonic).toBe('string');

      const words = mnemonic.split(' ');
      expect(words.length).toBe(24);
    });

    test('should generate unique mnemonics', async () => {
      const mnemonic1 = await keystoreService.generateMnemonic();
      const mnemonic2 = await keystoreService.generateMnemonic();

      expect(mnemonic1).not.toBe(mnemonic2);
    });
  });

  describe('Mnemonic Validation', () => {
    test('should validate correct mnemonic', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      const isValid = keystoreService.validateMnemonic(mnemonic);

      expect(isValid).toBe(true);
    });

    test('should reject invalid mnemonic', () => {
      const invalidMnemonic = 'invalid mnemonic phrase test';
      const isValid = keystoreService.validateMnemonic(invalidMnemonic);

      expect(isValid).toBe(false);
    });
  });

  describe('Wallet Creation', () => {
    test('should create wallet from mnemonic', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      const password = 'test-password-123';

      const wallet = await keystoreService.createWallet(mnemonic, password);

      expect(wallet).toBeDefined();
      expect(wallet.address).toBeDefined();
      expect(wallet.address).toMatch(/^paw1/);
      expect(wallet.publicKey).toBeDefined();
      expect(wallet.createdAt).toBeDefined();
    });

    test('should reject invalid mnemonic', async () => {
      const invalidMnemonic = 'invalid mnemonic phrase';
      const password = 'test-password-123';

      await expect(
        keystoreService.createWallet(invalidMnemonic, password)
      ).rejects.toThrow();
    });

    test('should save wallet to storage', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      const password = 'test-password-123';

      await keystoreService.createWallet(mnemonic, password);

      expect(window.electron.store.set).toHaveBeenCalled();
    });
  });

  describe('Wallet Retrieval', () => {
    test('should return null if no wallet exists', async () => {
      window.electron.store.get.mockResolvedValue(null);
      const wallet = await keystoreService.getWallet();

      expect(wallet).toBeNull();
    });

    test('should retrieve wallet data', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      const password = 'test-password-123';

      const created = await keystoreService.createWallet(mnemonic, password);

      const storedPayload = window.electron.store.set.mock.calls[0][1];
      window.electron.store.get.mockResolvedValue(storedPayload);

      const retrieved = await keystoreService.getWallet();

      expect(retrieved).toBeDefined();
      expect(retrieved.address).toBe(created.address);
      expect(retrieved.publicKey).toBe(created.publicKey);
    });
  });

  describe('Wallet Unlock', () => {
    test('should unlock wallet with correct password', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      const password = 'test-password-123';

      await keystoreService.createWallet(mnemonic, password);
      const storedPayload = window.electron.store.set.mock.calls[0][1];
      window.electron.store.get.mockResolvedValue(storedPayload);

      const unlocked = await keystoreService.unlockWallet(password);
      expect(unlocked.address).toBe(storedPayload.address);
      expect(unlocked.privateKey).toBe(mnemonic);
    });

    test('should reject incorrect password', async () => {
      window.electron.store.get.mockResolvedValue({
        address: 'paw1test',
        passwordHash: 'wrong-hash',
        encryptedMnemonic: 'encrypted'
      });

      await expect(
        keystoreService.unlockWallet('wrong-password')
      ).rejects.toThrow();
    });

    test('should reject if no wallet exists', async () => {
      window.electron.store.get.mockResolvedValue(null);
      await expect(
        keystoreService.unlockWallet('any-password')
      ).rejects.toThrow('No wallet found');
    });
  });

  describe('Password Hashing', () => {
    test('should hash password consistently', async () => {
      const password = 'test-password';

      const hash1 = await keystoreService.hashPassword(password);
      const hash2 = await keystoreService.hashPassword(password);

      expect(hash1).toBe(hash2);
    });

    test('should produce different hashes for different passwords', async () => {
      const hash1 = await keystoreService.hashPassword('password1');
      const hash2 = await keystoreService.hashPassword('password2');

      expect(hash1).not.toBe(hash2);
    });
  });

  describe('Encryption/Decryption', () => {
    test('should encrypt data with random salt and prefix', async () => {
      const data = 'sensitive data';
      const password = 'encryption-key';

      const encrypted1 = await keystoreService.encryptData(data, password);
      const encrypted2 = await keystoreService.encryptData(data, password);

      expect(encrypted1).toMatch(/^paw:v1:/);
      expect(encrypted1).not.toBe(data);
      expect(encrypted1).not.toBe(encrypted2);
    });

    test('should decrypt AES-GCM payload', async () => {
      const data = 'sensitive data';
      const password = 'encryption-key';

      const encrypted = await keystoreService.encryptData(data, password);
      const decrypted = await keystoreService.decryptData(encrypted, password);

      expect(decrypted).toBe(data);
    });

    test('should decrypt legacy XOR payloads', async () => {
      const data = 'legacy secret';
      const password = 'legacy-key';
      const key = await keystoreService.hashPassword(password);

      const legacyEncrypted = legacyXorEncrypt(data, key);
      const decrypted = await keystoreService.decryptData(legacyEncrypted, password);

      expect(decrypted).toBe(data);
    });
  });

  describe('Wallet Management', () => {
    test('should clear wallet', async () => {
      const mnemonic = await keystoreService.generateMnemonic();
      await keystoreService.createWallet(mnemonic, 'password');

      await keystoreService.clearWallet();

      expect(window.electron.store.delete).toHaveBeenCalled();
    });

    test('should check if wallet exists', async () => {
      window.electron.store.get
        .mockResolvedValueOnce(null)
        .mockResolvedValueOnce({ address: 'paw1test' });

      let exists = await keystoreService.hasWallet();
      expect(exists).toBe(false);

      exists = await keystoreService.hasWallet();
      expect(exists).toBe(true);
    });
  });
});

function legacyXorEncrypt(text, key) {
  let result = '';
  for (let i = 0; i < text.length; i++) {
    result += String.fromCharCode(
      text.charCodeAt(i) ^ key.charCodeAt(i % key.length)
    );
  }
  return Buffer.from(result, 'binary').toString('base64');
}
