/**
 * Wallet Service Tests
 */

import WalletService from '../src/services/WalletService';
import KeyStore from '../src/services/KeyStore';
import BiometricAuth from '../src/services/BiometricAuth';
import PawAPI from '../src/services/PawAPI';

// Mock dependencies
jest.mock('../src/services/KeyStore');
jest.mock('../src/services/BiometricAuth');
jest.mock('../src/services/PawAPI');

describe('WalletService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createWallet', () => {
    it('should create a new wallet successfully', async () => {
      const mockWallet = {
        address: 'paw1test',
        publicKey: 'pubkey',
        mnemonic: 'test mnemonic words here',
      };

      BiometricAuth.checkAvailability.mockResolvedValue({available: false});
      KeyStore.storeWallet.mockResolvedValue(true);
      KeyStore.storeMetadata.mockResolvedValue(true);

      const result = await WalletService.createWallet({
        walletName: 'Test Wallet',
        password: 'password123',
        useBiometric: false,
      });

      expect(result.address).toBeDefined();
      expect(result.publicKey).toBeDefined();
      expect(result.mnemonic).toBeDefined();
      expect(KeyStore.storeWallet).toHaveBeenCalled();
      expect(KeyStore.storeMetadata).toHaveBeenCalled();
    });

    it('should throw error with short password', async () => {
      await expect(
        WalletService.createWallet({
          walletName: 'Test',
          password: 'short',
        }),
      ).rejects.toThrow('Password must be at least 8 characters');
    });

    it('should setup biometric when requested', async () => {
      BiometricAuth.checkAvailability.mockResolvedValue({available: true});
      BiometricAuth.createKeys.mockResolvedValue({publicKey: 'key'});
      KeyStore.storeWallet.mockResolvedValue(true);
      KeyStore.storeMetadata.mockResolvedValue(true);

      await WalletService.createWallet({
        walletName: 'Test',
        password: 'password123',
        useBiometric: true,
      });

      expect(BiometricAuth.createKeys).toHaveBeenCalled();
    });
  });

  describe('importWallet', () => {
    const validMnemonic =
      'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';

    it('should import wallet from valid mnemonic', async () => {
      BiometricAuth.checkAvailability.mockResolvedValue({available: false});
      KeyStore.storeWallet.mockResolvedValue(true);
      KeyStore.storeMetadata.mockResolvedValue(true);

      const result = await WalletService.importWallet({
        mnemonic: validMnemonic,
        walletName: 'Imported',
        password: 'password123',
      });

      expect(result.address).toBeDefined();
      expect(result.publicKey).toBeDefined();
      expect(KeyStore.storeWallet).toHaveBeenCalled();
    });

    it('should throw error with invalid mnemonic', async () => {
      await expect(
        WalletService.importWallet({
          mnemonic: 'invalid mnemonic',
          walletName: 'Test',
          password: 'password123',
        }),
      ).rejects.toThrow();
    });
  });

  describe('getBalance', () => {
    it('should return formatted balance', async () => {
      KeyStore.getAddress.mockResolvedValue('paw1test');
      PawAPI.getBalance.mockResolvedValue({
        balances: [{denom: 'upaw', amount: '1000000'}],
      });

      const balance = await WalletService.getBalance();

      expect(balance.amount).toBe(1000000);
      expect(balance.formatted).toBe('1.000000');
      expect(balance.denom).toBe('PAW');
    });

    it('should handle zero balance', async () => {
      KeyStore.getAddress.mockResolvedValue('paw1test');
      PawAPI.getBalance.mockResolvedValue({balances: []});

      const balance = await WalletService.getBalance();

      expect(balance.amount).toBe(0);
      expect(balance.formatted).toBe('0.000000');
    });

    it('should throw error when no wallet', async () => {
      KeyStore.getAddress.mockResolvedValue(null);

      await expect(WalletService.getBalance()).rejects.toThrow('No wallet found');
    });
  });

  describe('hasWallet', () => {
    it('should return true when wallet exists', async () => {
      KeyStore.hasWallet.mockResolvedValue(true);

      const result = await WalletService.hasWallet();

      expect(result).toBe(true);
    });

    it('should return false when no wallet', async () => {
      KeyStore.hasWallet.mockResolvedValue(false);

      const result = await WalletService.hasWallet();

      expect(result).toBe(false);
    });
  });

  describe('getWalletInfo', () => {
    it('should return complete wallet info', async () => {
      KeyStore.retrieveMetadata.mockResolvedValue({
        createdAt: '2024-01-01',
        biometricEnabled: true,
      });
      KeyStore.getAddress.mockResolvedValue('paw1test');
      KeyStore.getName.mockResolvedValue('My Wallet');

      const info = await WalletService.getWalletInfo();

      expect(info.address).toBe('paw1test');
      expect(info.name).toBe('My Wallet');
      expect(info.biometricEnabled).toBe(true);
    });
  });

  describe('deleteWallet', () => {
    it('should delete wallet successfully', async () => {
      KeyStore.clearAll.mockResolvedValue(true);

      const result = await WalletService.deleteWallet();

      expect(result).toBe(true);
      expect(KeyStore.clearAll).toHaveBeenCalled();
    });
  });
});
