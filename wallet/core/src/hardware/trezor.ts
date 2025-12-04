/**
 * Trezor Hardware Wallet Integration for PAW Chain
 *
 * Supports Trezor One and Trezor Model T
 * Uses Trezor Connect for browser integration
 */

import TrezorConnect, { DEVICE_EVENT, UI_EVENT, Unsuccessful } from '@trezor/connect-web';
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

export class TrezorWallet implements IHardwareWallet {
  readonly type = HardwareWalletType.TREZOR;

  private config: Required<HardwareWalletConfig>;
  private initialized = false;
  private connected = false;

  constructor(config: HardwareWalletConfig = {}) {
    this.config = {
      timeout: config.timeout || DEFAULT_TIMEOUT,
      coinType: config.coinType || DEFAULT_COIN_TYPE,
      prefix: config.prefix || DEFAULT_PREFIX,
      debug: config.debug || false,
    };
  }

  /**
   * Initialize Trezor Connect
   */
  private async initialize(): Promise<void> {
    if (this.initialized) return;

    try {
      await TrezorConnect.init({
        lazyLoad: true,
        manifest: {
          email: 'support@pawchain.network',
          appUrl: 'https://wallet.pawchain.network',
        },
        debug: this.config.debug,
      });

      this.initialized = true;

      // Setup event listeners
      TrezorConnect.on(DEVICE_EVENT, (event) => {
        if (this.config.debug) {
          console.log('Trezor device event:', event);
        }

        if (event.type === 'device-connect') {
          this.connected = true;
        } else if (event.type === 'device-disconnect') {
          this.connected = false;
        }
      });

      TrezorConnect.on(UI_EVENT, (event) => {
        if (this.config.debug) {
          console.log('Trezor UI event:', event);
        }
      });
    } catch (error: any) {
      throw this.handleError(error, 'Failed to initialize Trezor Connect');
    }
  }

  /**
   * Check if device is connected
   */
  async isConnected(): Promise<boolean> {
    await this.initialize();
    return this.connected;
  }

  /**
   * Connect to Trezor device
   */
  async connect(): Promise<HardwareWalletInfo> {
    await this.initialize();

    try {
      // Get features to verify connection
      const result = await TrezorConnect.getFeatures();

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'CONNECTION_FAILED'
        );
      }

      const features = result.payload;
      this.connected = true;

      return {
        type: HardwareWalletType.TREZOR,
        model: this.getModelName(features.model),
        version: `${features.major_version}.${features.minor_version}.${features.patch_version}`,
        deviceId: features.device_id || undefined,
        status: DeviceConnectionStatus.CONNECTED,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to connect to Trezor device');
    }
  }

  /**
   * Disconnect from device
   */
  async disconnect(): Promise<void> {
    try {
      if (this.initialized) {
        TrezorConnect.dispose();
        this.initialized = false;
        this.connected = false;
      }
    } catch (error: any) {
      if (this.config.debug) {
        console.warn('Error during disconnect:', error);
      }
    }
  }

  /**
   * Get device information
   */
  async getDeviceInfo(): Promise<HardwareWalletInfo> {
    await this.initialize();

    try {
      const result = await TrezorConnect.getFeatures();

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'GET_INFO_FAILED'
        );
      }

      const features = result.payload;

      return {
        type: HardwareWalletType.TREZOR,
        model: this.getModelName(features.model),
        version: `${features.major_version}.${features.minor_version}.${features.patch_version}`,
        deviceId: features.device_id || undefined,
        status: this.connected ? DeviceConnectionStatus.CONNECTED : DeviceConnectionStatus.DISCONNECTED,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get device info');
    }
  }

  /**
   * Get public key for a derivation path
   */
  async getPublicKey(path: string, showOnDevice = false): Promise<Uint8Array> {
    await this.initialize();

    try {
      const result = await TrezorConnect.getPublicKey({
        path,
        coin: 'Cosmos',
        showOnTrezor: showOnDevice,
      });

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'GET_PUBKEY_FAILED'
        );
      }

      // Convert hex public key to Uint8Array
      return this.hexToBytes(result.payload.publicKey);
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get public key');
    }
  }

  /**
   * Get address for a derivation path
   */
  async getAddress(path: string, showOnDevice = false): Promise<string> {
    await this.initialize();

    try {
      const result = await TrezorConnect.cosmosGetAddress({
        path,
        showOnTrezor: showOnDevice,
      });

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'GET_ADDRESS_FAILED'
        );
      }

      const payload = result.payload;
      if (Array.isArray(payload)) {
        throw this.createError('Unexpected address bundle response', 'GET_ADDRESS_FAILED');
      }

      return payload.address;
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get address');
    }
  }

  /**
   * Get multiple addresses for account discovery
   */
  async getAddresses(paths: string[]): Promise<HardwareWalletAccount[]> {
    await this.initialize();

    const accounts: HardwareWalletAccount[] = [];

    // Get addresses in batch
    const bundle = paths.map(path => ({ path, showOnTrezor: false }));

    try {
      const result = await TrezorConnect.cosmosGetAddress({ bundle });

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'GET_ADDRESSES_FAILED'
        );
      }

      const payload = result.payload;
      if (!Array.isArray(payload)) {
        throw this.createError('Unexpected single response payload', 'GET_ADDRESSES_FAILED');
      }

      for (let i = 0; i < payload.length; i++) {
        const item = payload[i];
        if (item.success) {
          const pubkeyResult = await TrezorConnect.getPublicKey({
            path: paths[i],
            coin: 'Cosmos',
          });

          if (pubkeyResult.success) {
            accounts.push({
              address: item.payload.address,
              publicKey: this.hexToBytes(pubkeyResult.payload.publicKey),
              path: paths[i],
              index: i,
            });
          }
        } else if (this.config.debug) {
          console.warn(`Failed to fetch address for ${paths[i]}`);
        }
      }

      return accounts;
    } catch (error: any) {
      throw this.handleError(error, 'Failed to get addresses');
    }
  }

  /**
   * Sign a transaction
   */
  async signTransaction(
    path: string,
    txBytes: Uint8Array,
    _showOnDevice = true
  ): Promise<SignatureResult> {
    await this.initialize();

    try {
      // Parse transaction
      const tx = this.parseTransaction(txBytes);

      // Sign with Trezor
      const result = await TrezorConnect.cosmosSignTransaction({
        path,
        transaction: {
          chain_id: tx.chain_id,
          account_number: tx.account_number,
          sequence: tx.sequence,
          fee: tx.fee.amount[0]?.amount || '0',
          gas: tx.fee.gas,
          memo: tx.memo || '',
          msgs: tx.msgs.map(msg => ({
            type: msg.type || 'cosmos-sdk/MsgSend',
            ...msg.value,
          })),
        },
      });

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'SIGN_TX_FAILED'
        );
      }

      // Get public key
      const publicKey = await this.getPublicKey(path, false);

      return {
        signature: this.hexToBytes(result.payload.signature),
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
    await this.initialize();

    try {
      const messageStr = typeof message === 'string'
        ? message
        : Buffer.from(message).toString('utf8');

      // Trezor doesn't have native Cosmos message signing, use Ethereum-style
      const result = await TrezorConnect.ethereumSignMessage({
        path,
        message: messageStr,
        hex: false,
      });

      if (!result.success) {
        throw this.createError(
          (result as Unsuccessful).payload.error,
          'SIGN_MSG_FAILED'
        );
      }

      // Get public key
      const publicKey = await this.getPublicKey(path, false);

      return {
        signature: this.hexToBytes(result.payload.signature),
        publicKey,
      };
    } catch (error: any) {
      throw this.handleError(error, 'Failed to sign message');
    }
  }

  // ==================== Private Helper Methods ====================

  private parseTransaction(txBytes: Uint8Array): CosmosTransaction {
    try {
      const txString = Buffer.from(txBytes).toString('utf8');
      return JSON.parse(txString);
    } catch (error) {
      throw this.createError('Failed to parse transaction', 'INVALID_TX');
    }
  }

  private getModelName(model: string): string {
    switch (model) {
      case '1':
        return 'Trezor One';
      case 'T':
        return 'Trezor Model T';
      default:
        return `Trezor ${model}`;
    }
  }

  private hexToBytes(hex: string): Uint8Array {
    const cleanHex = hex.startsWith('0x') ? hex.slice(2) : hex;
    const bytes = new Uint8Array(cleanHex.length / 2);
    for (let i = 0; i < cleanHex.length; i += 2) {
      bytes[i / 2] = parseInt(cleanHex.substr(i, 2), 16);
    }
    return bytes;
  }

  private handleError(error: any, defaultMessage: string): HardwareWalletError {
    if (this.config.debug) {
      console.error('Trezor error:', error);
    }

    let message = defaultMessage;
    let code = 'UNKNOWN_ERROR';

    if (typeof error === 'string') {
      message = error;
    } else if (error.message) {
      message = error.message;

      // Map common errors
      if (error.message.includes('Cancelled')) {
        code = 'USER_REJECTED';
        message = 'User rejected on device';
      } else if (error.message.includes('not connected')) {
        code = 'NOT_CONNECTED';
        message = 'Device not connected';
      } else if (error.message.includes('PIN')) {
        code = 'DEVICE_LOCKED';
        message = 'Device locked - enter PIN';
      }
    }

    return this.createError(message, code);
  }

  private createError(message: string, code: string): HardwareWalletError {
    const error = new Error(message) as HardwareWalletError;
    error.code = code;
    error.deviceType = HardwareWalletType.TREZOR;
    return error;
  }
}

/**
 * Factory function to create Trezor wallet instance
 */
export function createTrezorWallet(config?: HardwareWalletConfig): TrezorWallet {
  return new TrezorWallet(config);
}

/**
 * Check if Trezor is supported in current environment
 */
export function isTrezorSupported(): boolean {
  // Trezor Connect works in all modern browsers
  return typeof window !== 'undefined' && !!window.navigator;
}
