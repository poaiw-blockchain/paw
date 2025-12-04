import * as Keychain from 'react-native-keychain';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {encrypt, decrypt} from '../utils/crypto';

const STORAGE_KEYS = {
  METADATA: '@PAW:wallet_metadata',
  ADDRESS: '@PAW:wallet_address',
  NAME: '@PAW:wallet_name',
  TRANSACTIONS: '@PAW:transactions',
};

class KeyStoreService {
  constructor() {
    this.isInitialized = false;
  }

  async initialize() {
    this.isInitialized = true;
    return true;
  }

  async storeWallet(privateKey, mnemonic, password) {
    try {
      const payload = {
        privateKey: encrypt(privateKey, password),
        mnemonic: encrypt(mnemonic, password),
      };
      await Keychain.setGenericPassword('paw_wallet', JSON.stringify(payload), {
        accessible: Keychain.ACCESSIBLE.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
      });
      return true;
    } catch (error) {
      throw new Error('Failed to store wallet credentials');
    }
  }

  async retrieveWallet(password) {
    try {
      const credentials = await Keychain.getGenericPassword();
      if (!credentials) {
        return null;
      }
      const payload = JSON.parse(credentials.password);
      return {
        privateKey: decrypt(payload.privateKey, password),
        mnemonic: decrypt(payload.mnemonic, password),
      };
    } catch (error) {
      throw new Error('Failed to retrieve wallet credentials');
    }
  }

  async hasWallet() {
    const credentials = await Keychain.getGenericPassword();
    return !!credentials;
  }

  async deleteWallet() {
    try {
      await Keychain.resetGenericPassword();
      await AsyncStorage.multiRemove([
        STORAGE_KEYS.METADATA,
        STORAGE_KEYS.ADDRESS,
        STORAGE_KEYS.NAME,
        STORAGE_KEYS.TRANSACTIONS,
      ]);
      return true;
    } catch (error) {
      throw new Error('Failed to delete wallet');
    }
  }

  async storeMetadata(metadata) {
    const payload = {
      ...metadata,
      createdAt: metadata.createdAt || new Date().toISOString(),
    };
    await AsyncStorage.setItem(STORAGE_KEYS.METADATA, JSON.stringify(payload));
    if (metadata.address) {
      await AsyncStorage.setItem(STORAGE_KEYS.ADDRESS, metadata.address);
    }
    if (metadata.name) {
      await AsyncStorage.setItem(STORAGE_KEYS.NAME, metadata.name);
    }
    return true;
  }

  async retrieveMetadata() {
    const stored = await AsyncStorage.getItem(STORAGE_KEYS.METADATA);
    return stored ? JSON.parse(stored) : null;
  }

  async getAddress() {
    return AsyncStorage.getItem(STORAGE_KEYS.ADDRESS);
  }

  async getName() {
    const name = await AsyncStorage.getItem(STORAGE_KEYS.NAME);
    return name || 'My Wallet';
  }

  async storeTransactions(transactions) {
    await AsyncStorage.setItem(
      STORAGE_KEYS.TRANSACTIONS,
      JSON.stringify(transactions),
    );
    return true;
  }

  async retrieveTransactions() {
    const stored = await AsyncStorage.getItem(STORAGE_KEYS.TRANSACTIONS);
    return stored ? JSON.parse(stored) : [];
  }

  async clearAll() {
    await this.deleteWallet();
    return true;
  }
}

const KeyStore = new KeyStoreService();
export default KeyStore;
