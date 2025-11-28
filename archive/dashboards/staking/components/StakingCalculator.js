// Staking Calculator Component

import { formatAmount, formatPercent } from '../utils/ui.js';

export class StakingCalculator {
    constructor(api) {
        this.api = api;
        this.amount = 1000;
        this.apy = 12.8;
        this.days = 365;
        this.compound = false;
    }

    async render() {
        const container = document.getElementById('calculator-content');
        if (!container) return;

        container.innerHTML = `
            <div class="calculator-panel">
                <h3>Input Parameters</h3>
                <form id="calculator-form">
                    <div class="form-group">
                        <label for="calc-amount">Staking Amount (PAW)</label>
                        <input
                            type="number"
                            id="calc-amount"
                            value="${this.amount}"
                            min="1"
                            step="1"
                            required
                        >
                    </div>

                    <div class="form-group">
                        <label for="calc-apy">Expected APY (%)</label>
                        <input
                            type="number"
                            id="calc-apy"
                            value="${this.apy}"
                            min="0"
                            max="100"
                            step="0.1"
                            required
                        >
                        <small>Current network average: ${this.apy}%</small>
                    </div>

                    <div class="form-group">
                        <label for="calc-period">Time Period</label>
                        <select id="calc-period">
                            <option value="7">1 Week</option>
                            <option value="30">1 Month</option>
                            <option value="90">3 Months</option>
                            <option value="180">6 Months</option>
                            <option value="365" selected>1 Year</option>
                            <option value="730">2 Years</option>
                            <option value="1095">3 Years</option>
                            <option value="custom">Custom Days</option>
                        </select>
                    </div>

                    <div class="form-group" id="custom-days-group" style="display: none;">
                        <label for="calc-custom-days">Custom Days</label>
                        <input
                            type="number"
                            id="calc-custom-days"
                            value="365"
                            min="1"
                            step="1"
                        >
                    </div>

                    <div class="form-group">
                        <label class="checkbox-label">
                            <input type="checkbox" id="calc-compound">
                            Auto-compound rewards (daily)
                        </label>
                    </div>

                    <button type="submit" class="btn btn-primary" style="width: 100%;">
                        <i class="fas fa-calculator"></i> Calculate Rewards
                    </button>
                </form>
            </div>

            <div class="results-panel" id="calculator-results">
                ${this.renderResults()}
            </div>
        `;

        this.setupFormHandlers();
        this.calculate();
    }

    setupFormHandlers() {
        const form = document.getElementById('calculator-form');
        const periodSelect = document.getElementById('calc-period');
        const customDaysGroup = document.getElementById('custom-days-group');

        periodSelect.addEventListener('change', (e) => {
            if (e.target.value === 'custom') {
                customDaysGroup.style.display = 'block';
            } else {
                customDaysGroup.style.display = 'none';
            }
        });

        form.addEventListener('submit', (e) => {
            e.preventDefault();
            this.calculate();
        });

        // Real-time calculation on input change
        form.querySelectorAll('input, select').forEach(input => {
            input.addEventListener('change', () => this.calculate());
        });
    }

    calculate() {
        // Get form values
        const amount = parseFloat(document.getElementById('calc-amount').value) || 0;
        const apy = parseFloat(document.getElementById('calc-apy').value) || 0;
        const periodSelect = document.getElementById('calc-period');
        const days = periodSelect.value === 'custom'
            ? parseFloat(document.getElementById('calc-custom-days').value)
            : parseFloat(periodSelect.value);
        const compound = document.getElementById('calc-compound')?.checked || false;

        this.amount = amount;
        this.apy = apy;
        this.days = days;
        this.compound = compound;

        let results;
        if (compound) {
            results = this.calculateCompoundRewards(amount, apy, days);
        } else {
            results = this.api.calculateRewards(amount, apy, days);
        }

        // Update results
        const resultsContainer = document.getElementById('calculator-results');
        if (resultsContainer) {
            resultsContainer.innerHTML = this.renderResults(results);
        }
    }

    calculateCompoundRewards(principal, annualRate, days) {
        const dailyRate = annualRate / 365 / 100;
        const finalAmount = principal * Math.pow(1 + dailyRate, days);
        const totalReward = finalAmount - principal;

        return {
            daily: principal * dailyRate,
            weekly: principal * Math.pow(1 + dailyRate, 7) - principal,
            monthly: principal * Math.pow(1 + dailyRate, 30) - principal,
            yearly: principal * Math.pow(1 + dailyRate, 365) - principal,
            total: totalReward,
            finalAmount: finalAmount
        };
    }

    renderResults(results = null) {
        if (!results) {
            results = this.api.calculateRewards(this.amount, this.apy, this.days);
        }

        const finalAmount = this.amount + results.total;
        const effectiveAPY = (results.total / this.amount) * (365 / this.days) * 100;

        return `
            <h3>Estimated Rewards</h3>

            <div class="result-item">
                <div class="result-label">Total Rewards (${this.days} days)</div>
                <div class="result-value">${formatAmount(results.total * 1e6)} PAW</div>
                <div class="result-sub">Initial: ${formatAmount(this.amount * 1e6)} PAW</div>
            </div>

            <div class="result-item">
                <div class="result-label">Final Amount</div>
                <div class="result-value">${formatAmount(finalAmount * 1e6)} PAW</div>
                <div class="result-sub">+${formatPercent((results.total / this.amount) * 100)} gain</div>
            </div>

            <div class="result-item">
                <div class="result-label">Daily Rewards</div>
                <div class="result-value">${formatAmount(results.daily * 1e6)} PAW</div>
                <div class="result-sub">${formatPercent(this.apy / 365)} daily rate</div>
            </div>

            <div class="result-item">
                <div class="result-label">Weekly Rewards</div>
                <div class="result-value">${formatAmount(results.weekly * 1e6)} PAW</div>
            </div>

            <div class="result-item">
                <div class="result-label">Monthly Rewards</div>
                <div class="result-value">${formatAmount(results.monthly * 1e6)} PAW</div>
            </div>

            <div class="result-item">
                <div class="result-label">Yearly Rewards</div>
                <div class="result-value">${formatAmount(results.yearly * 1e6)} PAW</div>
                <div class="result-sub">Effective APY: ${formatPercent(effectiveAPY)}</div>
            </div>

            ${this.compound ? `
                <div class="result-item">
                    <div class="result-label">Compounding Benefit</div>
                    <div class="result-value">+${formatAmount((results.total - this.api.calculateRewards(this.amount, this.apy, this.days).total) * 1e6)} PAW</div>
                    <div class="result-sub">vs. non-compounding</div>
                </div>
            ` : ''}

            <div style="margin-top: 2rem; padding-top: 1rem; border-top: 1px solid rgba(255,255,255,0.2); font-size: 0.875rem; opacity: 0.9;">
                <i class="fas fa-info-circle"></i> These are estimates based on current APY.
                Actual rewards may vary based on network conditions and validator performance.
            </div>
        `;
    }
}

export default StakingCalculator;
