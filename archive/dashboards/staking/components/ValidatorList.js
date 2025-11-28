// Validator List Component

import { formatAmount, formatPercent, debounce } from '../utils/ui.js';

export class ValidatorList {
    constructor(api) {
        this.api = api;
        this.validators = [];
        this.filteredValidators = [];
        this.sortBy = 'voting_power';
        this.sortDesc = true;
        this.searchQuery = '';
        this.activeOnly = false;
        this.eventHandlers = {};
        this.setupEventListeners();
    }

    on(event, handler) {
        this.eventHandlers[event] = handler;
    }

    emit(event, data) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event](data);
        }
    }

    setupEventListeners() {
        const searchInput = document.getElementById('validator-search');
        const sortSelect = document.getElementById('validator-sort');
        const activeFilter = document.getElementById('filter-active');

        if (searchInput) {
            searchInput.addEventListener('input', debounce((e) => {
                this.searchQuery = e.target.value.toLowerCase();
                this.filterAndSort();
            }, 300));
        }

        if (sortSelect) {
            sortSelect.addEventListener('change', (e) => {
                this.sortBy = e.target.value;
                this.filterAndSort();
            });
        }

        if (activeFilter) {
            activeFilter.addEventListener('change', (e) => {
                this.activeOnly = e.target.checked;
                this.filterAndSort();
            });
        }
    }

    async render() {
        const container = document.getElementById('validators-content');
        if (!container) return;

        container.innerHTML = '<div class="text-center">Loading validators...</div>';

        try {
            this.validators = await this.api.getValidators();
            this.filterAndSort();
        } catch (error) {
            console.error('Error loading validators:', error);
            container.innerHTML = '<div class="text-center text-danger">Failed to load validators</div>';
        }
    }

    filterAndSort() {
        // Filter
        this.filteredValidators = this.validators.filter(v => {
            const matchesSearch = !this.searchQuery ||
                v.moniker.toLowerCase().includes(this.searchQuery) ||
                v.operatorAddress.toLowerCase().includes(this.searchQuery);

            const matchesActive = !this.activeOnly ||
                v.status === 'BOND_STATUS_BONDED';

            return matchesSearch && matchesActive;
        });

        // Sort
        this.filteredValidators.sort((a, b) => {
            let comparison = 0;

            switch (this.sortBy) {
                case 'voting_power':
                    comparison = a.votingPower - b.votingPower;
                    break;
                case 'commission':
                    comparison = a.commission - b.commission;
                    break;
                case 'apy':
                    const apyA = this.api.calculateAPY(a, 7.5);
                    const apyB = this.api.calculateAPY(b, 7.5);
                    comparison = apyA - apyB;
                    break;
                case 'uptime':
                    // For now, random uptime (in real implementation, fetch from chain)
                    comparison = Math.random() - 0.5;
                    break;
            }

            return this.sortDesc ? -comparison : comparison;
        });

        this.renderTable();
    }

    renderTable() {
        const container = document.getElementById('validators-content');
        if (!container) return;

        if (this.filteredValidators.length === 0) {
            container.innerHTML = '<div class="text-center">No validators found</div>';
            return;
        }

        const table = document.createElement('table');
        table.className = 'validator-table';

        table.innerHTML = `
            <thead>
                <tr>
                    <th>Validator</th>
                    <th>Voting Power</th>
                    <th>Commission</th>
                    <th>APY</th>
                    <th>Status</th>
                    <th>Risk</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${this.filteredValidators.map(v => this.renderValidatorRow(v)).join('')}
            </tbody>
        `;

        container.innerHTML = '';
        container.appendChild(table);

        // Attach event listeners to delegate buttons
        container.querySelectorAll('.delegate-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const address = e.target.dataset.validator;
                const validator = this.validators.find(v => v.operatorAddress === address);
                if (validator) {
                    this.emit('delegate', validator);
                }
            });
        });
    }

    renderValidatorRow(validator) {
        const apy = this.api.calculateAPY(validator, 7.5);
        const riskScore = this.api.calculateRiskScore(validator);
        const riskLevel = this.api.getRiskLevel(riskScore);
        const statusClass = validator.status === 'BOND_STATUS_BONDED' ? 'status-active' : 'status-jailed';
        const statusText = validator.jailed ? 'Jailed' :
                          validator.status === 'BOND_STATUS_BONDED' ? 'Active' : 'Inactive';

        const logoText = validator.moniker.charAt(0).toUpperCase();

        return `
            <tr>
                <td>
                    <div class="validator-info">
                        <div class="validator-logo">${logoText}</div>
                        <div class="validator-details">
                            <h4>${validator.moniker}</h4>
                            <div class="validator-moniker">${validator.operatorAddress.slice(0, 20)}...</div>
                        </div>
                    </div>
                </td>
                <td>${formatAmount(validator.votingPower * 1e6)} PAW</td>
                <td>${formatPercent(validator.commission)}</td>
                <td>${formatPercent(apy)}</td>
                <td><span class="status-badge ${statusClass}">${statusText}</span></td>
                <td>
                    <div class="risk-indicator risk-${riskLevel}">
                        <i class="fas fa-circle"></i> ${riskLevel.toUpperCase()}
                    </div>
                </td>
                <td>
                    <button class="btn btn-primary btn-sm delegate-btn" data-validator="${validator.operatorAddress}">
                        Delegate
                    </button>
                </td>
            </tr>
        `;
    }
}

export default ValidatorList;
