/**
 * CreateProposal Component
 * Form for creating new governance proposals
 */

class CreateProposal {
    constructor(api, app) {
        this.api = api;
        this.app = app;
    }

    render() {
        const container = document.getElementById('createProposalForm');
        if (!container) return;

        container.innerHTML = `
            <div class="create-proposal-container">
                ${!this.app.walletConnected ? `
                    <div class="warning-message">
                        <i class="fas fa-exclamation-triangle"></i>
                        <p>Please connect your wallet to create a proposal</p>
                        <button class="btn btn-primary" onclick="window.governanceApp.connectWallet()">
                            Connect Wallet
                        </button>
                    </div>
                ` : `
                    <form id="proposalForm" class="proposal-form">
                        <div class="form-section">
                            <h3>Proposal Type</h3>
                            <div class="proposal-types">
                                <label class="type-option">
                                    <input type="radio" name="proposalType" value="text" checked>
                                    <div class="type-card">
                                        <i class="fas fa-file-alt"></i>
                                        <h4>Text Proposal</h4>
                                        <p>General purpose proposal for signaling</p>
                                    </div>
                                </label>
                                <label class="type-option">
                                    <input type="radio" name="proposalType" value="parameter">
                                    <div class="type-card">
                                        <i class="fas fa-sliders-h"></i>
                                        <h4>Parameter Change</h4>
                                        <p>Modify blockchain parameters</p>
                                    </div>
                                </label>
                                <label class="type-option">
                                    <input type="radio" name="proposalType" value="upgrade">
                                    <div class="type-card">
                                        <i class="fas fa-arrow-up"></i>
                                        <h4>Software Upgrade</h4>
                                        <p>Schedule a network upgrade</p>
                                    </div>
                                </label>
                                <label class="type-option">
                                    <input type="radio" name="proposalType" value="community">
                                    <div class="type-card">
                                        <i class="fas fa-hand-holding-usd"></i>
                                        <h4>Community Pool Spend</h4>
                                        <p>Allocate community pool funds</p>
                                    </div>
                                </label>
                            </div>
                        </div>

                        <div class="form-section">
                            <h3>Basic Information</h3>

                            <div class="form-group">
                                <label for="proposalTitle">
                                    Title <span class="required">*</span>
                                </label>
                                <input
                                    type="text"
                                    id="proposalTitle"
                                    name="title"
                                    required
                                    maxlength="140"
                                    placeholder="Enter a clear, concise title"
                                >
                                <small class="form-hint">Maximum 140 characters</small>
                            </div>

                            <div class="form-group">
                                <label for="proposalDescription">
                                    Description <span class="required">*</span>
                                </label>
                                <textarea
                                    id="proposalDescription"
                                    name="description"
                                    required
                                    rows="10"
                                    placeholder="Provide a detailed description of your proposal, including rationale and expected impact"
                                ></textarea>
                                <small class="form-hint">Markdown formatting supported</small>
                            </div>
                        </div>

                        <div id="parameterChangeFields" class="form-section hidden">
                            <h3>Parameter Change Details</h3>

                            <div class="form-group">
                                <label for="paramSubspace">Subspace</label>
                                <select id="paramSubspace">
                                    <option value="">Select subspace...</option>
                                    <option value="staking">Staking</option>
                                    <option value="gov">Governance</option>
                                    <option value="distribution">Distribution</option>
                                    <option value="slashing">Slashing</option>
                                    <option value="mint">Mint</option>
                                </select>
                            </div>

                            <div class="form-group">
                                <label for="paramKey">Parameter Key</label>
                                <input type="text" id="paramKey" placeholder="e.g., MaxValidators">
                            </div>

                            <div class="form-group">
                                <label for="paramValue">New Value</label>
                                <input type="text" id="paramValue" placeholder="Enter new value">
                            </div>
                        </div>

                        <div id="upgradeFields" class="form-section hidden">
                            <h3>Software Upgrade Details</h3>

                            <div class="form-group">
                                <label for="upgradeName">Upgrade Name</label>
                                <input type="text" id="upgradeName" placeholder="e.g., v2.0.0">
                            </div>

                            <div class="form-group">
                                <label for="upgradeHeight">Upgrade Height</label>
                                <input type="number" id="upgradeHeight" placeholder="Target block height">
                            </div>

                            <div class="form-group">
                                <label for="upgradeInfo">Upgrade Info (JSON)</label>
                                <textarea id="upgradeInfo" rows="5" placeholder='{"binaries": {...}}'></textarea>
                            </div>
                        </div>

                        <div id="communitySpendFields" class="form-section hidden">
                            <h3>Community Pool Spend Details</h3>

                            <div class="form-group">
                                <label for="recipientAddress">Recipient Address</label>
                                <input type="text" id="recipientAddress" placeholder="paw1...">
                            </div>

                            <div class="form-group">
                                <label for="spendAmount">Amount (PAW)</label>
                                <input type="number" id="spendAmount" min="1" step="0.000001">
                            </div>
                        </div>

                        <div class="form-section">
                            <h3>Initial Deposit</h3>

                            <div class="form-group">
                                <label for="initialDeposit">
                                    Deposit Amount (PAW) <span class="required">*</span>
                                </label>
                                <input
                                    type="number"
                                    id="initialDeposit"
                                    name="deposit"
                                    required
                                    min="1"
                                    step="0.000001"
                                    placeholder="Minimum: 10,000 PAW"
                                >
                                <small class="form-hint">
                                    Minimum deposit: 10,000 PAW. Your deposit will be returned if the proposal passes.
                                </small>
                            </div>

                            <div class="deposit-info-box">
                                <i class="fas fa-info-circle"></i>
                                <div>
                                    <strong>About Deposits:</strong>
                                    <ul>
                                        <li>Deposits are required to prevent spam</li>
                                        <li>If minimum deposit is not reached, the proposal will be rejected</li>
                                        <li>Deposits are returned when proposal passes</li>
                                        <li>Deposits are burned if proposal is vetoed or rejected</li>
                                    </ul>
                                </div>
                            </div>
                        </div>

                        <div class="form-section">
                            <h3>Preview</h3>
                            <div id="proposalPreview" class="proposal-preview">
                                <p class="preview-empty">Fill out the form to see a preview</p>
                            </div>
                        </div>

                        <div class="form-actions">
                            <button type="button" class="btn btn-secondary" onclick="window.governanceApp.navigateToSection('proposals')">
                                Cancel
                            </button>
                            <button type="button" class="btn btn-primary" id="previewBtn">
                                <i class="fas fa-eye"></i> Preview
                            </button>
                            <button type="submit" class="btn btn-primary">
                                <i class="fas fa-paper-plane"></i> Submit Proposal
                            </button>
                        </div>
                    </form>
                `}
            </div>
        `;

        if (this.app.walletConnected) {
            this.attachEventListeners();
        }
    }

    attachEventListeners() {
        // Proposal type change
        document.querySelectorAll('input[name="proposalType"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.handleTypeChange(e.target.value);
            });
        });

        // Form inputs for live preview
        const form = document.getElementById('proposalForm');
        if (form) {
            form.addEventListener('input', () => {
                this.updatePreview();
            });

            form.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleSubmit();
            });
        }

        // Preview button
        const previewBtn = document.getElementById('previewBtn');
        if (previewBtn) {
            previewBtn.addEventListener('click', () => {
                this.updatePreview();
            });
        }
    }

    handleTypeChange(type) {
        // Hide all type-specific fields
        document.getElementById('parameterChangeFields').classList.add('hidden');
        document.getElementById('upgradeFields').classList.add('hidden');
        document.getElementById('communitySpendFields').classList.add('hidden');

        // Show relevant fields
        switch (type) {
            case 'parameter':
                document.getElementById('parameterChangeFields').classList.remove('hidden');
                break;
            case 'upgrade':
                document.getElementById('upgradeFields').classList.remove('hidden');
                break;
            case 'community':
                document.getElementById('communitySpendFields').classList.remove('hidden');
                break;
        }

        this.updatePreview();
    }

    updatePreview() {
        const title = document.getElementById('proposalTitle').value;
        const description = document.getElementById('proposalDescription').value;
        const type = document.querySelector('input[name="proposalType"]:checked').value;
        const deposit = document.getElementById('initialDeposit').value;

        const previewContainer = document.getElementById('proposalPreview');

        if (!title && !description) {
            previewContainer.innerHTML = '<p class="preview-empty">Fill out the form to see a preview</p>';
            return;
        }

        const typeLabels = {
            text: 'Text Proposal',
            parameter: 'Parameter Change Proposal',
            upgrade: 'Software Upgrade Proposal',
            community: 'Community Pool Spend Proposal'
        };

        let additionalInfo = '';
        if (type === 'parameter') {
            const subspace = document.getElementById('paramSubspace').value;
            const key = document.getElementById('paramKey').value;
            const value = document.getElementById('paramValue').value;
            if (subspace && key && value) {
                additionalInfo = `
                    <div class="preview-params">
                        <strong>Parameter Change:</strong>
                        <code>${subspace}.${key} = ${value}</code>
                    </div>
                `;
            }
        } else if (type === 'upgrade') {
            const name = document.getElementById('upgradeName').value;
            const height = document.getElementById('upgradeHeight').value;
            if (name && height) {
                additionalInfo = `
                    <div class="preview-params">
                        <strong>Upgrade:</strong> ${name} at block ${height}
                    </div>
                `;
            }
        } else if (type === 'community') {
            const recipient = document.getElementById('recipientAddress').value;
            const amount = document.getElementById('spendAmount').value;
            if (recipient && amount) {
                additionalInfo = `
                    <div class="preview-params">
                        <strong>Recipient:</strong> ${recipient}<br>
                        <strong>Amount:</strong> ${amount} PAW
                    </div>
                `;
            }
        }

        previewContainer.innerHTML = `
            <div class="preview-content">
                <div class="preview-type">${typeLabels[type]}</div>
                <h3>${title || 'Untitled Proposal'}</h3>
                <p>${description || 'No description provided'}</p>
                ${additionalInfo}
                <div class="preview-deposit">
                    <strong>Initial Deposit:</strong> ${deposit ? parseFloat(deposit).toLocaleString() + ' PAW' : 'Not specified'}
                </div>
            </div>
        `;
    }

    async handleSubmit() {
        const form = document.getElementById('proposalForm');
        if (!form.checkValidity()) {
            form.reportValidity();
            return;
        }

        const type = document.querySelector('input[name="proposalType"]:checked').value;
        const title = document.getElementById('proposalTitle').value;
        const description = document.getElementById('proposalDescription').value;
        const deposit = document.getElementById('initialDeposit').value;

        // Validate minimum deposit
        if (parseFloat(deposit) < 10000) {
            this.app.showError('Minimum deposit is 10,000 PAW');
            return;
        }

        // Build proposal content based on type
        let proposalContent = {
            title: title,
            description: description
        };

        switch (type) {
            case 'text':
                proposalContent['@type'] = '/cosmos.gov.v1beta1.TextProposal';
                break;

            case 'parameter':
                proposalContent['@type'] = '/cosmos.params.v1beta1.ParameterChangeProposal';
                proposalContent.changes = [{
                    subspace: document.getElementById('paramSubspace').value,
                    key: document.getElementById('paramKey').value,
                    value: document.getElementById('paramValue').value
                }];
                break;

            case 'upgrade':
                proposalContent['@type'] = '/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal';
                proposalContent.plan = {
                    name: document.getElementById('upgradeName').value,
                    height: document.getElementById('upgradeHeight').value,
                    info: document.getElementById('upgradeInfo').value
                };
                break;

            case 'community':
                proposalContent['@type'] = '/cosmos.distribution.v1beta1.CommunityPoolSpendProposal';
                proposalContent.recipient = document.getElementById('recipientAddress').value;
                proposalContent.amount = [{
                    denom: 'paw',
                    amount: String(parseFloat(document.getElementById('spendAmount').value) * 1000000)
                }];
                break;
        }

        const depositAmount = [{
            denom: 'paw',
            amount: String(parseFloat(deposit) * 1000000)
        }];

        try {
            const result = await this.api.submitProposal(
                proposalContent,
                depositAmount,
                this.app.walletAddress
            );

            this.app.showSuccess(`Proposal submitted successfully! ID: ${result.proposal_id}`);

            // Reset form
            form.reset();
            this.updatePreview();

            // Reload proposals
            await this.app.loadProposals();

            // Navigate to proposals list
            setTimeout(() => {
                this.app.navigateToSection('proposals');
            }, 2000);

        } catch (error) {
            console.error('Failed to submit proposal:', error);
            this.app.showError('Failed to submit proposal: ' + error.message);
        }
    }
}
