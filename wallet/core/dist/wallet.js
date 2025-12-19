"use strict";
/**
 * Main Wallet class for PAW blockchain
 * Combines all functionalities: key management, transactions, and RPC
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.PAWWallet = void 0;
exports.createWallet = createWallet;
const crypto_1 = require("./crypto");
const keystore_1 = require("./keystore");
const transaction_1 = require("./transaction");
const rpc_1 = require("./rpc");
class PAWWallet {
    constructor(config) {
        this.rpcClient = (0, rpc_1.createRPCClient)(config.rpcConfig);
        this.hdPath = config.hdPath || crypto_1.DEFAULT_HD_PATH;
        this.prefix = config.prefix || crypto_1.PAW_BECH32_PREFIX;
    }
    // ==================== Wallet Creation & Import ====================
    /**
     * Create new wallet from mnemonic
     */
    async createFromMnemonic(mnemonic, password) {
        if (!(0, crypto_1.validateMnemonic)(mnemonic)) {
            throw new Error('Invalid mnemonic phrase');
        }
        this.mnemonic = mnemonic;
        this.privateKey = await (0, crypto_1.derivePrivateKey)(mnemonic, this.hdPath, password);
        this.publicKey = (0, crypto_1.derivePublicKey)(this.privateKey);
        this.address = (0, crypto_1.publicKeyToAddress)(this.publicKey, this.prefix);
        return {
            address: this.address,
            pubkey: this.publicKey,
            algo: 'secp256k1',
        };
    }
    /**
     * Generate new wallet
     */
    async generate(strength = 256) {
        const mnemonic = (0, crypto_1.generateMnemonic)(strength);
        const account = await this.createFromMnemonic(mnemonic);
        return { mnemonic, account };
    }
    /**
     * Import wallet from private key
     */
    async importPrivateKey(privateKey) {
        this.privateKey = privateKey;
        this.publicKey = (0, crypto_1.derivePublicKey)(privateKey);
        this.address = (0, crypto_1.publicKeyToAddress)(this.publicKey, this.prefix);
        return {
            address: this.address,
            pubkey: this.publicKey,
            algo: 'secp256k1',
        };
    }
    /**
     * Import wallet from keystore
     */
    async importKeystore(keystore, password) {
        const keystoreObj = typeof keystore === 'string' ? (0, keystore_1.importKeystore)(keystore) : keystore;
        const privateKey = await (0, keystore_1.decryptKeystore)(keystoreObj, password);
        return this.importPrivateKey(privateKey);
    }
    /**
     * Export wallet to keystore
     */
    async exportKeystore(password, name) {
        if (!this.privateKey || !this.address) {
            throw new Error('Wallet not initialized');
        }
        return (0, keystore_1.encryptKeystore)(this.privateKey, password, this.address, name);
    }
    /**
     * Get mnemonic (if available)
     */
    getMnemonic() {
        return this.mnemonic;
    }
    /**
     * Get wallet address
     */
    getAddress() {
        if (!this.address) {
            throw new Error('Wallet not initialized');
        }
        return this.address;
    }
    /**
     * Get public key
     */
    getPublicKey() {
        if (!this.publicKey) {
            throw new Error('Wallet not initialized');
        }
        return this.publicKey;
    }
    /**
     * Create HD wallet for multiple accounts
     */
    async deriveAccount(accountIndex) {
        if (!this.mnemonic) {
            throw new Error('Mnemonic required for HD derivation');
        }
        const hdPath = (0, crypto_1.deriveHDPath)({ addressIndex: accountIndex });
        const privateKey = await (0, crypto_1.derivePrivateKey)(this.mnemonic, hdPath);
        const publicKey = (0, crypto_1.derivePublicKey)(privateKey);
        const address = (0, crypto_1.publicKeyToAddress)(publicKey, this.prefix);
        return {
            address,
            pubkey: publicKey,
            algo: 'secp256k1',
        };
    }
    // ==================== Balance & Account Info ====================
    /**
     * Get balance
     */
    async getBalance(denom) {
        return this.rpcClient.getBalance(this.getAddress(), denom);
    }
    /**
     * Get account info
     */
    async getAccountInfo() {
        return this.rpcClient.getAccount(this.getAddress());
    }
    // ==================== Transaction Building & Signing ====================
    /**
     * Send tokens
     */
    async send(toAddress, amount, denom, options) {
        if (!(0, crypto_1.validateAddress)(toAddress, this.prefix)) {
            throw new Error('Invalid recipient address');
        }
        const message = (0, transaction_1.createMsgSend)(this.getAddress(), toAddress, amount, denom);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Delegate tokens
     */
    async delegate(validatorAddress, amount, denom, options) {
        const message = (0, transaction_1.createMsgDelegate)(this.getAddress(), validatorAddress, amount, denom);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Undelegate tokens
     */
    async undelegate(validatorAddress, amount, denom, options) {
        const message = (0, transaction_1.createMsgUndelegate)(this.getAddress(), validatorAddress, amount, denom);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Redelegate tokens
     */
    async redelegate(srcValidatorAddress, dstValidatorAddress, amount, denom, options) {
        const message = (0, transaction_1.createMsgRedelegate)(this.getAddress(), srcValidatorAddress, dstValidatorAddress, amount, denom);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Withdraw staking rewards
     */
    async withdrawRewards(validatorAddress, options) {
        const message = (0, transaction_1.createMsgWithdrawReward)(this.getAddress(), validatorAddress);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Vote on governance proposal
     */
    async vote(proposalId, option, options) {
        const message = (0, transaction_1.createMsgVote)(proposalId, this.getAddress(), option);
        return this.signAndBroadcast([message], options);
    }
    // ==================== DEX Operations ====================
    /**
     * Swap tokens
     */
    async swap(poolId, tokenIn, tokenOut, amountIn, minAmountOut, options) {
        const message = (0, transaction_1.createMsgSwap)(this.getAddress(), poolId, tokenIn, tokenOut, amountIn, minAmountOut);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Create liquidity pool
     */
    async createPool(tokenA, tokenB, amountA, amountB, options) {
        const message = (0, transaction_1.createMsgCreatePool)(this.getAddress(), tokenA, tokenB, amountA, amountB);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Add liquidity to pool
     */
    async addLiquidity(poolId, amountA, amountB, options) {
        const message = (0, transaction_1.createMsgAddLiquidity)(this.getAddress(), poolId, amountA, amountB);
        return this.signAndBroadcast([message], options);
    }
    /**
     * Remove liquidity from pool
     */
    async removeLiquidity(poolId, shares, options) {
        const message = (0, transaction_1.createMsgRemoveLiquidity)(this.getAddress(), poolId, shares);
        return this.signAndBroadcast([message], options);
    }
    // ==================== Query Functions ====================
    /**
     * Get validators
     */
    async getValidators(status) {
        return this.rpcClient.getValidators(status);
    }
    /**
     * Get delegations
     */
    async getDelegations() {
        return this.rpcClient.getDelegations(this.getAddress());
    }
    /**
     * Get rewards
     */
    async getRewards() {
        return this.rpcClient.getRewards(this.getAddress());
    }
    /**
     * Get pools
     */
    async getPools() {
        return this.rpcClient.getPools();
    }
    /**
     * Get pool
     */
    async getPool(poolId) {
        return this.rpcClient.getPool(poolId);
    }
    /**
     * Simulate swap
     */
    async simulateSwap(poolId, tokenIn, tokenOut, amountIn) {
        return this.rpcClient.simulateSwap(poolId, tokenIn, tokenOut, amountIn);
    }
    /**
     * Get transactions
     */
    async getTransactions(page = 1, limit = 10) {
        return this.rpcClient.getTransactionsByAddress(this.getAddress(), page, limit);
    }
    // ==================== Private Helper Methods ====================
    /**
     * Sign and broadcast transaction
     */
    async signAndBroadcast(messages, options) {
        if (!this.privateKey) {
            throw new Error('Wallet not initialized');
        }
        // Get account info
        const { accountNumber, sequence } = await this.getAccountInfo();
        // Get chain info
        const chainInfo = await this.rpcClient.getChainInfo();
        // Estimate gas if not provided
        let gasLimit;
        let feeAmount;
        let feeDenom;
        if (options?.fee) {
            gasLimit = parseInt(options.fee.gas);
            feeAmount = options.fee.amount[0]?.amount || '0';
            feeDenom = options.fee.amount[0]?.denom || 'upaw';
        }
        else {
            const estimation = (0, transaction_1.estimateGas)(messages);
            gasLimit = parseInt(estimation.gasLimit);
            feeAmount = estimation.feeAmount;
            feeDenom = estimation.feeDenom;
        }
        // Build transaction
        const txBody = (0, transaction_1.buildTxBody)(messages, options?.memo, options?.timeoutHeight);
        const authInfo = (0, transaction_1.buildAuthInfo)(this.publicKey, sequence, gasLimit, feeAmount, feeDenom);
        // Sign transaction
        const signedTx = await (0, transaction_1.signTransaction)(txBody, authInfo, chainInfo.chainId, accountNumber, this.privateKey);
        // Broadcast
        return this.rpcClient.broadcastTx(signedTx);
    }
    // ==================== RPC Client Access ====================
    /**
     * Get RPC client for advanced usage
     */
    getRPCClient() {
        return this.rpcClient;
    }
}
exports.PAWWallet = PAWWallet;
/**
 * Create PAW wallet instance
 */
function createWallet(config) {
    return new PAWWallet(config);
}
//# sourceMappingURL=wallet.js.map