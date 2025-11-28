/**
 * Ledger Hardware Wallet Integration for PAW Chain
 *
 * Supports Ledger Nano S, Nano X, and Nano S Plus
 * Requires Cosmos app to be installed and open on the device
 */

import TransportWebUSB from '@ledgerhq/hw-transport-webusb';
import CosmosApp from '@ledgerhq/hw-app-cosmos';
import { publicKeyToAddress } from '../crypto';
import {
  IHardwareWallet,
  HardwareWalletType,
  HardwareWalletInfo,
  HardwareWalletAccount,
  SignatureResult,
  DeviceConnectionStatus,
  HardwareWalletConfig,
  HardwareWalletError,
  CosmosTransaction,
} from './types';

const DEFAULT_COIN_TYPE = 118; // Cosmos coin type
const DEFAULT_PREFIX = 'paw';
const DEFAULT_TIMEOUT = 60000; // 60 seconds

export class LedgerWallet implements IHardwareWallet {
  readonly type = HardwareWalletType.LEDGER;

  private transport: TransportWebUSB | null = null;
  private app: CosmosApp | null = null;
  private config: Required<HardwareWalletConfig>;

  constructor(config: HardwareWalletConfig = {}) {
    this.config = {
      timeout: config.timeout || DEFAULT_TIMEOUT,
      coinType: config.coinType || DEFAULT_COIN_TYPE,
      prefix: config.prefix || DEFAULT_PREFIX,
      debug: config.debug || false,
    };
  }

  /**
   * Check if device is connected
   */
  async isConnected(): Promise<boolean> {
    try {
      if (!this.transport) {
        return false;
      }

      // Try to get app info to verify connection
      await this.transport.send(0xb0, 0x01, 0x00, 0x00);
      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * Connect to Ledger device
   */
  async connect(): Promise<HardwareWalletInfo> {
    try {
      // Close existing connection if any
      if (this.transport) {
        await this.disconnect();
      }

      // Request WebUSB device
      this.transport = await TransportWebUSB.create();

      // Set timeout
      this.transport.setExchangeTimeout(this.config.timeout);

      // Initialize Cosmos app
      this.app = new CosmosApp(this.transport);

      // Get app version to verify connection
      const appInfo = await this.app.getVersion();

      // Get device info
      const deviceModel = await this.getDeviceModel();

      return {
        type: HardwareWalletType.LEDGER,
        model: deviceModel,
        version: `${appInfo.major}.${appInfo.minor}.${appInfo.patch}`,
        status: DeviceConnectionStatus.CONNECTED,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to connect to Ledger device');
    }
  }

  /**
   * Disconnect from device
   */
  async disconnect(): Promise<void> {
    try {
      if (this.transport) {
        await this.transport.close();
        this.transport = null;
        this.app = null;
      }
    } catch (error: any) {
      if (this.config.debug) {
        console.warn('Error during disconnect:', error);
      }
      // Don't throw on disconnect errors
    }
  }

  /**
   * Get device information
   */
  async getDeviceInfo(): Promise<HardwareWalletInfo> {
    this.ensureConnected();

    try {
      const appInfo = await this.app!.getVersion();
      const deviceModel = await this.getDeviceModel();

      return {
        type: HardwareWalletType.LEDGER,
        model: deviceModel,
        version: `${appInfo.major}.${appInfo.minor}.${appInfo.patch}`,
        status: DeviceConnectionStatus.CONNECTED,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get device info');
    }
  }

  /**
   * Get public key for a derivation path
   */
  async getPublicKey(path: string, showOnDevice = false): Promise<Uint8Array> {
    this.ensureConnected();

    try {
      const hdPath = this.parsePath(path);
      const response = await this.app!.getAddress(hdPath, this.config.prefix, showOnDevice);

      // Convert hex string to Uint8Array
      return this.hexToBytes(response.publicKey);
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get public key');
    }
  }

  /**
   * Get address for a derivation path
   */
  async getAddress(path: string, showOnDevice = false): Promise<string> {
    this.ensureConnected();

    try {
      const hdPath = this.parsePath(path);
      const response = await this.app!.getAddress(hdPath, this.config.prefix, showOnDevice);

      return response.address;
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get address');
    }
  }

  /**
   * Get multiple addresses for account discovery
   */
  async getAddresses(paths: string[]): Promise<HardwareWalletAccount[]> {
    this.ensureConnected();

    const accounts: HardwareWalletAccount[] = [];

    for (let i = 0; i < paths.length; i++) {
      const path = paths[i];
      try {
        const hdPath = this.parsePath(path);
        const response = await this.app!.getAddress(hdPath, this.config.prefix, false);

        accounts.push({
          address: response.address,
          publicKey: this.hexToBytes(response.publicKey),
          path,
          index: i,
        });
      } catch (error: any) {
        if (this.config.debug) {
          console.warn(`Failed to get address for path ${path}:`, error);
        }
        // Continue with other addresses
      }
    }

    return accounts;
  }

  /**
   * Sign a transaction
   */
  async signTransaction(
    path: string,
    txBytes: Uint8Array,
    showOnDevice = true
  ): Promise<SignatureResult> {
    this.ensureConnected();

    try {
      // Parse the transaction to Cosmos format
      const tx = this.parseTransaction(txBytes);
      const hdPath = this.parsePath(path);

      // Sign with Ledger
      const response = await this.app!.sign(hdPath, JSON.stringify(tx));

      // Get public key
      const publicKey = await this.getPublicKey(path, false);

      return {
        signature: Buffer.from(response.signature, 'base64'),
        publicKey,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to sign transaction');
    }
  }

  /**
   * Sign arbitrary message
   */
  async signMessage(
    path: string,
    message: string | Uint8Array,
    showOnDevice = true
  ): Promise<SignatureResult> {
    this.ensureConnected();

    try {
      const messageStr = typeof message === 'string' ? message : Buffer.from(message).toString('utf8');
      const hdPath = this.parsePath(path);

      // Use Cosmos app's sign message feature
      const response = await this.app!.sign(hdPath, messageStr);
      const publicKey = await this.getPublicKey(path, false);

      return {
        signature: Buffer.from(response.signature, 'base64'),
        publicKey,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to sign message');
    }
  }

  // ==================== Private Helper Methods ====================

  private ensureConnected(): void {
    if (!this.transport || !this.app) {
      throw this.createError('Device not connected', 'NOT_CONNECTED');
    }
  }

  private parsePath(path: string): number[] {
    // Convert string path to number array
    // e.g., "m/44'/118'/0'/0/0" -> [44, 118, 0, 0, 0]
    const segments = path.replace(/^m\//, '').split('/');

    return segments.map((segment) => {
      const hardened = segment.endsWith("'");
      const index = parseInt(hardened ? segment.slice(0, -1) : segment);

      if (isNaN(index)) {
        throw this.createError(`Invalid path segment: ${segment}`, 'INVALID_PATH');
      }

      // Add hardening offset (0x80000000)
      return hardened ? index + 0x80000000 : index;
    });
  }

  private parseTransaction(txBytes: Uint8Array): CosmosTransaction {
    try {
      // Decode transaction bytes to JSON
      const txString = Buffer.from(txBytes).toString('utf8');
      return JSON.parse(txString);
    } catch (error) {
      throw this.createError('Failed to parse transaction', 'INVALID_TX');
    }
  }

  private async getDeviceModel(): Promise<string> {
    try {
      // Try to get device model from transport
      const deviceModel = await this.transport!.device.productName;
      return deviceModel || 'Ledger Device';
    } catch (error) {
      return 'Ledger Device';
    }
  }

  private hexToBytes(hex: string): Uint8Array {
    // Remove 0x prefix if present
    const cleanHex = hex.startsWith('0x') ? hex.slice(2) : hex;

    const bytes = new Uint8Array(cleanHex.length / 2);
    for (let i = 0; i < cleanHex.length; i += 2) {
      bytes[i / 2] = parseInt(cleanHex.substr(i, 2), 16);
    }
    return bytes;
  }

  private handleError(error: any, defaultMessage: string): HardwareWalletError {
    if (this.config.debug) {
      console.error('Ledger error:', error);
    }

    // Map Ledger error codes to user-friendly messages
    let message = defaultMessage;
    let code = 'UNKNOWN_ERROR';

    if (error.statusCode) {
      switch (error.statusCode) {
        case 0x6985:
          message = 'User rejected on device';
          code = 'USER_REJECTED';
          break;
        case 0x6a80:
          message = 'Invalid transaction data';
          code = 'INVALID_DATA';
          break;
        case 0x6d00:
          message = 'Cosmos app not open on device';
          code = 'APP_NOT_OPEN';
          break;
        case 0x6e00:
          message = 'Device locked';
          code = 'DEVICE_LOCKED';
          break;
        default:
          message = `${defaultMessage}: ${error.message || 'Unknown error'}`;
      }
    } else if (error.message) {
      message = error.message;
    }

    return this.createError(message, code);
  }

  private createError(message: string, code: string): HardwareWalletError {
    const error = new Error(message) as HardwareWalletError;
    error.code = code;
    error.deviceType = HardwareWalletType.LEDGER;
    return error;
  }
}

/**
 * Factory function to create Ledger wallet instance
 */
export function createLedgerWallet(config?: HardwareWalletConfig): LedgerWallet {
  return new LedgerWallet(config);
}

/**
 * Check if Ledger is supported in current environment
 */
export async function isLedgerSupported(): Promise<boolean> {
  try {
    return await TransportWebUSB.isSupported();
  } catch (error) {
    return false;
  }
}

/**
 * Request Ledger device connection (triggers browser permission)
 */
export async function requestLedgerDevice(): Promise<void> {
  try {
    await TransportWebUSB.request();
  } catch (error: any) {
    throw new Error(`Failed to request Ledger device: ${error.message}`);
  }
}
