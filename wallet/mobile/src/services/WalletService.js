import KeyStore from './KeyStore';
import StorageService from './StorageService';
import BiometricAuth from './BiometricAuth';
import PawAPI from './PawAPI';
import {
  generateWallet,
  importWalletFromMnemonic,
  importWalletFromPrivateKey,
} from '../utils/crypto';

class WalletServiceClass {
  async createWallet({walletName, password, useBiometric = false}) {
    if (!password || password.length < 8) {
      throw new Error('Password must be at least 8 characters');
    }

    const wallet = generateWallet();

    await KeyStore.storeWallet(wallet.privateKey, wallet.mnemonic, password);
    await KeyStore.storeMetadata({
      address: wallet.address,
      name: walletName || 'My Wallet',
      biometricEnabled: useBiometric,
    });

    if (useBiometric) {
      const availability = await BiometricAuth.checkAvailability();
      if (availability.available) {
        await BiometricAuth.createKeys();
      }
    }

    return wallet;
  }

  async importWallet({mnemonic, walletName, password}) {
    if (!password || password.length < 8) {
      throw new Error('Password must be at least 8 characters');
    }

    const wallet = importWalletFromMnemonic(mnemonic);
    await KeyStore.storeWallet(wallet.privateKey, wallet.mnemonic, password);
    await KeyStore.storeMetadata({
      address: wallet.address,
      name: walletName || 'My Wallet',
      biometricEnabled: false,
    });

    return wallet;
  }

  async importPrivateKey({privateKey, walletName, password}) {
    const wallet = importWalletFromPrivateKey(privateKey);
    await KeyStore.storeWallet(wallet.privateKey, '', password);
    await KeyStore.storeMetadata({
      address: wallet.address,
      name: walletName || 'My Wallet',
      biometricEnabled: false,
    });
    return wallet;
  }

  async getBalance() {
    const address = await KeyStore.getAddress();
    if (!address) {
      throw new Error('No wallet found');
    }

    const balanceData = await PawAPI.getBalance(address);
    const rawAmount =
      balanceData?.balances?.find(b => b.denom === 'upaw')?.amount || '0';
    const amount = parseInt(rawAmount, 10);
    const formatted = (amount / 1_000_000).toFixed(6);

    return {
      amount,
      formatted,
      denom: 'PAW',
    };
  }

  async hasWallet() {
    return KeyStore.hasWallet();
  }

  async getWalletInfo() {
    const metadata = (await KeyStore.retrieveMetadata()) || {};
    const address = await KeyStore.getAddress();
    const name = await KeyStore.getName();
    return {
      address,
      name,
      createdAt: metadata.createdAt,
      biometricEnabled: metadata.biometricEnabled || false,
    };
  }

  async getTransactions(limit = 20) {
    const address = await KeyStore.getAddress();
    if (!address) {
      return [];
    }
    const transactions = await PawAPI.getTransactionsByAddress(address, limit);
    await KeyStore.storeTransactions(transactions);
    return transactions;
  }

  async deleteWallet() {
    await KeyStore.clearAll();
    await StorageService.clear();
    return true;
  }
}

const WalletService = new WalletServiceClass();
export default WalletService;
