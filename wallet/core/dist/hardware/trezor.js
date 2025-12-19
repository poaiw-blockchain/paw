"use strict";
/**
 * Trezor Hardware Wallet Integration for PAW Chain
 *
 * Supports Trezor One and Trezor Model T
 * Uses Trezor Connect for browser integration
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
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.TrezorWallet = void 0;
exports.createTrezorWallet = createTrezorWallet;
exports.isTrezorSupported = isTrezorSupported;
const connect_web_1 = __importStar(require("@trezor/connect-web"));
const types_1 = require("./types");
const DEFAULT_COIN_TYPE = 118; // Cosmos coin type
const DEFAULT_PREFIX = 'paw';
const DEFAULT_TIMEOUT = 60000; // 60 seconds
class TrezorWallet {
    constructor(config = {}) {
        this.type = types_1.HardwareWalletType.TREZOR;
        this.initialized = false;
        this.connected = false;
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
    async initialize() {
        if (this.initialized)
            return;
        try {
            await connect_web_1.default.init({
                lazyLoad: true,
                manifest: {
                    email: 'support@pawchain.network',
                    appUrl: 'https://wallet.pawchain.network',
                },
                debug: this.config.debug,
            });
            this.initialized = true;
            // Setup event listeners
            connect_web_1.default.on(connect_web_1.DEVICE_EVENT, (event) => {
                if (this.config.debug) {
                    console.log('Trezor device event:', event);
                }
                if (event.type === 'device-connect') {
                    this.connected = true;
                }
                else if (event.type === 'device-disconnect') {
                    this.connected = false;
                }
            });
            connect_web_1.default.on(connect_web_1.UI_EVENT, (event) => {
                if (this.config.debug) {
                    console.log('Trezor UI event:', event);
                }
            });
        }
        catch (error) {
            throw this.handleError(error, 'Failed to initialize Trezor Connect');
        }
    }
    /**
     * Check if device is connected
     */
    async isConnected() {
        await this.initialize();
        return this.connected;
    }
    /**
     * Connect to Trezor device
     */
    async connect() {
        await this.initialize();
        try {
            // Get features to verify connection
            const result = await connect_web_1.default.getFeatures();
            if (!result.success) {
                throw this.createError(result.payload.error, 'CONNECTION_FAILED');
            }
            const features = result.payload;
            this.connected = true;
            return {
                type: types_1.HardwareWalletType.TREZOR,
                model: this.getModelName(features.model),
                version: `${features.major_version}.${features.minor_version}.${features.patch_version}`,
                deviceId: features.device_id || undefined,
                status: types_1.DeviceConnectionStatus.CONNECTED,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to connect to Trezor device');
        }
    }
    /**
     * Disconnect from device
     */
    async disconnect() {
        try {
            if (this.initialized) {
                connect_web_1.default.dispose();
                this.initialized = false;
                this.connected = false;
            }
        }
        catch (error) {
            if (this.config.debug) {
                console.warn('Error during disconnect:', error);
            }
        }
    }
    /**
     * Get device information
     */
    async getDeviceInfo() {
        await this.initialize();
        try {
            const result = await connect_web_1.default.getFeatures();
            if (!result.success) {
                throw this.createError(result.payload.error, 'GET_INFO_FAILED');
            }
            const features = result.payload;
            return {
                type: types_1.HardwareWalletType.TREZOR,
                model: this.getModelName(features.model),
                version: `${features.major_version}.${features.minor_version}.${features.patch_version}`,
                deviceId: features.device_id || undefined,
                status: this.connected ? types_1.DeviceConnectionStatus.CONNECTED : types_1.DeviceConnectionStatus.DISCONNECTED,
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
        await this.initialize();
        try {
            const result = await connect_web_1.default.getPublicKey({
                path,
                coin: 'Cosmos',
                showOnTrezor: showOnDevice,
            });
            if (!result.success) {
                throw this.createError(result.payload.error, 'GET_PUBKEY_FAILED');
            }
            // Convert hex public key to Uint8Array
            return this.hexToBytes(result.payload.publicKey);
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get public key');
        }
    }
    /**
     * Get address for a derivation path
     */
    async getAddress(path, showOnDevice = false) {
        await this.initialize();
        try {
            const result = await connect_web_1.default.cosmosGetAddress({
                path,
                showOnTrezor: showOnDevice,
            });
            if (!result.success) {
                throw this.createError(result.payload.error, 'GET_ADDRESS_FAILED');
            }
            const payload = result.payload;
            if (Array.isArray(payload)) {
                throw this.createError('Unexpected address bundle response', 'GET_ADDRESS_FAILED');
            }
            return payload.address;
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get address');
        }
    }
    /**
     * Get multiple addresses for account discovery
     */
    async getAddresses(paths) {
        await this.initialize();
        const accounts = [];
        // Get addresses in batch
        const bundle = paths.map(path => ({ path, showOnTrezor: false }));
        try {
            const result = await connect_web_1.default.cosmosGetAddress({ bundle });
            if (!result.success) {
                throw this.createError(result.payload.error, 'GET_ADDRESSES_FAILED');
            }
            const payload = result.payload;
            if (!Array.isArray(payload)) {
                throw this.createError('Unexpected single response payload', 'GET_ADDRESSES_FAILED');
            }
            for (let i = 0; i < payload.length; i++) {
                const item = payload[i];
                if (item.success) {
                    const pubkeyResult = await connect_web_1.default.getPublicKey({
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
                }
                else if (this.config.debug) {
                    console.warn(`Failed to fetch address for ${paths[i]}`);
                }
            }
            return accounts;
        }
        catch (error) {
            throw this.handleError(error, 'Failed to get addresses');
        }
    }
    /**
     * Sign a transaction
     */
    async signTransaction(path, txBytes, _showOnDevice = true) {
        await this.initialize();
        try {
            // Parse transaction
            const tx = this.parseTransaction(txBytes);
            // Sign with Trezor
            const result = await connect_web_1.default.cosmosSignTransaction({
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
                throw this.createError(result.payload.error, 'SIGN_TX_FAILED');
            }
            // Get public key
            const publicKey = await this.getPublicKey(path, false);
            return {
                signature: this.hexToBytes(result.payload.signature),
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
        await this.initialize();
        try {
            const messageStr = typeof message === 'string'
                ? message
                : Buffer.from(message).toString('utf8');
            // Trezor doesn't have native Cosmos message signing, use Ethereum-style
            const result = await connect_web_1.default.ethereumSignMessage({
                path,
                message: messageStr,
                hex: false,
            });
            if (!result.success) {
                throw this.createError(result.payload.error, 'SIGN_MSG_FAILED');
            }
            // Get public key
            const publicKey = await this.getPublicKey(path, false);
            return {
                signature: this.hexToBytes(result.payload.signature),
                publicKey,
            };
        }
        catch (error) {
            throw this.handleError(error, 'Failed to sign message');
        }
    }
    // ==================== Private Helper Methods ====================
    parseTransaction(txBytes) {
        try {
            const txString = Buffer.from(txBytes).toString('utf8');
            return JSON.parse(txString);
        }
        catch (error) {
            throw this.createError('Failed to parse transaction', 'INVALID_TX');
        }
    }
    getModelName(model) {
        switch (model) {
            case '1':
                return 'Trezor One';
            case 'T':
                return 'Trezor Model T';
            default:
                return `Trezor ${model}`;
        }
    }
    hexToBytes(hex) {
        const cleanHex = hex.startsWith('0x') ? hex.slice(2) : hex;
        const bytes = new Uint8Array(cleanHex.length / 2);
        for (let i = 0; i < cleanHex.length; i += 2) {
            bytes[i / 2] = parseInt(cleanHex.substr(i, 2), 16);
        }
        return bytes;
    }
    handleError(error, defaultMessage) {
        if (this.config.debug) {
            console.error('Trezor error:', error);
        }
        let message = defaultMessage;
        let code = 'UNKNOWN_ERROR';
        if (typeof error === 'string') {
            message = error;
        }
        else if (error.message) {
            message = error.message;
            // Map common errors
            if (error.message.includes('Cancelled')) {
                code = 'USER_REJECTED';
                message = 'User rejected on device';
            }
            else if (error.message.includes('not connected')) {
                code = 'NOT_CONNECTED';
                message = 'Device not connected';
            }
            else if (error.message.includes('PIN')) {
                code = 'DEVICE_LOCKED';
                message = 'Device locked - enter PIN';
            }
        }
        return this.createError(message, code);
    }
    createError(message, code) {
        const error = new Error(message);
        error.code = code;
        error.deviceType = types_1.HardwareWalletType.TREZOR;
        return error;
    }
}
exports.TrezorWallet = TrezorWallet;
/**
 * Factory function to create Trezor wallet instance
 */
function createTrezorWallet(config) {
    return new TrezorWallet(config);
}
/**
 * Check if Trezor is supported in current environment
 */
function isTrezorSupported() {
    // Trezor Connect works in all modern browsers
    return typeof window !== 'undefined' && !!window.navigator;
}
//# sourceMappingURL=trezor.js.map