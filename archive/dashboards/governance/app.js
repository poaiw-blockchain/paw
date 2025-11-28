/**
 * PAW Governance Portal - Main Application
 * Manages the governance interface, proposal viewing, voting, and analytics
 */

class GovernanceApp {
    constructor() {
        this.api = new GovernanceAPI();
        this.proposalList = null;
        this.proposalDetail = null;
        this.createProposal = null;
        this.votingPanel = null;
        this.tallyChart = null;
        this.currentSection = 'proposals';
        this.walletConnected = false;
        this.walletAddress = null;
        this.votingPower = 0;
        this.proposals = [];
        this.parameters = {};
        this.init();
    }

    async init() {
        console.log('Initializing PAW Governance Portal...');

        // Initialize components
        this.proposalList = new ProposalList(this.api, this);
        this.proposalDetail = new ProposalDetail(this.api, this);
        this.createProposal = new CreateProposal(this.api, this);
        this.votingPanel = new VotingPanel(this.api, this);
        this.tallyChart = new TallyChart();

        // Setup event listeners
        this.setupEventListeners();

        // Check connection status
        await this.checkConnection();

        // Load initial data
        await this.loadInitialData();

        console.log('Governance Portal initialized successfully');
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const section = e.currentTarget.dataset.section;
                this.navigateToSection(section);
            });
        });

        // Wallet connection
        const connectBtn = document.getElementById('connectWalletBtn');
        if (connectBtn) {
            connectBtn.addEventListener('click', () => this.connectWallet());
        }

        // Filters
        const statusFilter = document.getElementById('statusFilter');
        if (statusFilter) {
            statusFilter.addEventListener('change', () => this.filterProposals());
        }

        const typeFilter = document.getElementById('typeFilter');
        if (typeFilter) {
            typeFilter.addEventListener('change', () => this.filterProposals());
        }

        // Search
        const searchInput = document.getElementById('searchProposals');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => this.searchProposals(e.target.value));
        }

        // Modal close buttons
        document.querySelectorAll('.modal .close').forEach(closeBtn => {
            closeBtn.addEventListener('click', () => {
                closeBtn.closest('.modal').style.display = 'none';
            });
        });

        // Close modals on outside click
        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                e.target.style.display = 'none';
            }
        });
    }

    async checkConnection() {
        const statusElement = document.getElementById('connectionStatus');
        try {
            const isConnected = await this.api.checkConnection();
            if (isConnected) {
                statusElement.innerHTML = '<i class="fas fa-circle"></i> Connected';
                statusElement.classList.add('connected');
                statusElement.classList.remove('disconnected');
            } else {
                statusElement.innerHTML = '<i class="fas fa-circle"></i> Disconnected';
                statusElement.classList.add('disconnected');
                statusElement.classList.remove('connected');
            }
        } catch (error) {
            console.error('Connection check failed:', error);
            statusElement.innerHTML = '<i class="fas fa-circle"></i> Error';
            statusElement.classList.add('disconnected');
        }
    }

    async loadInitialData() {
        try {
            // Load proposals
            await this.loadProposals();

            // Load parameters
            await this.loadParameters();

            // Update stats
            await this.updateStats();

        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showError('Failed to load governance data. Please refresh the page.');
        }
    }

    async loadProposals() {
        try {
            this.proposals = await this.api.getAllProposals();
            if (this.currentSection === 'proposals') {
                this.proposalList.render(this.proposals);
            }
        } catch (error) {
            console.error('Failed to load proposals:', error);
            throw error;
        }
    }

    async loadParameters() {
        try {
            this.parameters = await this.api.getGovernanceParameters();
            if (this.currentSection === 'parameters') {
                this.renderParameters();
            }
        } catch (error) {
            console.error('Failed to load parameters:', error);
        }
    }

    async updateStats() {
        const activeProposals = this.proposals.filter(p => p.status === 'VOTING_PERIOD').length;
        const totalProposals = this.proposals.length;

        document.getElementById('activeProposals').textContent = activeProposals;
        document.getElementById('totalProposals').textContent = totalProposals;

        // Calculate participation rate
        const votingProposals = this.proposals.filter(p => p.status === 'VOTING_PERIOD');
        if (votingProposals.length > 0) {
            const avgParticipation = votingProposals.reduce((sum, p) => {
                return sum + this.calculateParticipationRate(p.final_tally_result);
            }, 0) / votingProposals.length;
            document.getElementById('participationRate').textContent = avgParticipation.toFixed(2) + '%';
        }

        // User votes (if wallet connected)
        if (this.walletConnected) {
            const userVotes = await this.api.getUserVotes(this.walletAddress);
            document.getElementById('userVotes').textContent = userVotes.length;
        }
    }

    calculateParticipationRate(tally) {
        if (!tally) return 0;
        const total = parseInt(tally.yes || 0) + parseInt(tally.no || 0) +
                     parseInt(tally.abstain || 0) + parseInt(tally.no_with_veto || 0);
        // Assume total bonded tokens (this would come from API in production)
        const totalBonded = 100000000;
        return (total / totalBonded) * 100;
    }

    navigateToSection(section) {
        // Update active nav link
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
            if (link.dataset.section === section) {
                link.classList.add('active');
            }
        });

        // Update content sections
        document.querySelectorAll('.content-section').forEach(sec => {
            sec.classList.remove('active');
        });

        const sectionElement = document.getElementById(`${section}Section`);
        if (sectionElement) {
            sectionElement.classList.add('active');
        }

        this.currentSection = section;

        // Load section-specific content
        switch (section) {
            case 'proposals':
                this.proposalList.render(this.proposals);
                break;
            case 'create':
                this.createProposal.render();
                break;
            case 'parameters':
                this.renderParameters();
                break;
            case 'analytics':
                this.renderAnalytics();
                break;
            case 'history':
                this.renderHistory();
                break;
        }
    }

    async connectWallet() {
        try {
            // Simulate wallet connection (integrate with actual wallet in production)
            this.walletAddress = 'paw1' + Math.random().toString(36).substring(2, 42);
            this.votingPower = Math.floor(Math.random() * 10000) + 1000;
            this.walletConnected = true;

            // Update UI
            document.getElementById('connectWalletBtn').classList.add('hidden');
            document.getElementById('walletInfo').classList.remove('hidden');
            document.getElementById('walletAddress').textContent =
                this.walletAddress.substring(0, 10) + '...' + this.walletAddress.substring(this.walletAddress.length - 6);
            document.getElementById('votingPower').textContent = this.votingPower.toLocaleString() + ' PAW';

            // Update stats
            await this.updateStats();

            this.showSuccess('Wallet connected successfully!');
        } catch (error) {
            console.error('Failed to connect wallet:', error);
            this.showError('Failed to connect wallet. Please try again.');
        }
    }

    filterProposals() {
        const statusFilter = document.getElementById('statusFilter').value;
        const typeFilter = document.getElementById('typeFilter').value;

        let filtered = this.proposals;

        if (statusFilter !== 'all') {
            filtered = filtered.filter(p => {
                const status = p.status.toLowerCase();
                if (statusFilter === 'voting') return status === 'voting_period';
                if (statusFilter === 'deposit') return status === 'deposit_period';
                if (statusFilter === 'passed') return status === 'passed';
                if (statusFilter === 'rejected') return status === 'rejected';
                if (statusFilter === 'failed') return status === 'failed';
                return true;
            });
        }

        if (typeFilter !== 'all') {
            filtered = filtered.filter(p => {
                const type = p.content['@type'] || '';
                return type.toLowerCase().includes(typeFilter.toLowerCase());
            });
        }

        this.proposalList.render(filtered);
    }

    searchProposals(query) {
        if (!query) {
            this.filterProposals();
            return;
        }

        const filtered = this.proposals.filter(p => {
            const title = (p.content.title || '').toLowerCase();
            const description = (p.content.description || '').toLowerCase();
            const id = p.proposal_id.toString();
            const q = query.toLowerCase();
            return title.includes(q) || description.includes(q) || id.includes(q);
        });

        this.proposalList.render(filtered);
    }

    showProposalDetail(proposalId) {
        const proposal = this.proposals.find(p => p.proposal_id === proposalId);
        if (proposal) {
            this.proposalDetail.render(proposal);
            document.getElementById('proposalDetailSection').classList.add('active');
            document.getElementById('proposalsSection').classList.remove('active');
        }
    }

    backToProposals() {
        document.getElementById('proposalDetailSection').classList.remove('active');
        document.getElementById('proposalsSection').classList.add('active');
    }

    renderParameters() {
        const container = document.getElementById('parametersList');
        if (!container) return;

        const html = `
            <div class="parameters-section">
                <h3>Deposit Parameters</h3>
                <div class="param-card">
                    <div class="param-label">Minimum Deposit</div>
                    <div class="param-value">${this.formatTokens(this.parameters.deposit?.min_deposit)}</div>
                </div>
                <div class="param-card">
                    <div class="param-label">Max Deposit Period</div>
                    <div class="param-value">${this.formatDuration(this.parameters.deposit?.max_deposit_period)}</div>
                </div>
            </div>

            <div class="parameters-section">
                <h3>Voting Parameters</h3>
                <div class="param-card">
                    <div class="param-label">Voting Period</div>
                    <div class="param-value">${this.formatDuration(this.parameters.voting?.voting_period)}</div>
                </div>
            </div>

            <div class="parameters-section">
                <h3>Tally Parameters</h3>
                <div class="param-card">
                    <div class="param-label">Quorum</div>
                    <div class="param-value">${this.formatPercentage(this.parameters.tally?.quorum)}</div>
                </div>
                <div class="param-card">
                    <div class="param-label">Threshold</div>
                    <div class="param-value">${this.formatPercentage(this.parameters.tally?.threshold)}</div>
                </div>
                <div class="param-card">
                    <div class="param-label">Veto Threshold</div>
                    <div class="param-value">${this.formatPercentage(this.parameters.tally?.veto_threshold)}</div>
                </div>
            </div>
        `;

        container.innerHTML = html;
    }

    renderAnalytics() {
        const container = document.getElementById('analyticsContent');
        if (!container) return;

        const html = `
            <div class="analytics-grid">
                <div class="analytics-card">
                    <h3>Proposal Success Rate</h3>
                    <canvas id="successRateChart"></canvas>
                </div>
                <div class="analytics-card">
                    <h3>Voting Trends</h3>
                    <canvas id="votingTrendsChart"></canvas>
                </div>
                <div class="analytics-card">
                    <h3>Participation Over Time</h3>
                    <canvas id="participationChart"></canvas>
                </div>
                <div class="analytics-card">
                    <h3>Top Voters</h3>
                    <div id="topVotersList"></div>
                </div>
            </div>
        `;

        container.innerHTML = html;

        // Render charts
        setTimeout(() => {
            this.renderSuccessRateChart();
            this.renderVotingTrendsChart();
            this.renderParticipationChart();
            this.renderTopVoters();
        }, 100);
    }

    renderSuccessRateChart() {
        const ctx = document.getElementById('successRateChart');
        if (!ctx) return;

        const passed = this.proposals.filter(p => p.status === 'PASSED').length;
        const rejected = this.proposals.filter(p => p.status === 'REJECTED').length;
        const failed = this.proposals.filter(p => p.status === 'FAILED').length;

        new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Passed', 'Rejected', 'Failed'],
                datasets: [{
                    data: [passed, rejected, failed],
                    backgroundColor: ['#4CAF50', '#f44336', '#FF9800']
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false
            }
        });
    }

    renderVotingTrendsChart() {
        const ctx = document.getElementById('votingTrendsChart');
        if (!ctx) return;

        // Generate sample data
        const labels = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'];
        const data = labels.map(() => Math.floor(Math.random() * 100));

        new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Proposals Created',
                    data: data,
                    borderColor: '#2196F3',
                    backgroundColor: 'rgba(33, 150, 243, 0.1)',
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false
            }
        });
    }

    renderParticipationChart() {
        const ctx = document.getElementById('participationChart');
        if (!ctx) return;

        const labels = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'];
        const data = labels.map(() => Math.floor(Math.random() * 100));

        new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Participation Rate (%)',
                    data: data,
                    backgroundColor: '#9C27B0'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });
    }

    renderTopVoters() {
        const container = document.getElementById('topVotersList');
        if (!container) return;

        const topVoters = [
            { address: 'paw1abc...xyz', votes: 45 },
            { address: 'paw1def...uvw', votes: 38 },
            { address: 'paw1ghi...rst', votes: 32 },
            { address: 'paw1jkl...opq', votes: 28 },
            { address: 'paw1mno...lmn', votes: 24 }
        ];

        const html = topVoters.map((voter, index) => `
            <div class="voter-item">
                <span class="voter-rank">#${index + 1}</span>
                <span class="voter-address">${voter.address}</span>
                <span class="voter-votes">${voter.votes} votes</span>
            </div>
        `).join('');

        container.innerHTML = html;
    }

    async renderHistory() {
        const container = document.getElementById('votingHistory');
        if (!container) return;

        if (!this.walletConnected) {
            container.innerHTML = '<div class="empty-state">Connect your wallet to view voting history</div>';
            return;
        }

        try {
            const votes = await this.api.getUserVotes(this.walletAddress);

            if (votes.length === 0) {
                container.innerHTML = '<div class="empty-state">No voting history found</div>';
                return;
            }

            const html = votes.map(vote => {
                const proposal = this.proposals.find(p => p.proposal_id === vote.proposal_id);
                return `
                    <div class="history-item">
                        <div class="history-proposal">
                            <h4>Proposal #${vote.proposal_id}</h4>
                            <p>${proposal?.content.title || 'Unknown Proposal'}</p>
                        </div>
                        <div class="history-vote">
                            <span class="vote-option vote-${vote.option.toLowerCase()}">${vote.option}</span>
                            <span class="vote-date">${new Date(vote.timestamp).toLocaleDateString()}</span>
                        </div>
                    </div>
                `;
            }).join('');

            container.innerHTML = html;
        } catch (error) {
            console.error('Failed to load voting history:', error);
            container.innerHTML = '<div class="error-state">Failed to load voting history</div>';
        }
    }

    formatTokens(amount) {
        if (!amount || !amount[0]) return '0 PAW';
        return parseInt(amount[0].amount).toLocaleString() + ' ' + amount[0].denom.toUpperCase();
    }

    formatDuration(duration) {
        if (!duration) return 'N/A';
        const seconds = parseInt(duration.replace('s', ''));
        const days = Math.floor(seconds / 86400);
        return `${days} days`;
    }

    formatPercentage(value) {
        if (!value) return '0%';
        return (parseFloat(value) * 100).toFixed(2) + '%';
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type) {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check-circle' : 'exclamation-circle'}"></i>
            <span>${message}</span>
        `;
        document.body.appendChild(notification);

        setTimeout(() => {
            notification.classList.add('show');
        }, 100);

        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.governanceApp = new GovernanceApp();
});
