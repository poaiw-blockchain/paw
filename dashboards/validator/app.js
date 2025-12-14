// PAW Validator Dashboard - Main Application

class ValidatorDashboard {
    constructor() {
        this.currentValidator = null;
        this.validators = this.loadValidators();
        this.wsConnection = null;
        this.refreshInterval = null;

        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.loadValidatorList();
        this.initializeWebSocket();

        // Load the first validator if available
        if (this.validators.length > 0) {
            this.selectValidator(this.validators[0].address);
        }

        // Start auto-refresh
        this.startAutoRefresh();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const section = link.getAttribute('data-section');
                this.showSection(section);
            });
        });

        // Validator selector
        const validatorSelect = document.getElementById('validatorSelect');
        validatorSelect.addEventListener('change', (e) => {
            this.selectValidator(e.target.value);
        });

        // Add validator button
        const addValidatorBtn = document.getElementById('addValidatorBtn');
        addValidatorBtn.addEventListener('click', () => {
            this.showAddValidatorModal();
        });

        // Modal controls
        const modal = document.getElementById('addValidatorModal');
        const closeButtons = modal.querySelectorAll('.modal-close');
        closeButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                this.hideAddValidatorModal();
            });
        });

        // Confirm add validator
        const confirmBtn = document.getElementById('confirmAddValidator');
        confirmBtn.addEventListener('click', () => {
            this.addValidator();
        });

        // Settings buttons
        document.getElementById('updateCommission')?.addEventListener('click', () => {
            this.updateCommission();
        });

        document.getElementById('saveSettings')?.addEventListener('click', () => {
            this.saveSettings();
        });

        document.getElementById('saveAlertSettings')?.addEventListener('click', () => {
            this.saveAlertSettings();
        });

        // Delegation controls
        document.getElementById('delegationSearch')?.addEventListener('input', (e) => {
            this.filterDelegations(e.target.value);
        });

        document.getElementById('delegationSort')?.addEventListener('change', (e) => {
            this.sortDelegations(e.target.value);
        });
    }

    showSection(sectionId) {
        // Update navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
        });
        document.querySelector(`[data-section="${sectionId}"]`)?.classList.add('active');

        // Show section
        document.querySelectorAll('.content-section').forEach(section => {
            section.classList.remove('active');
        });
        document.getElementById(sectionId)?.classList.add('active');
    }

    loadValidators() {
        const stored = localStorage.getItem('paw_validators');
        return stored ? JSON.parse(stored) : [];
    }

    saveValidators() {
        localStorage.setItem('paw_validators', JSON.stringify(this.validators));
    }

    loadValidatorList() {
        const select = document.getElementById('validatorSelect');
        select.innerHTML = '<option value="">Select Validator...</option>';

        this.validators.forEach(validator => {
            const option = document.createElement('option');
            option.value = validator.address;
            option.textContent = validator.name || validator.address.substring(0, 20) + '...';
            select.appendChild(option);
        });
    }

    async selectValidator(address) {
        if (!address) return;

        this.currentValidator = address;
        document.getElementById('validatorSelect').value = address;

        await this.loadValidatorData();
    }

    async loadValidatorData() {
        if (!this.currentValidator) return;

        try {
            // Load validator information
            const validatorInfo = await ValidatorAPI.getValidatorInfo(this.currentValidator);
            this.displayValidatorInfo(validatorInfo);

            // Load delegations
            const delegations = await ValidatorAPI.getDelegations(this.currentValidator);
            this.displayDelegations(delegations);

            // Load rewards
            const rewards = await ValidatorAPI.getRewards(this.currentValidator);
            this.displayRewards(rewards);

            // Load performance metrics
            const performance = await ValidatorAPI.getPerformance(this.currentValidator);
            this.displayPerformance(performance);

            // Load uptime data
            const uptime = await ValidatorAPI.getUptime(this.currentValidator);
            this.displayUptime(uptime);

            // Load signing stats
            const signingStats = await ValidatorAPI.getSigningStats(this.currentValidator);
            this.displaySigningStats(signingStats);

            // Load slash events
            const slashEvents = await ValidatorAPI.getSlashEvents(this.currentValidator);
            this.displaySlashEvents(slashEvents);

        } catch (error) {
            console.error('Error loading validator data:', error);
            this.showError('Failed to load validator data');
        }
    }

    displayValidatorInfo(info) {
        // Update overview stats
        document.getElementById('validatorStatus').textContent = info.status || 'Unknown';
        document.getElementById('totalStaked').textContent = this.formatAmount(info.tokens);
        document.getElementById('commission').textContent = `${(info.commission * 100).toFixed(2)}%`;
        document.getElementById('totalRewards').textContent = this.formatAmount(info.totalRewards);
        document.getElementById('delegatorCount').textContent = info.delegatorCount || '0';
        document.getElementById('uptime').textContent = `${(info.uptime * 100).toFixed(2)}%`;

        // Display detailed validator info
        const infoContainer = document.getElementById('validatorInfo');
        const validatorCard = new ValidatorCard(info);
        infoContainer.innerHTML = validatorCard.render();
    }

    displayDelegations(delegations) {
        const container = document.getElementById('delegationList');
        const delegationList = new DelegationList(delegations);
        container.innerHTML = delegationList.render();
    }

    displayRewards(rewards) {
        // Update rewards summary
        document.getElementById('totalDistributed').textContent = this.formatAmount(rewards.totalDistributed);
        document.getElementById('pendingRewards').textContent = this.formatAmount(rewards.pending);
        document.getElementById('commissionEarned').textContent = this.formatAmount(rewards.commissionEarned);

        // Render rewards chart
        const chartContainer = document.getElementById('rewardsChart');
        const rewardsChart = new RewardsChart(rewards.history);
        rewardsChart.render(chartContainer);

        // Display reward history
        const historyContainer = document.getElementById('rewardHistory');
        historyContainer.innerHTML = this.renderRewardHistory(rewards.history);
    }

    displayPerformance(performance) {
        document.getElementById('votingPower').textContent = `${performance.votingPower.toFixed(2)}%`;
        document.getElementById('blockProposals').textContent = performance.blockProposals || '0';
        document.getElementById('missRate').textContent = `${(performance.missRate * 100).toFixed(2)}%`;

        // Render mini charts (simplified for now)
        this.renderMiniChart('votingPowerChart', performance.votingPowerHistory);
        this.renderMiniChart('blockProposalsChart', performance.proposalHistory);
        this.renderMiniChart('missRateChart', performance.missRateHistory);
    }

    displayUptime(uptime) {
        const container = document.getElementById('uptimeMonitor');
        const uptimeMonitor = new UptimeMonitor(uptime);
        container.innerHTML = uptimeMonitor.render();

        // Display uptime alerts
        const alertsContainer = document.getElementById('uptimeAlerts');
        alertsContainer.innerHTML = this.renderUptimeAlerts(uptime.alerts);
    }

    displaySigningStats(stats) {
        document.getElementById('blocksSigned').textContent = stats.blocksSigned || '0';
        document.getElementById('blocksMissed').textContent = stats.blocksMissed || '0';

        const signRate = stats.blocksSigned / (stats.blocksSigned + stats.blocksMissed) * 100;
        document.getElementById('signRate').textContent = `${signRate.toFixed(2)}%`;

        // Display signing history
        const historyContainer = document.getElementById('signingHistory');
        historyContainer.innerHTML = this.renderSigningHistory(stats.history);
    }

    displaySlashEvents(events) {
        const container = document.getElementById('slashEvents');

        if (events.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-check-circle"></i>
                    <p>No slash events recorded</p>
                </div>
            `;
            return;
        }

        container.innerHTML = events.map(event => `
            <div class="slash-event">
                <div class="event-header">
                    <span class="event-type">${event.type}</span>
                    <span class="event-time">${new Date(event.timestamp).toLocaleString()}</span>
                </div>
                <div class="event-details">
                    <p>Height: ${event.height}</p>
                    <p>Amount Slashed: ${this.formatAmount(event.amount)}</p>
                    <p>Reason: ${event.reason}</p>
                </div>
            </div>
        `).join('');
    }

    renderRewardHistory(history) {
        if (!history || history.length === 0) {
            return '<div class="empty-state"><p>No reward history available</p></div>';
        }

        return `
            <div class="reward-history">
                <h3>Reward History</h3>
                <div class="history-list">
                    ${history.map(item => `
                        <div class="history-item">
                            <span class="date">${new Date(item.timestamp).toLocaleDateString()}</span>
                            <span class="amount">${this.formatAmount(item.amount)}</span>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    renderUptimeAlerts(alerts) {
        if (!alerts || alerts.length === 0) {
            return '<div class="empty-state"><p>No uptime alerts</p></div>';
        }

        return alerts.map(alert => `
            <div class="alert-item ${alert.level}">
                <div>
                    <strong>${alert.title}</strong>
                    <p>${alert.message}</p>
                </div>
                <span class="time">${new Date(alert.timestamp).toLocaleTimeString()}</span>
            </div>
        `).join('');
    }

    renderSigningHistory(history) {
        if (!history || history.length === 0) {
            return '<div class="empty-state"><p>No signing history available</p></div>';
        }

        return `
            <div class="signing-history-list">
                ${history.slice(0, 50).map((signed, index) => `
                    <div class="signing-block ${signed ? 'signed' : 'missed'}"
                         title="Block ${history.length - index}: ${signed ? 'Signed' : 'Missed'}">
                    </div>
                `).join('')}
            </div>
        `;
    }

    renderMiniChart(containerId, data) {
        // Simplified chart rendering - in production, use a charting library
        const container = document.getElementById(containerId);
        if (!container || !data) return;

        container.innerHTML = `
            <svg viewBox="0 0 100 50" style="width: 100%; height: 100%;">
                <polyline
                    points="${data.map((val, idx) =>
                        `${(idx / (data.length - 1)) * 100},${50 - (val * 50)}`
                    ).join(' ')}"
                    fill="none"
                    stroke="#3b82f6"
                    stroke-width="2"
                />
            </svg>
        `;
    }

    showAddValidatorModal() {
        const modal = document.getElementById('addValidatorModal');
        modal.classList.add('active');
    }

    hideAddValidatorModal() {
        const modal = document.getElementById('addValidatorModal');
        modal.classList.remove('active');
        document.getElementById('newValidatorAddress').value = '';
        document.getElementById('newValidatorName').value = '';
    }

    async addValidator() {
        const address = document.getElementById('newValidatorAddress').value.trim();
        const name = document.getElementById('newValidatorName').value.trim();

        if (!address) {
            this.showError('Please enter a validator address');
            return;
        }

        if (!address.startsWith('pawvaloper')) {
            this.showError('Invalid validator address format');
            return;
        }

        // Check if validator already exists
        if (this.validators.find(v => v.address === address)) {
            this.showError('Validator already added');
            return;
        }

        // Verify validator exists
        try {
            await ValidatorAPI.getValidatorInfo(address);

            this.validators.push({ address, name });
            this.saveValidators();
            this.loadValidatorList();
            this.selectValidator(address);
            this.hideAddValidatorModal();
            this.showSuccess('Validator added successfully');
        } catch (error) {
            this.showError('Validator not found or invalid address');
        }
    }

    async updateCommission() {
        const rate = parseFloat(document.getElementById('commissionRate').value);

        if (isNaN(rate) || rate < 0 || rate > 100) {
            this.showError('Invalid commission rate');
            return;
        }

        try {
            await ValidatorAPI.updateCommission(this.currentValidator, rate / 100);
            this.showSuccess('Commission updated successfully');
            await this.loadValidatorData();
        } catch (error) {
            this.showError('Failed to update commission');
        }
    }

    async saveSettings() {
        const moniker = document.getElementById('moniker').value.trim();
        const website = document.getElementById('website').value.trim();
        const details = document.getElementById('details').value.trim();

        try {
            await ValidatorAPI.updateValidatorInfo(this.currentValidator, {
                moniker,
                website,
                details
            });
            this.showSuccess('Settings saved successfully');
            await this.loadValidatorData();
        } catch (error) {
            this.showError('Failed to save settings');
        }
    }

    async saveAlertSettings() {
        const settings = {
            emailAlerts: document.getElementById('emailAlerts').checked,
            alertEmail: document.getElementById('alertEmail').value.trim(),
            uptimeAlerts: document.getElementById('uptimeAlerts').checked,
            slashingAlerts: document.getElementById('slashingAlerts').checked
        };

        localStorage.setItem('paw_alert_settings', JSON.stringify(settings));
        this.showSuccess('Alert settings saved');
    }

    filterDelegations(searchTerm) {
        const items = document.querySelectorAll('.delegation-item');
        items.forEach(item => {
            const address = item.querySelector('.delegator-address').textContent;
            if (address.toLowerCase().includes(searchTerm.toLowerCase())) {
                item.style.display = '';
            } else {
                item.style.display = 'none';
            }
        });
    }

    sortDelegations(sortBy) {
        // Reload delegations with sorting
        this.loadValidatorData();
    }

    initializeWebSocket() {
        this.wsConnection = new ValidatorWebSocket();

        this.wsConnection.on('connected', () => {
            this.updateConnectionStatus('connected');
        });

        this.wsConnection.on('disconnected', () => {
            this.updateConnectionStatus('disconnected');
        });

        this.wsConnection.on('validatorUpdate', (data) => {
            if (data.address === this.currentValidator) {
                this.handleValidatorUpdate(data);
            }
        });

        this.wsConnection.on('newBlock', (data) => {
            this.handleNewBlock(data);
        });

        this.wsConnection.connect();
    }

    updateConnectionStatus(status) {
        const statusElement = document.getElementById('connectionStatus');
        statusElement.className = `status-indicator ${status}`;

        const statusText = {
            connected: 'Connected',
            disconnected: 'Disconnected',
            connecting: 'Connecting...'
        };

        statusElement.innerHTML = `<i class="fas fa-circle"></i> ${statusText[status]}`;
    }

    handleValidatorUpdate(data) {
        // Update specific fields without full reload
        if (data.tokens) {
            document.getElementById('totalStaked').textContent = this.formatAmount(data.tokens);
        }
        if (data.uptime !== undefined) {
            document.getElementById('uptime').textContent = `${(data.uptime * 100).toFixed(2)}%`;
        }
    }

    handleNewBlock(data) {
        // Update block-related metrics
        // This would be enhanced based on actual block data structure
        console.log('New block:', data);
    }

    startAutoRefresh() {
        // Refresh data every 30 seconds
        this.refreshInterval = setInterval(() => {
            if (this.currentValidator) {
                this.loadValidatorData();
            }
        }, 30000);
    }

    formatAmount(amount) {
        if (!amount) return '0 PAW';
        const value = parseFloat(amount);
        if (value >= 1000000) {
            return `${(value / 1000000).toFixed(2)}M PAW`;
        } else if (value >= 1000) {
            return `${(value / 1000).toFixed(2)}K PAW`;
        }
        return `${value.toFixed(2)} PAW`;
    }

    showError(message) {
        // Simple error notification - enhance with a toast library in production
        alert(`Error: ${message}`);
    }

    showSuccess(message) {
        // Simple success notification - enhance with a toast library in production
        alert(`Success: ${message}`);
    }
}

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new ValidatorDashboard();
});
