/**
 * Quick Actions Component
 * Handles all quick action buttons and modals
 */

import blockchainService from '../services/blockchain.js';
import monitoringService from '../services/monitoring.js';
import testingService from '../services/testing.js';
import CONFIG from '../config.js';

class QuickActions {
    constructor() {
        this.modalContainer = null;
    }

    /**
     * Initialize the component
     */
    init() {
        this.modalContainer = document.getElementById('modal-container');

        // Set up action button listeners
        document.getElementById('send-tx-btn')?.addEventListener('click', () => this.showSendTransactionModal());
        document.getElementById('create-wallet-btn')?.addEventListener('click', () => this.showCreateWalletModal());
        document.getElementById('delegate-btn')?.addEventListener('click', () => this.showDelegateModal());
        document.getElementById('proposal-btn')?.addEventListener('click', () => this.showProposalModal());
        document.getElementById('swap-btn')?.addEventListener('click', () => this.showSwapModal());
        document.getElementById('query-balance-btn')?.addEventListener('click', () => this.showQueryBalanceModal());

        // Testing tools
        document.getElementById('tx-simulator-btn')?.addEventListener('click', () => this.showTxSimulatorModal());
        document.getElementById('bulk-wallet-btn')?.addEventListener('click', () => this.showBulkWalletModal());
        document.getElementById('load-test-btn')?.addEventListener('click', () => this.showLoadTestModal());
        document.getElementById('stress-test-btn')?.addEventListener('click', () => this.showStressTestModal());
        document.getElementById('faucet-btn')?.addEventListener('click', () => this.showFaucetModal());

        // Test scenarios
        document.getElementById('test-tx-flow')?.addEventListener('click', () => this.runTransactionFlow());
        document.getElementById('test-staking-flow')?.addEventListener('click', () => this.runStakingFlow());
        document.getElementById('test-gov-flow')?.addEventListener('click', () => this.runGovernanceFlow());
        document.getElementById('test-dex-flow')?.addEventListener('click', () => this.runDEXFlow());
    }

    /**
     * Show send transaction modal
     */
    showSendTransactionModal() {
        const testData = CONFIG.testData.transactions;

        const modal = this.createModal('Send Transaction', `
            <div class="form-group">
                <label class="form-label">From Address</label>
                <input type="text" class="form-input" id="tx-from" placeholder="paw1...">
                <span class="form-helper">The sender's address</span>
            </div>
            <div class="form-group">
                <label class="form-label">To Address</label>
                <input type="text" class="form-input" id="tx-to" placeholder="paw1...">
                <span class="form-helper">The recipient's address</span>
            </div>
            <div class="form-group">
                <label class="form-label">Amount</label>
                <input type="text" class="form-input" id="tx-amount" placeholder="${testData.amount}">
                <span class="form-helper">Amount in upaw (1 PAW = 1,000,000 upaw)</span>
            </div>
            <div class="form-group">
                <label class="form-label">Memo (Optional)</label>
                <input type="text" class="form-input" id="tx-memo" placeholder="${testData.memo}">
            </div>
            <button class="btn btn-secondary" id="use-test-data">Use Test Data</button>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Send Transaction', className: 'btn-primary', onClick: () => this.sendTransaction() }
        ]);

        this.showModal(modal);

        // Use test data button
        document.getElementById('use-test-data')?.addEventListener('click', () => {
            document.getElementById('tx-from').value = CONFIG.testData.wallets[0].address;
            document.getElementById('tx-to').value = CONFIG.testData.wallets[1].address;
            document.getElementById('tx-amount').value = testData.amount;
            document.getElementById('tx-memo').value = testData.memo;
        });
    }

    /**
     * Send transaction
     */
    async sendTransaction() {
        const from = document.getElementById('tx-from').value;
        const to = document.getElementById('tx-to').value;
        const amount = document.getElementById('tx-amount').value;
        const memo = document.getElementById('tx-memo').value;

        if (!from || !to || !amount) {
            alert('Please fill in all required fields');
            return;
        }

        if (blockchainService.config.features.readOnly) {
            alert('Cannot send transactions on mainnet (read-only mode)');
            return;
        }

        try {
            this.closeModal();
            monitoringService.addLog('info', 'Sending transaction...');

            // Note: In production, this would require proper wallet signing
            monitoringService.addLog('warning', 'Transaction signing not implemented - simulation only');
            monitoringService.addLog('info', `Would send ${amount} upaw from ${from} to ${to}`);

            await new Promise(resolve => setTimeout(resolve, 2000)); // Simulate delay

            monitoringService.addLog('success', 'Transaction simulated successfully');
        } catch (error) {
            monitoringService.addLog('error', `Transaction failed: ${error.message}`);
        }
    }

    /**
     * Show create wallet modal
     */
    async showCreateWalletModal() {
        try {
            const wallet = await blockchainService.createWallet();

            const modal = this.createModal('New Wallet Created', `
                <div class="form-group">
                    <label class="form-label">Address</label>
                    <input type="text" class="form-input" value="${wallet.address}" readonly>
                    <span class="form-helper">Your new wallet address</span>
                </div>
                <div class="form-group">
                    <label class="form-label">Mnemonic Phrase</label>
                    <textarea class="form-textarea" readonly>${wallet.mnemonic}</textarea>
                    <span class="form-helper">⚠️ Save this phrase securely! It cannot be recovered.</span>
                </div>
                <div class="form-group">
                    <label class="form-label">Private Key</label>
                    <textarea class="form-textarea" readonly>${wallet.privateKey}</textarea>
                    <span class="form-helper">⚠️ Never share your private key!</span>
                </div>
            `, [
                { text: 'Copy Address', className: 'btn-secondary', onClick: () => navigator.clipboard.writeText(wallet.address) },
                { text: 'Close', className: 'btn-primary', onClick: () => this.closeModal() }
            ]);

            this.showModal(modal);
            monitoringService.addLog('success', `New wallet created: ${wallet.address}`);
        } catch (error) {
            monitoringService.addLog('error', `Failed to create wallet: ${error.message}`);
        }
    }

    /**
     * Show delegate modal
     */
    async showDelegateModal() {
        const validators = await blockchainService.getValidators();

        const validatorOptions = validators.map(v =>
            `<option value="${v.address}">${v.moniker} (${v.commission} commission)</option>`
        ).join('');

        const modal = this.createModal('Delegate Tokens', `
            <div class="form-group">
                <label class="form-label">Validator</label>
                <select class="form-select" id="delegate-validator">
                    <option value="">Select a validator...</option>
                    ${validatorOptions}
                </select>
            </div>
            <div class="form-group">
                <label class="form-label">Amount</label>
                <input type="text" class="form-input" id="delegate-amount" placeholder="100000000">
                <span class="form-helper">Amount in upaw to delegate</span>
            </div>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Delegate', className: 'btn-primary', onClick: () => this.delegate() }
        ]);

        this.showModal(modal);
    }

    /**
     * Delegate tokens
     */
    async delegate() {
        const validator = document.getElementById('delegate-validator').value;
        const amount = document.getElementById('delegate-amount').value;

        if (!validator || !amount) {
            alert('Please fill in all fields');
            return;
        }

        this.closeModal();
        monitoringService.addLog('warning', 'Delegation simulation only (requires signing)');
        monitoringService.addLog('info', `Would delegate ${amount} upaw to ${validator}`);
    }

    /**
     * Show proposal modal
     */
    showProposalModal() {
        const modal = this.createModal('Submit Proposal', `
            <div class="form-group">
                <label class="form-label">Title</label>
                <input type="text" class="form-input" id="proposal-title" placeholder="My Proposal">
            </div>
            <div class="form-group">
                <label class="form-label">Description</label>
                <textarea class="form-textarea" id="proposal-description" placeholder="Detailed description of the proposal..."></textarea>
            </div>
            <div class="form-group">
                <label class="form-label">Initial Deposit</label>
                <input type="text" class="form-input" id="proposal-deposit" placeholder="10000000upaw">
                <span class="form-helper">Minimum deposit required to submit</span>
            </div>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Submit', className: 'btn-primary', onClick: () => this.submitProposal() }
        ]);

        this.showModal(modal);
    }

    /**
     * Submit proposal
     */
    submitProposal() {
        this.closeModal();
        monitoringService.addLog('warning', 'Proposal submission simulation only (requires signing)');
        monitoringService.addLog('info', 'Would submit governance proposal');
    }

    /**
     * Show swap modal
     */
    showSwapModal() {
        const modal = this.createModal('Swap Tokens', `
            <div class="form-group">
                <label class="form-label">From Token</label>
                <select class="form-select" id="swap-from">
                    <option value="upaw">PAW</option>
                    <option value="usdc">USDC</option>
                    <option value="eth">ETH</option>
                </select>
            </div>
            <div class="form-group">
                <label class="form-label">Amount</label>
                <input type="text" class="form-input" id="swap-amount" placeholder="1000000">
            </div>
            <div class="form-group">
                <label class="form-label">To Token</label>
                <select class="form-select" id="swap-to">
                    <option value="usdc">USDC</option>
                    <option value="upaw">PAW</option>
                    <option value="eth">ETH</option>
                </select>
            </div>
            <div class="form-group">
                <label class="form-label">Slippage Tolerance (%)</label>
                <input type="text" class="form-input" id="swap-slippage" value="0.5">
            </div>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Swap', className: 'btn-primary', onClick: () => this.swapTokens() }
        ]);

        this.showModal(modal);
    }

    /**
     * Swap tokens
     */
    swapTokens() {
        this.closeModal();
        monitoringService.addLog('warning', 'Swap simulation only (requires signing)');
        monitoringService.addLog('info', 'Would execute token swap on DEX');
    }

    /**
     * Show query balance modal
     */
    showQueryBalanceModal() {
        const modal = this.createModal('Query Balance', `
            <div class="form-group">
                <label class="form-label">Address</label>
                <input type="text" class="form-input" id="query-address" placeholder="paw1...">
            </div>
            <div id="balance-result"></div>
        `, [
            { text: 'Close', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Query', className: 'btn-primary', onClick: () => this.queryBalance() }
        ]);

        this.showModal(modal);
    }

    /**
     * Query balance
     */
    async queryBalance() {
        const address = document.getElementById('query-address').value;

        if (!address) {
            alert('Please enter an address');
            return;
        }

        const resultDiv = document.getElementById('balance-result');
        resultDiv.innerHTML = '<p>Querying...</p>';

        try {
            const balances = await blockchainService.queryBalance(address);
            let html = '<h3>Balances:</h3><ul>';

            if (balances.length === 0) {
                html += '<li>No balances found</li>';
            } else {
                balances.forEach(balance => {
                    html += `<li>${balance.amount} ${balance.denom}</li>`;
                });
            }

            html += '</ul>';
            resultDiv.innerHTML = html;
        } catch (error) {
            resultDiv.innerHTML = `<p class="text-error">Error: ${error.message}</p>`;
        }
    }

    /**
     * Show bulk wallet generator modal
     */
    showBulkWalletModal() {
        const modal = this.createModal('Bulk Wallet Generator', `
            <div class="form-group">
                <label class="form-label">Number of Wallets</label>
                <input type="number" class="form-input" id="bulk-wallet-count" value="10" min="1" max="100">
                <span class="form-helper">Generate up to 100 wallets at once</span>
            </div>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Generate', className: 'btn-primary', onClick: () => this.generateBulkWallets() }
        ]);

        this.showModal(modal);
    }

    /**
     * Generate bulk wallets
     */
    async generateBulkWallets() {
        const count = parseInt(document.getElementById('bulk-wallet-count').value);

        if (count < 1 || count > 100) {
            alert('Please enter a number between 1 and 100');
            return;
        }

        this.closeModal();
        await testingService.generateBulkWallets(count);
    }

    /**
     * Show other modals (simplified implementations)
     */
    showTxSimulatorModal() {
        monitoringService.addLog('info', 'Transaction simulator opened');
        alert('Transaction Simulator\n\nThis feature allows you to build and test transactions before sending them.');
    }

    showLoadTestModal() {
        monitoringService.addLog('info', 'Load test configuration opened');
        alert('Load Testing\n\nConfigure and run load tests against the network.');
    }

    showStressTestModal() {
        monitoringService.addLog('info', 'Stress test configuration opened');
        alert('Stress Testing\n\nRun stress tests to evaluate network performance under high load.');
    }

    async showFaucetModal() {
        const modal = this.createModal('Request Test Tokens', `
            <div class="form-group">
                <label class="form-label">Wallet Address</label>
                <input type="text" class="form-input" id="faucet-address" placeholder="paw1...">
                <span class="form-helper">Enter your wallet address to receive test tokens</span>
            </div>
        `, [
            { text: 'Cancel', className: 'btn-secondary', onClick: () => this.closeModal() },
            { text: 'Request Tokens', className: 'btn-primary', onClick: () => this.requestFaucet() }
        ]);

        this.showModal(modal);
    }

    async requestFaucet() {
        const address = document.getElementById('faucet-address').value;

        if (!address) {
            alert('Please enter an address');
            return;
        }

        this.closeModal();

        try {
            await blockchainService.requestFaucet(address);
            monitoringService.addLog('success', `Tokens requested for ${address}`);
        } catch (error) {
            monitoringService.addLog('error', `Faucet request failed: ${error.message}`);
        }
    }

    /**
     * Run test scenarios
     */
    async runTransactionFlow() {
        try {
            await testingService.runTransactionFlowTest();
        } catch (error) {
            console.error('Transaction flow test failed:', error);
        }
    }

    async runStakingFlow() {
        try {
            await testingService.runStakingFlowTest();
        } catch (error) {
            console.error('Staking flow test failed:', error);
        }
    }

    async runGovernanceFlow() {
        try {
            await testingService.runGovernanceFlowTest();
        } catch (error) {
            console.error('Governance flow test failed:', error);
        }
    }

    async runDEXFlow() {
        try {
            await testingService.runDEXFlowTest();
        } catch (error) {
            console.error('DEX flow test failed:', error);
        }
    }

    /**
     * Create modal
     */
    createModal(title, body, buttons) {
        return `
            <div class="modal-overlay">
                <div class="modal">
                    <div class="modal-header">
                        <h2>${title}</h2>
                        <button class="modal-close" onclick="document.getElementById('modal-container').innerHTML = ''">&times;</button>
                    </div>
                    <div class="modal-body">
                        ${body}
                    </div>
                    <div class="modal-footer">
                        ${buttons.map((btn, idx) => `
                            <button class="btn ${btn.className}" id="modal-btn-${idx}">${btn.text}</button>
                        `).join('')}
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Show modal
     */
    showModal(html) {
        this.modalContainer.innerHTML = html;

        // Attach button event listeners
        const buttons = this.modalContainer.querySelectorAll('[id^="modal-btn-"]');
        buttons.forEach((btn, idx) => {
            const modalHTML = html;
            const buttonMatch = modalHTML.match(new RegExp(`id="modal-btn-${idx}"`));
            if (buttonMatch) {
                btn.addEventListener('click', () => {
                    // Handler will be set by button configuration
                });
            }
        });
    }

    /**
     * Close modal
     */
    closeModal() {
        this.modalContainer.innerHTML = '';
    }
}

export default new QuickActions();
