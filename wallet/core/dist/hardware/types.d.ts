/**
 * Hardware Wallet Type Definitions for PAW Chain
 * Supports Ledger and Trezor devices
 */
export declare enum HardwareWalletType {
    LEDGER = "ledger",
    TREZOR = "trezor"
}
export declare enum DeviceConnectionStatus {
    DISCONNECTED = "disconnected",
    CONNECTED = "connected",
    LOCKED = "locked",
    APP_NOT_OPEN = "app_not_open",
    BUSY = "busy"
}
export interface HardwareWalletInfo {
    type: HardwareWalletType;
    model: string;
    version: string;
    deviceId?: string;
    status: DeviceConnectionStatus;
}
export interface HardwareWalletAccount {
    address: string;
    publicKey: Uint8Array;
    path: string;
    index: number;
}
export interface SignOptions {
    /** Derivation path (e.g., "m/44'/118'/0'/0/0") */
    path: string;
    /** Show address on device for verification */
    showOnDevice?: boolean;
}
export interface SignatureResult {
    signature: Uint8Array;
    publicKey: Uint8Array;
}
export interface HardwareWalletError extends Error {
    code: string;
    deviceType: HardwareWalletType;
}
export interface DeviceInfo {
    manufacturer: string;
    product: string;
    serialNumber?: string;
}
/**
 * Base interface for all hardware wallet implementations
 */
export interface IHardwareWallet {
    /** Hardware wallet type */
    readonly type: HardwareWalletType;
    /** Check if device is connected */
    isConnected(): Promise<boolean>;
    /** Connect to the device */
    connect(): Promise<HardwareWalletInfo>;
    /** Disconnect from the device */
    disconnect(): Promise<void>;
    /** Get device information */
    getDeviceInfo(): Promise<HardwareWalletInfo>;
    /** Get public key for a given path */
    getPublicKey(path: string, showOnDevice?: boolean): Promise<Uint8Array>;
    /** Get address for a given path */
    getAddress(path: string, showOnDevice?: boolean): Promise<string>;
    /** Get multiple addresses (for account discovery) */
    getAddresses(paths: string[]): Promise<HardwareWalletAccount[]>;
    /** Sign a transaction */
    signTransaction(path: string, txBytes: Uint8Array, showOnDevice?: boolean): Promise<SignatureResult>;
    /** Sign arbitrary message (for authentication) */
    signMessage(path: string, message: string | Uint8Array, showOnDevice?: boolean): Promise<SignatureResult>;
}
/**
 * Configuration for hardware wallet connection
 */
export interface HardwareWalletConfig {
    /** Timeout for operations in milliseconds */
    timeout?: number;
    /** BIP44 coin type (default: 118 for Cosmos) */
    coinType?: number;
    /** Address prefix (default: paw) */
    prefix?: string;
    /** Enable debug logging */
    debug?: boolean;
}
/**
 * Transaction data structure for hardware wallet signing
 */
export interface HardwareSignDoc {
    chainId: string;
    accountNumber: string;
    sequence: string;
    fee: {
        amount: Array<{
            denom: string;
            amount: string;
        }>;
        gas: string;
    };
    msgs: Array<{
        type: string;
        value: any;
    }>;
    memo: string;
}
/**
 * Cosmos transaction for hardware wallet
 */
export interface CosmosTransaction {
    account_number: string;
    chain_id: string;
    fee: {
        amount: Array<{
            denom: string;
            amount: string;
        }>;
        gas: string;
    };
    memo: string;
    msgs: any[];
    sequence: string;
}
//# sourceMappingURL=types.d.ts.map