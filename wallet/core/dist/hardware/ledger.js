"use strict";
/**
 * Ledger Hardware Wallet Integration for PAW Chain
 *
 * Supports Ledger Nano S, Nano X, and Nano S Plus
 * Requires Cosmos app to be installed and open on the device
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.LedgerWallet = void 0;
exports.createLedgerWallet = createLedgerWallet;
exports.isLedgerSupported = isLedgerSupported;
exports.requestLedgerDevice = requestLedgerDevice;
const hw_transport_webusb_1 = __importDefault(require("@ledgerhq/hw-transport-webusb"));
const hw_app_cosmos_1 = __importDefault(require("@ledgerhq/hw-app-cosmos"));
const types_1 = require("./types");
const DEFAULT_COIN_TYPE = 118; // Cosmos coin type
const DEFAULT_PREFIX = 'paw';
const DEFAULT_TIMEOUT = 60000; // 60 seconds
class LedgerWallet {
    constructor(config = {}) {
        this.type = types_1.HardwareWalletType.LEDGER;
        this.transport = null;
        this.app = null;
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
    async isConnected() {
        try {
            if (!this.transport) {
                return false;
            }
            // Try to get app info to verify connection
            await this.transport.send(0xb0, 0x01, 0x00, 0x00);
            return true;
        }
        catch (error) {
            return false;
        }
    }
    /**
     * Connect to Ledger device
     */
    async connect() {
        try {
            // Close existing connection if any
            if (this.transport) {
                await this.disconnect();
            }
            // Request WebUSB device
            this.transport = (await hw_transport_webusb_1.default.create());
            // Set timeout
            this.transport.setExchangeTimeout(this.config.timeout);
            // Initialize Cosmos app
            this.app = new hw_app_cosmos_1.default(this.transport);
            // Get app version to verify connection
            const appInfo = await this.app.getVersion();
            // Get device info
            const deviceModel = await this.getDeviceModel();
            return {
                type: types_1.HardwareWalletType.LEDGER,
                model: deviceModel,
                version: `${appInfo.major}.${appInfo.minor}.${appInfo.patch}`,
                status: types_1.DeviceConnectionStatus.CONNECTED,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to connect to Ledger device');
        }
    }
    /**
     * Disconnect from device
     */
    async disconnect() {
        try {
            if (this.transport) {
                await this.transport.close();
                this.transport = null;
                this.app = null;
            }
        }
        catch (error) {
            if (this.config.debug) {
                console.warn('Error during disconnect:', error);
            }
            // Don't throw on disconnect errors
        }
    }
    /**
     * Get device information
     */
    async getDeviceInfo() {
        this.ensureConnected();
        try {
            const appInfo = await this.app.getVersion();
            const deviceModel = await this.getDeviceModel();
            return {
                type: types_1.HardwareWalletType.LEDGER,
                model: deviceModel,
                version: `${appInfo.major}.${appInfo.minor}.${appInfo.patch}`,
                status: types_1.DeviceConnectionStatus.CONNECTED,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get device info');
        }
    }
    /**
     * Get public key for a derivation path
     */
    async getPublicKey(path, showOnDevice = false) {
        this.ensureConnected();
        try {
            const hdPath = this.normalizePath(path);
            const response = await this.app.getAddress(hdPath, this.config.prefix, showOnDevice);
            // Convert hex string to Uint8Array
            return this.hexToBytes(response.publicKey);
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get public key');
        }
    }
    /**
     * Get address for a derivation path
     */
    async getAddress(path, showOnDevice = false) {
        this.ensureConnected();
        try {
            const hdPath = this.normalizePath(path);
            const response = await this.app.getAddress(hdPath, this.config.prefix, showOnDevice);
            return response.address;
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get address');
        }
    }
    /**
     * Get multiple addresses for account discovery
     */
    async getAddresses(paths) {
        this.ensureConnected();
        const accounts = [];
        for (let i = 0; i < paths.length; i++) {
            const path = paths[i];
            try {
                const hdPath = this.normalizePath(path);
                const response = await this.app.getAddress(hdPath, this.config.prefix, false);
                accounts.push({
                    address: response.address,
                    publicKey: this.hexToBytes(response.publicKey),
                    path,
                    index: i,
                });
            }
            catch (error) {
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
    async signTransaction(path, txBytes, _showOnDevice = true) {
        this.ensureConnected();
        try {
            // Parse the transaction to Cosmos format
            const tx = this.parseTransaction(txBytes);
            const hdPath = this.normalizePath(path);
            // Sign with Ledger
            const response = await this.app.sign(hdPath, JSON.stringify(tx));
            // Get public key
            const publicKey = await this.getPublicKey(path, false);
            return {
                signature: Buffer.from(response.signature, 'base64'),
                publicKey,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to sign transaction');
        }
    }
    /**
     * Sign arbitrary message
     */
    async signMessage(path, message, _showOnDevice = true) {
        this.ensureConnected();
        try {
            const messageStr = typeof message === 'string' ? message : Buffer.from(message).toString('utf8');
            const hdPath = this.normalizePath(path);
            // Use Cosmos app's sign message feature
            const response = await this.app.sign(hdPath, messageStr);
            const publicKey = await this.getPublicKey(path, false);
            return {
                signature: Buffer.from(response.signature, 'base64'),
                publicKey,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to sign message');
        }
    }
    // ==================== Private Helper Methods ====================
    ensureConnected() {
        if (!this.transport || !this.app) {
            throw this.createError('Device not connected', 'NOT_CONNECTED');
        }
    }
    normalizePath(path) {
        const sanitized = path.startsWith('m/') ? path.slice(2) : path;
        const segments = sanitized.split('/');
        if (segments.length !== 5) {
            throw this.createError(`Invalid derivation path: ${path}`, 'INVALID_PATH');
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
    parseTransaction(txBytes) {
        try {
            // Decode transaction bytes to JSON
            const txString = Buffer.from(txBytes).toString('utf8');
            return JSON.parse(txString);
        }
        catch (error) {
            throw this.createError('Failed to parse transaction', 'INVALID_TX');
        }
    }
    async getDeviceModel() {
        const productName = (this.transport && this.transport.device?.productName) || null;
        return productName || 'Ledger Device';
    }
    hexToBytes(hex) {
        // Remove 0x prefix if present
        const cleanHex = hex.startsWith('0x') ? hex.slice(2) : hex;
        const bytes = new Uint8Array(cleanHex.length / 2);
        for (let i = 0; i < cleanHex.length; i += 2) {
            bytes[i / 2] = parseInt(cleanHex.substr(i, 2), 16);
        }
        return bytes;
    }
    handleError(error, defaultMessage) {
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
        }
        else if (error.message) {
            message = error.message;
        }
        return this.createError(message, code);
    }
    createError(message, code) {
        const error = new Error(message);
        error.code = code;
        error.deviceType = types_1.HardwareWalletType.LEDGER;
        return error;
    }
}
exports.LedgerWallet = LedgerWallet;
/**
 * Factory function to create Ledger wallet instance
 */
function createLedgerWallet(config) {
    return new LedgerWallet(config);
}
/**
 * Check if Ledger is supported in current environment
 */
async function isLedgerSupported() {
    try {
        return await hw_transport_webusb_1.default.isSupported();
    }
    catch (error) {
        return false;
    }
}
/**
 * Request Ledger device connection (triggers browser permission)
 */
async function requestLedgerDevice() {
    try {
        await hw_transport_webusb_1.default.request();
    }
    catch (error) {
        throw new Error(`Failed to request Ledger device: ${error.message}`);
    }
}
//# sourceMappingURL=ledger.js.map