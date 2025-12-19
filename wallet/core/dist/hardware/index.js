"use strict";
/**
 * Hardware Wallet Support for PAW Chain
 *
 * Main export file for hardware wallet functionality
 * Supports Ledger and Trezor devices
 */
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.HardwareWalletManager = exports.HardwareWalletUtils = exports.HardwareWalletFactory = exports.TrezorWallet = exports.LedgerWallet = void 0;
exports.connectLedger = connectLedger;
exports.connectTrezor = connectTrezor;
exports.connectHardwareWallet = connectHardwareWallet;
exports.checkHardwareWalletSupport = checkHardwareWalletSupport;
const ledger_1 = require("./ledger");
Object.defineProperty(exports, "LedgerWallet", { enumerable: true, get: function () { return ledger_1.LedgerWallet; } });
const trezor_1 = require("./trezor");
Object.defineProperty(exports, "TrezorWallet", { enumerable: true, get: function () { return trezor_1.TrezorWallet; } });
const types_1 = require("./types");
// Re-export types
__exportStar(require("./types"), exports);
/**
 * Hardware wallet factory
 * Creates appropriate wallet instance based on type
 */
class HardwareWalletFactory {
    /**
     * Create hardware wallet instance
     */
    static create(type, config) {
        switch (type) {
            case types_1.HardwareWalletType.LEDGER:
                return (0, ledger_1.createLedgerWallet)(config);
            case types_1.HardwareWalletType.TREZOR:
                return (0, trezor_1.createTrezorWallet)(config);
            default:
                throw new Error(`Unsupported hardware wallet type: ${type}`);
        }
    }
    /**
     * Check which hardware wallets are supported
     */
    static async getSupportedWallets() {
        const supported = [];
        if (await (0, ledger_1.isLedgerSupported)()) {
            supported.push(types_1.HardwareWalletType.LEDGER);
        }
        if ((0, trezor_1.isTrezorSupported)()) {
            supported.push(types_1.HardwareWalletType.TREZOR);
        }
        return supported;
    }
    /**
     * Detect connected hardware wallets
     */
    static async detectWallets() {
        const detected = [];
        // Try Ledger
        try {
            const ledger = (0, ledger_1.createLedgerWallet)();
            if (await ledger.isConnected()) {
                const info = await ledger.getDeviceInfo();
                detected.push(info);
            }
        }
        catch (error) {
            // Ledger not connected
        }
        // Try Trezor
        try {
            const trezor = (0, trezor_1.createTrezorWallet)();
            if (await trezor.isConnected()) {
                const info = await trezor.getDeviceInfo();
                detected.push(info);
            }
        }
        catch (error) {
            // Trezor not connected
        }
        return detected;
    }
}
exports.HardwareWalletFactory = HardwareWalletFactory;
/**
 * Utility functions for hardware wallets
 */
class HardwareWalletUtils {
    /**
     * Generate standard Cosmos derivation paths
     */
    static generatePaths(coinType = 118, accountCount = 10, account = 0) {
        const paths = [];
        for (let i = 0; i < accountCount; i++) {
            paths.push(`m/44'/${coinType}'/${account}'/0/${i}`);
        }
        return paths;
    }
    /**
     * Parse derivation path
     */
    static parsePath(path) {
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
    static isValidPath(path) {
        try {
            this.parsePath(path);
            return true;
        }
        catch {
            return false;
        }
    }
    /**
     * Get user-friendly error message
     */
    static getErrorMessage(error) {
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
exports.HardwareWalletUtils = HardwareWalletUtils;
/**
 * Hardware wallet manager for handling multiple devices
 */
class HardwareWalletManager {
    constructor() {
        this.wallets = new Map();
    }
    /**
     * Add a hardware wallet
     */
    async addWallet(type, config) {
        const wallet = HardwareWalletFactory.create(type, config);
        const info = await wallet.connect();
        const id = `${type}-${info.deviceId || Date.now()}`;
        this.wallets.set(id, wallet);
        return id;
    }
    /**
     * Get a wallet by ID
     */
    getWallet(id) {
        return this.wallets.get(id);
    }
    /**
     * Remove a wallet
     */
    async removeWallet(id) {
        const wallet = this.wallets.get(id);
        if (wallet) {
            await wallet.disconnect();
            this.wallets.delete(id);
        }
    }
    /**
     * Get all wallets
     */
    getAllWallets() {
        return this.wallets;
    }
    /**
     * Disconnect all wallets
     */
    async disconnectAll() {
        const promises = Array.from(this.wallets.values()).map(wallet => wallet.disconnect());
        await Promise.all(promises);
        this.wallets.clear();
    }
}
exports.HardwareWalletManager = HardwareWalletManager;
/**
 * Main export - convenience functions
 */
/**
 * Connect to Ledger device
 */
async function connectLedger(config) {
    const wallet = (0, ledger_1.createLedgerWallet)(config);
    await wallet.connect();
    return wallet;
}
/**
 * Connect to Trezor device
 */
async function connectTrezor(config) {
    const wallet = (0, trezor_1.createTrezorWallet)(config);
    await wallet.connect();
    return wallet;
}
/**
 * Auto-detect and connect to hardware wallet
 */
async function connectHardwareWallet(preferredType, config) {
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
        }
        catch (error) {
            // Try next wallet type
        }
    }
    throw new Error('Failed to connect to any hardware wallet');
}
/**
 * Check hardware wallet support
 */
async function checkHardwareWalletSupport() {
    const ledger = await (0, ledger_1.isLedgerSupported)();
    const trezor = (0, trezor_1.isTrezorSupported)();
    const supported = await HardwareWalletFactory.getSupportedWallets();
    return {
        ledger,
        trezor,
        supported,
    };
}
//# sourceMappingURL=index.js.map