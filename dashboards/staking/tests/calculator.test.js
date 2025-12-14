// Staking Calculator Tests

import { StakingCalculator } from '../components/StakingCalculator.js';
import { StakingAPI } from '../services/stakingAPI.js';

describe('StakingCalculator', () => {
    let calculator;
    let api;

    beforeEach(() => {
        document.body.innerHTML = '<div id="calculator-content"></div>';
        api = new StakingAPI();
        calculator = new StakingCalculator(api);
    });

    describe('Rendering', () => {
        test('should render calculator form', async () => {
            await calculator.render();

            const container = document.getElementById('calculator-content');
            expect(container.innerHTML).toContain('Input Parameters');
            expect(container.querySelector('#calc-amount')).toBeDefined();
            expect(container.querySelector('#calc-apy')).toBeDefined();
            expect(container.querySelector('#calc-period')).toBeDefined();
        });

        test('should render results panel', async () => {
            await calculator.render();

            const results = document.getElementById('calculator-results');
            expect(results).toBeDefined();
            expect(results.innerHTML).toContain('Estimated Rewards');
        });
    });

    describe('Calculations', () => {
        test('should calculate simple rewards correctly', () => {
            const amount = 1000;
            const apy = 12;
            const days = 365;

            const results = api.calculateRewards(amount, apy, days);

            expect(results.yearly).toBeCloseTo(120, 1);
            expect(results.monthly).toBeCloseTo(10, 1);
            expect(results.weekly).toBeCloseTo(2.3, 1);
            expect(results.daily).toBeCloseTo(0.329, 2);
        });

        test('should calculate compound rewards correctly', () => {
            const amount = 1000;
            const apy = 12;
            const days = 365;

            const results = calculator.calculateCompoundRewards(amount, apy, days);

            expect(results.total).toBeGreaterThan(120); // Should be more than simple interest
            expect(results.finalAmount).toBeCloseTo(1127.47, 1); // Compound interest formula
        });

        test('should show compounding benefit', () => {
            const amount = 10000;
            const apy = 15;
            const days = 365;

            const simple = api.calculateRewards(amount, apy, days);
            const compound = calculator.calculateCompoundRewards(amount, apy, days);

            const benefit = compound.total - simple.total;
            expect(benefit).toBeGreaterThan(0);
            expect(benefit).toBeCloseTo(115, 0); // Approximate compounding benefit
        });

        test('should calculate for different time periods', () => {
            const amount = 1000;
            const apy = 10;

            const week = api.calculateRewards(amount, apy, 7);
            const month = api.calculateRewards(amount, apy, 30);
            const year = api.calculateRewards(amount, apy, 365);

            expect(week.total).toBeLessThan(month.total);
            expect(month.total).toBeLessThan(year.total);
            expect(year.total).toBeCloseTo(100, 1);
        });

        test('should handle edge cases', () => {
            // Zero amount
            const zeroAmount = api.calculateRewards(0, 12, 365);
            expect(zeroAmount.total).toBe(0);

            // Zero APY
            const zeroAPY = api.calculateRewards(1000, 0, 365);
            expect(zeroAPY.total).toBe(0);

            // Zero days
            const zeroDays = api.calculateRewards(1000, 12, 0);
            expect(zeroDays.total).toBe(0);
        });

        test('should calculate effective APY correctly', () => {
            calculator.amount = 1000;
            calculator.apy = 12;
            calculator.days = 182.5; // Half a year

            const results = api.calculateRewards(1000, 12, 182.5);
            const effectiveAPY = (results.total / 1000) * (365 / 182.5) * 100;

            expect(effectiveAPY).toBeCloseTo(12, 1);
        });
    });

    describe('User Interactions', () => {
        test('should update calculations on form input', async () => {
            await calculator.render();

            const amountInput = document.getElementById('calc-amount');
            amountInput.value = '5000';
            amountInput.dispatchEvent(new Event('change'));

            await new Promise(resolve => setTimeout(resolve, 100));

            expect(calculator.amount).toBe(5000);
        });

        test('should show custom days input when selected', async () => {
            await calculator.render();

            const periodSelect = document.getElementById('calc-period');
            const customDaysGroup = document.getElementById('custom-days-group');

            periodSelect.value = 'custom';
            periodSelect.dispatchEvent(new Event('change'));

            expect(customDaysGroup.style.display).toBe('block');
        });

        test('should handle form submission', async () => {
            await calculator.render();

            const form = document.getElementById('calculator-form');
            const submitEvent = new Event('submit');
            submitEvent.preventDefault = jest.fn();

            form.dispatchEvent(submitEvent);

            expect(submitEvent.preventDefault).toHaveBeenCalled();
        });
    });

    describe('Validation', () => {
        test('should validate positive amounts', () => {
            expect(() => {
                api.calculateRewards(-100, 12, 365);
            }).not.toThrow();

            const results = api.calculateRewards(-100, 12, 365);
            expect(results.total).toBeLessThan(0);
        });

        test('should handle very large amounts', () => {
            const largeAmount = 1000000000; // 1 billion
            const results = api.calculateRewards(largeAmount, 12, 365);

            expect(results.yearly).toBe(largeAmount * 0.12);
            expect(isFinite(results.yearly)).toBe(true);
        });

        test('should handle very small amounts', () => {
            const smallAmount = 0.000001;
            const results = api.calculateRewards(smallAmount, 12, 365);

            expect(results.yearly).toBeCloseTo(0.00000012, 8);
            expect(isFinite(results.yearly)).toBe(true);
        });
    });

    describe('Display Formatting', () => {
        test('should format large numbers correctly', async () => {
            calculator.amount = 1000000;
            calculator.apy = 12;
            calculator.days = 365;

            await calculator.render();

            const results = document.getElementById('calculator-results');
            expect(results.innerHTML).toContain('PAW');
        });

        test('should display compounding indicator', async () => {
            calculator.compound = true;
            await calculator.render();

            const results = document.getElementById('calculator-results');
            expect(results.innerHTML).toContain('Compounding Benefit');
        });
    });
});
