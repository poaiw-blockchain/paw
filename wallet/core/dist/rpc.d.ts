/**
 * RPC client for PAW blockchain
 * Handles REST API and WebSocket connections
 */
import { Balance, Validator, Delegation, Pool, PriceData, BroadcastResult, Transaction, NodeInfo, BlockInfo, SignedTransaction, ChainInfo } from './types';
export interface RPCConfig {
    restUrl: string;
    rpcUrl?: string;
    wsUrl?: string;
    timeout?: number;
    headers?: Record<string, string>;
}
export declare class PAWRPCClient {
    private restClient;
    private restUrl;
    private rpcUrl?;
    private wsUrl?;
    private wsConnection?;
    private eventHandlers;
    constructor(config: RPCConfig);
    /**
     * Get node information
     */
    getNodeInfo(): Promise<NodeInfo>;
    /**
     * Get latest block information
     */
    getLatestBlock(): Promise<BlockInfo>;
    /**
     * Get block by height
     */
    getBlockByHeight(height: number): Promise<BlockInfo>;
    /**
     * Get account balances
     */
    getBalance(address: string, denom?: string): Promise<Balance[]>;
    /**
     * Get account information
     */
    getAccount(address: string): Promise<{
        accountNumber: number;
        sequence: number;
    }>;
    /**
     * Get total supply
     */
    getTotalSupply(denom?: string): Promise<Balance[]>;
    /**
     * Get all validators
     */
    getValidators(status?: 'BOND_STATUS_BONDED' | 'BOND_STATUS_UNBONDED' | 'BOND_STATUS_UNBONDING'): Promise<Validator[]>;
    /**
     * Get validator by address
     */
    getValidator(validatorAddress: string): Promise<Validator>;
    /**
     * Get delegations for an address
     */
    getDelegations(delegatorAddress: string): Promise<Delegation[]>;
    /**
     * Get delegation to specific validator
     */
    getDelegation(delegatorAddress: string, validatorAddress: string): Promise<Delegation | null>;
    /**
     * Get staking rewards
     */
    getRewards(delegatorAddress: string): Promise<{
        total: Balance[];
        rewards: any[];
    }>;
    /**
     * Get governance proposals
     */
    getProposals(status?: string): Promise<any[]>;
    /**
     * Get proposal by ID
     */
    getProposal(proposalId: string): Promise<any>;
    /**
     * Get votes for a proposal
     */
    getProposalVotes(proposalId: string): Promise<any[]>;
    /**
     * Get all pools
     */
    getPools(): Promise<Pool[]>;
    /**
     * Get pool by ID
     */
    getPool(poolId: string): Promise<Pool>;
    /**
     * Get pool by token pair
     */
    getPoolByTokens(tokenA: string, tokenB: string): Promise<Pool>;
    /**
     * Simulate swap
     */
    simulateSwap(poolId: string, tokenIn: string, tokenOut: string, amountIn: string): Promise<{
        amountOut: string;
    }>;
    /**
     * Get liquidity for provider
     */
    getLiquidity(poolId: string, provider: string): Promise<{
        shares: string;
    }>;
    /**
     * Get price data for asset
     */
    getPrice(asset: string): Promise<PriceData>;
    /**
     * Get all active prices
     */
    getAllPrices(): Promise<PriceData[]>;
    /**
     * Get oracle parameters
     */
    getOracleParams(): Promise<any>;
    /**
     * Broadcast transaction
     */
    broadcastTx(signedTx: SignedTransaction, mode?: 'sync' | 'async' | 'block'): Promise<BroadcastResult>;
    /**
     * Get transaction by hash
     */
    getTransaction(hash: string): Promise<Transaction>;
    /**
     * Search transactions
     */
    searchTransactions(query: string, page?: number, limit?: number): Promise<Transaction[]>;
    /**
     * Get transactions by address
     */
    getTransactionsByAddress(address: string, page?: number, limit?: number): Promise<Transaction[]>;
    /**
     * Connect to WebSocket
     */
    connectWebSocket(): void;
    /**
     * Disconnect WebSocket
     */
    disconnectWebSocket(): void;
    /**
     * Subscribe to new blocks
     */
    subscribeToBlocks(callback: (block: any) => void): () => void;
    /**
     * Subscribe to transactions
     */
    subscribeToTransactions(callback: (tx: any) => void): () => void;
    /**
     * Add event listener
     */
    private addEventListener;
    /**
     * Remove event listener
     */
    private removeEventListener;
    /**
     * Handle WebSocket message
     */
    private handleWebSocketMessage;
    /**
     * Check if node is healthy
     */
    healthCheck(): Promise<boolean>;
    /**
     * Get chain info
     */
    getChainInfo(): Promise<ChainInfo>;
    /**
     * Estimate transaction fee
     */
    estimateFee(gasLimit: number): Promise<{
        amount: string;
        denom: string;
    }>;
}
/**
 * Create PAW RPC client
 */
export declare function createRPCClient(config: RPCConfig): PAWRPCClient;
//# sourceMappingURL=rpc.d.ts.map