// Staking API Service

export class StakingAPI {
    constructor() {
        this.baseURL = window.PAW_API_URL || 'http://localhost:1317';
        this.rpcURL = window.PAW_RPC_URL || 'http://localhost:26657';
        this.cache = new Map();
        this.cacheTimeout = 30000; // 30 seconds
    }

    async fetchWithCache(url, options = {}) {
        const cacheKey = `${url}_${JSON.stringify(options)}`;
        const cached = this.cache.get(cacheKey);

        if (cached && Date.now() - cached.timestamp < this.cacheTimeout) {
            return cached.data;
        }

        const response = await fetch(url, options);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        this.cache.set(cacheKey, { data, timestamp: Date.now() });
        return data;
    }

    // Network Statistics
    async getNetworkStats() {
        try {
            const [pool, params, validators] = await Promise.all([
                this.getStakingPool(),
                this.getStakingParams(),
                this.getValidators()
            ]);

            const totalStaked = parseFloat(pool.pool.bonded_tokens) / 1e6;
            const activeValidators = validators.filter(v => v.status === 'BOND_STATUS_BONDED').length;
            const inflationRate = parseFloat(params.params.inflation_rate_change) * 100;
            const averageAPY = this.calculateAverageAPY(validators, inflationRate);

            return {
                totalStaked,
                activeValidators,
                inflationRate,
                averageAPY
            };
        } catch (error) {
            console.error('Error fetching network stats:', error);
            return this.getMockNetworkStats();
        }
    }

    async getStakingPool() {
        const url = `${this.baseURL}/cosmos/staking/v1beta1/pool`;
        return this.fetchWithCache(url);
    }

    async getStakingParams() {
        const url = `${this.baseURL}/cosmos/staking/v1beta1/params`;
        return this.fetchWithCache(url);
    }

    // Validators
    async getValidators(status = '') {
        try {
            const url = `${this.baseURL}/cosmos/staking/v1beta1/validators?status=${status}`;
            const response = await this.fetchWithCache(url);

            return response.validators.map(v => this.formatValidator(v));
        } catch (error) {
            console.error('Error fetching validators:', error);
            return this.getMockValidators();
        }
    }

    formatValidator(validator) {
        const commission = parseFloat(validator.commission.commission_rates.rate) * 100;
        const votingPower = parseFloat(validator.tokens) / 1e6;

        return {
            operatorAddress: validator.operator_address,
            moniker: validator.description.moniker,
            identity: validator.description.identity,
            website: validator.description.website,
            details: validator.description.details,
            commission: commission,
            maxCommission: parseFloat(validator.commission.commission_rates.max_rate) * 100,
            maxCommissionChange: parseFloat(validator.commission.commission_rates.max_change_rate) * 100,
            votingPower: votingPower,
            status: validator.status,
            jailed: validator.jailed,
            tokens: validator.tokens,
            delegatorShares: validator.delegator_shares
        };
    }

    async getValidatorDetails(validatorAddress) {
        try {
            const url = `${this.baseURL}/cosmos/staking/v1beta1/validators/${validatorAddress}`;
            const response = await this.fetchWithCache(url);
            return this.formatValidator(response.validator);
        } catch (error) {
            console.error('Error fetching validator details:', error);
            return null;
        }
    }

    // Delegations
    async getDelegations(delegatorAddress) {
        try {
            const url = `${this.baseURL}/cosmos/staking/v1beta1/delegations/${delegatorAddress}`;
            const response = await this.fetchWithCache(url);

            return response.delegation_responses.map(d => ({
                validatorAddress: d.delegation.validator_address,
                shares: d.delegation.shares,
                balance: parseFloat(d.balance.amount) / 1e6
            }));
        } catch (error) {
            console.error('Error fetching delegations:', error);
            return [];
        }
    }

    async getUnbondingDelegations(delegatorAddress) {
        try {
            const url = `${this.baseURL}/cosmos/staking/v1beta1/delegators/${delegatorAddress}/unbonding_delegations`;
            const response = await this.fetchWithCache(url);

            return response.unbonding_responses.map(u => ({
                validatorAddress: u.validator_address,
                entries: u.entries.map(e => ({
                    creationHeight: e.creation_height,
                    completionTime: e.completion_time,
                    initialBalance: parseFloat(e.initial_balance) / 1e6,
                    balance: parseFloat(e.balance) / 1e6
                }))
            }));
        } catch (error) {
            console.error('Error fetching unbonding delegations:', error);
            return [];
        }
    }

    // Rewards
    async getDelegationRewards(delegatorAddress, validatorAddress = null) {
        try {
            const url = validatorAddress
                ? `${this.baseURL}/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards/${validatorAddress}`
                : `${this.baseURL}/cosmos/distribution/v1beta1/delegators/${delegatorAddress}/rewards`;

            const response = await this.fetchWithCache(url);

            if (validatorAddress) {
                return {
                    rewards: response.rewards.map(r => ({
                        denom: r.denom,
                        amount: parseFloat(r.amount) / 1e6
                    }))
                };
            }

            return {
                total: response.total.map(r => ({
                    denom: r.denom,
                    amount: parseFloat(r.amount) / 1e6
                })),
                rewards: response.rewards.map(r => ({
                    validatorAddress: r.validator_address,
                    reward: r.reward.map(rw => ({
                        denom: rw.denom,
                        amount: parseFloat(rw.amount) / 1e6
                    }))
                }))
            };
        } catch (error) {
            console.error('Error fetching rewards:', error);
            return { total: [], rewards: [] };
        }
    }

    // Account Balance
    async getBalance(address) {
        try {
            const url = `${this.baseURL}/cosmos/bank/v1beta1/balances/${address}`;
            const response = await this.fetchWithCache(url);

            const pawBalance = response.balances.find(b => b.denom === 'upaw');
            return pawBalance ? parseFloat(pawBalance.amount) / 1e6 : 0;
        } catch (error) {
            console.error('Error fetching balance:', error);
            return 0;
        }
    }

    // Calculations
    calculateAPY(validator, inflationRate) {
        const commission = validator.commission / 100;
        const baseAPY = inflationRate * 0.85; // Assuming 85% of inflation goes to staking rewards
        return baseAPY * (1 - commission);
    }

    calculateAverageAPY(validators, inflationRate) {
        const activeValidators = validators.filter(v => v.status === 'BOND_STATUS_BONDED');
        if (activeValidators.length === 0) return 0;

        const totalAPY = activeValidators.reduce((sum, v) => {
            return sum + this.calculateAPY(v, inflationRate);
        }, 0);

        return totalAPY / activeValidators.length;
    }

    calculateRewards(amount, apy, days) {
        const dailyRate = apy / 365 / 100;
        const totalReward = amount * dailyRate * days;
        return {
            daily: amount * dailyRate,
            weekly: amount * dailyRate * 7,
            monthly: amount * dailyRate * 30,
            yearly: amount * (apy / 100),
            total: totalReward
        };
    }

    calculateRiskScore(validator) {
        let score = 100;

        // Commission risk (higher commission = higher risk)
        if (validator.commission > 10) score -= 20;
        else if (validator.commission > 5) score -= 10;

        // Jailed status
        if (validator.jailed) score -= 30;

        // Voting power concentration (too high = centralization risk)
        if (validator.votingPower > 10000000) score -= 15;
        else if (validator.votingPower > 5000000) score -= 10;

        // Status
        if (validator.status !== 'BOND_STATUS_BONDED') score -= 25;

        return Math.max(0, Math.min(100, score));
    }

    getRiskLevel(score) {
        if (score >= 80) return 'low';
        if (score >= 60) return 'medium';
        return 'high';
    }

    // Mock Data for Testing
    getMockNetworkStats() {
        return {
            totalStaked: 125000000,
            activeValidators: 45,
            inflationRate: 7.5,
            averageAPY: 12.8
        };
    }

    getMockValidators() {
        return [
            {
                operatorAddress: 'pawvaloper1abc123',
                moniker: 'Alpha Validator',
                identity: '',
                website: 'https://alpha.validator',
                details: 'Professional validator service',
                commission: 5.0,
                maxCommission: 20.0,
                maxCommissionChange: 1.0,
                votingPower: 1250000,
                status: 'BOND_STATUS_BONDED',
                jailed: false
            },
            {
                operatorAddress: 'pawvaloper1def456',
                moniker: 'Beta Validator',
                identity: '',
                website: 'https://beta.validator',
                details: 'Trusted validator since genesis',
                commission: 3.0,
                maxCommission: 10.0,
                maxCommissionChange: 0.5,
                votingPower: 2100000,
                status: 'BOND_STATUS_BONDED',
                jailed: false
            },
            {
                operatorAddress: 'pawvaloper1ghi789',
                moniker: 'Gamma Validator',
                identity: '',
                website: 'https://gamma.validator',
                details: 'Secure and reliable',
                commission: 7.5,
                maxCommission: 15.0,
                maxCommissionChange: 1.0,
                votingPower: 850000,
                status: 'BOND_STATUS_BONDED',
                jailed: false
            }
        ];
    }

    clearCache() {
        this.cache.clear();
    }
}

export default StakingAPI;
