import { DirectSecp256k1HdWallet, OfflineDirectSigner } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import * as bip39 from 'bip39';
import { WalletAccount } from './types';

export class PawWallet {
  private wallet: DirectSecp256k1HdWallet | null = null;
  private prefix: string;

  constructor(prefix: string = 'paw') {
    this.prefix = prefix;
  }

  /**
   * Generate a new 24-word mnemonic
   */
  static generateMnemonic(): string {
    return bip39.generateMnemonic(256);
  }

  /**
   * Validate a mnemonic phrase
   */
  static validateMnemonic(mnemonic: string): boolean {
    return bip39.validateMnemonic(mnemonic);
  }

  /**
   * Create wallet from mnemonic
   */
  async fromMnemonic(mnemonic: string, hdPath?: string): Promise<void> {
    if (!PawWallet.validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase');
    }

    const options = hdPath
      ? { hdPaths: [stringToPath(hdPath)], prefix: this.prefix }
      : { prefix: this.prefix };

    this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, options);
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
   * Get first account address
   */
  async getAddress(): Promise<string> {
    const accounts = await this.getAccounts();
    if (accounts.length === 0) {
      throw new Error('No accounts in wallet');
    }
    return accounts[0].address;
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
   * Export mnemonic (use with caution!)
   */
  async exportMnemonic(): Promise<string> {
    if (!this.wallet) {
      throw new Error('Wallet not initialized');
    }
    return this.wallet.mnemonic;
  }
}
