// PAW Blockchain API Client

export class APIClient {
    constructor(network = 'testnet') {
        this.network = network;
        this.customEndpoint = null;
        this.endpoints = {
            local: 'http://localhost:1317',
            testnet: 'https://testnet-api.paw.zone',
            mainnet: 'https://api.paw.zone'
        };
        this.chainIds = {
            local: 'paw-local',
            testnet: 'paw-testnet-1',
            mainnet: 'paw-1'
        };
    }

    getEndpoint() {
        return this.customEndpoint || this.endpoints[this.network];
    }

    setNetwork(network) {
        if (!this.endpoints[network]) {
            throw new Error(`Invalid network: ${network}`);
        }
        this.network = network;
        this.customEndpoint = null;
    }

    setCustomEndpoint(endpoint) {
        this.customEndpoint = endpoint;
    }

    getChainId() {
        return this.chainIds[this.network] || 'paw-1';
    }

    async request(path, options = {}) {
        const endpoint = this.getEndpoint();
        const url = `${endpoint}${path}`;

        try {
            const response = await fetch(url, {
                ...options,
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                }
            });

            if (!response.ok) {
                const error = await response.text();
                throw new Error(`API request failed: ${response.status} ${error}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API request error:', error);
            throw error;
        }
    }

    // Bank Module
    async getBalance(address, denom = null) {
        const path = denom
            ? `/cosmos/bank/v1beta1/balances/${address}/by_denom?denom=${denom}`
            : `/cosmos/bank/v1beta1/balances/${address}`;
        return await this.request(path);
    }

    async getAllBalances(address) {
        return await this.request(`/cosmos/bank/v1beta1/balances/${address}`);
    }

    async getSupply(denom = null) {
        const path = denom
            ? `/cosmos/bank/v1beta1/supply/by_denom?denom=${denom}`
            : `/cosmos/bank/v1beta1/supply`;
        return await this.request(path);
    }

    async getDenomMetadata(denom) {
        return await this.request(`/cosmos/bank/v1beta1/denoms_metadata/${denom}`);
    }

    // Staking Module
    async getValidators(status = null) {
        const path = status
            ? `/cosmos/staking/v1beta1/validators?status=${status}`
            : `/cosmos/staking/v1beta1/validators`;
        return await this.request(path);
    }

    async getValidator(validatorAddr) {
        return await this.request(`/cosmos/staking/v1beta1/validators/${validatorAddr}`);
    }

    async getDelegations(delegatorAddr) {
        return await this.request(`/cosmos/staking/v1beta1/delegations/${delegatorAddr}`);
    }

    async getValidatorDelegations(validatorAddr) {
        return await this.request(`/cosmos/staking/v1beta1/validators/${validatorAddr}/delegations`);
    }

    async getUnbondingDelegations(delegatorAddr) {
        return await this.request(`/cosmos/staking/v1beta1/delegators/${delegatorAddr}/unbonding_delegations`);
    }

    async getStakingPool() {
        return await this.request('/cosmos/staking/v1beta1/pool');
    }

    async getStakingParams() {
        return await this.request('/cosmos/staking/v1beta1/params');
    }

    // Distribution Module
    async getDelegationRewards(delegatorAddr, validatorAddr = null) {
        const path = validatorAddr
            ? `/cosmos/distribution/v1beta1/delegators/${delegatorAddr}/rewards/${validatorAddr}`
            : `/cosmos/distribution/v1beta1/delegators/${delegatorAddr}/rewards`;
        return await this.request(path);
    }

    async getValidatorCommission(validatorAddr) {
        return await this.request(`/cosmos/distribution/v1beta1/validators/${validatorAddr}/commission`);
    }

    async getValidatorOutstandingRewards(validatorAddr) {
        return await this.request(`/cosmos/distribution/v1beta1/validators/${validatorAddr}/outstanding_rewards`);
    }

    // Governance Module
    async getProposals(status = null) {
        const path = status
            ? `/cosmos/gov/v1beta1/proposals?proposal_status=${status}`
            : `/cosmos/gov/v1beta1/proposals`;
        return await this.request(path);
    }

    async getProposal(proposalId) {
        return await this.request(`/cosmos/gov/v1beta1/proposals/${proposalId}`);
    }

    async getProposalVotes(proposalId) {
        return await this.request(`/cosmos/gov/v1beta1/proposals/${proposalId}/votes`);
    }

    async getProposalTally(proposalId) {
        return await this.request(`/cosmos/gov/v1beta1/proposals/${proposalId}/tally`);
    }

    async getGovParams(paramsType = 'voting') {
        return await this.request(`/cosmos/gov/v1beta1/params/${paramsType}`);
    }

    // DEX Module (Custom)
    async getPools() {
        return await this.request('/paw/dex/v1/pools');
    }

    async getPool(poolId) {
        return await this.request(`/paw/dex/v1/pools/${poolId}`);
    }

    async getPoolLiquidity(poolId) {
        return await this.request(`/paw/dex/v1/pools/${poolId}/liquidity`);
    }

    async estimateSwap(poolId, tokenIn, amountIn) {
        return await this.request(`/paw/dex/v1/pools/${poolId}/estimate_swap`, {
            method: 'POST',
            body: JSON.stringify({
                token_in: tokenIn,
                amount_in: amountIn
            })
        });
    }

    // Transaction queries
    async getTx(hash) {
        return await this.request(`/cosmos/tx/v1beta1/txs/${hash}`);
    }

    async getTxsByEvents(events) {
        const params = new URLSearchParams();
        events.forEach(event => {
            params.append('events', event);
        });
        return await this.request(`/cosmos/tx/v1beta1/txs?${params.toString()}`);
    }

    // Node info
    async getNodeInfo() {
        return await this.request('/cosmos/base/tendermint/v1beta1/node_info');
    }

    async getLatestBlock() {
        return await this.request('/cosmos/base/tendermint/v1beta1/blocks/latest');
    }

    async getBlockByHeight(height) {
        return await this.request(`/cosmos/base/tendermint/v1beta1/blocks/${height}`);
    }

    async getSyncing() {
        return await this.request('/cosmos/base/tendermint/v1beta1/syncing');
    }

    // Generic query method
    async query(path, options = {}) {
        return await this.request(path, options);
    }
}
