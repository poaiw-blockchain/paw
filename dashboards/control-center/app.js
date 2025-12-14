/**
 * PAW Testing Control Panel - Main Application
 * Coordinates all components and services
 */

import CONFIG from './config.js';
import blockchainService from './services/blockchain.js';
import monitoringService from './services/monitoring.js';
import networkSelector from './components/NetworkSelector.js';
import quickActions from './components/QuickActions.js';
import logViewer from './components/LogViewer.js';
import metricsDisplay from './components/MetricsDisplay.js';

class App {
    constructor() {
        this.currentTab = 'blocks';
        this.refreshIntervals = {};
    }

    /**
     * Initialize the application
     */
    async init() {
        console.log('Initializing PAW Testing Control Panel...');

        // Initialize components
        networkSelector.init();
        quickActions.init();
        logViewer.init();
        metricsDisplay.init();

        // Set up UI event listeners
        this.setupTabNavigation();
        this.setupThemeToggle();
        this.setupHelpButton();

        // Load initial data
        await this.loadInitialData();

        // Start monitoring
        monitoringService.startMonitoring();

        // Set up auto-refresh for active tab
        this.startAutoRefresh();

        // Listen for network changes
        window.addEventListener('networkChanged', () => {
            this.loadInitialData();
        });

        console.log('Application initialized successfully');
    }

    /**
     * Set up tab navigation
     */
    setupTabNavigation() {
        const tabButtons = document.querySelectorAll('.tab-btn');
        const tabPanes = document.querySelectorAll('.tab-pane');

        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                const tabName = button.getAttribute('data-tab');

                // Update active states
                tabButtons.forEach(btn => btn.classList.remove('active'));
                tabPanes.forEach(pane => pane.classList.remove('active'));

                button.classList.add('active');
                document.getElementById(tabName + '-pane')?.classList.add('active');

                // Update current tab and load data
                this.currentTab = tabName;
                this.loadTabData(tabName);
            });
        });
    }

    /**
     * Set up theme toggle
     */
    setupThemeToggle() {
        const themeToggleBtn = document.getElementById('theme-toggle-btn');
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

        // Set initial theme
        const savedTheme = localStorage.getItem('theme') || (prefersDark ? 'dark' : 'light');
        document.documentElement.setAttribute('data-theme', savedTheme);
        this.updateThemeIcon(savedTheme);

        // Toggle theme on button click
        themeToggleBtn?.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';

            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            this.updateThemeIcon(newTheme);
        });
    }

    /**
     * Update theme icon
     */
    updateThemeIcon(theme) {
        const icon = document.querySelector('#theme-toggle-btn i');
        if (icon) {
            icon.className = theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
        }
    }

    /**
     * Set up help button
     */
    setupHelpButton() {
        const helpBtn = document.getElementById('help-btn');

        helpBtn?.addEventListener('click', () => {
            this.showHelpModal();
        });
    }

    /**
     * Show help modal
     */
    showHelpModal() {
        const modalContainer = document.getElementById('modal-container');
        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h2>Help & Documentation</h2>
                        <button class="modal-close" onclick="document.getElementById('modal-container').innerHTML = ''">&times;</button>
                    </div>
                    <div class="modal-body">
                        <h3>Quick Start Guide</h3>
                        <ol>
                            <li><strong>Select Network:</strong> Use the network dropdown to switch between Local Testnet, Public Testnet, or Mainnet</li>
                            <li><strong>Check Status:</strong> The green indicator shows you're connected</li>
                            <li><strong>Quick Actions:</strong> Use the left sidebar for common tasks like sending transactions or creating wallets</li>
                            <li><strong>Test Scenarios:</strong> Run pre-built test flows to test complete features</li>
                            <li><strong>Monitor:</strong> Watch live logs and events in the right sidebar</li>
                        </ol>

                        <h3>Common Tasks</h3>
                        <ul>
                            <li><strong>Create Wallet:</strong> Click "Create Wallet" and save the mnemonic phrase securely</li>
                            <li><strong>Get Test Tokens:</strong> Click "Request Tokens" and enter your wallet address</li>
                            <li><strong>Send Transaction:</strong> Click "Send Transaction" and fill in the form (or use test data)</li>
                            <li><strong>Check Balance:</strong> Click "Query Balance" and enter any wallet address</li>
                        </ul>

                        <h3>Test Scenarios</h3>
                        <p>Pre-built test flows that run automatically:</p>
                        <ul>
                            <li><strong>Transaction Flow:</strong> Creates wallet, requests tokens, sends transaction</li>
                            <li><strong>Staking Flow:</strong> Fetches validators, simulates delegation</li>
                            <li><strong>Governance Flow:</strong> Lists proposals, simulates voting</li>
                            <li><strong>DEX Trading Flow:</strong> Shows pools, simulates swap</li>
                        </ul>

                        <h3>Troubleshooting</h3>
                        <ul>
                            <li><strong>Red Status Indicator:</strong> Check if your local node is running</li>
                            <li><strong>No Data Loading:</strong> Try switching networks or refreshing</li>
                            <li><strong>Transaction Failed:</strong> Check the logs for error messages</li>
                        </ul>

                        <h3>Need More Help?</h3>
                        <p>Visit the <a href="https://docs.paw.network" target="_blank">PAW Documentation</a> for detailed guides and API references.</p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-primary" onclick="document.getElementById('modal-container').innerHTML = ''">Got it!</button>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Load initial data
     */
    async loadInitialData() {
        monitoringService.addLog('info', 'Loading initial data...');

        try {
            // Load data for current tab
            await this.loadTabData(this.currentTab);

            monitoringService.addLog('success', 'Initial data loaded');
        } catch (error) {
            monitoringService.addLog('error', `Failed to load initial data: ${error.message}`);
        }
    }

    /**
     * Load data for specific tab
     */
    async loadTabData(tabName) {
        try {
            switch (tabName) {
                case 'blocks':
                    await this.loadBlocks();
                    break;
                case 'transactions':
                    await this.loadTransactions();
                    break;
                case 'validators':
                    await this.loadValidators();
                    break;
                case 'proposals':
                    await this.loadProposals();
                    break;
                case 'liquidity':
                    await this.loadLiquidityPools();
                    break;
            }
        } catch (error) {
            console.error(`Error loading ${tabName} data:`, error);
        }
    }

    /**
     * Load blocks
     */
    async loadBlocks() {
        const tbody = document.getElementById('blocks-table-body');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="5" class="loading">Loading blocks...</td></tr>';

        try {
            const blocks = await blockchainService.getRecentBlocks(10);

            if (blocks.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="loading">No blocks found</td></tr>';
                return;
            }

            tbody.innerHTML = blocks.map(block => `
                <tr>
                    <td>${block.height}</td>
                    <td>${block.hash}</td>
                    <td>${block.proposer}</td>
                    <td>${block.txCount}</td>
                    <td>${block.time}</td>
                </tr>
            `).join('');
        } catch (error) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Error loading blocks</td></tr>';
        }
    }

    /**
     * Load transactions
     */
    async loadTransactions() {
        const tbody = document.getElementById('transactions-table-body');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="6" class="loading">Loading transactions...</td></tr>';

        try {
            const txs = await blockchainService.getRecentTransactions(10);

            if (txs.length === 0) {
                tbody.innerHTML = '<tr><td colspan="6" class="loading">No transactions found</td></tr>';
                return;
            }

            tbody.innerHTML = txs.map(tx => `
                <tr>
                    <td>${tx.hash}</td>
                    <td>${tx.type}</td>
                    <td>-</td>
                    <td>-</td>
                    <td>-</td>
                    <td><span class="badge badge-${tx.status === 'Success' ? 'success' : 'error'}">${tx.status}</span></td>
                </tr>
            `).join('');
        } catch (error) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">Error loading transactions</td></tr>';
        }
    }

    /**
     * Load validators
     */
    async loadValidators() {
        const tbody = document.getElementById('validators-table-body');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="5" class="loading">Loading validators...</td></tr>';

        try {
            const validators = await blockchainService.getValidators();
            const stakingInfo = await blockchainService.getStakingInfo();

            // Update stats
            if (stakingInfo) {
                const totalStaked = document.getElementById('total-staked');
                if (totalStaked) {
                    totalStaked.textContent = this.formatTokenAmount(stakingInfo.bondedTokens);
                }
            }

            const activeValidators = document.getElementById('active-validators');
            if (activeValidators) {
                activeValidators.textContent = validators.filter(v => v.status === 'Active').length;
            }

            if (validators.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="loading">No validators found</td></tr>';
                return;
            }

            tbody.innerHTML = validators.map(val => `
                <tr>
                    <td>${val.moniker}</td>
                    <td><span class="badge badge-${val.status === 'Active' ? 'success' : 'warning'}">${val.status}</span></td>
                    <td>${this.formatTokenAmount(val.votingPower)}</td>
                    <td>${val.commission}</td>
                    <td><button class="btn btn-sm btn-primary">Delegate</button></td>
                </tr>
            `).join('');
        } catch (error) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Error loading validators</td></tr>';
        }
    }

    /**
     * Load proposals
     */
    async loadProposals() {
        const tbody = document.getElementById('proposals-table-body');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="5" class="loading">Loading proposals...</td></tr>';

        try {
            const proposals = await blockchainService.getProposals();

            if (proposals.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="loading">No proposals found</td></tr>';
                return;
            }

            tbody.innerHTML = proposals.map(prop => `
                <tr>
                    <td>${prop.id}</td>
                    <td>${prop.title}</td>
                    <td><span class="badge badge-info">${prop.status}</span></td>
                    <td>${prop.votingEndTime}</td>
                    <td><button class="btn btn-sm btn-primary">Vote</button></td>
                </tr>
            `).join('');
        } catch (error) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Error loading proposals</td></tr>';
        }
    }

    /**
     * Load liquidity pools
     */
    async loadLiquidityPools() {
        const tbody = document.getElementById('liquidity-table-body');
        if (!tbody) return;

        tbody.innerHTML = '<tr><td colspan="5" class="loading">Loading liquidity pools...</td></tr>';

        try {
            const pools = await blockchainService.getLiquidityPools();

            if (pools.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" class="loading">No pools found</td></tr>';
                return;
            }

            tbody.innerHTML = pools.map(pool => `
                <tr>
                    <td>${pool.id}</td>
                    <td>${pool.tokenPair}</td>
                    <td>$${pool.liquidity}</td>
                    <td>$${pool.volume24h}</td>
                    <td>
                        <button class="btn btn-sm btn-primary">Add Liquidity</button>
                        <button class="btn btn-sm btn-secondary">Swap</button>
                    </td>
                </tr>
            `).join('');
        } catch (error) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Error loading pools</td></tr>';
        }
    }

    /**
     * Start auto-refresh
     */
    startAutoRefresh() {
        if (!CONFIG.ui.autoRefresh) return;

        // Refresh current tab every 10 seconds
        this.refreshIntervals.tab = setInterval(() => {
            this.loadTabData(this.currentTab);
        }, 10000);
    }

    /**
     * Format token amount
     */
    formatTokenAmount(amount) {
        const num = parseInt(amount) / 1000000; // Convert upaw to PAW
        if (num >= 1000000) {
            return (num / 1000000).toFixed(2) + 'M PAW';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(2) + 'K PAW';
        }
        return num.toFixed(2) + ' PAW';
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    const app = new App();
    app.init();

    // Make app globally accessible for debugging
    window.pawApp = app;
});

export default App;
