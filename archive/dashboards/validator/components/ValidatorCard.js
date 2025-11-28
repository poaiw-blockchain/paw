// ValidatorCard Component - Displays detailed validator information

class ValidatorCard {
    constructor(validatorData) {
        this.data = validatorData;
    }

    render() {
        const {
            address,
            moniker,
            website,
            details,
            identity,
            tokens,
            delegatorShares,
            commission,
            status,
            jailed,
            unbondingHeight,
            unbondingTime
        } = this.data;

        return `
            <div class="validator-card">
                <div class="validator-header">
                    ${this.renderIdentityIcon(identity)}
                    <div class="validator-title">
                        <h3>${this.escapeHtml(moniker || 'Unknown Validator')}</h3>
                        <span class="validator-address">${this.escapeHtml(address)}</span>
                    </div>
                    ${this.renderStatusBadge(status, jailed)}
                </div>

                <div class="validator-details">
                    ${website ? `
                        <div class="detail-item">
                            <i class="fas fa-globe"></i>
                            <a href="${this.escapeHtml(website)}" target="_blank" rel="noopener noreferrer">
                                ${this.escapeHtml(website)}
                            </a>
                        </div>
                    ` : ''}

                    ${details ? `
                        <div class="detail-item">
                            <i class="fas fa-info-circle"></i>
                            <p>${this.escapeHtml(details)}</p>
                        </div>
                    ` : ''}

                    <div class="detail-row">
                        <div class="detail-item">
                            <label>Total Tokens</label>
                            <span class="value">${this.formatTokens(tokens)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Delegator Shares</label>
                            <span class="value">${this.formatShares(delegatorShares)}</span>
                        </div>
                    </div>

                    <div class="detail-row">
                        <div class="detail-item">
                            <label>Commission Rate</label>
                            <span class="value">${this.formatCommission(commission.rate)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Max Rate</label>
                            <span class="value">${this.formatCommission(commission.maxRate)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Max Change Rate</label>
                            <span class="value">${this.formatCommission(commission.maxChangeRate)}</span>
                        </div>
                    </div>

                    ${jailed ? `
                        <div class="jailed-warning">
                            <i class="fas fa-exclamation-triangle"></i>
                            <div>
                                <strong>Validator is Jailed</strong>
                                <p>Unbonding Height: ${unbondingHeight || 'N/A'}</p>
                                <p>Unbonding Time: ${unbondingTime ? new Date(unbondingTime).toLocaleString() : 'N/A'}</p>
                            </div>
                        </div>
                    ` : ''}
                </div>

                <div class="validator-actions">
                    <button class="btn btn-primary" onclick="dashboard.showSection('settings')">
                        <i class="fas fa-cog"></i> Edit Settings
                    </button>
                    <button class="btn btn-secondary" onclick="window.open('https://explorer.paw.network/validators/${address}', '_blank')">
                        <i class="fas fa-external-link-alt"></i> View in Explorer
                    </button>
                </div>
            </div>

            <style>
                .validator-card {
                    background-color: var(--card-bg);
                }

                .validator-header {
                    display: flex;
                    align-items: center;
                    gap: 1rem;
                    margin-bottom: 1.5rem;
                    padding-bottom: 1.5rem;
                    border-bottom: 1px solid var(--border-color);
                }

                .validator-title {
                    flex: 1;
                }

                .validator-title h3 {
                    font-size: 1.5rem;
                    margin-bottom: 0.25rem;
                    color: var(--text-primary);
                }

                .validator-address {
                    font-family: monospace;
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    word-break: break-all;
                }

                .identity-icon {
                    width: 64px;
                    height: 64px;
                    border-radius: 50%;
                    background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    color: white;
                    font-size: 1.5rem;
                    font-weight: bold;
                }

                .status-badge {
                    padding: 0.5rem 1rem;
                    border-radius: 9999px;
                    font-size: 0.875rem;
                    font-weight: 500;
                    text-transform: uppercase;
                }

                .status-badge.bonded {
                    background-color: rgba(16, 185, 129, 0.1);
                    color: var(--success-color);
                }

                .status-badge.unbonding {
                    background-color: rgba(245, 158, 11, 0.1);
                    color: var(--warning-color);
                }

                .status-badge.unbonded {
                    background-color: rgba(107, 114, 128, 0.1);
                    color: var(--text-secondary);
                }

                .status-badge.jailed {
                    background-color: rgba(239, 68, 68, 0.1);
                    color: var(--danger-color);
                }

                .validator-details {
                    display: flex;
                    flex-direction: column;
                    gap: 1rem;
                }

                .detail-item {
                    display: flex;
                    align-items: flex-start;
                    gap: 0.75rem;
                }

                .detail-item i {
                    color: var(--primary-color);
                    margin-top: 0.25rem;
                }

                .detail-item label {
                    display: block;
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    margin-bottom: 0.25rem;
                }

                .detail-item .value {
                    font-weight: 600;
                    color: var(--text-primary);
                }

                .detail-item a {
                    color: var(--primary-color);
                    text-decoration: none;
                }

                .detail-item a:hover {
                    text-decoration: underline;
                }

                .detail-row {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
                    gap: 1rem;
                    padding: 1rem;
                    background-color: var(--light-bg);
                    border-radius: 0.5rem;
                }

                .jailed-warning {
                    display: flex;
                    gap: 1rem;
                    padding: 1rem;
                    background-color: rgba(239, 68, 68, 0.1);
                    border-left: 4px solid var(--danger-color);
                    border-radius: 0.5rem;
                }

                .jailed-warning i {
                    color: var(--danger-color);
                    font-size: 1.5rem;
                }

                .jailed-warning strong {
                    display: block;
                    margin-bottom: 0.5rem;
                    color: var(--danger-color);
                }

                .jailed-warning p {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    margin: 0.25rem 0;
                }

                .validator-actions {
                    display: flex;
                    gap: 1rem;
                    margin-top: 1.5rem;
                    padding-top: 1.5rem;
                    border-top: 1px solid var(--border-color);
                }

                @media (max-width: 768px) {
                    .validator-actions {
                        flex-direction: column;
                    }

                    .detail-row {
                        grid-template-columns: 1fr;
                    }
                }
            </style>
        `;
    }

    renderIdentityIcon(identity) {
        if (identity) {
            // In production, fetch actual validator logo from Keybase
            return `<div class="identity-icon">${identity.substring(0, 2).toUpperCase()}</div>`;
        }

        // Use first letter of address as fallback
        return `<div class="identity-icon">${this.data.address.substring(10, 11).toUpperCase()}</div>`;
    }

    renderStatusBadge(status, jailed) {
        if (jailed) {
            return '<span class="status-badge jailed">Jailed</span>';
        }

        const statusMap = {
            'BOND_STATUS_BONDED': 'bonded',
            'BOND_STATUS_UNBONDING': 'unbonding',
            'BOND_STATUS_UNBONDED': 'unbonded'
        };

        const statusClass = statusMap[status] || 'unbonded';
        const statusText = statusClass.charAt(0).toUpperCase() + statusClass.slice(1);

        return `<span class="status-badge ${statusClass}">${statusText}</span>`;
    }

    formatTokens(tokens) {
        if (!tokens) return '0 PAW';
        const value = parseFloat(tokens) / 1000000; // Assuming micro-units
        return `${value.toLocaleString()} PAW`;
    }

    formatShares(shares) {
        if (!shares) return '0';
        const value = parseFloat(shares) / 1000000;
        return value.toLocaleString();
    }

    formatCommission(rate) {
        if (!rate) return '0%';
        const value = parseFloat(rate) * 100;
        return `${value.toFixed(2)}%`;
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ValidatorCard;
}
