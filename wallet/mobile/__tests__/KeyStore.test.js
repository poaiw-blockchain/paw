/**
 * KeyStore service tests
 */

import KeyStore from '../src/services/KeyStore';
import * as Keychain from 'react-native-keychain';
import AsyncStorage from '@react-native-async-storage/async-storage';

describe('KeyStore Service', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      await KeyStore.initialize();
      expect(KeyStore.isInitialized).toBe(true);
    });
  });

  describe('Wallet Storage', () => {
    test('should store wallet credentials', async () => {
      const privateKey = 'test_private_key';
      const mnemonic = 'test mnemonic phrase';
      const password = 'test_password';

      const result = await KeyStore.storeWallet(privateKey, mnemonic, password);
      expect(result).toBe(true);
      expect(Keychain.setGenericPassword).toHaveBeenCalled();
    });

    test('should retrieve wallet credentials', async () => {
      const password = 'test_password';

      // Mock successful retrieval with properly encrypted data
      const {encrypt} = require('../src/utils/crypto');
      Keychain.getGenericPassword.mockResolvedValue({
        username: 'paw_wallet',
        password: JSON.stringify({
          privateKey: encrypt('test_private_key', password),
          mnemonic: encrypt('test mnemonic phrase', password),
        }),
      });

      const result = await KeyStore.retrieveWallet(password);
      expect(result).toBeDefined();
      expect(result.privateKey).toBe('test_private_key');
      expect(result.mnemonic).toBe('test mnemonic phrase');
    });

    test('should return null when no wallet exists', async () => {
      Keychain.getGenericPassword.mockResolvedValue(false);

      const result = await KeyStore.retrieveWallet('password');
      expect(result).toBeNull();
    });

    test('should check if wallet exists', async () => {
      Keychain.getGenericPassword.mockResolvedValue({
        username: 'paw_wallet',
        password: 'data',
      });

      const exists = await KeyStore.hasWallet();
      expect(exists).toBe(true);
    });

    test('should return false when wallet does not exist', async () => {
      Keychain.getGenericPassword.mockResolvedValue(false);

      const exists = await KeyStore.hasWallet();
      expect(exists).toBe(false);
    });

    test('should delete wallet', async () => {
      const result = await KeyStore.deleteWallet();
      expect(result).toBe(true);
      expect(Keychain.resetGenericPassword).toHaveBeenCalled();
      expect(AsyncStorage.removeItem).toHaveBeenCalled();
    });
  });

  describe('Metadata Storage', () => {
    test('should store wallet metadata', async () => {
      const metadata = {
        address: 'paw1test',
        name: 'Test Wallet',
      };

      const result = await KeyStore.storeMetadata(metadata);
      expect(result).toBe(true);
      expect(AsyncStorage.setItem).toHaveBeenCalled();
    });

    test('should retrieve wallet metadata', async () => {
      const metadata = {
        address: 'paw1test',
        name: 'Test Wallet',
      };

      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(metadata));

      const result = await KeyStore.retrieveMetadata();
      expect(result).toEqual(metadata);
    });

    test('should return null when no metadata exists', async () => {
      AsyncStorage.getItem.mockResolvedValue(null);

      const result = await KeyStore.retrieveMetadata();
      expect(result).toBeNull();
    });

    test('should get wallet address', async () => {
      const address = 'paw1test';
      AsyncStorage.getItem.mockResolvedValue(address);

      const result = await KeyStore.getAddress();
      expect(result).toBe(address);
    });

    test('should get wallet name', async () => {
      const name = 'Test Wallet';
      AsyncStorage.getItem.mockResolvedValue(name);

      const result = await KeyStore.getName();
      expect(result).toBe(name);
    });

    test('should return default name when none exists', async () => {
      AsyncStorage.getItem.mockResolvedValue(null);

      const result = await KeyStore.getName();
      expect(result).toBe('My Wallet');
    });
  });

  describe('Transaction Storage', () => {
    test('should store transactions', async () => {
      const transactions = [
        {txhash: 'ABC123', amount: '100'},
        {txhash: 'DEF456', amount: '200'},
      ];

      const result = await KeyStore.storeTransactions(transactions);
      expect(result).toBe(true);
      expect(AsyncStorage.setItem).toHaveBeenCalledWith(
        '@PAW:transactions',
        JSON.stringify(transactions),
      );
    });

    test('should retrieve transactions', async () => {
      const transactions = [
        {txhash: 'ABC123', amount: '100'},
      ];

      AsyncStorage.getItem.mockResolvedValue(JSON.stringify(transactions));

      const result = await KeyStore.retrieveTransactions();
      expect(result).toEqual(transactions);
    });

    test('should return empty array when no transactions', async () => {
      AsyncStorage.getItem.mockResolvedValue(null);

      const result = await KeyStore.retrieveTransactions();
      expect(result).toEqual([]);
    });
  });

  describe('Clear All', () => {
    test('should clear all data', async () => {
      const result = await KeyStore.clearAll();
      expect(result).toBe(true);
      expect(Keychain.resetGenericPassword).toHaveBeenCalled();
      expect(AsyncStorage.clear).toHaveBeenCalled();
    });
  });

  describe('Error Handling', () => {
    test('should handle storage errors', async () => {
      Keychain.setGenericPassword.mockRejectedValue(new Error('Storage error'));

      await expect(
        KeyStore.storeWallet('key', 'mnemonic', 'password'),
      ).rejects.toThrow('Failed to store wallet credentials');
    });

    test('should handle retrieval errors', async () => {
      Keychain.getGenericPassword.mockRejectedValue(new Error('Retrieval error'));

      await expect(KeyStore.retrieveWallet('password')).rejects.toThrow(
        'Failed to retrieve wallet credentials',
      );
    });

    test('should handle deletion errors', async () => {
      Keychain.resetGenericPassword.mockRejectedValue(new Error('Delete error'));

      await expect(KeyStore.deleteWallet()).rejects.toThrow('Failed to delete wallet');
    });
  });
});
