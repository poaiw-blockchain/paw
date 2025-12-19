/**
 * Trezor Hardware Wallet Integration for PAW Chain
 *
 * Supports Trezor One and Trezor Model T
 * Uses Trezor Connect for browser integration
 */
import { IHardwareWallet, HardwareWalletType, HardwareWalletInfo, HardwareWalletAccount, SignatureResult, HardwareWalletConfig } from './types';
export declare class TrezorWallet implements IHardwareWallet {
    readonly type = HardwareWalletType.TREZOR;
    private config;
    private initialized;
    private connected;
    constructor(config?: HardwareWalletConfig);
    /**
     * Initialize Trezor Connect
     */
    private initialize;
    /**
     * Check if device is connected
     */
    isConnected(): Promise<boolean>;
    /**
     * Connect to Trezor device
     */
    connect(): Promise<HardwareWalletInfo>;
    /**
     * Disconnect from device
     */
    disconnect(): Promise<void>;
    /**
     * Get device information
     */
    getDeviceInfo(): Promise<HardwareWalletInfo>;
    /**
     * Get public key for a derivation path
     */
    getPublicKey(path: string, showOnDevice?: boolean): Promise<Uint8Array>;
    /**
     * Get address for a derivation path
     */
    getAddress(path: string, showOnDevice?: boolean): Promise<string>;
    /**
     * Get multiple addresses for account discovery
     */
    getAddresses(paths: string[]): Promise<HardwareWalletAccount[]>;
    /**
     * Sign a transaction
     */
    signTransaction(path: string, txBytes: Uint8Array, _showOnDevice?: boolean): Promise<SignatureResult>;
    /**
     * Sign arbitrary message
     */
    signMessage(path: string, message: string | Uint8Array, _showOnDevice?: boolean): Promise<SignatureResult>;
    private parseTransaction;
    private getModelName;
    private hexToBytes;
    private handleError;
    private createError;
}
/**
 * Factory function to create Trezor wallet instance
 */
export declare function createTrezorWallet(config?: HardwareWalletConfig): TrezorWallet;
/**
 * Check if Trezor is supported in current environment
 */
export declare function isTrezorSupported(): boolean;
//# sourceMappingURL=trezor.d.ts.map