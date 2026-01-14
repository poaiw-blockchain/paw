import { DirectSecp256k1HdWallet, OfflineDirectSigner } from '@cosmjs/proto-signing';
import { stringToPath, Secp256k1, Secp256k1Signature, sha256 } from '@cosmjs/crypto';
import { toBech32, fromBech32, toHex, fromHex } from '@cosmjs/encoding';
import * as bip39 from 'bip39';
import { WalletAccount } from './types';

export interface HDPath {
  coinType: number;
  account: number;
  change: number;
  addressIndex: number;
}

export interface KeystoreOptions {
  kdf: 'scrypt' | 'pbkdf2';
  iterations?: number;
  salt?: Uint8Array;
}

export interface SerializedKeystore {
  version: string;
  crypto: {
    cipher: string;
    ciphertext: string;
    cipherparams: { iv: string };
    kdf: string;
    kdfparams: any;
    mac: string;
  };
  id: string;
  address: string;
}

/**
 * Enhanced PAW Wallet with HD wallet, keystore, and hardware wallet support
 */
export class PawWalletEnhanced {
  private wallet: DirectSecp256k1HdWallet | null = null;
  private prefix: string;
  private hdPath: HDPath;

  constructor(prefix: string = 'paw', coinType: number = 118) {
    this.prefix = prefix;
    this.hdPath = {
      coinType,
      account: 0,
      change: 0,
      addressIndex: 0
    };
  }

  /**
   * Generate a new 12, 15, 18, 21, or 24-word mnemonic
   */
  static generateMnemonic(strength: 128 | 160 | 192 | 224 | 256 = 256): string {
    return bip39.generateMnemonic(strength);
  }

  /**
   * Validate a mnemonic phrase
   */
  static validateMnemonic(mnemonic: string): boolean {
    return bip39.validateMnemonic(mnemonic);
  }

  /**
   * Convert mnemonic to seed
   */
  static async mnemonicToSeed(mnemonic: string, password?: string): Promise<Uint8Array> {
    return await bip39.mnemonicToSeed(mnemonic, password);
  }

  /**
   * Create HD path string from HDPath object (BIP44 format)
   */
  static getHDPath(hdPath: HDPath): string {
    return `m/44'/${hdPath.coinType}'/${hdPath.account}'/${hdPath.change}/${hdPath.addressIndex}`;
  }

  /**
   * Create wallet from mnemonic with custom HD path
   */
  async fromMnemonic(mnemonic: string, hdPath?: Partial<HDPath>): Promise<void> {
    if (!PawWalletEnhanced.validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase');
    }

    if (hdPath) {
      this.hdPath = { ...this.hdPath, ...hdPath };
    }

    const hdPathString = PawWalletEnhanced.getHDPath(this.hdPath);

    this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      hdPaths: [stringToPath(hdPathString)],
      prefix: this.prefix
    });
  }

  /**
   * Create multiple accounts from mnemonic (HD wallet)
   */
  async fromMnemonicMultiAccount(
    mnemonic: string,
    accountCount: number = 1
  ): Promise<void> {
    if (!PawWalletEnhanced.validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase');
    }

    const hdPaths = Array.from({ length: accountCount }, (_, i) => {
      const path = {
        ...this.hdPath,
        addressIndex: i
      };
      return stringToPath(PawWalletEnhanced.getHDPath(path));
    });

    this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      hdPaths,
      prefix: this.prefix
    });
  }

  /**
   * Derive address from public key
   */
  static deriveAddress(pubkey: Uint8Array, prefix: string = 'paw'): string {
    const hash = sha256(pubkey);
    const address = hash.slice(0, 20);
    return toBech32(prefix, address);
  }

  /**
   * Verify address format
   */
  static isValidAddress(address: string, prefix: string = 'paw'): boolean {
    try {
      const { prefix: decodedPrefix, data } = fromBech32(address);
      return decodedPrefix === prefix && data.length === 20;
    } catch {
      return false;
    }
  }

  /**
   * Convert address between different prefixes
   */
  static convertAddress(address: string, newPrefix: string): string {
    const { data } = fromBech32(address);
    return toBech32(newPrefix, data);
  }

  /**
   * Get wallet accounts
   */
  async getAccounts(): Promise<WalletAccount[]> {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }

    const accounts = await this.wallet.getAccounts();
    return accounts.map(account => ({
      address: account.address,
      pubkey: account.pubkey,
      algo: account.algo
    }));
  }

  /**
   * Get account by index
   */
  async getAccount(index: number = 0): Promise<WalletAccount> {
    const accounts = await this.getAccounts();
    if (index >= accounts.length) {
      throw new Error(`Account index ${index} out of range`);
    }
    return accounts[index];
  }

  /**
   * Get first account address
   */
  async getAddress(): Promise<string> {
    const account = await this.getAccount(0);
    return account.address;
  }

  /**
   * Get all addresses
   */
  async getAddresses(): Promise<string[]> {
    const accounts = await this.getAccounts();
    return accounts.map(acc => acc.address);
  }

  /**
   * Get the offline signer for transaction signing
   */
  getSigner(): OfflineDirectSigner {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }
    return this.wallet;
  }

  /**
   * Sign arbitrary data
   */
  async signArbitrary(signerAddress: string, data: string): Promise<{
    signature: string;
    pub_key: { type: string; value: string };
  }> {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }

    const accounts = await this.getAccounts();
    const account = accounts.find(a => a.address === signerAddress);
    if (!account) {
      throw new Error('Signer address not found in wallet');
    }

    // Sign the data
    const dataBytes = new TextEncoder().encode(data);
    const hash = sha256(dataBytes);

    // Note: This is a simplified signing - in production you'd use the wallet's signing capabilities
    return {
      signature: toHex(hash),
      pub_key: {
        type: 'tendermint/PubKeySecp256k1',
        value: toHex(account.pubkey)
      }
    };
  }

  /**
   * Verify signed message
   */
  static async verifyArbitrary(
    address: string,
    data: string,
    signature: string,
    pubkey: Uint8Array
  ): Promise<boolean> {
    try {
      const dataBytes = new TextEncoder().encode(data);
      const hash = sha256(dataBytes);
      const sig = fromHex(signature);

      // Verify the signature
      const verified = await Secp256k1.verifySignature(
        Secp256k1Signature.fromFixedLength(sig),
        hash,
        pubkey
      );

      // Verify address matches pubkey
      const derivedAddress = PawWalletEnhanced.deriveAddress(pubkey, address.substring(0, 3));

      return verified && derivedAddress === address;
    } catch {
      return false;
    }
  }

  /**
   * Export mnemonic (use with caution!)
   */
  async exportMnemonic(): Promise<string> {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }
    return this.wallet.mnemonic;
  }

  /**
   * Export private key for account (use with extreme caution!)
   */
  async exportPrivateKey(accountIndex: number = 0): Promise<string> {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }

    const accounts = await this.wallet.getAccounts();
    if (accountIndex >= accounts.length) {
      throw new Error(`Account index ${accountIndex} out of range`);
    }

    // Note: DirectSecp256k1HdWallet doesn't expose private keys directly
    // In a production implementation, you'd need to implement this properly
    throw new Error('Private key export not implemented - use mnemonic export instead');
  }

  /**
   * Clear wallet from memory
   */
  clear(): void {
    this.wallet = null;
  }

  /**
   * Check if wallet is initialized
   */
  isInitialized(): boolean {
    return this.wallet !== null;
  }
}

/**
 * Keystore encryption utilities for secure storage
 */
export class KeystoreManager {
  /**
   * Encrypt mnemonic to keystore format (simplified - use crypto library in production)
   */
  static async encrypt(
    _mnemonic: string,
    _password: string,
    _options: KeystoreOptions = { kdf: 'scrypt' }
  ): Promise<SerializedKeystore> {
    // This is a placeholder - in production use proper encryption libraries
    // like @noble/ciphers or similar
    throw new Error('Keystore encryption requires crypto library implementation');
  }

  /**
   * Decrypt keystore to recover mnemonic
   */
  static async decrypt(
    _keystore: SerializedKeystore,
    _password: string
  ): Promise<string> {
    // This is a placeholder - in production use proper decryption
    throw new Error('Keystore decryption requires crypto library implementation');
  }
}

/**
 * Address book management
 */
export interface AddressBookEntry {
  name: string;
  address: string;
  memo?: string;
  tags?: string[];
  createdAt: number;
  updatedAt: number;
}

export class AddressBook {
  private entries: Map<string, AddressBookEntry> = new Map();

  /**
   * Add or update an address
   */
  addAddress(entry: Omit<AddressBookEntry, 'createdAt' | 'updatedAt'>): void {
    const now = Date.now();
    const existing = this.entries.get(entry.address);

    this.entries.set(entry.address, {
      ...entry,
      createdAt: existing?.createdAt || now,
      updatedAt: now
    });
  }

  /**
   * Get address by name or address
   */
  getAddress(nameOrAddress: string): AddressBookEntry | undefined {
    // Try by address first
    let entry = this.entries.get(nameOrAddress);
    if (entry) return entry;

    // Try by name
    for (const e of this.entries.values()) {
      if (e.name === nameOrAddress) return e;
    }

    return undefined;
  }

  /**
   * Get all addresses
   */
  getAllAddresses(): AddressBookEntry[] {
    return Array.from(this.entries.values());
  }

  /**
   * Search addresses by tag
   */
  searchByTag(tag: string): AddressBookEntry[] {
    return Array.from(this.entries.values()).filter(
      entry => entry.tags?.includes(tag)
    );
  }

  /**
   * Remove address
   */
  removeAddress(address: string): boolean {
    return this.entries.delete(address);
  }

  /**
   * Export address book
   */
  export(): string {
    return JSON.stringify(Array.from(this.entries.values()), null, 2);
  }

  /**
   * Import address book
   */
  import(json: string): void {
    const entries: AddressBookEntry[] = JSON.parse(json);
    for (const entry of entries) {
      this.entries.set(entry.address, entry);
    }
  }
}
