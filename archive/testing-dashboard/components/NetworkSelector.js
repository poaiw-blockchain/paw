/**
 * Network Selector Component
 * Handles network switching and status display
 */

import blockchainService from '../services/blockchain.js';
import monitoringService from '../services/monitoring.js';

class NetworkSelector {
    constructor() {
        this.currentNetwork = 'local';
        this.statusIndicator = null;
        this.statusText = null;
        this.networkSelect = null;
    }

    /**
     * Initialize the component
     */
    init() {
        this.networkSelect = document.getElementById('network-select');
        this.statusIndicator = document.getElementById('status-indicator');
        this.statusText = document.getElementById('status-text');

        // Set up event listeners
        this.networkSelect.addEventListener('change', (e) => this.handleNetworkChange(e));

        // Initial network check
        this.checkNetworkStatus();

        // Periodic status check
        setInterval(() => this.checkNetworkStatus(), 10000);
    }

    /**
     * Handle network selection change
     */
    async handleNetworkChange(event) {
        const network = event.target.value;

        // Confirm if switching to mainnet
        if (network === 'mainnet') {
            const confirmed = confirm(
                'WARNING: You are switching to Mainnet.\n\n' +
                'Mainnet operates in READ-ONLY mode in this dashboard.\n' +
                'Real transactions with real value cannot be sent.\n\n' +
                'Continue?'
            );

            if (!confirmed) {
                event.target.value = this.currentNetwork;
                return;
            }
        }

        this.updateStatus('connecting', 'Switching network...');

        try {
            await blockchainService.switchNetwork(network);
            this.currentNetwork = network;

            const networkInfo = blockchainService.getNetworkInfo();
            monitoringService.addLog('success', `Switched to ${networkInfo.name}`);

            // Restart monitoring with new network
            monitoringService.stopMonitoring();
            monitoringService.startMonitoring();

            // Update UI
            this.checkNetworkStatus();

            // Emit network change event
            window.dispatchEvent(new CustomEvent('networkChanged', {
                detail: { network, networkInfo }
            }));
        } catch (error) {
            monitoringService.addLog('error', `Failed to switch network: ${error.message}`);
            this.updateStatus('disconnected', 'Connection failed');
            event.target.value = this.currentNetwork;
        }
    }

    /**
     * Check network connection status
     */
    async checkNetworkStatus() {
        try {
            const result = await blockchainService.checkConnection();

            if (result.connected) {
                this.updateStatus('connected', 'Connected');
            } else {
                this.updateStatus('disconnected', result.error || 'Disconnected');
            }
        } catch (error) {
            this.updateStatus('disconnected', 'Connection error');
        }
    }

    /**
     * Update status indicator
     */
    updateStatus(status, text) {
        if (this.statusIndicator) {
            this.statusIndicator.className = 'status-indicator ' + status;
        }

        if (this.statusText) {
            this.statusText.textContent = text;
        }
    }

    /**
     * Get current network
     */
    getCurrentNetwork() {
        return this.currentNetwork;
    }
}

export default new NetworkSelector();
