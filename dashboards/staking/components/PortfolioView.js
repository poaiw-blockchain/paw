// Portfolio View Component

import { formatAmount, formatPercent, formatDate } from '../utils/ui.js';

export class PortfolioView {
    constructor(api) {
        this.api = api;
        this.delegatorAddress = null;
        this.eventHandlers = {};
    }

    on(event, handler) {
        this.eventHandlers[event] = handler;
    }

    emit(event, data) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event](data);
        }
    }

    async render(delegatorAddress) {
        this.delegatorAddress = delegatorAddress;

        const container = document.getElementById('portfolio-content');
        if (!container) return;

        container.innerHTML = '<div class="text-center">Loading portfolio...</div>';

        try {
            const [balance, delegations, unbonding, rewards] = await Promise.all([
                this.api.getBalance(delegatorAddress),
                this.api.getDelegations(delegatorAddress),
                this.api.getUnbondingDelegations(delegatorAddress),
                this.api.getDelegationRewards(delegatorAddress)
            ]);

            const totalDelegated = delegations.reduce((sum, d) => sum + d.balance, 0);
            const totalUnbonding = unbonding.reduce((sum, u) =>
                sum + u.entries.reduce((s, e) => s + e.balance, 0), 0
            );
            const totalRewards = rewards.total.reduce((sum, r) =>
                r.denom === 'upaw' ? sum + r.amount : sum, 0
            );

            container.innerHTML = `
                ${this.renderSummary(balance, totalDelegated, totalUnbonding, totalRewards)}
                ${this.renderDelegations(delegations, rewards)}
                ${unbonding.length > 0 ? this.renderUnbonding(unbonding) : ''}
                ${this.renderHistory()}
            `;

            this.setupEventListeners();
        } catch (error) {
            console.error('Error loading portfolio:', error);
            container.innerHTML = '<div class="text-center text-danger">Failed to load portfolio</div>';
        }
    }

    renderSummary(balance, totalDelegated, totalUnbonding, totalRewards) {
        const totalValue = balance + totalDelegated + totalUnbonding + totalRewards;

        return `
            <div class="portfolio-summary">
                <div class="stat-card">
                    <div class="stat-icon"><i class="fas fa-wallet"></i></div>
                    <div class="stat-info">
                        <div class="stat-label">Available Balance</div>
                        <div class="stat-value">${formatAmount(balance * 1e6)} PAW</div>
                    </div>
                </div>

                <div class="stat-card">
                    <div class="stat-icon"><i class="fas fa-coins"></i></div>
                    <div class="stat-info">
                        <div class="stat-label">Total Delegated</div>
                        <div class="stat-value">${formatAmount(totalDelegated * 1e6)} PAW</div>
                    </div>
                </div>

                <div class="stat-card">
                    <div class="stat-icon"><i class="fas fa-clock"></i></div>
                    <div class="stat-info">
                        <div class="stat-label">Unbonding</div>
                        <div class="stat-value">${formatAmount(totalUnbonding * 1e6)} PAW</div>
                    </div>
                </div>

                <div class="stat-card">
                    <div class="stat-icon"><i class="fas fa-gift"></i></div>
                    <div class="stat-info">
                        <div class="stat-label">Pending Rewards</div>
                        <div class="stat-value" style="color: var(--success-color);">
                            ${formatAmount(totalRewards * 1e6)} PAW
                        </div>
                    </div>
                </div>
            </div>

            <div class="stat-card" style="margin-bottom: 2rem;">
                <div class="stat-icon" style="background: linear-gradient(135deg, var(--success-color), #059669);">
                    <i class="fas fa-chart-pie"></i>
                </div>
                <div class="stat-info">
                    <div class="stat-label">Total Portfolio Value</div>
                    <div class="stat-value">${formatAmount(totalValue * 1e6)} PAW</div>
                    <div style="font-size: 0.875rem; color: var(--text-secondary); margin-top: 0.25rem;">
                        ${formatPercent((totalDelegated / totalValue) * 100)} staked
                    </div>
                </div>
                <button class="btn btn-primary" id="claim-rewards-btn" ${totalRewards === 0 ? 'disabled' : ''}>
                    <i class="fas fa-gift"></i> Claim Rewards
                </button>
            </div>
        `;
    }

    renderDelegations(delegations, rewards) {
        if (delegations.length === 0) {
            return `
                <div class="delegation-list">
                    <h3>Active Delegations</h3>
                    <div class="text-center" style="padding: 2rem; color: var(--text-secondary);">
                        <i class="fas fa-inbox" style="font-size: 3rem; margin-bottom: 1rem;"></i>
                        <p>No active delegations</p>
                        <button class="btn btn-primary" onclick="document.querySelector('[data-view=validators]').click()">
                            <i class="fas fa-plus"></i> Start Staking
                        </button>
                    </div>
                </div>
            `;
        }

        return `
            <div class="delegation-list">
                <h3>Active Delegations (${delegations.length})</h3>
                ${delegations.map(d => this.renderDelegationItem(d, rewards)).join('')}
            </div>
        `;
    }

    renderDelegationItem(delegation, rewards) {
        const validatorReward = rewards.rewards.find(r =>
            r.validatorAddress === delegation.validatorAddress
        );
        const pawReward = validatorReward?.reward.find(r => r.denom === 'upaw');
        const rewardAmount = pawReward ? pawReward.amount : 0;

        return `
            <div class="delegation-item">
                <div style="flex: 1;">
                    <div style="font-weight: 600; margin-bottom: 0.25rem;">
                        ${delegation.validatorAddress.slice(0, 25)}...
                    </div>
                    <div style="font-size: 0.875rem; color: var(--text-secondary);">
                        Delegated: ${formatAmount(delegation.balance * 1e6)} PAW
                    </div>
                    ${rewardAmount > 0 ? `
                        <div style="font-size: 0.875rem; color: var(--success-color); margin-top: 0.25rem;">
                            <i class="fas fa-coins"></i> Rewards: ${formatAmount(rewardAmount * 1e6)} PAW
                        </div>
                    ` : ''}
                </div>
                <div class="delegation-actions">
                    <button class="btn btn-sm btn-primary add-delegation-btn" data-validator="${delegation.validatorAddress}">
                        <i class="fas fa-plus"></i>
                    </button>
                    <button class="btn btn-sm btn-secondary remove-delegation-btn" data-validator="${delegation.validatorAddress}">
                        <i class="fas fa-minus"></i>
                    </button>
                </div>
            </div>
        `;
    }

    renderUnbonding(unbonding) {
        const totalEntries = unbonding.reduce((sum, u) => sum + u.entries.length, 0);

        return `
            <div class="delegation-list" style="margin-top: 2rem;">
                <h3>Unbonding Delegations (${totalEntries})</h3>
                ${unbonding.map(u => this.renderUnbondingItem(u)).join('')}
            </div>
        `;
    }

    renderUnbondingItem(unbonding) {
        return unbonding.entries.map(entry => `
            <div class="delegation-item">
                <div style="flex: 1;">
                    <div style="font-weight: 600; margin-bottom: 0.25rem;">
                        ${unbonding.validatorAddress.slice(0, 25)}...
                    </div>
                    <div style="font-size: 0.875rem; color: var(--text-secondary);">
                        Amount: ${formatAmount(entry.balance * 1e6)} PAW
                    </div>
                    <div style="font-size: 0.875rem; color: var(--warning-color); margin-top: 0.25rem;">
                        <i class="fas fa-clock"></i> Completes: ${formatDate(entry.completionTime)}
                    </div>
                </div>
                <div class="delegation-actions">
                    <span class="status-badge" style="background: var(--bg-tertiary); color: var(--text-primary);">
                        Unbonding
                    </span>
                </div>
            </div>
        `).join('');
    }

    renderHistory() {
        // Mock historical data for demonstration
        const mockHistory = [
            { date: new Date(Date.now() - 86400000), action: 'Delegate', amount: 1000, validator: 'Alpha Validator' },
            { date: new Date(Date.now() - 172800000), action: 'Claim Rewards', amount: 25.5, validator: 'Multiple' },
            { date: new Date(Date.now() - 259200000), action: 'Delegate', amount: 500, validator: 'Beta Validator' }
        ];

        return `
            <div class="delegation-list" style="margin-top: 2rem;">
                <h3>Recent Activity</h3>
                ${mockHistory.map(h => `
                    <div class="delegation-item">
                        <div style="flex: 1;">
                            <div style="font-weight: 600; margin-bottom: 0.25rem;">
                                ${h.action}
                            </div>
                            <div style="font-size: 0.875rem; color: var(--text-secondary);">
                                ${h.validator} • ${formatDate(h.date)}
                            </div>
                        </div>
                        <div style="text-align: right;">
                            <div style="font-weight: 600; color: ${h.action === 'Claim Rewards' ? 'var(--success-color)' : 'var(--text-primary)'};">
                                ${h.action === 'Claim Rewards' ? '+' : ''}${formatAmount(h.amount * 1e6)} PAW
                            </div>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;
    }

    setupEventListeners() {
        // Claim rewards button
        const claimBtn = document.getElementById('claim-rewards-btn');
        if (claimBtn) {
            claimBtn.addEventListener('click', () => {
                this.emit('claim-rewards');
            });
        }

        // Add delegation buttons
        document.querySelectorAll('.add-delegation-btn').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const validatorAddress = e.target.closest('button').dataset.validator;
                const validator = await this.api.getValidatorDetails(validatorAddress);
                if (validator) {
                    this.emit('delegate', validator);
                }
            });
        });

        // Remove delegation buttons
        document.querySelectorAll('.remove-delegation-btn').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const validatorAddress = e.target.closest('button').dataset.validator;
                const validator = await this.api.getValidatorDetails(validatorAddress);
                if (validator) {
                    this.emit('delegate', validator);
                }
            });
        });
    }

    async refresh() {
        if (this.delegatorAddress) {
            await this.render(this.delegatorAddress);
        }
    }
}

export default PortfolioView;
