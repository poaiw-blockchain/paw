/**
 * RPC client for PAW blockchain
 * Handles REST API and WebSocket connections
 */

import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import {
  Balance,
  Validator,
  Delegation,
  Pool,
  PriceData,
  BroadcastResult,
  Transaction,
  NodeInfo,
  BlockInfo,
  SignedTransaction,
  ChainInfo,
} from './types';
import { encodeTxBase64 } from './transaction';

export interface RPCConfig {
  restUrl: string;
  rpcUrl?: string;
  wsUrl?: string;
  timeout?: number;
  headers?: Record<string, string>;
}

export class PAWRPCClient {
  private restClient: AxiosInstance;
  private restUrl: string;
  private rpcUrl?: string;
  private wsUrl?: string;
  private wsConnection?: WebSocket;
  private eventHandlers: Map<string, Set<(data: any) => void>>;

  constructor(config: RPCConfig) {
    this.restUrl = config.restUrl;
    this.rpcUrl = config.rpcUrl;
    this.wsUrl = config.wsUrl;
    this.eventHandlers = new Map();

    const axiosConfig: AxiosRequestConfig = {
      baseURL: config.restUrl,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
        ...config.headers,
      },
    };

    this.restClient = axios.create(axiosConfig);

    // Add response interceptor for error handling
    this.restClient.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response) {
          throw new Error(`API Error: ${error.response.status} - ${JSON.stringify(error.response.data)}`);
        } else if (error.request) {
          throw new Error('Network Error: No response received');
        } else {
          throw new Error(`Request Error: ${error.message}`);
        }
      }
    );
  }

  // ==================== Node Information ====================

  /**
   * Get node information
   */
  async getNodeInfo(): Promise<NodeInfo> {
    const response = await this.restClient.get('/cosmos/base/tendermint/v1beta1/node_info');
    const data = response.data;
    return {
      nodeId: data.default_node_info.default_node_id,
      listenAddr: data.default_node_info.listen_addr,
      network: data.default_node_info.network,
      version: data.default_node_info.version,
      channels: data.default_node_info.channels,
      moniker: data.default_node_info.moniker,
      other: data.default_node_info.other,
    };
  }

  /**
   * Get latest block information
   */
  async getLatestBlock(): Promise<BlockInfo> {
    const response = await this.restClient.get('/cosmos/base/tendermint/v1beta1/blocks/latest');
    const block = response.data.block;
    return {
      chainId: block.header.chain_id,
      height: parseInt(block.header.height),
      time: block.header.time,
      lastBlockId: block.header.last_block_id,
      proposerAddress: block.header.proposer_address,
    };
  }

  /**
   * Get block by height
   */
  async getBlockByHeight(height: number): Promise<BlockInfo> {
    const response = await this.restClient.get(`/cosmos/base/tendermint/v1beta1/blocks/${height}`);
    const block = response.data.block;
    return {
      chainId: block.header.chain_id,
      height: parseInt(block.header.height),
      time: block.header.time,
      lastBlockId: block.header.last_block_id,
      proposerAddress: block.header.proposer_address,
    };
  }

  // ==================== Account & Balance ====================

  /**
   * Get account balances
   */
  async getBalance(address: string, denom?: string): Promise<Balance[]> {
    if (denom) {
      const response = await this.restClient.get(`/cosmos/bank/v1beta1/balances/${address}/by_denom?denom=${denom}`);
      return response.data.balance ? [response.data.balance] : [];
    } else {
      const response = await this.restClient.get(`/cosmos/bank/v1beta1/balances/${address}`);
      return response.data.balances || [];
    }
  }

  /**
   * Get account information
   */
  async getAccount(address: string): Promise<{ accountNumber: number; sequence: number }> {
    const response = await this.restClient.get(`/cosmos/auth/v1beta1/accounts/${address}`);
    const account = response.data.account;

    return {
      accountNumber: parseInt(account.account_number),
      sequence: parseInt(account.sequence),
    };
  }

  /**
   * Get total supply
   */
  async getTotalSupply(denom?: string): Promise<Balance[]> {
    if (denom) {
      const response = await this.restClient.get(`/cosmos/bank/v1beta1/supply/by_denom?denom=${denom}`);
      return [response.data.amount];
    } else {
      const response = await this.restClient.get('/cosmos/bank/v1beta1/supply');
      return response.data.supply || [];
    }
  }

  // ==================== Staking ====================

  /**
   * Get all validators
   */
  async getValidators(status?: 'BOND_STATUS_BONDED' | 'BOND_STATUS_UNBONDED' | 'BOND_STATUS_UNBONDING'): Promise<Validator[]> {
    const params = status ? `?status=${status}` : '';
    const response = await this.restClient.get(`/cosmos/staking/v1beta1/validators${params}`);
    return response.data.validators || [];
  }

  /**
   * Get validator by address
   */
  async getValidator(validatorAddress: string): Promise<Validator> {
    const response = await this.restClient.get(`/cosmos/staking/v1beta1/validators/${validatorAddress}`);
    return response.data.validator;
  }

  /**
   * Get delegations for an address
   */
  async getDelegations(delegatorAddress: string): Promise<Delegation[]> {
    const response = await this.restClient.get(`/cosmos/staking/v1beta1/delegations/${delegatorAddress}`);
    return response.data.delegation_responses || [];
  }

  /**
   * Get delegation to specific validator
   */
  async getDelegation(delegatorAddress: string, validatorAddress: string): Promise<Delegation | null> {
    try {
      const response = await this.restClient.get(
        `/cosmos/staking/v1beta1/validators/${validatorAddress}/delegations/${delegatorAddress}`
      );
      return response.data.delegation_response;
    } catch (error: any) {
      if (error.response?.status === 404) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Get staking rewards
   */
  async getRewards(delegatorAddress: string): Promise<{ total: Balance[]; rewards: any[] }> {
    const response = await this.restClient.get(`/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards`);
    return {
      total: response.data.total || [],
      rewards: response.data.rewards || [],
    };
  }

  // ==================== Governance ====================

  /**
   * Get governance proposals
   */
  async getProposals(status?: string): Promise<any[]> {
    const params = status ? `?proposal_status=${status}` : '';
    const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals${params}`);
    return response.data.proposals || [];
  }

  /**
   * Get proposal by ID
   */
  async getProposal(proposalId: string): Promise<any> {
    const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals/${proposalId}`);
    return response.data.proposal;
  }

  /**
   * Get votes for a proposal
   */
  async getProposalVotes(proposalId: string): Promise<any[]> {
    const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals/${proposalId}/votes`);
    return response.data.votes || [];
  }

  // ==================== DEX Module ====================

  /**
   * Get all pools
   */
  async getPools(): Promise<Pool[]> {
    const response = await this.restClient.get('/paw/dex/v1/pools');
    return response.data.pools || [];
  }

  /**
   * Get pool by ID
   */
  async getPool(poolId: string): Promise<Pool> {
    const response = await this.restClient.get(`/paw/dex/v1/pools/${poolId}`);
    return response.data.pool;
  }

  /**
   * Get pool by token pair
   */
  async getPoolByTokens(tokenA: string, tokenB: string): Promise<Pool> {
    const response = await this.restClient.get(`/paw/dex/v1/pools/by-tokens/${tokenA}/${tokenB}`);
    return response.data.pool;
  }

  /**
   * Simulate swap
   */
  async simulateSwap(poolId: string, tokenIn: string, tokenOut: string, amountIn: string): Promise<{ amountOut: string }> {
    const response = await this.restClient.get(
      `/paw/dex/v1/simulate-swap/${poolId}?token_in=${tokenIn}&token_out=${tokenOut}&amount_in=${amountIn}`
    );
    return { amountOut: response.data.amount_out };
  }

  /**
   * Get liquidity for provider
   */
  async getLiquidity(poolId: string, provider: string): Promise<{ shares: string }> {
    const response = await this.restClient.get(`/paw/dex/v1/liquidity/${poolId}/${provider}`);
    return { shares: response.data.shares };
  }

  // ==================== Oracle Module ====================

  /**
   * Get price data for asset
   */
  async getPrice(asset: string): Promise<PriceData> {
    const response = await this.restClient.get(`/paw/oracle/v1/prices/${asset}`);
    return response.data.price_data;
  }

  /**
   * Get all active prices
   */
  async getAllPrices(): Promise<PriceData[]> {
    const response = await this.restClient.get('/paw/oracle/v1/prices');
    return response.data.prices || [];
  }

  /**
   * Get oracle parameters
   */
  async getOracleParams(): Promise<any> {
    const response = await this.restClient.get('/paw/oracle/v1/params');
    return response.data.params;
  }

  // ==================== Transaction Broadcasting ====================

  /**
   * Broadcast transaction
   */
  async broadcastTx(signedTx: SignedTransaction, mode: 'sync' | 'async' | 'block' = 'sync'): Promise<BroadcastResult> {
    const txBytes = encodeTxBase64(signedTx);

    const response = await this.restClient.post('/cosmos/tx/v1beta1/txs', {
      tx_bytes: txBytes,
      mode: `BROADCAST_MODE_${mode.toUpperCase()}`,
    });

    const txResponse = response.data.tx_response;

    return {
      code: txResponse.code,
      transactionHash: txResponse.txhash,
      rawLog: txResponse.raw_log,
      height: txResponse.height ? parseInt(txResponse.height) : undefined,
      gasUsed: txResponse.gas_used ? parseInt(txResponse.gas_used) : undefined,
      gasWanted: txResponse.gas_wanted ? parseInt(txResponse.gas_wanted) : undefined,
    };
  }

  /**
   * Get transaction by hash
   */
  async getTransaction(hash: string): Promise<Transaction> {
    const response = await this.restClient.get(`/cosmos/tx/v1beta1/txs/${hash}`);
    const tx = response.data.tx_response;

    return {
      hash: tx.txhash,
      height: parseInt(tx.height),
      timestamp: tx.timestamp,
      success: tx.code === 0,
      memo: tx.tx?.body?.memo,
      fee: {
        amount: tx.tx?.auth_info?.fee?.amount || [],
        gas: tx.tx?.auth_info?.fee?.gas_limit || '0',
      },
      messages: tx.tx?.body?.messages || [],
    };
  }

  /**
   * Search transactions
   */
  async searchTransactions(query: string, page: number = 1, limit: number = 10): Promise<Transaction[]> {
    const response = await this.restClient.get(
      `/cosmos/tx/v1beta1/txs?events=${encodeURIComponent(query)}&page=${page}&limit=${limit}`
    );

    return (response.data.tx_responses || []).map((tx: any) => ({
      hash: tx.txhash,
      height: parseInt(tx.height),
      timestamp: tx.timestamp,
      success: tx.code === 0,
      memo: tx.tx?.body?.memo,
      fee: {
        amount: tx.tx?.auth_info?.fee?.amount || [],
        gas: tx.tx?.auth_info?.fee?.gas_limit || '0',
      },
      messages: tx.tx?.body?.messages || [],
    }));
  }

  /**
   * Get transactions by address
   */
  async getTransactionsByAddress(address: string, page: number = 1, limit: number = 10): Promise<Transaction[]> {
    const query = `message.sender='${address}'`;
    return this.searchTransactions(query, page, limit);
  }

  // ==================== WebSocket Subscriptions ====================

  /**
   * Connect to WebSocket
   */
  connectWebSocket(): void {
    if (!this.wsUrl) {
      throw new Error('WebSocket URL not configured');
    }

    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      return; // Already connected
    }

    this.wsConnection = new WebSocket(this.wsUrl);

    this.wsConnection.onopen = () => {
      console.log('WebSocket connected');
    };

    this.wsConnection.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.handleWebSocketMessage(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    this.wsConnection.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.wsConnection.onclose = () => {
      console.log('WebSocket disconnected');
      // Auto-reconnect after 5 seconds
      setTimeout(() => this.connectWebSocket(), 5000);
    };
  }

  /**
   * Disconnect WebSocket
   */
  disconnectWebSocket(): void {
    if (this.wsConnection) {
      this.wsConnection.close();
      this.wsConnection = undefined;
    }
  }

  /**
   * Subscribe to new blocks
   */
  subscribeToBlocks(callback: (block: any) => void): () => void {
    this.addEventListener('NewBlock', callback);

    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      this.wsConnection.send(JSON.stringify({
        jsonrpc: '2.0',
        method: 'subscribe',
        id: 'blocks',
        params: {
          query: "tm.event='NewBlock'",
        },
      }));
    }

    return () => this.removeEventListener('NewBlock', callback);
  }

  /**
   * Subscribe to transactions
   */
  subscribeToTransactions(callback: (tx: any) => void): () => void {
    this.addEventListener('Tx', callback);

    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      this.wsConnection.send(JSON.stringify({
        jsonrpc: '2.0',
        method: 'subscribe',
        id: 'txs',
        params: {
          query: "tm.event='Tx'",
        },
      }));
    }

    return () => this.removeEventListener('Tx', callback);
  }

  /**
   * Add event listener
   */
  private addEventListener(event: string, callback: (data: any) => void): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(callback);
  }

  /**
   * Remove event listener
   */
  private removeEventListener(event: string, callback: (data: any) => void): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.delete(callback);
    }
  }

  /**
   * Handle WebSocket message
   */
  private handleWebSocketMessage(data: any): void {
    if (data.result?.events) {
      const eventType = data.result.events['tm.event']?.[0];
      if (eventType) {
        const handlers = this.eventHandlers.get(eventType);
        if (handlers) {
          handlers.forEach((callback) => callback(data.result.data));
        }
      }
    }
  }

  // ==================== Utilities ====================

  /**
   * Check if node is healthy
   */
  async healthCheck(): Promise<boolean> {
    try {
      await this.getNodeInfo();
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Get chain info
   */
  async getChainInfo(): Promise<ChainInfo> {
    const nodeInfo = await this.getNodeInfo();

    return {
      chainId: nodeInfo.network,
      chainName: 'PAW',
      rpc: this.rpcUrl || '',
      rest: this.restUrl,
      bech32Prefix: 'paw',
      coinType: 118,
      stakeCurrency: {
        coinDenom: 'PAW',
        coinMinimalDenom: 'upaw',
        coinDecimals: 6,
      },
      feeCurrencies: [
        {
          coinDenom: 'PAW',
          coinMinimalDenom: 'upaw',
          coinDecimals: 6,
          gasPriceStep: {
            low: 0.01,
            average: 0.025,
            high: 0.04,
          },
        },
      ],
    };
  }

  /**
   * Estimate transaction fee
   */
  async estimateFee(gasLimit: number): Promise<{ amount: string; denom: string }> {
    const gasPrice = 0.025; // Default gas price
    const amount = Math.ceil(gasLimit * gasPrice);

    return {
      amount: amount.toString(),
      denom: 'upaw',
    };
  }
}

/**
 * Create PAW RPC client
 */
export function createRPCClient(config: RPCConfig): PAWRPCClient {
  return new PAWRPCClient(config);
}
