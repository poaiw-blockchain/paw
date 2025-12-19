import * as Keychain from 'react-native-keychain';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  encrypt,
  decrypt,
  decryptWithMigration,
  isLegacyCiphertext,
} from '../utils/crypto';

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
      const privateKeyResult = decryptWithMigration(payload.privateKey, password);
      const mnemonicResult = decryptWithMigration(payload.mnemonic, password);
      if (!privateKeyResult.plaintext || !mnemonicResult.plaintext) {
        throw new Error('Invalid password or corrupted wallet data');
      }

      // If either field is legacy, re-encrypt to the hardened format.
      if (isLegacyCiphertext(payload.privateKey) || isLegacyCiphertext(payload.mnemonic)) {
        const updatedPayload = {
          privateKey: privateKeyResult.migratedCiphertext || payload.privateKey,
          mnemonic: mnemonicResult.migratedCiphertext || payload.mnemonic,
        };
        await Keychain.setGenericPassword('paw_wallet', JSON.stringify(updatedPayload), {
          accessible: Keychain.ACCESSIBLE.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
        });
      }

      return {
        privateKey: privateKeyResult.plaintext,
        mnemonic: mnemonicResult.plaintext,
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

// Hardware signing helper (BLE Ledger) with signer address guard
export async function signWithLedgerAmino(signDoc, accountIndex = 0) {
  const { signWithLedger } = require('./LedgerHardwareSigner');
  return signWithLedger(signDoc, accountIndex);
}

const KeyStore = new KeyStoreService();
export default KeyStore;
export { signWithLedgerAmino };
