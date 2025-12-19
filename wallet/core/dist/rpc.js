"use strict";
/**
 * RPC client for PAW blockchain
 * Handles REST API and WebSocket connections
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.PAWRPCClient = void 0;
exports.createRPCClient = createRPCClient;
const axios_1 = __importDefault(require("axios"));
const transaction_1 = require("./transaction");
class PAWRPCClient {
    constructor(config) {
        this.restUrl = config.restUrl;
        this.rpcUrl = config.rpcUrl;
        this.wsUrl = config.wsUrl;
        this.eventHandlers = new Map();
        const axiosConfig = {
            baseURL: config.restUrl,
            timeout: config.timeout || 30000,
            headers: {
                'Content-Type': 'application/json',
                ...config.headers,
            },
        };
        this.restClient = axios_1.default.create(axiosConfig);
        // Add response interceptor for error handling
        this.restClient.interceptors.response.use((response) => response, (error) => {
            if (error.response) {
                throw new Error(`API Error: ${error.response.status} - ${JSON.stringify(error.response.data)}`);
            }
            else if (error.request) {
                throw new Error('Network Error: No response received');
            }
            else {
                throw new Error(`Request Error: ${error.message}`);
            }
        });
    }
    // ==================== Node Information ====================
    /**
     * Get node information
     */
    async getNodeInfo() {
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
    async getLatestBlock() {
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
    async getBlockByHeight(height) {
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
    async getBalance(address, denom) {
        if (denom) {
            const response = await this.restClient.get(`/cosmos/bank/v1beta1/balances/${address}/by_denom?denom=${denom}`);
            return response.data.balance ? [response.data.balance] : [];
        }
        else {
            const response = await this.restClient.get(`/cosmos/bank/v1beta1/balances/${address}`);
            return response.data.balances || [];
        }
    }
    /**
     * Get account information
     */
    async getAccount(address) {
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
    async getTotalSupply(denom) {
        if (denom) {
            const response = await this.restClient.get(`/cosmos/bank/v1beta1/supply/by_denom?denom=${denom}`);
            return [response.data.amount];
        }
        else {
            const response = await this.restClient.get('/cosmos/bank/v1beta1/supply');
            return response.data.supply || [];
        }
    }
    // ==================== Staking ====================
    /**
     * Get all validators
     */
    async getValidators(status) {
        const params = status ? `?status=${status}` : '';
        const response = await this.restClient.get(`/cosmos/staking/v1beta1/validators${params}`);
        return response.data.validators || [];
    }
    /**
     * Get validator by address
     */
    async getValidator(validatorAddress) {
        const response = await this.restClient.get(`/cosmos/staking/v1beta1/validators/${validatorAddress}`);
        return response.data.validator;
    }
    /**
     * Get delegations for an address
     */
    async getDelegations(delegatorAddress) {
        const response = await this.restClient.get(`/cosmos/staking/v1beta1/delegations/${delegatorAddress}`);
        return response.data.delegation_responses || [];
    }
    /**
     * Get delegation to specific validator
     */
    async getDelegation(delegatorAddress, validatorAddress) {
        try {
            const response = await this.restClient.get(`/cosmos/staking/v1beta1/validators/${validatorAddress}/delegations/${delegatorAddress}`);
            return response.data.delegation_response;
        }
        catch (error) {
            if (error.response?.status === 404) {
                return null;
            }
            throw error;
        }
    }
    /**
     * Get staking rewards
     */
    async getRewards(delegatorAddress) {
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
    async getProposals(status) {
        const params = status ? `?proposal_status=${status}` : '';
        const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals${params}`);
        return response.data.proposals || [];
    }
    /**
     * Get proposal by ID
     */
    async getProposal(proposalId) {
        const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals/${proposalId}`);
        return response.data.proposal;
    }
    /**
     * Get votes for a proposal
     */
    async getProposalVotes(proposalId) {
        const response = await this.restClient.get(`/cosmos/gov/v1beta1/proposals/${proposalId}/votes`);
        return response.data.votes || [];
    }
    // ==================== DEX Module ====================
    /**
     * Get all pools
     */
    async getPools() {
        const response = await this.restClient.get('/paw/dex/v1/pools');
        return response.data.pools || [];
    }
    /**
     * Get pool by ID
     */
    async getPool(poolId) {
        const response = await this.restClient.get(`/paw/dex/v1/pools/${poolId}`);
        return response.data.pool;
    }
    /**
     * Get pool by token pair
     */
    async getPoolByTokens(tokenA, tokenB) {
        const response = await this.restClient.get(`/paw/dex/v1/pools/by-tokens/${tokenA}/${tokenB}`);
        return response.data.pool;
    }
    /**
     * Simulate swap
     */
    async simulateSwap(poolId, tokenIn, tokenOut, amountIn) {
        const response = await this.restClient.get(`/paw/dex/v1/simulate-swap/${poolId}?token_in=${tokenIn}&token_out=${tokenOut}&amount_in=${amountIn}`);
        return { amountOut: response.data.amount_out };
    }
    /**
     * Get liquidity for provider
     */
    async getLiquidity(poolId, provider) {
        const response = await this.restClient.get(`/paw/dex/v1/liquidity/${poolId}/${provider}`);
        return { shares: response.data.shares };
    }
    // ==================== Oracle Module ====================
    /**
     * Get price data for asset
     */
    async getPrice(asset) {
        const response = await this.restClient.get(`/paw/oracle/v1/prices/${asset}`);
        return response.data.price_data;
    }
    /**
     * Get all active prices
     */
    async getAllPrices() {
        const response = await this.restClient.get('/paw/oracle/v1/prices');
        return response.data.prices || [];
    }
    /**
     * Get oracle parameters
     */
    async getOracleParams() {
        const response = await this.restClient.get('/paw/oracle/v1/params');
        return response.data.params;
    }
    // ==================== Transaction Broadcasting ====================
    /**
     * Broadcast transaction
     */
    async broadcastTx(signedTx, mode = 'sync') {
        const txBytes = (0, transaction_1.encodeTxBase64)(signedTx);
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
    async getTransaction(hash) {
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
    async searchTransactions(query, page = 1, limit = 10) {
        const response = await this.restClient.get(`/cosmos/tx/v1beta1/txs?events=${encodeURIComponent(query)}&page=${page}&limit=${limit}`);
        return (response.data.tx_responses || []).map((tx) => ({
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
    async getTransactionsByAddress(address, page = 1, limit = 10) {
        const query = `message.sender='${address}'`;
        return this.searchTransactions(query, page, limit);
    }
    // ==================== WebSocket Subscriptions ====================
    /**
     * Connect to WebSocket
     */
    connectWebSocket() {
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
            }
            catch (error) {
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
    disconnectWebSocket() {
        if (this.wsConnection) {
            this.wsConnection.close();
            this.wsConnection = undefined;
        }
    }
    /**
     * Subscribe to new blocks
     */
    subscribeToBlocks(callback) {
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
    subscribeToTransactions(callback) {
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
    addEventListener(event, callback) {
        if (!this.eventHandlers.has(event)) {
            this.eventHandlers.set(event, new Set());
        }
        this.eventHandlers.get(event).add(callback);
    }
    /**
     * Remove event listener
     */
    removeEventListener(event, callback) {
        const handlers = this.eventHandlers.get(event);
        if (handlers) {
            handlers.delete(callback);
        }
    }
    /**
     * Handle WebSocket message
     */
    handleWebSocketMessage(data) {
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
    async healthCheck() {
        try {
            await this.getNodeInfo();
            return true;
        }
        catch {
            return false;
        }
    }
    /**
     * Get chain info
     */
    async getChainInfo() {
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
    async estimateFee(gasLimit) {
        const gasPrice = 0.025; // Default gas price
        const amount = Math.ceil(gasLimit * gasPrice);
        return {
            amount: amount.toString(),
            denom: 'upaw',
        };
    }
}
exports.PAWRPCClient = PAWRPCClient;
/**
 * Create PAW RPC client
 */
function createRPCClient(config) {
    return new PAWRPCClient(config);
}
//# sourceMappingURL=rpc.js.map