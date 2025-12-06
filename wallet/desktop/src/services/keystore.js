import * as bip39 from 'bip39';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { toBech32 } from '@cosmjs/encoding';
import { sha256 } from '@cosmjs/crypto';

const ENCRYPTION_PREFIX = 'paw:v1:';
const SALT_BYTES = 16;
const IV_BYTES = 12;
const PBKDF2_ITERATIONS = 210000;

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
      const crypto = this.getCrypto();
      const encoder = new TextEncoder();
      const salt = crypto.getRandomValues(new Uint8Array(SALT_BYTES));
      const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
      const key = await this.deriveEncryptionKey(password, salt);
      const encryptedBuffer = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv },
        key,
        encoder.encode(data)
      );

      const payload = this.concatUint8Arrays([
        salt,
        iv,
        new Uint8Array(encryptedBuffer)
      ]);

      return `${ENCRYPTION_PREFIX}${this.uint8ToBase64(payload)}`;
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
      if (!encryptedData) {
        throw new Error('No encrypted payload found');
      }

      if (encryptedData.startsWith(ENCRYPTION_PREFIX)) {
        const crypto = this.getCrypto();
        const payload = this.base64ToUint8(
          encryptedData.slice(ENCRYPTION_PREFIX.length)
        );

        if (payload.length <= SALT_BYTES + IV_BYTES) {
          throw new Error('Invalid encrypted payload');
        }

        const salt = payload.slice(0, SALT_BYTES);
        const iv = payload.slice(SALT_BYTES, SALT_BYTES + IV_BYTES);
        const ciphertext = payload.slice(SALT_BYTES + IV_BYTES);

        const key = await this.deriveEncryptionKey(password, salt);
        const decryptedBuffer = await crypto.subtle.decrypt(
          { name: 'AES-GCM', iv },
          key,
          ciphertext
        );

        return new TextDecoder().decode(decryptedBuffer);
      }

      // Legacy fallback for XOR-based storage
      return this.legacyDecrypt(encryptedData, password);
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

  getCrypto() {
    if (typeof window !== 'undefined' && window.crypto?.subtle) {
      return window.crypto;
    }
    if (typeof globalThis !== 'undefined' && globalThis.crypto?.subtle) {
      return globalThis.crypto;
    }
    if (typeof require === 'function') {
      try {
        // eslint-disable-next-line global-require
        const { webcrypto } = require('crypto');
        if (webcrypto?.subtle) {
          return webcrypto;
        }
      } catch (error) {
        // Ignore, fallback below will throw
      }
    }
    throw new Error('Secure crypto APIs are not available');
  }

  async deriveEncryptionKey(password, salt) {
    const crypto = this.getCrypto();
    const encoder = new TextEncoder();
    const keyMaterial = await crypto.subtle.importKey(
      'raw',
      encoder.encode(password),
      { name: 'PBKDF2' },
      false,
      ['deriveBits', 'deriveKey']
    );

    return crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt,
        iterations: PBKDF2_ITERATIONS,
        hash: 'SHA-256'
      },
      keyMaterial,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt', 'decrypt']
    );
  }

  concatUint8Arrays(chunks) {
    const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    chunks.forEach((chunk) => {
      result.set(chunk, offset);
      offset += chunk.length;
    });
    return result;
  }

  uint8ToBase64(buffer) {
    let binary = '';
    const chunkSize = 0x8000;
    for (let i = 0; i < buffer.length; i += chunkSize) {
      const chunk = buffer.subarray(i, i + chunkSize);
      binary += String.fromCharCode.apply(null, chunk);
    }
    return btoa(binary);
  }

  base64ToUint8(base64) {
    const binary = atob(base64);
    const output = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      output[i] = binary.charCodeAt(i);
    }
    return output;
  }

  async legacyDecrypt(encryptedData, password) {
    const key = await this.hashPassword(password);
    const cipherBytes = this.base64ToUint8(encryptedData);
    let result = '';
    for (let i = 0; i < cipherBytes.length; i++) {
      result += String.fromCharCode(
        cipherBytes[i] ^ key.charCodeAt(i % key.length)
      );
    }
    return result;
  }
}
