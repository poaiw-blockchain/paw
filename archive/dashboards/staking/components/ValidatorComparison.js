// Validator Comparison Component

import { formatAmount, formatPercent } from '../utils/ui.js';

export class ValidatorComparison {
    constructor(api) {
        this.api = api;
        this.validators = [];
        this.selectedValidators = [];
        this.maxComparisons = 4;
    }

    async render() {
        const container = document.getElementById('comparison-content');
        if (!container) return;

        try {
            this.validators = await this.api.getValidators();

            container.innerHTML = `
                <div class="comparison-selector">
                    <h3>Select Validators to Compare (up to ${this.maxComparisons})</h3>
                    <div class="form-group">
                        <select id="validator-selector" class="select-input">
                            <option value="">-- Select a validator --</option>
                            ${this.validators.map(v => `
                                <option value="${v.operatorAddress}">
                                    ${v.moniker} - ${formatPercent(v.commission)} commission
                                </option>
                            `).join('')}
                        </select>
                        <button id="add-validator-btn" class="btn btn-primary" style="margin-left: 1rem;">
                            <i class="fas fa-plus"></i> Add
                        </button>
                        <button id="clear-comparison-btn" class="btn btn-secondary" style="margin-left: 0.5rem;">
                            <i class="fas fa-trash"></i> Clear All
                        </button>
                    </div>
                </div>

                <div id="comparison-results" class="comparison-grid">
                    ${this.renderComparisonCards()}
                </div>
            `;

            this.setupEventListeners();
        } catch (error) {
            console.error('Error loading validators:', error);
            container.innerHTML = '<div class="text-center text-danger">Failed to load validators</div>';
        }
    }

    setupEventListeners() {
        const addBtn = document.getElementById('add-validator-btn');
        const clearBtn = document.getElementById('clear-comparison-btn');
        const selector = document.getElementById('validator-selector');

        addBtn.addEventListener('click', () => {
            const validatorAddress = selector.value;
            if (validatorAddress) {
                this.addValidator(validatorAddress);
            }
        });

        clearBtn.addEventListener('click', () => {
            this.selectedValidators = [];
            this.updateComparisonCards();
        });
    }

    addValidator(validatorAddress) {
        if (this.selectedValidators.length >= this.maxComparisons) {
            alert(`You can only compare up to ${this.maxComparisons} validators at once`);
            return;
        }

        if (this.selectedValidators.find(v => v.operatorAddress === validatorAddress)) {
            alert('This validator is already selected');
            return;
        }

        const validator = this.validators.find(v => v.operatorAddress === validatorAddress);
        if (validator) {
            this.selectedValidators.push(validator);
            this.updateComparisonCards();
        }
    }

    removeValidator(validatorAddress) {
        this.selectedValidators = this.selectedValidators.filter(
            v => v.operatorAddress !== validatorAddress
        );
        this.updateComparisonCards();
    }

    updateComparisonCards() {
        const container = document.getElementById('comparison-results');
        if (container) {
            container.innerHTML = this.renderComparisonCards();
        }
    }

    renderComparisonCards() {
        if (this.selectedValidators.length === 0) {
            return '<div class="text-center">Select validators to compare</div>';
        }

        return this.selectedValidators.map(v => this.renderValidatorCard(v)).join('');
    }

    renderValidatorCard(validator) {
        const apy = this.api.calculateAPY(validator, 7.5);
        const riskScore = this.api.calculateRiskScore(validator);
        const riskLevel = this.api.getRiskLevel(riskScore);
        const uptime = this.calculateUptime(validator);

        return `
            <div class="comparison-card selected">
                <div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 1rem;">
                    <div>
                        <h4>${validator.moniker}</h4>
                        <div style="font-size: 0.875rem; color: var(--text-secondary);">
                            ${validator.operatorAddress.slice(0, 20)}...
                        </div>
                    </div>
                    <button
                        class="btn btn-sm btn-secondary"
                        onclick="window.stakingDashboard.components.validatorComparison.removeValidator('${validator.operatorAddress}')"
                    >
                        <i class="fas fa-times"></i>
                    </button>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Status</span>
                    <span class="metric-value">
                        <span class="status-badge ${validator.jailed ? 'status-jailed' : 'status-active'}">
                            ${validator.jailed ? 'Jailed' : 'Active'}
                        </span>
                    </span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Voting Power</span>
                    <span class="metric-value">${formatAmount(validator.votingPower * 1e6)} PAW</span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Commission</span>
                    <span class="metric-value">${formatPercent(validator.commission)}</span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Max Commission</span>
                    <span class="metric-value">${formatPercent(validator.maxCommission)}</span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Max Change Rate</span>
                    <span class="metric-value">${formatPercent(validator.maxCommissionChange)}</span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Estimated APY</span>
                    <span class="metric-value" style="color: var(--success-color); font-weight: bold;">
                        ${formatPercent(apy)}
                    </span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Uptime</span>
                    <span class="metric-value">${formatPercent(uptime)}</span>
                </div>

                <div class="metric-row">
                    <span class="metric-label">Risk Score</span>
                    <span class="metric-value">
                        <div class="risk-indicator risk-${riskLevel}">
                            <i class="fas fa-circle"></i> ${riskScore}/100 (${riskLevel.toUpperCase()})
                        </div>
                    </span>
                </div>

                ${validator.website ? `
                    <div class="metric-row">
                        <span class="metric-label">Website</span>
                        <span class="metric-value">
                            <a href="${validator.website}" target="_blank" rel="noopener">
                                <i class="fas fa-external-link-alt"></i>
                            </a>
                        </span>
                    </div>
                ` : ''}

                <div style="margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--border-color);">
                    <button
                        class="btn btn-primary"
                        style="width: 100%;"
                        onclick="window.stakingDashboard.showDelegationModal(${JSON.stringify(validator).replace(/"/g, '&quot;')})"
                    >
                        <i class="fas fa-hand-holding-usd"></i> Delegate
                    </button>
                </div>
            </div>
        `;
    }

    calculateUptime(validator) {
        // In a real implementation, this would fetch actual uptime data
        // For now, return a simulated value based on risk factors
        const riskScore = this.api.calculateRiskScore(validator);
        const baseUptime = 99.5;
        const uptimeVariation = (riskScore - 80) * 0.1;
        return Math.max(95, Math.min(100, baseUptime + uptimeVariation));
    }

    getBestValidator() {
        if (this.selectedValidators.length === 0) return null;

        return this.selectedValidators.reduce((best, current) => {
            const bestScore = this.api.calculateRiskScore(best);
            const currentScore = this.api.calculateRiskScore(current);

            const bestAPY = this.api.calculateAPY(best, 7.5);
            const currentAPY = this.api.calculateAPY(current, 7.5);

            // Weight: 60% risk score, 40% APY
            const bestOverall = bestScore * 0.6 + bestAPY * 0.4;
            const currentOverall = currentScore * 0.6 + currentAPY * 0.4;

            return currentOverall > bestOverall ? current : best;
        });
    }
}

export default ValidatorComparison;
