/**
 * Ledger Hardware Wallet Integration for PAW Chain
 *
 * Supports Ledger Nano S, Nano X, and Nano S Plus
 * Requires Cosmos app to be installed and open on the device
 */
import { IHardwareWallet, HardwareWalletType, HardwareWalletInfo, HardwareWalletAccount, SignatureResult, HardwareWalletConfig } from './types';
export declare class LedgerWallet implements IHardwareWallet {
    readonly type = HardwareWalletType.LEDGER;
    private transport;
    private app;
    private config;
    constructor(config?: HardwareWalletConfig);
    /**
     * Check if device is connected
     */
    isConnected(): Promise<boolean>;
    /**
     * Connect to Ledger device
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
    private ensureConnected;
    private normalizePath;
    private parseTransaction;
    private getDeviceModel;
    private hexToBytes;
    private handleError;
    private createError;
}
/**
 * Factory function to create Ledger wallet instance
 */
export declare function createLedgerWallet(config?: HardwareWalletConfig): LedgerWallet;
/**
 * Check if Ledger is supported in current environment
 */
export declare function isLedgerSupported(): Promise<boolean>;
/**
 * Request Ledger device connection (triggers browser permission)
 */
export declare function requestLedgerDevice(): Promise<void>;
//# sourceMappingURL=ledger.d.ts.map