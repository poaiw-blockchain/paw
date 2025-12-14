// PAW Staking Dashboard - Main Application

import { StakingAPI } from './services/stakingAPI.js';
import { ValidatorList } from './components/ValidatorList.js';
import { ValidatorComparison } from './components/ValidatorComparison.js';
import { StakingCalculator } from './components/StakingCalculator.js';
import { DelegationPanel } from './components/DelegationPanel.js';
import { RewardsPanel } from './components/RewardsPanel.js';
import { PortfolioView } from './components/PortfolioView.js';
import { showToast, showLoading, hideLoading } from './utils/ui.js';

class StakingDashboard {
    constructor() {
        this.api = new StakingAPI();
        this.walletAddress = null;
        this.components = {};
        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.initializeComponents();
        await this.loadNetworkStats();
        this.checkWalletConnection();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => this.handleNavigation(e));
        });

        // Wallet Connection
        document.getElementById('connect-wallet')?.addEventListener('click', () => this.connectWallet());
        document.getElementById('disconnect-wallet')?.addEventListener('click', () => this.disconnectWallet());

        // Modal Close Buttons
        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', (e) => this.closeModal(e.target.closest('.modal')));
        });

        // Click outside modal to close
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal(modal);
                }
            });
        });

        // Refresh Portfolio
        document.getElementById('refresh-portfolio')?.addEventListener('click', () => {
            if (this.components.portfolio) {
                this.components.portfolio.refresh();
            }
        });
    }

    initializeComponents() {
        // Initialize all components
        this.components = {
            validatorList: new ValidatorList(this.api),
            validatorComparison: new ValidatorComparison(this.api),
            stakingCalculator: new StakingCalculator(this.api),
            delegationPanel: new DelegationPanel(this.api),
            rewardsPanel: new RewardsPanel(this.api),
            portfolio: new PortfolioView(this.api)
        };

        // Set up component event listeners
        this.components.validatorList.on('delegate', (validator) => this.showDelegationModal(validator));
        this.components.portfolio.on('claim-rewards', () => this.showRewardsModal());
        this.components.portfolio.on('delegate', (validator) => this.showDelegationModal(validator));
    }

    handleNavigation(e) {
        const view = e.currentTarget.dataset.view;

        // Update nav active state
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        e.currentTarget.classList.add('active');

        // Update view visibility
        document.querySelectorAll('.view-container').forEach(container => {
            container.classList.remove('active');
        });
        document.getElementById(`${view}-view`)?.classList.add('active');

        // Trigger component refresh if needed
        switch(view) {
            case 'validators':
                this.components.validatorList.render();
                break;
            case 'calculator':
                this.components.stakingCalculator.render();
                break;
            case 'comparison':
                this.components.validatorComparison.render();
                break;
            case 'portfolio':
                if (this.walletAddress) {
                    this.components.portfolio.render(this.walletAddress);
                }
                break;
        }
    }

    async loadNetworkStats() {
        try {
            const stats = await this.api.getNetworkStats();

            document.getElementById('total-staked').textContent =
                `${this.formatNumber(stats.totalStaked)} PAW`;
            document.getElementById('avg-apy').textContent =
                `${stats.averageAPY.toFixed(2)}%`;
            document.getElementById('active-validators').textContent =
                stats.activeValidators.toString();
            document.getElementById('inflation-rate').textContent =
                `${stats.inflationRate.toFixed(2)}%`;
        } catch (error) {
            console.error('Failed to load network stats:', error);
            showToast('Failed to load network statistics', 'error');
        }
    }

    async connectWallet() {
        try {
            showLoading('Connecting wallet...');

            // Check if Keplr is installed
            if (!window.keplr) {
                throw new Error('Keplr wallet not found. Please install Keplr extension.');
            }

            // Request chain add (if needed)
            await this.suggestChain();

            // Enable wallet
            await window.keplr.enable('paw-testnet');

            // Get offline signer
            const offlineSigner = window.keplr.getOfflineSigner('paw-testnet');
            const accounts = await offlineSigner.getAccounts();

            if (accounts.length === 0) {
                throw new Error('No accounts found in wallet');
            }

            this.walletAddress = accounts[0].address;

            // Update UI
            document.getElementById('connect-wallet').style.display = 'none';
            const walletConnected = document.getElementById('wallet-connected');
            walletConnected.style.display = 'flex';
            walletConnected.querySelector('.wallet-address').textContent =
                this.formatAddress(this.walletAddress);

            // Save to localStorage
            localStorage.setItem('paw_wallet_address', this.walletAddress);

            // Load portfolio
            await this.components.portfolio.render(this.walletAddress);

            hideLoading();
            showToast('Wallet connected successfully', 'success');
        } catch (error) {
            hideLoading();
            console.error('Wallet connection failed:', error);
            showToast(error.message || 'Failed to connect wallet', 'error');
        }
    }

    async suggestChain() {
        const chainConfig = {
            chainId: 'paw-testnet',
            chainName: 'PAW Testnet',
            rpc: 'http://localhost:26657',
            rest: 'http://localhost:1317',
            bip44: {
                coinType: 118,
            },
            bech32Config: {
                bech32PrefixAccAddr: 'paw',
                bech32PrefixAccPub: 'pawpub',
                bech32PrefixValAddr: 'pawvaloper',
                bech32PrefixValPub: 'pawvaloperpub',
                bech32PrefixConsAddr: 'pawvalcons',
                bech32PrefixConsPub: 'pawvalconspub',
            },
            currencies: [
                {
                    coinDenom: 'PAW',
                    coinMinimalDenom: 'upaw',
                    coinDecimals: 6,
                },
            ],
            feeCurrencies: [
                {
                    coinDenom: 'PAW',
                    coinMinimalDenom: 'upaw',
                    coinDecimals: 6,
                },
            ],
            stakeCurrency: {
                coinDenom: 'PAW',
                coinMinimalDenom: 'upaw',
                coinDecimals: 6,
            },
        };

        try {
            await window.keplr.experimentalSuggestChain(chainConfig);
        } catch (error) {
            console.error('Failed to suggest chain:', error);
        }
    }

    disconnectWallet() {
        this.walletAddress = null;
        localStorage.removeItem('paw_wallet_address');

        document.getElementById('connect-wallet').style.display = 'flex';
        document.getElementById('wallet-connected').style.display = 'none';

        // Clear portfolio view
        document.getElementById('portfolio-content').innerHTML =
            '<p class="text-center">Please connect your wallet to view your staking portfolio.</p>';

        showToast('Wallet disconnected', 'info');
    }

    checkWalletConnection() {
        const savedAddress = localStorage.getItem('paw_wallet_address');
        if (savedAddress && window.keplr) {
            this.connectWallet();
        }
    }

    showDelegationModal(validator) {
        if (!this.walletAddress) {
            showToast('Please connect your wallet first', 'error');
            return;
        }

        const modal = document.getElementById('delegation-modal');
        modal.classList.add('active');
        this.components.delegationPanel.render(validator, this.walletAddress);
    }

    showRewardsModal() {
        if (!this.walletAddress) {
            showToast('Please connect your wallet first', 'error');
            return;
        }

        const modal = document.getElementById('rewards-modal');
        modal.classList.add('active');
        this.components.rewardsPanel.render(this.walletAddress);
    }

    closeModal(modal) {
        modal.classList.remove('active');
    }

    formatAddress(address) {
        if (!address) return '';
        return `${address.slice(0, 10)}...${address.slice(-6)}`;
    }

    formatNumber(num) {
        if (num >= 1e9) return `${(num / 1e9).toFixed(2)}B`;
        if (num >= 1e6) return `${(num / 1e6).toFixed(2)}M`;
        if (num >= 1e3) return `${(num / 1e3).toFixed(2)}K`;
        return num.toFixed(2);
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.stakingDashboard = new StakingDashboard();
});

export default StakingDashboard;
