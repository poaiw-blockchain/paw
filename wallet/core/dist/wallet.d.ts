/**
 * Main Wallet class for PAW blockchain
 * Combines all functionalities: key management, transactions, and RPC
 */
import { PAWRPCClient, RPCConfig } from './rpc';
import { WalletAccount, SecureKeystore, Balance, Validator, Pool, TransactionOptions, BroadcastResult } from './types';
export interface WalletConfig {
    rpcConfig: RPCConfig;
    hdPath?: string;
    prefix?: string;
}
export declare class PAWWallet {
    private rpcClient;
    private hdPath;
    private prefix;
    private privateKey?;
    private publicKey?;
    private address?;
    private mnemonic?;
    constructor(config: WalletConfig);
    /**
     * Create new wallet from mnemonic
     */
    createFromMnemonic(mnemonic: string, password?: string): Promise<WalletAccount>;
    /**
     * Generate new wallet
     */
    generate(strength?: 128 | 160 | 192 | 224 | 256): Promise<{
        mnemonic: string;
        account: WalletAccount;
    }>;
    /**
     * Import wallet from private key
     */
    importPrivateKey(privateKey: Uint8Array): Promise<WalletAccount>;
    /**
     * Import wallet from keystore
     */
    importKeystore(keystore: SecureKeystore | string, password: string): Promise<WalletAccount>;
    /**
     * Export wallet to keystore
     */
    exportKeystore(password: string, name?: string): Promise<SecureKeystore>;
    /**
     * Get mnemonic (if available)
     */
    getMnemonic(): string | undefined;
    /**
     * Get wallet address
     */
    getAddress(): string;
    /**
     * Get public key
     */
    getPublicKey(): Uint8Array;
    /**
     * Create HD wallet for multiple accounts
     */
    deriveAccount(accountIndex: number): Promise<WalletAccount>;
    /**
     * Get balance
     */
    getBalance(denom?: string): Promise<Balance[]>;
    /**
     * Get account info
     */
    getAccountInfo(): Promise<{
        accountNumber: number;
        sequence: number;
    }>;
    /**
     * Send tokens
     */
    send(toAddress: string, amount: string, denom: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Delegate tokens
     */
    delegate(validatorAddress: string, amount: string, denom: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Undelegate tokens
     */
    undelegate(validatorAddress: string, amount: string, denom: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Redelegate tokens
     */
    redelegate(srcValidatorAddress: string, dstValidatorAddress: string, amount: string, denom: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Withdraw staking rewards
     */
    withdrawRewards(validatorAddress: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Vote on governance proposal
     */
    vote(proposalId: string, option: 1 | 2 | 3 | 4, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Swap tokens
     */
    swap(poolId: number, tokenIn: string, tokenOut: string, amountIn: string, minAmountOut: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Create liquidity pool
     */
    createPool(tokenA: string, tokenB: string, amountA: string, amountB: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Add liquidity to pool
     */
    addLiquidity(poolId: number, amountA: string, amountB: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Remove liquidity from pool
     */
    removeLiquidity(poolId: number, shares: string, options?: TransactionOptions): Promise<BroadcastResult>;
    /**
     * Get validators
     */
    getValidators(status?: 'BOND_STATUS_BONDED' | 'BOND_STATUS_UNBONDED' | 'BOND_STATUS_UNBONDING'): Promise<Validator[]>;
    /**
     * Get delegations
     */
    getDelegations(): Promise<import("./types").Delegation[]>;
    /**
     * Get rewards
     */
    getRewards(): Promise<{
        total: Balance[];
        rewards: any[];
    }>;
    /**
     * Get pools
     */
    getPools(): Promise<Pool[]>;
    /**
     * Get pool
     */
    getPool(poolId: string): Promise<Pool>;
    /**
     * Simulate swap
     */
    simulateSwap(poolId: string, tokenIn: string, tokenOut: string, amountIn: string): Promise<{
        amountOut: string;
    }>;
    /**
     * Get transactions
     */
    getTransactions(page?: number, limit?: number): Promise<import("./types").Transaction[]>;
    /**
     * Sign and broadcast transaction
     */
    private signAndBroadcast;
    /**
     * Get RPC client for advanced usage
     */
    getRPCClient(): PAWRPCClient;
}
/**
 * Create PAW wallet instance
 */
export declare function createWallet(config: WalletConfig): PAWWallet;
//# sourceMappingURL=wallet.d.ts.map