/**
 * Hardware Wallet Support for PAW Chain
 *
 * Main export file for hardware wallet functionality
 * Supports Ledger and Trezor devices
 */

import { LedgerWallet, createLedgerWallet, isLedgerSupported } from './ledger';
import { TrezorWallet, createTrezorWallet, isTrezorSupported } from './trezor';
import {
  IHardwareWallet,
  HardwareWalletType,
  HardwareWalletInfo,
  HardwareWalletConfig,
  HardwareWalletError,
} from './types';
import { assertBech32Prefix, validateSignDocBasics } from './guards';

// Re-export types
export * from './types';

// Re-export wallet classes
export { LedgerWallet, TrezorWallet };

/**
 * Hardware wallet factory
 * Creates appropriate wallet instance based on type
 */
export class HardwareWalletFactory {
  /**
   * Create hardware wallet instance
   */
  static create(
    type: HardwareWalletType,
    config?: HardwareWalletConfig
  ): IHardwareWallet {
    switch (type) {
      case HardwareWalletType.LEDGER:
        return createLedgerWallet(config);
      case HardwareWalletType.TREZOR:
        return createTrezorWallet(config);
      default:
        throw new Error(`Unsupported hardware wallet type: ${type}`);
    }
  }

  /**
   * Check which hardware wallets are supported
   */
  static async getSupportedWallets(): Promise<HardwareWalletType[]> {
    const supported: HardwareWalletType[] = [];

    if (await isLedgerSupported()) {
      supported.push(HardwareWalletType.LEDGER);
    }

    if (isTrezorSupported()) {
      supported.push(HardwareWalletType.TREZOR);
    }

    return supported;
  }

  /**
   * Detect connected hardware wallets
   */
  static async detectWallets(): Promise<HardwareWalletInfo[]> {
    const detected: HardwareWalletInfo[] = [];

    // Try Ledger
    try {
      const ledger = createLedgerWallet();
      if (await ledger.isConnected()) {
        const info = await ledger.getDeviceInfo();
        detected.push(info);
      }
    } catch (error) {
      // Ledger not connected
    }

    // Try Trezor
    try {
      const trezor = createTrezorWallet();
      if (await trezor.isConnected()) {
        const info = await trezor.getDeviceInfo();
        detected.push(info);
      }
    } catch (error) {
      // Trezor not connected
    }

    return detected;
  }
}

/**
 * Utility functions for hardware wallets
 */
export class HardwareWalletUtils {
  /**
   * Build a standard path matrix (accounts 0..maxAccountIndex)
   */
  static buildDefaultPathMatrix(
    coinType = 118,
    maxAccountIndex = 4,
    account = 0
  ): string[] {
    const upperBound = Math.max(0, maxAccountIndex);
    return this.generatePaths(coinType, upperBound + 1, account);
  }

  /**
   * Generate standard Cosmos derivation paths
   */
  static generatePaths(
    coinType = 118,
    accountCount = 10,
    account = 0
  ): string[] {
    const paths: string[] = [];

    for (let i = 0; i < accountCount; i++) {
      paths.push(`m/44'/${coinType}'/${account}'/0/${i}`);
    }

    return paths;
  }

  /**
   * Parse derivation path
   */
  static parsePath(path: string): {
    coinType: number;
    account: number;
    change: number;
    index: number;
  } {
    const regex = /^m\/44'\/(\d+)'\/(\d+)'\/(\d+)\/(\d+)$/;
    const match = path.match(regex);

    if (!match) {
      throw new Error(`Invalid derivation path: ${path}`);
    }

    return {
      coinType: parseInt(match[1]),
      account: parseInt(match[2]),
      change: parseInt(match[3]),
      index: parseInt(match[4]),
    };
  }

  /**
   * Validate derivation path
   */
  static isValidPath(path: string): boolean {
    try {
      this.parsePath(path);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Validate a Bech32 address prefix
   */
  static assertBech32Prefix(address: string, expectedPrefix: string): void {
    assertBech32Prefix(address, expectedPrefix);
  }

  /**
   * Validate fee + chain-id constraints before sending to a device
   */
  static validateSignDocBasics(
    doc: {
      chain_id?: string;
      fee?: { amount?: Array<{ denom?: string; amount?: string }>; gas?: string };
    },
    {
      enforceChainId,
      allowedFeeDenoms = ['upaw'],
    }: Pick<HardwareWalletConfig, 'enforceChainId' | 'allowedFeeDenoms'>
  ): void {
    validateSignDocBasics(doc, { enforceChainId, allowedFeeDenoms });
  }

  /**
   * Get user-friendly error message
   */
  static getErrorMessage(error: HardwareWalletError): string {
    switch (error.code) {
      case 'USER_REJECTED':
        return 'Transaction was rejected on the device';
      case 'NOT_CONNECTED':
        return 'Hardware wallet is not connected';
      case 'DEVICE_LOCKED':
        return 'Please unlock your device';
      case 'APP_NOT_OPEN':
        return 'Please open the Cosmos app on your device';
      case 'INVALID_DATA':
        return 'Invalid transaction data';
      case 'INVALID_PATH':
        return 'Invalid derivation path';
      default:
        return error.message || 'An unknown error occurred';
    }
  }
}

/**
 * Hardware wallet manager for handling multiple devices
 */
export class HardwareWalletManager {
  private wallets: Map<string, IHardwareWallet> = new Map();

  /**
   * Add a hardware wallet
   */
  async addWallet(
    type: HardwareWalletType,
    config?: HardwareWalletConfig
  ): Promise<string> {
    const wallet = HardwareWalletFactory.create(type, config);
    const info = await wallet.connect();
    const id = `${type}-${info.deviceId || Date.now()}`;

    this.wallets.set(id, wallet);

    return id;
  }

  /**
   * Get a wallet by ID
   */
  getWallet(id: string): IHardwareWallet | undefined {
    return this.wallets.get(id);
  }

  /**
   * Remove a wallet
   */
  async removeWallet(id: string): Promise<void> {
    const wallet = this.wallets.get(id);
    if (wallet) {
      await wallet.disconnect();
      this.wallets.delete(id);
    }
  }

  /**
   * Get all wallets
   */
  getAllWallets(): Map<string, IHardwareWallet> {
    return this.wallets;
  }

  /**
   * Disconnect all wallets
   */
  async disconnectAll(): Promise<void> {
    const promises = Array.from(this.wallets.values()).map(wallet =>
      wallet.disconnect()
    );
    await Promise.all(promises);
    this.wallets.clear();
  }
}

/**
 * Main export - convenience functions
 */

/**
 * Connect to Ledger device
 */
export async function connectLedger(
  config?: HardwareWalletConfig
): Promise<LedgerWallet> {
  const wallet = createLedgerWallet(config);
  await wallet.connect();
  return wallet;
}

/**
 * Connect to Trezor device
 */
export async function connectTrezor(
  config?: HardwareWalletConfig
): Promise<TrezorWallet> {
  const wallet = createTrezorWallet(config);
  await wallet.connect();
  return wallet;
}

/**
 * Auto-detect and connect to hardware wallet
 */
export async function connectHardwareWallet(
  preferredType?: HardwareWalletType,
  config?: HardwareWalletConfig
): Promise<IHardwareWallet> {
  // If preferred type specified, try that first
  if (preferredType) {
    const wallet = HardwareWalletFactory.create(preferredType, config);
    await wallet.connect();
    return wallet;
  }

  // Otherwise, try to detect and connect to any available wallet
  const supported = await HardwareWalletFactory.getSupportedWallets();

  if (supported.length === 0) {
    throw new Error('No supported hardware wallets found');
  }

  // Try each supported wallet
  for (const type of supported) {
    try {
      const wallet = HardwareWalletFactory.create(type, config);
      await wallet.connect();
      return wallet;
    } catch (error) {
      // Try next wallet type
    }
  }

  throw new Error('Failed to connect to any hardware wallet');
}

/**
 * Check hardware wallet support
 */
export async function checkHardwareWalletSupport(): Promise<{
  ledger: boolean;
  trezor: boolean;
  supported: HardwareWalletType[];
}> {
  const ledger = await isLedgerSupported();
  const trezor = isTrezorSupported();
  const supported = await HardwareWalletFactory.getSupportedWallets();

  return {
    ledger,
    trezor,
    supported,
  };
}
