/**
 * PAW Wallet Core SDK
 * @packageDocumentation
 */
export { PAWWallet, createWallet, WalletConfig } from './wallet';
export { PAWRPCClient, createRPCClient, RPCConfig } from './rpc';
export { generateMnemonic, validateMnemonic, mnemonicToSeed, derivePrivateKey, derivePublicKey, deriveKeyPair, deriveHDPath, parseHDPath, publicKeyToAddress, validateAddress, createWalletAccount, signData, verifySignature, randomBytes, hash256, toHexString, fromHexString, encryptAES, decryptAES, encryptMnemonic, decryptMnemonic, DEFAULT_COIN_TYPE, DEFAULT_HD_PATH, PAW_BECH32_PREFIX, } from './crypto';
export { secureRandom, secureRandomHex, constantTimeCompare, constantTimeCompareString, validatePasswordStrength, secureWipe, secureWipeString, generateUUID, sha256, sha512, hmacSha256, hmacSha512, verifyHmacSha256, checkPasswordCompromise, RateLimiter, sanitizeError, } from './security';
export { deriveKey, deriveKeyBytes, deriveKeyArgon2, deriveEncryptionAndMacKeys, generateSalt, calculateOptimalIterations, verifyDerivedKey, getKeyDerivationInfo, compareKeyDerivationSecurity, MIN_PBKDF2_ITERATIONS, RECOMMENDED_PBKDF2_ITERATIONS, } from './keyDerivation';
export { encryptKeystore, decryptKeystore, exportKeystore, importKeystore, validateKeystore, changeKeystorePassword, generateKeystoreFilename, verifyKeystorePassword, getKeystoreSecurityLevel, sanitizeKeystore, estimateDecryptionTime, } from './keystore';
export { createRegistry, buildTxBody, buildAuthInfo, signTransaction, serializeSignedTx, encodeTxBase64, decodeTxBase64, estimateGas, createMsgSend, createMsgDelegate, createMsgUndelegate, createMsgRedelegate, createMsgWithdrawReward, createMsgVote, createMsgSwap, createMsgCreatePool, createMsgAddLiquidity, createMsgRemoveLiquidity, calculateTxHash, simulateTransaction, } from './transaction';
export { LedgerWallet, TrezorWallet, HardwareWalletFactory, HardwareWalletUtils, HardwareWalletManager, connectLedger, connectTrezor, connectHardwareWallet, checkHardwareWalletSupport, HardwareWalletType, DeviceConnectionStatus, IHardwareWallet, HardwareWalletInfo, HardwareWalletAccount, HardwareWalletConfig, HardwareWalletError, SignatureResult, } from './hardware';
export * from './types';
//# sourceMappingURL=index.d.ts.map