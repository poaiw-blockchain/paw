// Rewards Panel Component

import { formatAmount, showToast, showLoading, hideLoading } from '../utils/ui.js';

export class RewardsPanel {
    constructor(api) {
        this.api = api;
        this.delegatorAddress = null;
        this.rewards = null;
    }

    async render(delegatorAddress) {
        this.delegatorAddress = delegatorAddress;

        const container = document.getElementById('rewards-panel');
        if (!container) return;

        container.innerHTML = '<div class="text-center">Loading rewards...</div>';

        try {
            this.rewards = await this.api.getDelegationRewards(delegatorAddress);
            const delegations = await this.api.getDelegations(delegatorAddress);

            const totalRewards = this.calculateTotalRewards(this.rewards);

            container.innerHTML = `
                <div class="rewards-summary" style="margin-bottom: 2rem;">
                    <div style="text-align: center; padding: 2rem; background: linear-gradient(135deg, var(--primary-color), var(--secondary-color)); color: white; border-radius: var(--radius);">
                        <div style="font-size: 0.875rem; opacity: 0.9; margin-bottom: 0.5rem;">Total Pending Rewards</div>
                        <div style="font-size: 2.5rem; font-weight: 700; margin-bottom: 1rem;">
                            ${formatAmount(totalRewards * 1e6)} PAW
                        </div>
                        <button id="claim-all-btn" class="btn btn-primary" style="background: white; color: var(--primary-color);" ${totalRewards === 0 ? 'disabled' : ''}>
                            <i class="fas fa-gift"></i> Claim All Rewards
                        </button>
                    </div>
                </div>

                <h4 style="margin-bottom: 1rem;">Rewards by Validator</h4>
                <div id="rewards-list">
                    ${this.renderRewardsList(delegations)}
                </div>

                ${totalRewards > 0 ? `
                    <div style="margin-top: 2rem; padding: 1rem; background: var(--bg-tertiary); border-radius: var(--radius);">
                        <label class="checkbox-label">
                            <input type="checkbox" id="auto-compound-checkbox">
                            Auto-compound rewards (stake immediately after claiming)
                        </label>
                    </div>
                ` : ''}
            `;

            this.setupEventListeners();
        } catch (error) {
            console.error('Error loading rewards:', error);
            container.innerHTML = '<div class="text-center text-danger">Failed to load rewards</div>';
        }
    }

    calculateTotalRewards(rewards) {
        if (!rewards.total || rewards.total.length === 0) return 0;

        return rewards.total.reduce((sum, r) => {
            if (r.denom === 'upaw') {
                return sum + r.amount;
            }
            return sum;
        }, 0);
    }

    renderRewardsList(delegations) {
        if (!this.rewards.rewards || this.rewards.rewards.length === 0) {
            return '<div class="text-center" style="padding: 2rem; color: var(--text-secondary);">No pending rewards</div>';
        }

        return this.rewards.rewards.map(r => {
            const delegation = delegations.find(d => d.validatorAddress === r.validatorAddress);
            const pawReward = r.reward.find(rw => rw.denom === 'upaw');
            const rewardAmount = pawReward ? pawReward.amount : 0;

            return `
                <div class="delegation-item">
                    <div>
                        <div style="font-weight: 600; margin-bottom: 0.25rem;">
                            ${r.validatorAddress.slice(0, 20)}...
                        </div>
                        <div style="font-size: 0.875rem; color: var(--text-secondary);">
                            Delegated: ${delegation ? formatAmount(delegation.balance * 1e6) : '0'} PAW
                        </div>
                    </div>
                    <div style="text-align: right;">
                        <div style="font-size: 1.125rem; font-weight: 600; color: var(--success-color);">
                            ${formatAmount(rewardAmount * 1e6)} PAW
                        </div>
                        <button
                            class="btn btn-sm btn-primary claim-single-btn"
                            data-validator="${r.validatorAddress}"
                            ${rewardAmount === 0 ? 'disabled' : ''}
                        >
                            <i class="fas fa-gift"></i> Claim
                        </button>
                    </div>
                </div>
            `;
        }).join('');
    }

    setupEventListeners() {
        const claimAllBtn = document.getElementById('claim-all-btn');
        if (claimAllBtn) {
            claimAllBtn.addEventListener('click', () => this.claimAllRewards());
        }

        document.querySelectorAll('.claim-single-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const validatorAddress = e.target.closest('button').dataset.validator;
                this.claimRewards(validatorAddress);
            });
        });
    }

    async claimAllRewards() {
        const autoCompound = document.getElementById('auto-compound-checkbox')?.checked || false;

        try {
            showLoading('Claiming all rewards...');

            // In a real implementation, this would broadcast the transaction
            await this.simulateClaim();

            if (autoCompound) {
                showLoading('Auto-compounding rewards...');
                await this.simulateCompound();
            }

            hideLoading();
            showToast('Successfully claimed all rewards!', 'success');

            // Close modal and refresh
            document.getElementById('rewards-modal').classList.remove('active');
            if (window.stakingDashboard?.components?.portfolio) {
                window.stakingDashboard.components.portfolio.refresh();
            }
        } catch (error) {
            hideLoading();
            showToast(error.message || 'Failed to claim rewards', 'error');
        }
    }

    async claimRewards(validatorAddress) {
        try {
            showLoading('Claiming rewards...');

            // In a real implementation, this would broadcast the transaction
            await this.simulateClaim(validatorAddress);

            hideLoading();
            showToast('Successfully claimed rewards!', 'success');

            // Refresh rewards display
            await this.render(this.delegatorAddress);
        } catch (error) {
            hideLoading();
            showToast(error.message || 'Failed to claim rewards', 'error');
        }
    }

    async simulateClaim(validatorAddress = null) {
        // Simulate network delay
        return new Promise((resolve, reject) => {
            setTimeout(() => {
                if (Math.random() > 0.05) { // 95% success rate
                    resolve();
                } else {
                    reject(new Error('Transaction simulation failed'));
                }
            }, 2000);
        });
    }

    async simulateCompound() {
        return new Promise((resolve) => {
            setTimeout(() => resolve(), 1500);
        });
    }
}

export default RewardsPanel;
