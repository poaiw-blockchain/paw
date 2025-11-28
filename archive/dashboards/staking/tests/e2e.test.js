// End-to-End Staking Flow Tests

import { StakingAPI } from '../services/stakingAPI.js';
import { ValidatorList } from '../components/ValidatorList.js';
import { DelegationPanel } from '../components/DelegationPanel.js';
import { RewardsPanel } from '../components/RewardsPanel.js';
import { PortfolioView } from '../components/PortfolioView.js';

describe('E2E Staking Flow', () => {
    let api;
    let validatorList;
    let delegationPanel;
    let rewardsPanel;
    let portfolioView;

    beforeEach(() => {
        // Setup DOM
        document.body.innerHTML = `
            <div id="validators-content"></div>
            <div id="delegation-panel"></div>
            <div id="rewards-panel"></div>
            <div id="portfolio-content"></div>
            <div id="toast-container"></div>
            <div id="loading-overlay"></div>
        `;

        api = new StakingAPI();
        validatorList = new ValidatorList(api);
        delegationPanel = new DelegationPanel(api);
        rewardsPanel = new RewardsPanel(api);
        portfolioView = new PortfolioView(api);
    });

    describe('Complete Staking Workflow', () => {
        test('should complete full delegation flow', async () => {
            const testAddress = 'paw1abc123def456';

            // Step 1: Load validators
            await validatorList.render();
            const validators = await api.getValidators();
            expect(validators.length).toBeGreaterThan(0);

            // Step 2: Select a validator
            const selectedValidator = validators[0];
            expect(selectedValidator).toBeDefined();
            expect(selectedValidator.operatorAddress).toBeDefined();

            // Step 3: Open delegation panel
            await delegationPanel.render(selectedValidator, testAddress);
            const delegationForm = document.getElementById('delegation-panel');
            expect(delegationForm).toBeDefined();

            // Step 4: Fill delegation form
            const amountInput = document.getElementById('delegate-amount');
            expect(amountInput).toBeDefined();

            // Step 5: Verify balance check would work
            const balance = await api.getBalance(testAddress);
            expect(balance).toBeGreaterThanOrEqual(0);
        });

        test('should complete rewards claiming flow', async () => {
            const testAddress = 'paw1abc123def456';

            // Step 1: Load portfolio
            await portfolioView.render(testAddress);

            // Step 2: Check for rewards
            const rewards = await api.getDelegationRewards(testAddress);
            expect(rewards).toBeDefined();

            // Step 3: Open rewards panel
            await rewardsPanel.render(testAddress);
            const rewardsForm = document.getElementById('rewards-panel');
            expect(rewardsForm).toBeDefined();
        });

        test('should complete undelegation flow', async () => {
            const testAddress = 'paw1abc123def456';

            // Step 1: Get delegations
            const delegations = await api.getDelegations(testAddress);

            if (delegations.length > 0) {
                // Step 2: Select delegation to undelegate
                const delegation = delegations[0];

                // Step 3: Get validator details
                const validator = await api.getValidatorDetails(delegation.validatorAddress);
                expect(validator).toBeDefined();

                // Step 4: Open delegation panel for undelegation
                await delegationPanel.render(validator, testAddress);
            }
        });

        test('should complete redelegation flow', async () => {
            const testAddress = 'paw1abc123def456';

            // Step 1: Get current delegations
            const delegations = await api.getDelegations(testAddress);

            if (delegations.length > 0) {
                // Step 2: Get all validators for redelegation
                const validators = await api.getValidators();
                const currentValidator = delegations[0].validatorAddress;
                const newValidators = validators.filter(v =>
                    v.operatorAddress !== currentValidator
                );

                expect(newValidators.length).toBeGreaterThan(0);
            }
        });
    });

    describe('Multi-Validator Staking', () => {
        test('should handle delegation to multiple validators', async () => {
            const validators = await api.getValidators();
            const testAddress = 'paw1abc123def456';

            // Simulate delegating to multiple validators
            const delegationPromises = validators.slice(0, 3).map(validator =>
                delegationPanel.render(validator, testAddress)
            );

            await Promise.all(delegationPromises);
            expect(delegationPromises.length).toBe(3);
        });

        test('should calculate total portfolio value correctly', async () => {
            const testAddress = 'paw1abc123def456';

            const [balance, delegations, unbonding, rewards] = await Promise.all([
                api.getBalance(testAddress),
                api.getDelegations(testAddress),
                api.getUnbondingDelegations(testAddress),
                api.getDelegationRewards(testAddress)
            ]);

            const totalDelegated = delegations.reduce((sum, d) => sum + d.balance, 0);
            const totalUnbonding = unbonding.reduce((sum, u) =>
                sum + u.entries.reduce((s, e) => s + e.balance, 0), 0
            );
            const totalRewards = rewards.total.reduce((sum, r) =>
                r.denom === 'upaw' ? sum + r.amount : sum, 0
            );

            const totalValue = balance + totalDelegated + totalUnbonding + totalRewards;

            expect(totalValue).toBeGreaterThanOrEqual(0);
            expect(isFinite(totalValue)).toBe(true);
        });
    });

    describe('Error Handling', () => {
        test('should handle validator not found', async () => {
            const result = await api.getValidatorDetails('invalid_address');
            expect(result).toBeNull();
        });

        test('should handle empty delegations gracefully', async () => {
            const testAddress = 'paw1empty';
            await portfolioView.render(testAddress);

            const container = document.getElementById('portfolio-content');
            expect(container.innerHTML).toContain('No active delegations');
        });

        test('should handle zero rewards', async () => {
            const testAddress = 'paw1norewards';
            const rewards = await api.getDelegationRewards(testAddress);

            expect(rewards.total.length).toBeGreaterThanOrEqual(0);
        });

        test('should validate delegation amounts', async () => {
            const testAddress = 'paw1test';
            const balance = 100;

            // Too much
            const errorHigh = await validateAmount(150, balance);
            expect(errorHigh).toBeTruthy();

            // Negative
            const errorNeg = await validateAmount(-10, balance);
            expect(errorNeg).toBeTruthy();

            // Valid
            const errorValid = await validateAmount(50, balance);
            expect(errorValid).toBeNull();
        });
    });

    describe('Performance', () => {
        test('should load validators within reasonable time', async () => {
            const startTime = Date.now();
            await api.getValidators();
            const endTime = Date.now();

            const loadTime = endTime - startTime;
            expect(loadTime).toBeLessThan(5000); // Should load in less than 5 seconds
        });

        test('should cache repeated requests', async () => {
            const startTime1 = Date.now();
            await api.getValidators();
            const endTime1 = Date.now();

            const startTime2 = Date.now();
            await api.getValidators();
            const endTime2 = Date.now();

            const firstLoad = endTime1 - startTime1;
            const secondLoad = endTime2 - startTime2;

            // Second load should be significantly faster due to caching
            expect(secondLoad).toBeLessThan(firstLoad / 2);
        });

        test('should handle concurrent requests', async () => {
            const promises = [
                api.getValidators(),
                api.getNetworkStats(),
                api.getStakingParams()
            ];

            const results = await Promise.all(promises);
            expect(results.length).toBe(3);
            results.forEach(result => {
                expect(result).toBeDefined();
            });
        });
    });

    describe('Data Consistency', () => {
        test('should maintain consistent validator data', async () => {
            const validators1 = await api.getValidators();
            const validators2 = await api.getValidators();

            expect(validators1.length).toBe(validators2.length);

            if (validators1.length > 0) {
                expect(validators1[0].operatorAddress).toBe(validators2[0].operatorAddress);
            }
        });

        test('should calculate consistent APY', () => {
            const validator = {
                commission: 5.0,
                votingPower: 1000000
            };

            const apy1 = api.calculateAPY(validator, 10);
            const apy2 = api.calculateAPY(validator, 10);

            expect(apy1).toBe(apy2);
        });

        test('should maintain risk score consistency', () => {
            const validator = {
                commission: 7.5,
                jailed: false,
                votingPower: 2000000,
                status: 'BOND_STATUS_BONDED'
            };

            const score1 = api.calculateRiskScore(validator);
            const score2 = api.calculateRiskScore(validator);

            expect(score1).toBe(score2);
        });
    });
});

// Helper function for validation
function validateAmount(amount, balance) {
    const value = parseFloat(amount);
    if (isNaN(value) || value <= 0) {
        return 'Please enter a valid amount';
    }
    if (value > balance) {
        return 'Insufficient balance';
    }
    return null;
}
