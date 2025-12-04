/**
 * Main Wallet class for PAW blockchain
 * Combines all functionalities: key management, transactions, and RPC
 */

import {
  generateMnemonic,
  validateMnemonic,
  derivePrivateKey,
  derivePublicKey,
  publicKeyToAddress,
  validateAddress,
  deriveHDPath,
  DEFAULT_HD_PATH,
  PAW_BECH32_PREFIX,
} from './crypto';
import {
  encryptKeystore,
  decryptKeystore,
  importKeystore,
} from './keystore';
import {
  buildTxBody,
  buildAuthInfo,
  signTransaction,
  estimateGas,
  createMsgSend,
  createMsgDelegate,
  createMsgUndelegate,
  createMsgRedelegate,
  createMsgWithdrawReward,
  createMsgVote,
  createMsgSwap,
  createMsgCreatePool,
  createMsgAddLiquidity,
  createMsgRemoveLiquidity,
} from './transaction';
import { PAWRPCClient, RPCConfig, createRPCClient } from './rpc';
import {
  WalletAccount,
  SecureKeystore,
  Balance,
  Validator,
  Pool,
  TransactionOptions,
  BroadcastResult,
} from './types';

export interface WalletConfig {
  rpcConfig: RPCConfig;
  hdPath?: string;
  prefix?: string;
}

export class PAWWallet {
  private rpcClient: PAWRPCClient;
  private hdPath: string;
  private prefix: string;
  private privateKey?: Uint8Array;
  private publicKey?: Uint8Array;
  private address?: string;
  private mnemonic?: string;

  constructor(config: WalletConfig) {
    this.rpcClient = createRPCClient(config.rpcConfig);
    this.hdPath = config.hdPath || DEFAULT_HD_PATH;
    this.prefix = config.prefix || PAW_BECH32_PREFIX;
  }

  // ==================== Wallet Creation & Import ====================

  /**
   * Create new wallet from mnemonic
   */
  async createFromMnemonic(mnemonic: string, password?: string): Promise<WalletAccount> {
    if (!validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase');
    }

    this.mnemonic = mnemonic;
    this.privateKey = await derivePrivateKey(mnemonic, this.hdPath, password);
    this.publicKey = derivePublicKey(this.privateKey);
    this.address = publicKeyToAddress(this.publicKey, this.prefix);

    return {
      address: this.address,
      pubkey: this.publicKey,
      algo: 'secp256k1',
    };
  }

  /**
   * Generate new wallet
   */
  async generate(strength: 128 | 160 | 192 | 224 | 256 = 256): Promise<{ mnemonic: string; account: WalletAccount }> {
    const mnemonic = generateMnemonic(strength);
    const account = await this.createFromMnemonic(mnemonic);

    return { mnemonic, account };
  }

  /**
   * Import wallet from private key
   */
  async importPrivateKey(privateKey: Uint8Array): Promise<WalletAccount> {
    this.privateKey = privateKey;
    this.publicKey = derivePublicKey(privateKey);
    this.address = publicKeyToAddress(this.publicKey, this.prefix);

    return {
      address: this.address,
      pubkey: this.publicKey,
      algo: 'secp256k1',
    };
  }

  /**
   * Import wallet from keystore
   */
  async importKeystore(keystore: SecureKeystore | string, password: string): Promise<WalletAccount> {
    const keystoreObj = typeof keystore === 'string' ? importKeystore(keystore) : keystore;
    const privateKey = await decryptKeystore(keystoreObj, password);

    return this.importPrivateKey(privateKey);
  }

  /**
   * Export wallet to keystore
   */
  async exportKeystore(password: string, name?: string): Promise<SecureKeystore> {
    if (!this.privateKey || !this.address) {
      throw new Error('Wallet not initialized');
    }

    return encryptKeystore(this.privateKey, password, this.address, name);
  }

  /**
   * Get mnemonic (if available)
   */
  getMnemonic(): string | undefined {
    return this.mnemonic;
  }

  /**
   * Get wallet address
   */
  getAddress(): string {
    if (!this.address) {
      throw new Error('Wallet not initialized');
    }
    return this.address;
  }

  /**
   * Get public key
   */
  getPublicKey(): Uint8Array {
    if (!this.publicKey) {
      throw new Error('Wallet not initialized');
    }
    return this.publicKey;
  }

  /**
   * Create HD wallet for multiple accounts
   */
  async deriveAccount(accountIndex: number): Promise<WalletAccount> {
    if (!this.mnemonic) {
      throw new Error('Mnemonic required for HD derivation');
    }

    const hdPath = deriveHDPath({ addressIndex: accountIndex });
    const privateKey = await derivePrivateKey(this.mnemonic, hdPath);
    const publicKey = derivePublicKey(privateKey);
    const address = publicKeyToAddress(publicKey, this.prefix);

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
  async getBalance(denom?: string): Promise<Balance[]> {
    return this.rpcClient.getBalance(this.getAddress(), denom);
  }

  /**
   * Get account info
   */
  async getAccountInfo(): Promise<{ accountNumber: number; sequence: number }> {
    return this.rpcClient.getAccount(this.getAddress());
  }

  // ==================== Transaction Building & Signing ====================

  /**
   * Send tokens
   */
  async send(
    toAddress: string,
    amount: string,
    denom: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    if (!validateAddress(toAddress, this.prefix)) {
      throw new Error('Invalid recipient address');
    }

    const message = createMsgSend(this.getAddress(), toAddress, amount, denom);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Delegate tokens
   */
  async delegate(
    validatorAddress: string,
    amount: string,
    denom: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgDelegate(this.getAddress(), validatorAddress, amount, denom);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Undelegate tokens
   */
  async undelegate(
    validatorAddress: string,
    amount: string,
    denom: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgUndelegate(this.getAddress(), validatorAddress, amount, denom);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Redelegate tokens
   */
  async redelegate(
    srcValidatorAddress: string,
    dstValidatorAddress: string,
    amount: string,
    denom: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgRedelegate(
      this.getAddress(),
      srcValidatorAddress,
      dstValidatorAddress,
      amount,
      denom
    );
    return this.signAndBroadcast([message], options);
  }

  /**
   * Withdraw staking rewards
   */
  async withdrawRewards(validatorAddress: string, options?: TransactionOptions): Promise<BroadcastResult> {
    const message = createMsgWithdrawReward(this.getAddress(), validatorAddress);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Vote on governance proposal
   */
  async vote(
    proposalId: string,
    option: 1 | 2 | 3 | 4,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgVote(proposalId, this.getAddress(), option);
    return this.signAndBroadcast([message], options);
  }

  // ==================== DEX Operations ====================

  /**
   * Swap tokens
   */
  async swap(
    poolId: number,
    tokenIn: string,
    tokenOut: string,
    amountIn: string,
    minAmountOut: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgSwap(this.getAddress(), poolId, tokenIn, tokenOut, amountIn, minAmountOut);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Create liquidity pool
   */
  async createPool(
    tokenA: string,
    tokenB: string,
    amountA: string,
    amountB: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgCreatePool(this.getAddress(), tokenA, tokenB, amountA, amountB);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Add liquidity to pool
   */
  async addLiquidity(
    poolId: number,
    amountA: string,
    amountB: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgAddLiquidity(this.getAddress(), poolId, amountA, amountB);
    return this.signAndBroadcast([message], options);
  }

  /**
   * Remove liquidity from pool
   */
  async removeLiquidity(
    poolId: number,
    shares: string,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    const message = createMsgRemoveLiquidity(this.getAddress(), poolId, shares);
    return this.signAndBroadcast([message], options);
  }

  // ==================== Query Functions ====================

  /**
   * Get validators
   */
  async getValidators(status?: 'BOND_STATUS_BONDED' | 'BOND_STATUS_UNBONDED' | 'BOND_STATUS_UNBONDING'): Promise<Validator[]> {
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
  async getPools(): Promise<Pool[]> {
    return this.rpcClient.getPools();
  }

  /**
   * Get pool
   */
  async getPool(poolId: string): Promise<Pool> {
    return this.rpcClient.getPool(poolId);
  }

  /**
   * Simulate swap
   */
  async simulateSwap(poolId: string, tokenIn: string, tokenOut: string, amountIn: string) {
    return this.rpcClient.simulateSwap(poolId, tokenIn, tokenOut, amountIn);
  }

  /**
   * Get transactions
   */
  async getTransactions(page: number = 1, limit: number = 10) {
    return this.rpcClient.getTransactionsByAddress(this.getAddress(), page, limit);
  }

  // ==================== Private Helper Methods ====================

  /**
   * Sign and broadcast transaction
   */
  private async signAndBroadcast(
    messages: Array<{ typeUrl: string; value: any }>,
    options?: TransactionOptions
  ): Promise<BroadcastResult> {
    if (!this.privateKey) {
      throw new Error('Wallet not initialized');
    }

    // Get account info
    const { accountNumber, sequence } = await this.getAccountInfo();

    // Get chain info
    const chainInfo = await this.rpcClient.getChainInfo();

    // Estimate gas if not provided
    let gasLimit: number;
    let feeAmount: string;
    let feeDenom: string;

    if (options?.fee) {
      gasLimit = parseInt(options.fee.gas);
      feeAmount = options.fee.amount[0]?.amount || '0';
      feeDenom = options.fee.amount[0]?.denom || 'upaw';
    } else {
      const estimation = estimateGas(messages);
      gasLimit = parseInt(estimation.gasLimit);
      feeAmount = estimation.feeAmount;
      feeDenom = estimation.feeDenom;
    }

    // Build transaction
    const txBody = buildTxBody(messages, options?.memo, options?.timeoutHeight);
    const authInfo = buildAuthInfo(this.publicKey!, sequence, gasLimit, feeAmount, feeDenom);

    // Sign transaction
    const signedTx = await signTransaction(
      txBody,
      authInfo,
      chainInfo.chainId,
      accountNumber,
      this.privateKey
    );

    // Broadcast
    return this.rpcClient.broadcastTx(signedTx);
  }

  // ==================== RPC Client Access ====================

  /**
   * Get RPC client for advanced usage
   */
  getRPCClient(): PAWRPCClient {
    return this.rpcClient;
  }
}

/**
 * Create PAW wallet instance
 */
export function createWallet(config: WalletConfig): PAWWallet {
  return new PAWWallet(config);
}
