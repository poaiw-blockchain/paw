import * as bip39 from 'bip39';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { toBech32 } from '@cosmjs/encoding';
import { sha256 } from '@cosmjs/crypto';

export class KeystoreService {
  constructor() {
    this.storageKey = 'paw-wallet';
    this.prefix = 'paw';
  }

  /**
   * Generate a new 24-word mnemonic phrase
   */
  async generateMnemonic() {
    try {
      // Generate 256-bit entropy for 24 words
      const mnemonic = bip39.generateMnemonic(256);
      return mnemonic;
    } catch (error) {
      console.error('Failed to generate mnemonic:', error);
      throw new Error('Failed to generate mnemonic phrase');
    }
  }

  /**
   * Validate mnemonic phrase
   */
  validateMnemonic(mnemonic) {
    return bip39.validateMnemonic(mnemonic);
  }

  /**
   * Create wallet from mnemonic and save encrypted
   */
  async createWallet(mnemonic, password) {
    try {
      // Validate mnemonic
      if (!this.validateMnemonic(mnemonic)) {
        throw new Error('Invalid mnemonic phrase');
      }

      // Create wallet from mnemonic
      const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
        prefix: this.prefix
      });

      // Get accounts
      const accounts = await wallet.getAccounts();
      if (!accounts || accounts.length === 0) {
        throw new Error('Failed to derive account from mnemonic');
      }

      const account = accounts[0];

      // Create wallet data
      const walletData = {
        address: account.address,
        publicKey: Buffer.from(account.pubkey).toString('hex'),
        createdAt: new Date().toISOString()
      };

      // Encrypt and save mnemonic
      const encryptedMnemonic = await this.encryptData(mnemonic, password);
      const passwordHash = await this.hashPassword(password);

      if (window.electron?.store) {
        await window.electron.store.set(this.storageKey, {
          ...walletData,
          encryptedMnemonic,
          passwordHash
        });
      } else {
        // Fallback to localStorage for testing
        localStorage.setItem(this.storageKey, JSON.stringify({
          ...walletData,
          encryptedMnemonic,
          passwordHash
        }));
      }

      return walletData;
    } catch (error) {
      console.error('Failed to create wallet:', error);
      throw new Error(error.message || 'Failed to create wallet');
    }
  }

  /**
   * Get wallet data (without private key)
   */
  async getWallet() {
    try {
      let data;
      if (window.electron?.store) {
        data = await window.electron.store.get(this.storageKey);
      } else {
        // Fallback to localStorage
        const stored = localStorage.getItem(this.storageKey);
        data = stored ? JSON.parse(stored) : null;
      }

      if (!data) {
        return null;
      }

      return {
        address: data.address,
        publicKey: data.publicKey,
        createdAt: data.createdAt
      };
    } catch (error) {
      console.error('Failed to get wallet:', error);
      return null;
    }
  }

  /**
   * Unlock wallet with password and get mnemonic/private key
   */
  async unlockWallet(password) {
    try {
      let data;
      if (window.electron?.store) {
        data = await window.electron.store.get(this.storageKey);
      } else {
        const stored = localStorage.getItem(this.storageKey);
        data = stored ? JSON.parse(stored) : null;
      }

      if (!data) {
        throw new Error('No wallet found');
      }

      // Verify password
      const passwordHash = await this.hashPassword(password);
      if (passwordHash !== data.passwordHash) {
        throw new Error('Invalid password');
      }

      // Decrypt mnemonic
      const mnemonic = await this.decryptData(data.encryptedMnemonic, password);

      // Recreate wallet
      const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
        prefix: this.prefix
      });

      const accounts = await wallet.getAccounts();

      return {
        address: accounts[0].address,
        publicKey: Buffer.from(accounts[0].pubkey).toString('hex'),
        privateKey: mnemonic // Return mnemonic as privateKey for signing
      };
    } catch (error) {
      console.error('Failed to unlock wallet:', error);
      throw new Error(error.message || 'Failed to unlock wallet');
    }
  }

  /**
   * Get mnemonic phrase (requires password verification first via unlockWallet)
   */
  async getMnemonic() {
    try {
      let data;
      if (window.electron?.store) {
        data = await window.electron.store.get(this.storageKey);
      } else {
        const stored = localStorage.getItem(this.storageKey);
        data = stored ? JSON.parse(stored) : null;
      }

      if (!data || !data.encryptedMnemonic) {
        throw new Error('No wallet found');
      }

      // For this method, we need the password to decrypt
      // This should be called after password verification
      throw new Error('Use unlockWallet method to get mnemonic with password');
    } catch (error) {
      console.error('Failed to get mnemonic:', error);
      throw error;
    }
  }

  /**
   * Clear wallet data
   */
  async clearWallet() {
    try {
      if (window.electron?.store) {
        await window.electron.store.delete(this.storageKey);
      } else {
        localStorage.removeItem(this.storageKey);
      }
      return true;
    } catch (error) {
      console.error('Failed to clear wallet:', error);
      throw new Error('Failed to clear wallet');
    }
  }

  /**
   * Encrypt data using password
   */
  async encryptData(data, password) {
    try {
      // Simple XOR encryption with password hash
      // In production, use a proper encryption library like crypto-js
      const key = await this.hashPassword(password);
      const encrypted = this.xorEncrypt(data, key);
      return encrypted;
    } catch (error) {
      console.error('Encryption failed:', error);
      throw new Error('Failed to encrypt data');
    }
  }

  /**
   * Decrypt data using password
   */
  async decryptData(encryptedData, password) {
    try {
      const key = await this.hashPassword(password);
      const decrypted = this.xorEncrypt(encryptedData, key);
      return decrypted;
    } catch (error) {
      console.error('Decryption failed:', error);
      throw new Error('Failed to decrypt data');
    }
  }

  /**
   * Hash password using SHA-256
   */
  async hashPassword(password) {
    try {
      const encoder = new TextEncoder();
      const data = encoder.encode(password);
      const hash = sha256(data);
      return Buffer.from(hash).toString('hex');
    } catch (error) {
      console.error('Hashing failed:', error);
      throw new Error('Failed to hash password');
    }
  }

  /**
   * Simple XOR encryption/decryption
   * NOTE: This is a basic implementation. For production, use a proper encryption library.
   */
  xorEncrypt(text, key) {
    let result = '';
    for (let i = 0; i < text.length; i++) {
      result += String.fromCharCode(
        text.charCodeAt(i) ^ key.charCodeAt(i % key.length)
      );
    }
    return Buffer.from(result).toString('base64');
  }

  /**
   * Derive address from public key
   */
  async deriveAddress(publicKey, prefix = 'paw') {
    try {
      const pubkeyBytes = Buffer.from(publicKey, 'hex');
      const hash = sha256(pubkeyBytes);
      const address = toBech32(prefix, hash.slice(0, 20));
      return address;
    } catch (error) {
      console.error('Failed to derive address:', error);
      throw new Error('Failed to derive address');
    }
  }

  /**
   * Check if wallet exists
   */
  async hasWallet() {
    try {
      let data;
      if (window.electron?.store) {
        data = await window.electron.store.get(this.storageKey);
      } else {
        const stored = localStorage.getItem(this.storageKey);
        data = stored ? JSON.parse(stored) : null;
      }
      return !!data;
    } catch (error) {
      console.error('Failed to check wallet:', error);
      return false;
    }
  }

  /**
   * Export wallet to JSON (for backup)
   */
  async exportWallet(password) {
    try {
      const wallet = await this.unlockWallet(password);
      return {
        address: wallet.address,
        mnemonic: wallet.privateKey,
        exportedAt: new Date().toISOString(),
        version: '1.0'
      };
    } catch (error) {
      console.error('Failed to export wallet:', error);
      throw new Error('Failed to export wallet');
    }
  }

  /**
   * Import wallet from JSON backup
   */
  async importWallet(walletData, password) {
    try {
      if (!walletData.mnemonic) {
        throw new Error('Invalid wallet backup file');
      }

      return await this.createWallet(walletData.mnemonic, password);
    } catch (error) {
      console.error('Failed to import wallet:', error);
      throw new Error('Failed to import wallet');
    }
  }
}
