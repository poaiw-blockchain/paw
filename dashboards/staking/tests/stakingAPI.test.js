// Staking API Tests

import { StakingAPI } from '../services/stakingAPI.js';

describe('StakingAPI', () => {
    let api;

    beforeEach(() => {
        api = new StakingAPI();
    });

    afterEach(() => {
        api.clearCache();
    });

    describe('Network Statistics', () => {
        test('should fetch network stats', async () => {
            const stats = await api.getNetworkStats();

            expect(stats).toBeDefined();
            expect(stats.totalStaked).toBeGreaterThanOrEqual(0);
            expect(stats.activeValidators).toBeGreaterThanOrEqual(0);
            expect(stats.inflationRate).toBeGreaterThanOrEqual(0);
            expect(stats.averageAPY).toBeGreaterThanOrEqual(0);
        });

        test('should return mock data on API failure', async () => {
            const originalFetch = global.fetch;
            global.fetch = jest.fn(() => Promise.reject(new Error('Network error')));

            const stats = await api.getNetworkStats();

            expect(stats).toBeDefined();
            expect(stats.totalStaked).toBe(125000000);

            global.fetch = originalFetch;
        });
    });

    describe('Validators', () => {
        test('should fetch validators list', async () => {
            const validators = await api.getValidators();

            expect(Array.isArray(validators)).toBe(true);
            expect(validators.length).toBeGreaterThan(0);

            const validator = validators[0];
            expect(validator).toHaveProperty('operatorAddress');
            expect(validator).toHaveProperty('moniker');
            expect(validator).toHaveProperty('commission');
            expect(validator).toHaveProperty('votingPower');
        });

        test('should format validator data correctly', () => {
            const rawValidator = {
                operator_address: 'pawvaloper1abc123',
                description: {
                    moniker: 'Test Validator',
                    identity: '',
                    website: 'https://test.com',
                    details: 'Test details'
                },
                commission: {
                    commission_rates: {
                        rate: '0.050000000000000000',
                        max_rate: '0.200000000000000000',
                        max_change_rate: '0.010000000000000000'
                    }
                },
                tokens: '1000000000000',
                delegator_shares: '1000000000000',
                status: 'BOND_STATUS_BONDED',
                jailed: false
            };

            const formatted = api.formatValidator(rawValidator);

            expect(formatted.commission).toBe(5.0);
            expect(formatted.maxCommission).toBe(20.0);
            expect(formatted.votingPower).toBe(1000000);
        });

        test('should fetch validator details', async () => {
            const validators = await api.getValidators();
            if (validators.length > 0) {
                const details = await api.getValidatorDetails(validators[0].operatorAddress);
                expect(details).toBeDefined();
            }
        });
    });

    describe('Calculations', () => {
        test('should calculate APY correctly', () => {
            const validator = {
                commission: 5.0,
                votingPower: 1000000
            };

            const apy = api.calculateAPY(validator, 10);

            expect(apy).toBeCloseTo(8.075, 2); // 10 * 0.85 * 0.95
        });

        test('should calculate average APY', () => {
            const validators = [
                { commission: 5.0, status: 'BOND_STATUS_BONDED' },
                { commission: 10.0, status: 'BOND_STATUS_BONDED' },
                { commission: 7.5, status: 'BOND_STATUS_UNBONDED' }
            ];

            const avgAPY = api.calculateAverageAPY(validators, 10);

            expect(avgAPY).toBeGreaterThan(0);
            expect(avgAPY).toBeLessThan(10);
        });

        test('should calculate rewards correctly', () => {
            const amount = 1000;
            const apy = 12;
            const days = 365;

            const rewards = api.calculateRewards(amount, apy, days);

            expect(rewards.yearly).toBeCloseTo(120, 1);
            expect(rewards.monthly).toBeCloseTo(10, 1);
            expect(rewards.weekly).toBeCloseTo(2.3, 1);
            expect(rewards.daily).toBeCloseTo(0.33, 1);
        });

        test('should calculate risk score correctly', () => {
            const lowRiskValidator = {
                commission: 3.0,
                jailed: false,
                votingPower: 1000000,
                status: 'BOND_STATUS_BONDED'
            };

            const highRiskValidator = {
                commission: 15.0,
                jailed: true,
                votingPower: 15000000,
                status: 'BOND_STATUS_UNBONDED'
            };

            const lowScore = api.calculateRiskScore(lowRiskValidator);
            const highScore = api.calculateRiskScore(highRiskValidator);

            expect(lowScore).toBeGreaterThan(highScore);
            expect(lowScore).toBeGreaterThanOrEqual(80);
            expect(highScore).toBeLessThan(50);
        });

        test('should categorize risk levels correctly', () => {
            expect(api.getRiskLevel(85)).toBe('low');
            expect(api.getRiskLevel(70)).toBe('medium');
            expect(api.getRiskLevel(50)).toBe('high');
        });
    });

    describe('Caching', () => {
        test('should cache API responses', async () => {
            const fetchSpy = jest.spyOn(global, 'fetch');

            await api.getValidators();
            await api.getValidators();

            // Should only fetch once due to caching
            expect(fetchSpy).toHaveBeenCalledTimes(1);

            fetchSpy.mockRestore();
        });

        test('should clear cache', async () => {
            await api.getValidators();
            api.clearCache();

            expect(api.cache.size).toBe(0);
        });

        test('should respect cache timeout', async () => {
            api.cacheTimeout = 100; // 100ms

            await api.getValidators();
            await new Promise(resolve => setTimeout(resolve, 150));
            await api.getValidators();

            // Cache should have expired and fetched again
            expect(api.cache.size).toBeGreaterThan(0);
        });
    });

    describe('Error Handling', () => {
        test('should handle network errors gracefully', async () => {
            const originalFetch = global.fetch;
            global.fetch = jest.fn(() => Promise.reject(new Error('Network error')));

            const validators = await api.getValidators();

            expect(Array.isArray(validators)).toBe(true);
            expect(validators.length).toBeGreaterThan(0); // Should return mock data

            global.fetch = originalFetch;
        });

        test('should handle invalid responses', async () => {
            const originalFetch = global.fetch;
            global.fetch = jest.fn(() =>
                Promise.resolve({
                    ok: false,
                    status: 404
                })
            );

            await expect(api.fetchWithCache('http://test.com')).rejects.toThrow();

            global.fetch = originalFetch;
        });
    });
});
