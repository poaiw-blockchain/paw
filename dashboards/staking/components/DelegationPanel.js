// Delegation Panel Component

import { formatAmount, validateAmount, showToast, showLoading, hideLoading } from '../utils/ui.js';

export class DelegationPanel {
    constructor(api) {
        this.api = api;
        this.validator = null;
        this.delegatorAddress = null;
        this.actionType = 'delegate'; // delegate, undelegate, redelegate
    }

    async render(validator, delegatorAddress) {
        this.validator = validator;
        this.delegatorAddress = delegatorAddress;

        const container = document.getElementById('delegation-panel');
        if (!container) return;

        const balance = await this.api.getBalance(delegatorAddress);
        const delegations = await this.api.getDelegations(delegatorAddress);
        const currentDelegation = delegations.find(d =>
            d.validatorAddress === validator.operatorAddress
        );

        container.innerHTML = `
            <div class="delegation-info" style="margin-bottom: 1.5rem;">
                <h4>${validator.moniker}</h4>
                <p style="color: var(--text-secondary); font-size: 0.875rem;">
                    ${validator.operatorAddress}
                </p>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 1rem;">
                    <div>
                        <div style="font-size: 0.875rem; color: var(--text-secondary);">Commission</div>
                        <div style="font-weight: 600;">${validator.commission.toFixed(2)}%</div>
                    </div>
                    <div>
                        <div style="font-size: 0.875rem; color: var(--text-secondary);">Your Delegation</div>
                        <div style="font-weight: 600;">
                            ${currentDelegation ? formatAmount(currentDelegation.balance * 1e6) : '0'} PAW
                        </div>
                    </div>
                </div>
            </div>

            <div class="action-selector" style="margin-bottom: 1.5rem;">
                <div style="display: flex; gap: 0.5rem;">
                    <button class="btn btn-primary action-btn active" data-action="delegate">
                        <i class="fas fa-plus"></i> Delegate
                    </button>
                    <button class="btn btn-secondary action-btn" data-action="undelegate" ${!currentDelegation ? 'disabled' : ''}>
                        <i class="fas fa-minus"></i> Undelegate
                    </button>
                    <button class="btn btn-secondary action-btn" data-action="redelegate" ${!currentDelegation ? 'disabled' : ''}>
                        <i class="fas fa-exchange-alt"></i> Redelegate
                    </button>
                </div>
            </div>

            <form id="delegation-form">
                <div id="delegate-form">
                    <div class="form-group">
                        <label for="delegate-amount">Amount to Delegate</label>
                        <input
                            type="number"
                            id="delegate-amount"
                            placeholder="0.00"
                            min="0"
                            step="0.000001"
                            required
                        >
                        <small>Available: ${formatAmount(balance * 1e6)} PAW</small>
                    </div>
                </div>

                <div id="undelegate-form" style="display: none;">
                    <div class="form-group">
                        <label for="undelegate-amount">Amount to Undelegate</label>
                        <input
                            type="number"
                            id="undelegate-amount"
                            placeholder="0.00"
                            min="0"
                            step="0.000001"
                            required
                        >
                        <small>
                            Delegated: ${currentDelegation ? formatAmount(currentDelegation.balance * 1e6) : '0'} PAW
                        </small>
                    </div>
                    <div style="background: var(--bg-tertiary); padding: 1rem; border-radius: var(--radius); margin-top: 1rem;">
                        <i class="fas fa-info-circle"></i>
                        <small>Unbonding period: 21 days. Tokens will not earn rewards during this time.</small>
                    </div>
                </div>

                <div id="redelegate-form" style="display: none;">
                    <div class="form-group">
                        <label for="redelegate-validator">New Validator</label>
                        <select id="redelegate-validator" class="select-input" required>
                            <option value="">-- Select validator --</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="redelegate-amount">Amount to Redelegate</label>
                        <input
                            type="number"
                            id="redelegate-amount"
                            placeholder="0.00"
                            min="0"
                            step="0.000001"
                            required
                        >
                        <small>
                            Delegated: ${currentDelegation ? formatAmount(currentDelegation.balance * 1e6) : '0'} PAW
                        </small>
                    </div>
                </div>

                <div id="estimation" style="background: var(--bg-tertiary); padding: 1rem; border-radius: var(--radius); margin: 1rem 0;">
                    <div style="font-weight: 600; margin-bottom: 0.5rem;">Estimated Annual Rewards</div>
                    <div id="estimated-rewards" style="font-size: 1.25rem; color: var(--success-color); font-weight: bold;">
                        0 PAW
                    </div>
                </div>

                <button type="submit" class="btn btn-primary" style="width: 100%;" id="submit-btn">
                    <i class="fas fa-paper-plane"></i> Submit Transaction
                </button>
            </form>
        `;

        this.setupFormHandlers(balance, currentDelegation);
    }

    setupFormHandlers(balance, currentDelegation) {
        const form = document.getElementById('delegation-form');
        const actionBtns = document.querySelectorAll('.action-btn');

        // Action type switching
        actionBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                actionBtns.forEach(b => {
                    b.classList.remove('btn-primary', 'active');
                    b.classList.add('btn-secondary');
                });
                e.target.classList.remove('btn-secondary');
                e.target.classList.add('btn-primary', 'active');

                this.actionType = e.target.dataset.action;
                this.showActionForm(this.actionType);
            });
        });

        // Amount input estimation
        const delegateAmount = document.getElementById('delegate-amount');
        const undelegateAmount = document.getElementById('undelegate-amount');
        const redelegateAmount = document.getElementById('redelegate-amount');

        [delegateAmount, undelegateAmount, redelegateAmount].forEach(input => {
            if (input) {
                input.addEventListener('input', () => this.updateEstimation());
            }
        });

        // Form submission
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            await this.submitTransaction(balance, currentDelegation);
        });

        // Load validators for redelegation
        this.loadValidatorsForRedelegate();
    }

    async loadValidatorsForRedelegate() {
        const select = document.getElementById('redelegate-validator');
        if (!select) return;

        const validators = await this.api.getValidators();
        const otherValidators = validators.filter(v =>
            v.operatorAddress !== this.validator.operatorAddress &&
            v.status === 'BOND_STATUS_BONDED'
        );

        otherValidators.forEach(v => {
            const option = document.createElement('option');
            option.value = v.operatorAddress;
            option.textContent = `${v.moniker} - ${v.commission.toFixed(2)}% commission`;
            select.appendChild(option);
        });
    }

    showActionForm(action) {
        document.getElementById('delegate-form').style.display =
            action === 'delegate' ? 'block' : 'none';
        document.getElementById('undelegate-form').style.display =
            action === 'undelegate' ? 'block' : 'none';
        document.getElementById('redelegate-form').style.display =
            action === 'redelegate' ? 'block' : 'none';

        const submitBtn = document.getElementById('submit-btn');
        const icons = {
            delegate: 'fa-paper-plane',
            undelegate: 'fa-minus-circle',
            redelegate: 'fa-exchange-alt'
        };
        submitBtn.innerHTML = `<i class="fas ${icons[action]}"></i> ${action === 'delegate' ? 'Delegate' : action === 'undelegate' ? 'Undelegate' : 'Redelegate'}`;

        this.updateEstimation();
    }

    updateEstimation() {
        const estimationEl = document.getElementById('estimated-rewards');
        if (!estimationEl) return;

        let amount = 0;
        if (this.actionType === 'delegate') {
            amount = parseFloat(document.getElementById('delegate-amount')?.value || 0);
        } else if (this.actionType === 'undelegate') {
            amount = -parseFloat(document.getElementById('undelegate-amount')?.value || 0);
        } else if (this.actionType === 'redelegate') {
            amount = parseFloat(document.getElementById('redelegate-amount')?.value || 0);
        }

        const apy = this.api.calculateAPY(this.validator, 7.5);
        const annualRewards = Math.abs(amount) * (apy / 100);

        if (this.actionType === 'undelegate') {
            estimationEl.textContent = `-${formatAmount(annualRewards * 1e6)} PAW`;
            estimationEl.style.color = 'var(--danger-color)';
        } else {
            estimationEl.textContent = `${formatAmount(annualRewards * 1e6)} PAW`;
            estimationEl.style.color = 'var(--success-color)';
        }
    }

    async submitTransaction(balance, currentDelegation) {
        let amount = 0;
        let error = null;

        // Validate based on action type
        if (this.actionType === 'delegate') {
            amount = parseFloat(document.getElementById('delegate-amount').value);
            error = validateAmount(amount, balance);
        } else if (this.actionType === 'undelegate') {
            amount = parseFloat(document.getElementById('undelegate-amount').value);
            const delegated = currentDelegation ? currentDelegation.balance : 0;
            error = validateAmount(amount, delegated);
        } else if (this.actionType === 'redelegate') {
            const newValidator = document.getElementById('redelegate-validator').value;
            if (!newValidator) {
                error = 'Please select a validator';
            } else {
                amount = parseFloat(document.getElementById('redelegate-amount').value);
                const delegated = currentDelegation ? currentDelegation.balance : 0;
                error = validateAmount(amount, delegated);
            }
        }

        if (error) {
            showToast(error, 'error');
            return;
        }

        try {
            showLoading('Processing transaction...');

            // In a real implementation, this would call the actual chain
            await this.simulateTransaction(amount);

            hideLoading();
            showToast(`Successfully ${this.actionType}d ${amount} PAW`, 'success');

            // Close modal
            document.getElementById('delegation-modal').classList.remove('active');

            // Refresh portfolio
            if (window.stakingDashboard?.components?.portfolio) {
                window.stakingDashboard.components.portfolio.refresh();
            }
        } catch (error) {
            hideLoading();
            showToast(error.message || 'Transaction failed', 'error');
        }
    }

    async simulateTransaction(amount) {
        // Simulate network delay
        return new Promise((resolve, reject) => {
            setTimeout(() => {
                if (Math.random() > 0.1) { // 90% success rate for testing
                    resolve();
                } else {
                    reject(new Error('Transaction simulation failed'));
                }
            }, 2000);
        });
    }
}

export default DelegationPanel;
