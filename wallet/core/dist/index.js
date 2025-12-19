"use strict";
/**
 * PAW Wallet Core SDK
 * @packageDocumentation
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
exports.calculateOptimalIterations = exports.generateSalt = exports.deriveEncryptionAndMacKeys = exports.deriveKeyArgon2 = exports.deriveKeyBytes = exports.deriveKey = exports.sanitizeError = exports.RateLimiter = exports.checkPasswordCompromise = exports.verifyHmacSha256 = exports.hmacSha512 = exports.hmacSha256 = exports.sha512 = exports.sha256 = exports.generateUUID = exports.secureWipeString = exports.secureWipe = exports.validatePasswordStrength = exports.constantTimeCompareString = exports.constantTimeCompare = exports.secureRandomHex = exports.secureRandom = exports.PAW_BECH32_PREFIX = exports.DEFAULT_HD_PATH = exports.DEFAULT_COIN_TYPE = exports.decryptMnemonic = exports.encryptMnemonic = exports.decryptAES = exports.encryptAES = exports.fromHexString = exports.toHexString = exports.hash256 = exports.randomBytes = exports.verifySignature = exports.signData = exports.createWalletAccount = exports.validateAddress = exports.publicKeyToAddress = exports.parseHDPath = exports.deriveHDPath = exports.deriveKeyPair = exports.derivePublicKey = exports.derivePrivateKey = exports.mnemonicToSeed = exports.validateMnemonic = exports.generateMnemonic = exports.createRPCClient = exports.PAWRPCClient = exports.createWallet = exports.PAWWallet = void 0;
exports.DeviceConnectionStatus = exports.HardwareWalletType = exports.checkHardwareWalletSupport = exports.connectHardwareWallet = exports.connectTrezor = exports.connectLedger = exports.HardwareWalletManager = exports.HardwareWalletUtils = exports.HardwareWalletFactory = exports.TrezorWallet = exports.LedgerWallet = exports.simulateTransaction = exports.calculateTxHash = exports.createMsgRemoveLiquidity = exports.createMsgAddLiquidity = exports.createMsgCreatePool = exports.createMsgSwap = exports.createMsgVote = exports.createMsgWithdrawReward = exports.createMsgRedelegate = exports.createMsgUndelegate = exports.createMsgDelegate = exports.createMsgSend = exports.estimateGas = exports.decodeTxBase64 = exports.encodeTxBase64 = exports.serializeSignedTx = exports.signTransaction = exports.buildAuthInfo = exports.buildTxBody = exports.createRegistry = exports.estimateDecryptionTime = exports.sanitizeKeystore = exports.getKeystoreSecurityLevel = exports.verifyKeystorePassword = exports.generateKeystoreFilename = exports.changeKeystorePassword = exports.validateKeystore = exports.importKeystore = exports.exportKeystore = exports.decryptKeystore = exports.encryptKeystore = exports.RECOMMENDED_PBKDF2_ITERATIONS = exports.MIN_PBKDF2_ITERATIONS = exports.compareKeyDerivationSecurity = exports.getKeyDerivationInfo = exports.verifyDerivedKey = void 0;
// Main wallet class
var wallet_1 = require("./wallet");
Object.defineProperty(exports, "PAWWallet", { enumerable: true, get: function () { return wallet_1.PAWWallet; } });
Object.defineProperty(exports, "createWallet", { enumerable: true, get: function () { return wallet_1.createWallet; } });
// RPC Client
var rpc_1 = require("./rpc");
Object.defineProperty(exports, "PAWRPCClient", { enumerable: true, get: function () { return rpc_1.PAWRPCClient; } });
Object.defineProperty(exports, "createRPCClient", { enumerable: true, get: function () { return rpc_1.createRPCClient; } });
// Cryptography functions
var crypto_1 = require("./crypto");
Object.defineProperty(exports, "generateMnemonic", { enumerable: true, get: function () { return crypto_1.generateMnemonic; } });
Object.defineProperty(exports, "validateMnemonic", { enumerable: true, get: function () { return crypto_1.validateMnemonic; } });
Object.defineProperty(exports, "mnemonicToSeed", { enumerable: true, get: function () { return crypto_1.mnemonicToSeed; } });
Object.defineProperty(exports, "derivePrivateKey", { enumerable: true, get: function () { return crypto_1.derivePrivateKey; } });
Object.defineProperty(exports, "derivePublicKey", { enumerable: true, get: function () { return crypto_1.derivePublicKey; } });
Object.defineProperty(exports, "deriveKeyPair", { enumerable: true, get: function () { return crypto_1.deriveKeyPair; } });
Object.defineProperty(exports, "deriveHDPath", { enumerable: true, get: function () { return crypto_1.deriveHDPath; } });
Object.defineProperty(exports, "parseHDPath", { enumerable: true, get: function () { return crypto_1.parseHDPath; } });
Object.defineProperty(exports, "publicKeyToAddress", { enumerable: true, get: function () { return crypto_1.publicKeyToAddress; } });
Object.defineProperty(exports, "validateAddress", { enumerable: true, get: function () { return crypto_1.validateAddress; } });
Object.defineProperty(exports, "createWalletAccount", { enumerable: true, get: function () { return crypto_1.createWalletAccount; } });
Object.defineProperty(exports, "signData", { enumerable: true, get: function () { return crypto_1.signData; } });
Object.defineProperty(exports, "verifySignature", { enumerable: true, get: function () { return crypto_1.verifySignature; } });
Object.defineProperty(exports, "randomBytes", { enumerable: true, get: function () { return crypto_1.randomBytes; } });
Object.defineProperty(exports, "hash256", { enumerable: true, get: function () { return crypto_1.hash256; } });
Object.defineProperty(exports, "toHexString", { enumerable: true, get: function () { return crypto_1.toHexString; } });
Object.defineProperty(exports, "fromHexString", { enumerable: true, get: function () { return crypto_1.fromHexString; } });
Object.defineProperty(exports, "encryptAES", { enumerable: true, get: function () { return crypto_1.encryptAES; } });
Object.defineProperty(exports, "decryptAES", { enumerable: true, get: function () { return crypto_1.decryptAES; } });
Object.defineProperty(exports, "encryptMnemonic", { enumerable: true, get: function () { return crypto_1.encryptMnemonic; } });
Object.defineProperty(exports, "decryptMnemonic", { enumerable: true, get: function () { return crypto_1.decryptMnemonic; } });
Object.defineProperty(exports, "DEFAULT_COIN_TYPE", { enumerable: true, get: function () { return crypto_1.DEFAULT_COIN_TYPE; } });
Object.defineProperty(exports, "DEFAULT_HD_PATH", { enumerable: true, get: function () { return crypto_1.DEFAULT_HD_PATH; } });
Object.defineProperty(exports, "PAW_BECH32_PREFIX", { enumerable: true, get: function () { return crypto_1.PAW_BECH32_PREFIX; } });
// Security utilities
var security_1 = require("./security");
Object.defineProperty(exports, "secureRandom", { enumerable: true, get: function () { return security_1.secureRandom; } });
Object.defineProperty(exports, "secureRandomHex", { enumerable: true, get: function () { return security_1.secureRandomHex; } });
Object.defineProperty(exports, "constantTimeCompare", { enumerable: true, get: function () { return security_1.constantTimeCompare; } });
Object.defineProperty(exports, "constantTimeCompareString", { enumerable: true, get: function () { return security_1.constantTimeCompareString; } });
Object.defineProperty(exports, "validatePasswordStrength", { enumerable: true, get: function () { return security_1.validatePasswordStrength; } });
Object.defineProperty(exports, "secureWipe", { enumerable: true, get: function () { return security_1.secureWipe; } });
Object.defineProperty(exports, "secureWipeString", { enumerable: true, get: function () { return security_1.secureWipeString; } });
Object.defineProperty(exports, "generateUUID", { enumerable: true, get: function () { return security_1.generateUUID; } });
Object.defineProperty(exports, "sha256", { enumerable: true, get: function () { return security_1.sha256; } });
Object.defineProperty(exports, "sha512", { enumerable: true, get: function () { return security_1.sha512; } });
Object.defineProperty(exports, "hmacSha256", { enumerable: true, get: function () { return security_1.hmacSha256; } });
Object.defineProperty(exports, "hmacSha512", { enumerable: true, get: function () { return security_1.hmacSha512; } });
Object.defineProperty(exports, "verifyHmacSha256", { enumerable: true, get: function () { return security_1.verifyHmacSha256; } });
Object.defineProperty(exports, "checkPasswordCompromise", { enumerable: true, get: function () { return security_1.checkPasswordCompromise; } });
Object.defineProperty(exports, "RateLimiter", { enumerable: true, get: function () { return security_1.RateLimiter; } });
Object.defineProperty(exports, "sanitizeError", { enumerable: true, get: function () { return security_1.sanitizeError; } });
// Key derivation functions
var keyDerivation_1 = require("./keyDerivation");
Object.defineProperty(exports, "deriveKey", { enumerable: true, get: function () { return keyDerivation_1.deriveKey; } });
Object.defineProperty(exports, "deriveKeyBytes", { enumerable: true, get: function () { return keyDerivation_1.deriveKeyBytes; } });
Object.defineProperty(exports, "deriveKeyArgon2", { enumerable: true, get: function () { return keyDerivation_1.deriveKeyArgon2; } });
Object.defineProperty(exports, "deriveEncryptionAndMacKeys", { enumerable: true, get: function () { return keyDerivation_1.deriveEncryptionAndMacKeys; } });
Object.defineProperty(exports, "generateSalt", { enumerable: true, get: function () { return keyDerivation_1.generateSalt; } });
Object.defineProperty(exports, "calculateOptimalIterations", { enumerable: true, get: function () { return keyDerivation_1.calculateOptimalIterations; } });
Object.defineProperty(exports, "verifyDerivedKey", { enumerable: true, get: function () { return keyDerivation_1.verifyDerivedKey; } });
Object.defineProperty(exports, "getKeyDerivationInfo", { enumerable: true, get: function () { return keyDerivation_1.getKeyDerivationInfo; } });
Object.defineProperty(exports, "compareKeyDerivationSecurity", { enumerable: true, get: function () { return keyDerivation_1.compareKeyDerivationSecurity; } });
Object.defineProperty(exports, "MIN_PBKDF2_ITERATIONS", { enumerable: true, get: function () { return keyDerivation_1.MIN_PBKDF2_ITERATIONS; } });
Object.defineProperty(exports, "RECOMMENDED_PBKDF2_ITERATIONS", { enumerable: true, get: function () { return keyDerivation_1.RECOMMENDED_PBKDF2_ITERATIONS; } });
// Keystore functions
var keystore_1 = require("./keystore");
Object.defineProperty(exports, "encryptKeystore", { enumerable: true, get: function () { return keystore_1.encryptKeystore; } });
Object.defineProperty(exports, "decryptKeystore", { enumerable: true, get: function () { return keystore_1.decryptKeystore; } });
Object.defineProperty(exports, "exportKeystore", { enumerable: true, get: function () { return keystore_1.exportKeystore; } });
Object.defineProperty(exports, "importKeystore", { enumerable: true, get: function () { return keystore_1.importKeystore; } });
Object.defineProperty(exports, "validateKeystore", { enumerable: true, get: function () { return keystore_1.validateKeystore; } });
Object.defineProperty(exports, "changeKeystorePassword", { enumerable: true, get: function () { return keystore_1.changeKeystorePassword; } });
Object.defineProperty(exports, "generateKeystoreFilename", { enumerable: true, get: function () { return keystore_1.generateKeystoreFilename; } });
Object.defineProperty(exports, "verifyKeystorePassword", { enumerable: true, get: function () { return keystore_1.verifyKeystorePassword; } });
Object.defineProperty(exports, "getKeystoreSecurityLevel", { enumerable: true, get: function () { return keystore_1.getKeystoreSecurityLevel; } });
Object.defineProperty(exports, "sanitizeKeystore", { enumerable: true, get: function () { return keystore_1.sanitizeKeystore; } });
Object.defineProperty(exports, "estimateDecryptionTime", { enumerable: true, get: function () { return keystore_1.estimateDecryptionTime; } });
// Transaction functions
var transaction_1 = require("./transaction");
Object.defineProperty(exports, "createRegistry", { enumerable: true, get: function () { return transaction_1.createRegistry; } });
Object.defineProperty(exports, "buildTxBody", { enumerable: true, get: function () { return transaction_1.buildTxBody; } });
Object.defineProperty(exports, "buildAuthInfo", { enumerable: true, get: function () { return transaction_1.buildAuthInfo; } });
Object.defineProperty(exports, "signTransaction", { enumerable: true, get: function () { return transaction_1.signTransaction; } });
Object.defineProperty(exports, "serializeSignedTx", { enumerable: true, get: function () { return transaction_1.serializeSignedTx; } });
Object.defineProperty(exports, "encodeTxBase64", { enumerable: true, get: function () { return transaction_1.encodeTxBase64; } });
Object.defineProperty(exports, "decodeTxBase64", { enumerable: true, get: function () { return transaction_1.decodeTxBase64; } });
Object.defineProperty(exports, "estimateGas", { enumerable: true, get: function () { return transaction_1.estimateGas; } });
Object.defineProperty(exports, "createMsgSend", { enumerable: true, get: function () { return transaction_1.createMsgSend; } });
Object.defineProperty(exports, "createMsgDelegate", { enumerable: true, get: function () { return transaction_1.createMsgDelegate; } });
Object.defineProperty(exports, "createMsgUndelegate", { enumerable: true, get: function () { return transaction_1.createMsgUndelegate; } });
Object.defineProperty(exports, "createMsgRedelegate", { enumerable: true, get: function () { return transaction_1.createMsgRedelegate; } });
Object.defineProperty(exports, "createMsgWithdrawReward", { enumerable: true, get: function () { return transaction_1.createMsgWithdrawReward; } });
Object.defineProperty(exports, "createMsgVote", { enumerable: true, get: function () { return transaction_1.createMsgVote; } });
Object.defineProperty(exports, "createMsgSwap", { enumerable: true, get: function () { return transaction_1.createMsgSwap; } });
Object.defineProperty(exports, "createMsgCreatePool", { enumerable: true, get: function () { return transaction_1.createMsgCreatePool; } });
Object.defineProperty(exports, "createMsgAddLiquidity", { enumerable: true, get: function () { return transaction_1.createMsgAddLiquidity; } });
Object.defineProperty(exports, "createMsgRemoveLiquidity", { enumerable: true, get: function () { return transaction_1.createMsgRemoveLiquidity; } });
Object.defineProperty(exports, "calculateTxHash", { enumerable: true, get: function () { return transaction_1.calculateTxHash; } });
Object.defineProperty(exports, "simulateTransaction", { enumerable: true, get: function () { return transaction_1.simulateTransaction; } });
// Hardware wallet support
var hardware_1 = require("./hardware");
Object.defineProperty(exports, "LedgerWallet", { enumerable: true, get: function () { return hardware_1.LedgerWallet; } });
Object.defineProperty(exports, "TrezorWallet", { enumerable: true, get: function () { return hardware_1.TrezorWallet; } });
Object.defineProperty(exports, "HardwareWalletFactory", { enumerable: true, get: function () { return hardware_1.HardwareWalletFactory; } });
Object.defineProperty(exports, "HardwareWalletUtils", { enumerable: true, get: function () { return hardware_1.HardwareWalletUtils; } });
Object.defineProperty(exports, "HardwareWalletManager", { enumerable: true, get: function () { return hardware_1.HardwareWalletManager; } });
Object.defineProperty(exports, "connectLedger", { enumerable: true, get: function () { return hardware_1.connectLedger; } });
Object.defineProperty(exports, "connectTrezor", { enumerable: true, get: function () { return hardware_1.connectTrezor; } });
Object.defineProperty(exports, "connectHardwareWallet", { enumerable: true, get: function () { return hardware_1.connectHardwareWallet; } });
Object.defineProperty(exports, "checkHardwareWalletSupport", { enumerable: true, get: function () { return hardware_1.checkHardwareWalletSupport; } });
Object.defineProperty(exports, "HardwareWalletType", { enumerable: true, get: function () { return hardware_1.HardwareWalletType; } });
Object.defineProperty(exports, "DeviceConnectionStatus", { enumerable: true, get: function () { return hardware_1.DeviceConnectionStatus; } });
// Type exports
__exportStar(require("./types"), exports);
//# sourceMappingURL=index.js.map