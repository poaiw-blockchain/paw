// DelegationList Component - Displays list of delegations

class DelegationList {
    constructor(delegations) {
        this.delegations = delegations || [];
        this.filteredDelegations = [...this.delegations];
        this.sortBy = 'amount';
        this.sortOrder = 'desc';
    }

    render() {
        if (this.delegations.length === 0) {
            return this.renderEmptyState();
        }

        const sortedDelegations = this.sortDelegations(this.filteredDelegations);

        return `
            <div class="delegation-list-container">
                <div class="delegation-summary">
                    <div class="summary-item">
                        <span class="label">Total Delegations:</span>
                        <span class="value">${this.delegations.length}</span>
                    </div>
                    <div class="summary-item">
                        <span class="label">Total Amount:</span>
                        <span class="value">${this.formatTotalAmount()}</span>
                    </div>
                    <div class="summary-item">
                        <span class="label">Average Delegation:</span>
                        <span class="value">${this.formatAverageAmount()}</span>
                    </div>
                </div>

                <div class="delegation-table">
                    <div class="table-header">
                        <div class="header-cell" data-sort="delegator">
                            Delegator
                            <i class="fas fa-sort"></i>
                        </div>
                        <div class="header-cell" data-sort="amount">
                            Amount
                            <i class="fas fa-sort"></i>
                        </div>
                        <div class="header-cell" data-sort="rewards">
                            Pending Rewards
                            <i class="fas fa-sort"></i>
                        </div>
                        <div class="header-cell" data-sort="timestamp">
                            Delegation Date
                            <i class="fas fa-sort"></i>
                        </div>
                    </div>

                    <div class="table-body">
                        ${sortedDelegations.map(delegation => this.renderDelegationRow(delegation)).join('')}
                    </div>
                </div>

                ${this.renderPagination()}
            </div>

            <style>
                .delegation-list-container {
                    background-color: var(--card-bg);
                    border: 1px solid var(--border-color);
                    border-radius: 0.75rem;
                    overflow: hidden;
                }

                .delegation-summary {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                    gap: 1.5rem;
                    padding: 1.5rem;
                    background-color: var(--light-bg);
                    border-bottom: 1px solid var(--border-color);
                }

                .summary-item {
                    display: flex;
                    flex-direction: column;
                    gap: 0.25rem;
                }

                .summary-item .label {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                }

                .summary-item .value {
                    font-size: 1.25rem;
                    font-weight: 600;
                    color: var(--primary-color);
                }

                .delegation-table {
                    overflow-x: auto;
                }

                .table-header {
                    display: grid;
                    grid-template-columns: 2fr 1fr 1fr 1fr;
                    gap: 1rem;
                    padding: 1rem 1.5rem;
                    background-color: var(--light-bg);
                    border-bottom: 2px solid var(--border-color);
                    font-weight: 600;
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    text-transform: uppercase;
                }

                .header-cell {
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                    cursor: pointer;
                    user-select: none;
                }

                .header-cell:hover {
                    color: var(--primary-color);
                }

                .header-cell i {
                    font-size: 0.75rem;
                }

                .table-body {
                    max-height: 600px;
                    overflow-y: auto;
                }

                .delegation-row {
                    display: grid;
                    grid-template-columns: 2fr 1fr 1fr 1fr;
                    gap: 1rem;
                    padding: 1rem 1.5rem;
                    border-bottom: 1px solid var(--border-color);
                    align-items: center;
                    transition: background-color 0.2s;
                }

                .delegation-row:hover {
                    background-color: var(--light-bg);
                }

                .delegation-row:last-child {
                    border-bottom: none;
                }

                .delegator-info {
                    display: flex;
                    align-items: center;
                    gap: 0.75rem;
                }

                .delegator-avatar {
                    width: 40px;
                    height: 40px;
                    border-radius: 50%;
                    background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    color: white;
                    font-weight: bold;
                    font-size: 0.875rem;
                }

                .delegator-details {
                    flex: 1;
                    min-width: 0;
                }

                .delegator-name {
                    font-weight: 500;
                    color: var(--text-primary);
                    margin-bottom: 0.125rem;
                }

                .delegator-address {
                    font-family: monospace;
                    font-size: 0.75rem;
                    color: var(--text-secondary);
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }

                .amount-cell {
                    font-weight: 600;
                    color: var(--primary-color);
                }

                .rewards-cell {
                    color: var(--success-color);
                    font-weight: 500;
                }

                .date-cell {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                }

                .pagination {
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    gap: 1rem;
                    padding: 1rem;
                    border-top: 1px solid var(--border-color);
                }

                .pagination button {
                    padding: 0.5rem 1rem;
                    border: 1px solid var(--border-color);
                    border-radius: 0.375rem;
                    background-color: white;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .pagination button:hover:not(:disabled) {
                    background-color: var(--primary-color);
                    color: white;
                    border-color: var(--primary-color);
                }

                .pagination button:disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }

                .pagination-info {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                }

                @media (max-width: 768px) {
                    .table-header,
                    .delegation-row {
                        grid-template-columns: 1fr;
                        gap: 0.5rem;
                    }

                    .header-cell {
                        display: none;
                    }

                    .delegation-row > div:before {
                        content: attr(data-label);
                        display: block;
                        font-weight: 600;
                        font-size: 0.75rem;
                        color: var(--text-secondary);
                        margin-bottom: 0.25rem;
                    }
                }
            </style>
        `;
    }

    renderDelegationRow(delegation) {
        return `
            <div class="delegation-row">
                <div class="delegator-info" data-label="Delegator">
                    <div class="delegator-avatar">
                        ${this.getInitials(delegation.delegatorAddress)}
                    </div>
                    <div class="delegator-details">
                        <div class="delegator-name">${this.getDelegatorName(delegation)}</div>
                        <div class="delegator-address" title="${delegation.delegatorAddress}">
                            ${delegation.delegatorAddress}
                        </div>
                    </div>
                </div>
                <div class="amount-cell" data-label="Amount">
                    ${this.formatAmount(delegation.shares)}
                </div>
                <div class="rewards-cell" data-label="Pending Rewards">
                    ${this.formatAmount(delegation.pendingRewards || 0)}
                </div>
                <div class="date-cell" data-label="Delegation Date">
                    ${this.formatDate(delegation.timestamp)}
                </div>
            </div>
        `;
    }

    renderEmptyState() {
        return `
            <div class="empty-state">
                <i class="fas fa-users"></i>
                <p>No delegations found</p>
            </div>
        `;
    }

    renderPagination() {
        // Simplified pagination - enhance based on actual needs
        const totalPages = Math.ceil(this.delegations.length / 20);
        const currentPage = 1;

        if (totalPages <= 1) return '';

        return `
            <div class="pagination">
                <button onclick="this.previousPage()" ${currentPage === 1 ? 'disabled' : ''}>
                    <i class="fas fa-chevron-left"></i> Previous
                </button>
                <span class="pagination-info">Page ${currentPage} of ${totalPages}</span>
                <button onclick="this.nextPage()" ${currentPage === totalPages ? 'disabled' : ''}>
                    Next <i class="fas fa-chevron-right"></i>
                </button>
            </div>
        `;
    }

    sortDelegations(delegations) {
        return [...delegations].sort((a, b) => {
            let aValue, bValue;

            switch (this.sortBy) {
                case 'amount':
                    aValue = parseFloat(a.shares || 0);
                    bValue = parseFloat(b.shares || 0);
                    break;
                case 'rewards':
                    aValue = parseFloat(a.pendingRewards || 0);
                    bValue = parseFloat(b.pendingRewards || 0);
                    break;
                case 'timestamp':
                    aValue = new Date(a.timestamp || 0).getTime();
                    bValue = new Date(b.timestamp || 0).getTime();
                    break;
                case 'delegator':
                    aValue = a.delegatorAddress;
                    bValue = b.delegatorAddress;
                    break;
                default:
                    return 0;
            }

            if (this.sortOrder === 'asc') {
                return aValue > bValue ? 1 : -1;
            } else {
                return aValue < bValue ? 1 : -1;
            }
        });
    }

    getInitials(address) {
        if (!address || address.length < 12) return '??';
        return address.substring(10, 12).toUpperCase();
    }

    getDelegatorName(delegation) {
        // In production, this could fetch names from a registry
        return delegation.name || `Delegator ${delegation.delegatorAddress.substring(0, 10)}...`;
    }

    formatAmount(amount) {
        if (!amount) return '0 PAW';
        const value = parseFloat(amount) / 1000000; // Assuming micro-units
        return `${value.toLocaleString(undefined, { maximumFractionDigits: 2 })} PAW`;
    }

    formatTotalAmount() {
        const total = this.delegations.reduce((sum, d) => sum + parseFloat(d.shares || 0), 0);
        return this.formatAmount(total);
    }

    formatAverageAmount() {
        if (this.delegations.length === 0) return '0 PAW';
        const total = this.delegations.reduce((sum, d) => sum + parseFloat(d.shares || 0), 0);
        const average = total / this.delegations.length;
        return this.formatAmount(average);
    }

    formatDate(timestamp) {
        if (!timestamp) return 'N/A';
        const date = new Date(timestamp);
        return date.toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    }

    filter(searchTerm) {
        this.filteredDelegations = this.delegations.filter(d =>
            d.delegatorAddress.toLowerCase().includes(searchTerm.toLowerCase())
        );
    }

    sort(sortBy) {
        if (this.sortBy === sortBy) {
            this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
        } else {
            this.sortBy = sortBy;
            this.sortOrder = 'desc';
        }
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DelegationList;
}
