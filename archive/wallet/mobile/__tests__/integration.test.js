/**
 * Integration Tests
 * Test complete workflows
 */

import WalletService from '../src/services/WalletService';
import KeyStore from '../src/services/KeyStore';
import {validateMnemonic} from '../src/utils/crypto';

// Mock dependencies
jest.mock('../src/services/KeyStore');
jest.mock('../src/services/PawAPI');
jest.mock('../src/services/BiometricAuth');

describe('Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Wallet Creation Flow', () => {
    it('should create and store wallet successfully', async () => {
      KeyStore.storeWallet.mockResolvedValue(true);
      KeyStore.storeMetadata.mockResolvedValue(true);

      const wallet = await WalletService.createWallet({
        walletName: 'Test Wallet',
        password: 'SecurePassword123',
        useBiometric: false,
      });

      // Verify wallet was created
      expect(wallet.address).toBeDefined();
      expect(wallet.address).toMatch(/^paw1/);
      expect(wallet.publicKey).toBeDefined();
      expect(wallet.mnemonic).toBeDefined();

      // Verify mnemonic is valid
      expect(validateMnemonic(wallet.mnemonic)).toBe(true);

      // Verify storage was called
      expect(KeyStore.storeWallet).toHaveBeenCalled();
      expect(KeyStore.storeMetadata).toHaveBeenCalledWith(
        expect.objectContaining({
          address: wallet.address,
          name: 'Test Wallet',
        }),
      );
    });
  });

  describe('Wallet Import Flow', () => {
    it('should import wallet from mnemonic', async () => {
      const mnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';

      KeyStore.storeWallet.mockResolvedValue(true);
      KeyStore.storeMetadata.mockResolvedValue(true);

      const wallet = await WalletService.importWallet({
        mnemonic,
        walletName: 'Imported Wallet',
        password: 'SecurePassword123',
      });

      expect(wallet.address).toBeDefined();
      expect(wallet.address).toMatch(/^paw1/);
      expect(KeyStore.storeWallet).toHaveBeenCalled();
    });
  });

  describe('Balance Check Flow', () => {
    it('should retrieve and format balance', async () => {
      KeyStore.getAddress.mockResolvedValue('paw1testaddress');

      const PawAPI = require('../src/services/PawAPI').default;
      PawAPI.getBalance.mockResolvedValue({
        balances: [{denom: 'upaw', amount: '5000000'}],
      });

      const balance = await WalletService.getBalance();

      expect(balance.amount).toBe(5000000);
      expect(balance.formatted).toBe('5.000000');
      expect(balance.denom).toBe('PAW');
    });
  });
});
