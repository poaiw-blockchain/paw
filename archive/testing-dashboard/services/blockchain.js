/**
 * Blockchain Service
 * Handles all blockchain interactions and RPC/REST API calls
 */

import CONFIG from '../config.js';

class BlockchainService {
    constructor() {
        this.currentNetwork = 'local';
        this.config = CONFIG.networks[this.currentNetwork];
        this.isConnected = false;
        this.latestBlock = null;
    }

    /**
     * Switch to a different network
     */
    switchNetwork(network) {
        if (!CONFIG.networks[network]) {
            throw new Error(`Unknown network: ${network}`);
        }
        this.currentNetwork = network;
        this.config = CONFIG.networks[network];
        this.isConnected = false;
        return this.checkConnection();
    }

    /**
     * Check if node is reachable
     */
    async checkConnection() {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/base/tendermint/v1beta1/node_info`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                this.isConnected = true;
                return { connected: true, network: this.currentNetwork };
            }

            this.isConnected = false;
            return { connected: false, error: 'Node not responding' };
        } catch (error) {
            this.isConnected = false;
            return { connected: false, error: error.message };
        }
    }

    /**
     * Get latest block information
     */
    async getLatestBlock() {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/base/tendermint/v1beta1/blocks/latest`);
            const data = await response.json();
            this.latestBlock = data.block;
            return {
                height: data.block.header.height,
                hash: data.block_id.hash,
                time: data.block.header.time,
                proposer: data.block.header.proposer_address,
                txCount: data.block.data.txs ? data.block.data.txs.length : 0
            };
        } catch (error) {
            console.error('Error fetching latest block:', error);
            return null;
        }
    }

    /**
     * Get recent blocks
     */
    async getRecentBlocks(count = 10) {
        try {
            const blocks = [];
            const latest = await this.getLatestBlock();
            if (!latest) return [];

            const startHeight = Math.max(1, parseInt(latest.height) - count + 1);

            for (let i = 0; i < count && startHeight + i <= latest.height; i++) {
                const height = startHeight + i;
                const response = await fetch(`${this.config.restUrl}/cosmos/base/tendermint/v1beta1/blocks/${height}`);
                const data = await response.json();

                blocks.push({
                    height: data.block.header.height,
                    hash: data.block_id.hash.substring(0, 16) + '...',
                    proposer: data.block.header.proposer_address.substring(0, 16) + '...',
                    txCount: data.block.data.txs ? data.block.data.txs.length : 0,
                    time: new Date(data.block.header.time).toLocaleString()
                });
            }

            return blocks.reverse();
        } catch (error) {
            console.error('Error fetching recent blocks:', error);
            return [];
        }
    }

    /**
     * Get recent transactions
     */
    async getRecentTransactions(limit = 10) {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/tx/v1beta1/txs?limit=${limit}`);
            const data = await response.json();

            if (!data.tx_responses) return [];

            return data.tx_responses.map(tx => ({
                hash: tx.txhash.substring(0, 16) + '...',
                type: this.extractTxType(tx),
                height: tx.height,
                status: tx.code === 0 ? 'Success' : 'Failed',
                timestamp: tx.timestamp
            }));
        } catch (error) {
            console.error('Error fetching transactions:', error);
            return [];
        }
    }

    /**
     * Get validators
     */
    async getValidators() {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/staking/v1beta1/validators`);
            const data = await response.json();

            if (!data.validators) return [];

            return data.validators.map(val => ({
                moniker: val.description.moniker,
                address: val.operator_address,
                status: val.status === 'BOND_STATUS_BONDED' ? 'Active' : 'Inactive',
                votingPower: val.tokens,
                commission: (parseFloat(val.commission.commission_rates.rate) * 100).toFixed(2) + '%',
                jailed: val.jailed
            }));
        } catch (error) {
            console.error('Error fetching validators:', error);
            return [];
        }
    }

    /**
     * Get staking info
     */
    async getStakingInfo() {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/staking/v1beta1/pool`);
            const data = await response.json();

            return {
                bondedTokens: data.pool.bonded_tokens,
                notBondedTokens: data.pool.not_bonded_tokens
            };
        } catch (error) {
            console.error('Error fetching staking info:', error);
            return null;
        }
    }

    /**
     * Get governance proposals
     */
    async getProposals() {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/gov/v1beta1/proposals`);
            const data = await response.json();

            if (!data.proposals) return [];

            return data.proposals.map(prop => ({
                id: prop.proposal_id,
                title: prop.content.title || 'Unknown',
                status: prop.status,
                submitTime: new Date(prop.submit_time).toLocaleString(),
                votingEndTime: new Date(prop.voting_end_time).toLocaleString()
            }));
        } catch (error) {
            console.error('Error fetching proposals:', error);
            return [];
        }
    }

    /**
     * Get liquidity pools
     */
    async getLiquidityPools() {
        try {
            const response = await fetch(`${this.config.restUrl}/paw/dex/v1/pools`);
            const data = await response.json();

            if (!data.pools) return [];

            return data.pools.map(pool => ({
                id: pool.id,
                tokenPair: `${pool.token0}/${pool.token1}`,
                liquidity: pool.liquidity || '0',
                volume24h: pool.volume_24h || '0'
            }));
        } catch (error) {
            console.error('Error fetching liquidity pools:', error);
            // Return mock data for testing
            return [
                { id: '1', tokenPair: 'PAW/USDC', liquidity: '1,000,000', volume24h: '50,000' },
                { id: '2', tokenPair: 'PAW/ETH', liquidity: '500,000', volume24h: '25,000' }
            ];
        }
    }

    /**
     * Query balance for an address
     */
    async queryBalance(address) {
        try {
            const response = await fetch(`${this.config.restUrl}/cosmos/bank/v1beta1/balances/${address}`);
            const data = await response.json();

            if (!data.balances) return [];

            return data.balances.map(balance => ({
                denom: balance.denom,
                amount: balance.amount
            }));
        } catch (error) {
            console.error('Error querying balance:', error);
            throw new Error('Failed to query balance: ' + error.message);
        }
    }

    /**
     * Send transaction
     */
    async sendTransaction(txData) {
        if (this.config.features.readOnly) {
            throw new Error('Cannot send transactions on mainnet (read-only mode)');
        }

        try {
            // This is a simplified version - in production, you'd need proper signing
            const tx = {
                body: {
                    messages: [{
                        "@type": "/cosmos.bank.v1beta1.MsgSend",
                        from_address: txData.from,
                        to_address: txData.to,
                        amount: [{
                            denom: txData.denom || 'upaw',
                            amount: txData.amount
                        }]
                    }],
                    memo: txData.memo || ''
                },
                auth_info: {
                    fee: {
                        amount: [{
                            denom: 'upaw',
                            amount: '5000'
                        }],
                        gas_limit: txData.gas || '200000'
                    }
                }
            };

            const response = await fetch(`${this.config.restUrl}/cosmos/tx/v1beta1/txs`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ tx_bytes: btoa(JSON.stringify(tx)), mode: 'BROADCAST_MODE_SYNC' })
            });

            const data = await response.json();
            return {
                success: true,
                txHash: data.tx_response?.txhash,
                height: data.tx_response?.height
            };
        } catch (error) {
            console.error('Error sending transaction:', error);
            throw new Error('Failed to send transaction: ' + error.message);
        }
    }

    /**
     * Create wallet
     */
    async createWallet() {
        // In a real implementation, this would use crypto libraries
        const randomBytes = new Uint8Array(32);
        crypto.getRandomValues(randomBytes);

        const address = 'paw1' + Array.from(randomBytes.slice(0, 20))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('')
            .substring(0, 38);

        return {
            address,
            mnemonic: 'Example mnemonic - in production use proper BIP39 generation',
            privateKey: Array.from(randomBytes).map(b => b.toString(16).padStart(2, '0')).join('')
        };
    }

    /**
     * Extract transaction type from tx data
     */
    extractTxType(tx) {
        if (!tx.tx || !tx.tx.body || !tx.tx.body.messages || !tx.tx.body.messages[0]) {
            return 'Unknown';
        }

        const msgType = tx.tx.body.messages[0]['@type'] || '';
        const parts = msgType.split('.');
        return parts[parts.length - 1] || 'Unknown';
    }

    /**
     * Get network info
     */
    getNetworkInfo() {
        return {
            name: this.config.name,
            chainId: this.config.chainId,
            rpcUrl: this.config.rpcUrl,
            restUrl: this.config.restUrl,
            features: this.config.features
        };
    }

    /**
     * Request tokens from faucet
     */
    async requestFaucet(address) {
        if (!this.config.faucetUrl) {
            throw new Error('Faucet not available on this network');
        }

        try {
            const response = await fetch(`${this.config.faucetUrl}/request`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ address })
            });

            const data = await response.json();
            return {
                success: true,
                txHash: data.txHash,
                amount: data.amount
            };
        } catch (error) {
            console.error('Error requesting from faucet:', error);
            throw new Error('Failed to request tokens from faucet: ' + error.message);
        }
    }
}

export default new BlockchainService();
