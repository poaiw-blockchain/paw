/**
 * Ledger Hardware Wallet Integration for PAW Chain
 *
 * Supports Ledger Nano S, Nano X, and Nano S Plus
 * Requires Cosmos app to be installed and open on the device
 */

import TransportWebUSB from '@ledgerhq/hw-transport-webusb';
import CosmosApp from '@ledgerhq/hw-app-cosmos';
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
import { assertBech32Prefix, validateSignDocBasics } from './guards';

const DEFAULT_COIN_TYPE = 118; // Cosmos coin type
const DEFAULT_PREFIX = 'paw';
const DEFAULT_TIMEOUT = 60000; // 60 seconds
const DEFAULT_SIGN_MODE: NonNullable<HardwareWalletConfig['signMode']> = 'amino';
const DEFAULT_ALLOWED_FEE_DENOMS = ['upaw'];
const DEFAULT_MAX_ACCOUNT_INDEX = 4;

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
      signMode: config.signMode || DEFAULT_SIGN_MODE,
      enforceChainId: config.enforceChainId,
      allowedFeeDenoms: config.allowedFeeDenoms || DEFAULT_ALLOWED_FEE_DENOMS,
      maxAccountIndex: config.maxAccountIndex ?? DEFAULT_MAX_ACCOUNT_INDEX,
      allowedManufacturers: config.allowedManufacturers || ['Ledger'],
      allowedProductNames: config.allowedProductNames || ['Nano S', 'Nano X', 'Nano S Plus', 'Ledger Device'],
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
      this.transport = (await TransportWebUSB.create()) as TransportWebUSB;

      // Set timeout
      this.transport.setExchangeTimeout(this.config.timeout);

      // Initialize Cosmos app
      this.app = new CosmosApp(this.transport);

      // Get app version to verify connection
      const appInfo = await this.app.getVersion();

      // Get device info
      const deviceModel = await this.getDeviceModel();
      this.basicAttestationCheck(deviceModel);

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
      const hdPath = this.normalizePath(path);
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
      const hdPath = this.normalizePath(path);
      const response = await this.app!.getAddress(hdPath, this.config.prefix, showOnDevice);

      assertBech32Prefix(response.address, this.config.prefix);

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
        const hdPath = this.normalizePath(path);
        const response = await this.app!.getAddress(hdPath, this.config.prefix, false);

        assertBech32Prefix(response.address, this.config.prefix);

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
    _showOnDevice = true
  ): Promise<SignatureResult> {
    this.ensureConnected();

    try {
      // Parse the transaction to Cosmos format
      const tx = this.parseTransaction(txBytes);
      if (this.config.signMode === 'direct') {
        throw this.createError(
          'Direct sign is not available over WebUSB; fallback to amino/legacy JSON payloads',
          'UNSUPPORTED_SIGN_MODE'
        );
      }
      validateSignDocBasics(
        {
          chain_id: tx.chain_id,
          fee: tx.fee,
        },
        {
          enforceChainId: this.config.enforceChainId,
          allowedFeeDenoms: this.config.allowedFeeDenoms,
        }
      );
      this.validateMsgAddresses(tx.msgs);
      const hdPath = this.normalizePath(path);

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
    _showOnDevice = true
  ): Promise<SignatureResult> {
    this.ensureConnected();

    try {
      const messageStr = typeof message === 'string' ? message : Buffer.from(message).toString('utf8');
      const hdPath = this.normalizePath(path);

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

  private normalizePath(path: string): string {
    const sanitized = path.startsWith('m/') ? path.slice(2) : path;
    const segments = sanitized.split('/');

    if (segments.length !== 5) {
      throw this.createError(`Invalid derivation path: ${path}`, 'INVALID_PATH');
    }

    const accountSegment = segments[2];
    const accountValue = parseInt(accountSegment.endsWith("'") ? accountSegment.slice(0, -1) : accountSegment, 10);
    if (Number.isNaN(accountValue) || accountValue > this.config.maxAccountIndex) {
      throw this.createError(
        `Invalid path: account index exceeds max (${this.config.maxAccountIndex})`,
        'INVALID_PATH'
      );
    }

    return segments
      .map((segment, index) => {
        const requiresHardened = index < 3;
        const hardened = segment.endsWith("'");
        const value = parseInt(hardened ? segment.slice(0, -1) : segment, 10);

        if (Number.isNaN(value)) {
          throw this.createError(`Invalid path segment: ${segment}`, 'INVALID_PATH');
        }

        if (requiresHardened && !hardened) {
          throw this.createError(`Path segment must be hardened: ${segment}`, 'INVALID_PATH');
        }

        return hardened || requiresHardened ? `${value}'` : `${value}`;
      })
      .join('/');
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

  private validateMsgAddresses(msgs: any[]): void {
    if (!Array.isArray(msgs)) return;

    for (const msg of msgs) {
      if (msg?.value && typeof msg.value === 'object') {
        const value = msg.value;
        Object.entries(value).forEach(([key, val]) => {
          if (typeof val === 'string' && key.toLowerCase().includes('address')) {
            try {
              assertBech32Prefix(val, this.config.prefix);
            } catch (err: any) {
              throw this.createError(`Invalid bech32 prefix for ${key}: ${err.message}`, 'INVALID_DATA');
            }
          }
        });
      }
    }
  }

  private async getDeviceModel(): Promise<string> {
    const productName =
      (this.transport && (this.transport as TransportWebUSB).device?.productName) || null;
    return productName || 'Ledger Device';
  }

  private basicAttestationCheck(deviceModel: string): void {
    const manufacturer =
      (this.transport && (this.transport as TransportWebUSB).device?.manufacturerName) || '';
    const allowedManufacturers = this.config.allowedManufacturers || [];
    const allowedProductNames = this.config.allowedProductNames || [];

    if (
      allowedManufacturers.length > 0 &&
      manufacturer &&
      !allowedManufacturers.some((m) => manufacturer.toLowerCase().includes(m.toLowerCase()))
    ) {
      throw this.createError(
        `Unexpected manufacturer: ${manufacturer}`,
        'DEVICE_ATTESTATION_FAILED'
      );
    }

    if (
      allowedProductNames.length > 0 &&
      deviceModel &&
      !allowedProductNames.some((m) => deviceModel.toLowerCase().includes(m.toLowerCase()))
    ) {
      throw this.createError(
        `Unexpected device model: ${deviceModel}`,
        'DEVICE_ATTESTATION_FAILED'
      );
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
