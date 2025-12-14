// Unit tests for ValidatorAPI

const ValidatorAPI = require('../../services/validatorAPI');

describe('ValidatorAPI', () => {
    beforeEach(() => {
        // Reset any mocks or state
        global.fetch = jest.fn();
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    describe('request', () => {
        it('should make a successful API request', async () => {
            const mockResponse = { data: 'test' };
            global.fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockResponse
            });

            const result = await ValidatorAPI.request('/test');
            expect(result).toEqual(mockResponse);
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/test'),
                expect.any(Object)
            );
        });

        it('should handle API errors', async () => {
            global.fetch.mockResolvedValueOnce({
                ok: false,
                status: 404,
                statusText: 'Not Found'
            });

            await expect(ValidatorAPI.request('/test')).rejects.toThrow('HTTP 404');
        });

        it('should handle timeout', async () => {
            global.fetch.mockImplementationOnce(() =>
                new Promise(resolve => setTimeout(resolve, 15000))
            );

            await expect(ValidatorAPI.request('/test')).rejects.toThrow('timeout');
        });
    });

    describe('getValidatorInfo', () => {
        it('should fetch and format validator information', async () => {
            const mockValidator = {
                validator: {
                    operator_address: 'pawvaloper1test',
                    description: {
                        moniker: 'Test Validator',
                        website: 'https://test.com',
                        details: 'Test details',
                        identity: ''
                    },
                    tokens: '1000000',
                    delegator_shares: '1000000',
                    commission: {
                        commission_rates: {
                            rate: '0.05',
                            max_rate: '0.20',
                            max_change_rate: '0.01'
                        }
                    },
                    status: 'BOND_STATUS_BONDED',
                    jailed: false,
                    unbonding_height: '0',
                    unbonding_time: null
                }
            };

            global.fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockValidator
            });

            // Mock dependent calls
            jest.spyOn(ValidatorAPI, 'calculateUptime').mockResolvedValue(0.99);
            jest.spyOn(ValidatorAPI, 'getTotalRewards').mockResolvedValue(1000000);
            jest.spyOn(ValidatorAPI, 'getDelegatorCount').mockResolvedValue(100);

            const result = await ValidatorAPI.getValidatorInfo('pawvaloper1test');

            expect(result.address).toBe('pawvaloper1test');
            expect(result.moniker).toBe('Test Validator');
            expect(result.commission.rate).toBe(0.05);
            expect(result.jailed).toBe(false);
        });

        it('should fall back to mock data on error', async () => {
            global.fetch.mockRejectedValueOnce(new Error('Network error'));

            const result = await ValidatorAPI.getValidatorInfo('pawvaloper1test');

            expect(result).toBeDefined();
            expect(result.address).toBe('pawvaloper1test');
        });
    });

    describe('getDelegations', () => {
        it('should fetch and format delegations', async () => {
            const mockDelegations = {
                delegation_responses: [
                    {
                        delegation: {
                            delegator_address: 'paw1delegator1',
                            shares: '1000000'
                        }
                    },
                    {
                        delegation: {
                            delegator_address: 'paw1delegator2',
                            shares: '2000000'
                        }
                    }
                ]
            };

            global.fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockDelegations
            });

            const result = await ValidatorAPI.getDelegations('pawvaloper1test');

            expect(result).toHaveLength(2);
            expect(result[0].delegatorAddress).toBe('paw1delegator1');
            expect(result[1].shares).toBe('2000000');
        });
    });

    describe('getRewards', () => {
        it('should fetch validator rewards', async () => {
            const mockRewards = {
                rewards: {
                    rewards: [{ amount: '100000' }]
                }
            };

            global.fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockRewards
            });

            jest.spyOn(ValidatorAPI, 'getRewardHistory').mockResolvedValue([]);
            jest.spyOn(ValidatorAPI, 'getCommissionEarned').mockResolvedValue(5000);

            const result = await ValidatorAPI.getRewards('pawvaloper1test');

            expect(result.totalDistributed).toBe(100000);
            expect(result.commissionEarned).toBe(5000);
        });
    });

    describe('Helper methods', () => {
        it('should calculate uptime correctly', async () => {
            const mockSigningInfo = {
                val_signing_info: {
                    indexOffset: 1000,
                    missedBlocksCounter: 10
                }
            };

            global.fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockSigningInfo
            });

            const uptime = await ValidatorAPI.calculateUptime('pawvaloper1test');

            expect(uptime).toBeCloseTo(0.99, 2);
        });

        it('should calculate consecutive misses', () => {
            const blocks = [
                { signed: true },
                { signed: false },
                { signed: false },
                { signed: false },
                { signed: true },
                { signed: false },
                { signed: false }
            ];

            const result = ValidatorAPI.calculateConsecutiveMisses(blocks);
            expect(result).toBe(3);
        });

        it('should calculate longest streak', () => {
            const blocks = [
                { signed: true },
                { signed: true },
                { signed: false },
                { signed: true },
                { signed: true },
                { signed: true },
                { signed: true }
            ];

            const result = ValidatorAPI.calculateLongestStreak(blocks);
            expect(result).toBe(4);
        });

        it('should generate uptime alerts for low uptime', () => {
            const alerts = ValidatorAPI.generateUptimeAlerts(93, {
                missedBlocksCounter: 450
            });

            expect(alerts).toHaveLength(2);
            expect(alerts[0].level).toBe('danger');
            expect(alerts[1].level).toBe('warning');
        });
    });

    describe('Mock data generators', () => {
        it('should generate mock validator info', () => {
            const mock = ValidatorAPI.getMockValidatorInfo('pawvaloper1test');

            expect(mock.address).toBe('pawvaloper1test');
            expect(mock.moniker).toBeDefined();
            expect(mock.commission.rate).toBeGreaterThan(0);
        });

        it('should generate mock delegations', () => {
            const mock = ValidatorAPI.getMockDelegations();

            expect(Array.isArray(mock)).toBe(true);
            expect(mock.length).toBeGreaterThan(0);
            expect(mock[0].delegatorAddress).toMatch(/^paw1/);
        });

        it('should generate mock rewards with history', () => {
            const mock = ValidatorAPI.getMockRewards();

            expect(mock.totalDistributed).toBeGreaterThan(0);
            expect(mock.history).toHaveLength(30);
        });
    });
});

// Run tests if this file is executed directly
if (require.main === module) {
    console.log('ValidatorAPI tests would run here with Jest');
}

module.exports = {};
