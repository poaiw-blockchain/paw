/**
 * Hardware Wallet Support for PAW Chain
 *
 * Main export file for hardware wallet functionality
 * Supports Ledger and Trezor devices
 */
import { LedgerWallet } from './ledger';
import { TrezorWallet } from './trezor';
import { IHardwareWallet, HardwareWalletType, HardwareWalletInfo, HardwareWalletConfig, HardwareWalletError } from './types';
export * from './types';
export { LedgerWallet, TrezorWallet };
/**
 * Hardware wallet factory
 * Creates appropriate wallet instance based on type
 */
export declare class HardwareWalletFactory {
    /**
     * Create hardware wallet instance
     */
    static create(type: HardwareWalletType, config?: HardwareWalletConfig): IHardwareWallet;
    /**
     * Check which hardware wallets are supported
     */
    static getSupportedWallets(): Promise<HardwareWalletType[]>;
    /**
     * Detect connected hardware wallets
     */
    static detectWallets(): Promise<HardwareWalletInfo[]>;
}
/**
 * Utility functions for hardware wallets
 */
export declare class HardwareWalletUtils {
    /**
     * Generate standard Cosmos derivation paths
     */
    static generatePaths(coinType?: number, accountCount?: number, account?: number): string[];
    /**
     * Parse derivation path
     */
    static parsePath(path: string): {
        coinType: number;
        account: number;
        change: number;
        index: number;
    };
    /**
     * Validate derivation path
     */
    static isValidPath(path: string): boolean;
    /**
     * Get user-friendly error message
     */
    static getErrorMessage(error: HardwareWalletError): string;
}
/**
 * Hardware wallet manager for handling multiple devices
 */
export declare class HardwareWalletManager {
    private wallets;
    /**
     * Add a hardware wallet
     */
    addWallet(type: HardwareWalletType, config?: HardwareWalletConfig): Promise<string>;
    /**
     * Get a wallet by ID
     */
    getWallet(id: string): IHardwareWallet | undefined;
    /**
     * Remove a wallet
     */
    removeWallet(id: string): Promise<void>;
    /**
     * Get all wallets
     */
    getAllWallets(): Map<string, IHardwareWallet>;
    /**
     * Disconnect all wallets
     */
    disconnectAll(): Promise<void>;
}
/**
 * Main export - convenience functions
 */
/**
 * Connect to Ledger device
 */
export declare function connectLedger(config?: HardwareWalletConfig): Promise<LedgerWallet>;
/**
 * Connect to Trezor device
 */
export declare function connectTrezor(config?: HardwareWalletConfig): Promise<TrezorWallet>;
/**
 * Auto-detect and connect to hardware wallet
 */
export declare function connectHardwareWallet(preferredType?: HardwareWalletType, config?: HardwareWalletConfig): Promise<IHardwareWallet>;
/**
 * Check hardware wallet support
 */
export declare function checkHardwareWalletSupport(): Promise<{
    ledger: boolean;
    trezor: boolean;
    supported: HardwareWalletType[];
}>;
//# sourceMappingURL=index.d.ts.map