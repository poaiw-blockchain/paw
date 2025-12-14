/**
 * ProposalDetail Component
 * Displays detailed information about a specific proposal
 */

class ProposalDetail {
    constructor(api, app) {
        this.api = api;
        this.app = app;
        this.currentProposal = null;
    }

    async render(proposal) {
        this.currentProposal = proposal;
        const container = document.getElementById('proposalDetail');
        if (!container) return;

        // Load additional data
        const [votes, deposits, tally] = await Promise.all([
            this.api.getProposalVotes(proposal.proposal_id),
            this.api.getProposalDeposits(proposal.proposal_id),
            this.api.getProposalTally(proposal.proposal_id)
        ]);

        const status = this.getStatusInfo(proposal.status);
        const type = this.getProposalType(proposal.content['@type']);

        container.innerHTML = `
            <div class="proposal-detail-container">
                <div class="detail-header">
                    <button class="btn btn-back" onclick="window.governanceApp.backToProposals()">
                        <i class="fas fa-arrow-left"></i> Back to Proposals
                    </button>
                    <div class="detail-meta">
                        <span class="detail-id">Proposal #${proposal.proposal_id}</span>
                        <span class="detail-status ${status.class}">
                            <i class="${status.icon}"></i> ${status.text}
                        </span>
                        <span class="detail-type">
                            <i class="${type.icon}"></i> ${type.text}
                        </span>
                    </div>
                </div>

                <div class="detail-content">
                    <h1 class="detail-title">${this.escapeHtml(proposal.content.title)}</h1>

                    <div class="detail-timeline">
                        ${this.renderTimeline(proposal)}
                    </div>

                    <div class="detail-description">
                        <h3>Description</h3>
                        <div class="description-content">
                            ${this.formatDescription(proposal.content.description)}
                        </div>
                    </div>

                    ${proposal.status === 'VOTING_PERIOD' || proposal.status === 'PASSED' || proposal.status === 'REJECTED' || proposal.status === 'FAILED' ? `
                        <div class="detail-voting">
                            <h3>Voting Results</h3>
                            <div class="voting-container">
                                <div class="tally-chart-container">
                                    <canvas id="proposalTallyChart"></canvas>
                                </div>
                                <div class="tally-stats">
                                    ${this.renderTallyStats(tally)}
                                </div>
                            </div>
                            ${proposal.status === 'VOTING_PERIOD' && this.app.walletConnected ? `
                                <div class="voting-actions">
                                    <button class="btn btn-vote btn-yes" data-vote="VOTE_OPTION_YES">
                                        <i class="fas fa-thumbs-up"></i> Vote Yes
                                    </button>
                                    <button class="btn btn-vote btn-no" data-vote="VOTE_OPTION_NO">
                                        <i class="fas fa-thumbs-down"></i> Vote No
                                    </button>
                                    <button class="btn btn-vote btn-abstain" data-vote="VOTE_OPTION_ABSTAIN">
                                        <i class="fas fa-minus"></i> Abstain
                                    </button>
                                    <button class="btn btn-vote btn-veto" data-vote="VOTE_OPTION_NO_WITH_VETO">
                                        <i class="fas fa-ban"></i> No With Veto
                                    </button>
                                </div>
                            ` : ''}
                        </div>
                    ` : ''}

                    ${proposal.status === 'DEPOSIT_PERIOD' ? `
                        <div class="detail-deposits">
                            <h3>Deposits</h3>
                            <div class="deposit-info">
                                <div class="deposit-progress-bar">
                                    ${this.renderDepositProgress(proposal, deposits)}
                                </div>
                                ${this.app.walletConnected ? `
                                    <button class="btn btn-primary btn-deposit" data-proposal-id="${proposal.proposal_id}">
                                        <i class="fas fa-coins"></i> Add Deposit
                                    </button>
                                ` : ''}
                            </div>
                            <div class="deposits-list">
                                ${this.renderDepositsList(deposits)}
                            </div>
                        </div>
                    ` : ''}

                    ${votes.length > 0 ? `
                        <div class="detail-votes">
                            <h3>Recent Votes (${votes.length})</h3>
                            <div class="votes-list">
                                ${this.renderVotesList(votes.slice(0, 10))}
                            </div>
                            ${votes.length > 10 ? `
                                <button class="btn btn-secondary" id="showAllVotes">
                                    Show All ${votes.length} Votes
                                </button>
                            ` : ''}
                        </div>
                    ` : ''}
                </div>
            </div>
        `;

        // Render tally chart
        if (proposal.status === 'VOTING_PERIOD' || proposal.status === 'PASSED' || proposal.status === 'REJECTED' || proposal.status === 'FAILED') {
            setTimeout(() => {
                this.app.tallyChart.render('proposalTallyChart', tally);
            }, 100);
        }

        // Attach event listeners
        this.attachEventListeners();
    }

    renderTimeline(proposal) {
        const events = [];

        // Submit event
        events.push({
            icon: 'fa-plus-circle',
            label: 'Submitted',
            date: proposal.submit_time,
            status: 'complete'
        });

        // Deposit end event
        if (proposal.status !== 'DEPOSIT_PERIOD') {
            events.push({
                icon: 'fa-coins',
                label: 'Deposit Period Ended',
                date: proposal.deposit_end_time,
                status: 'complete'
            });
        } else {
            events.push({
                icon: 'fa-coins',
                label: 'Deposit Period Ends',
                date: proposal.deposit_end_time,
                status: 'pending'
            });
        }

        // Voting events
        if (proposal.status === 'VOTING_PERIOD') {
            events.push({
                icon: 'fa-vote-yea',
                label: 'Voting Started',
                date: proposal.voting_start_time,
                status: 'complete'
            });
            events.push({
                icon: 'fa-flag-checkered',
                label: 'Voting Ends',
                date: proposal.voting_end_time,
                status: 'pending'
            });
        } else if (proposal.status === 'PASSED' || proposal.status === 'REJECTED' || proposal.status === 'FAILED') {
            events.push({
                icon: 'fa-vote-yea',
                label: 'Voting Period',
                date: proposal.voting_start_time,
                status: 'complete'
            });
            events.push({
                icon: proposal.status === 'PASSED' ? 'fa-check-circle' : 'fa-times-circle',
                label: proposal.status.charAt(0) + proposal.status.slice(1).toLowerCase(),
                date: proposal.voting_end_time,
                status: 'complete'
            });
        }

        return `
            <div class="timeline">
                ${events.map(event => `
                    <div class="timeline-event ${event.status}">
                        <div class="timeline-icon">
                            <i class="fas ${event.icon}"></i>
                        </div>
                        <div class="timeline-content">
                            <div class="timeline-label">${event.label}</div>
                            <div class="timeline-date">${this.formatDateTime(event.date)}</div>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;
    }

    renderTallyStats(tally) {
        const yes = parseInt(tally.yes || 0);
        const no = parseInt(tally.no || 0);
        const abstain = parseInt(tally.abstain || 0);
        const veto = parseInt(tally.no_with_veto || 0);
        const total = yes + no + abstain + veto;

        return `
            <div class="stat-item stat-yes">
                <div class="stat-label">Yes</div>
                <div class="stat-value">${this.formatVotingPower(yes)}</div>
                <div class="stat-percent">${total > 0 ? ((yes / total) * 100).toFixed(2) : 0}%</div>
            </div>
            <div class="stat-item stat-no">
                <div class="stat-label">No</div>
                <div class="stat-value">${this.formatVotingPower(no)}</div>
                <div class="stat-percent">${total > 0 ? ((no / total) * 100).toFixed(2) : 0}%</div>
            </div>
            <div class="stat-item stat-abstain">
                <div class="stat-label">Abstain</div>
                <div class="stat-value">${this.formatVotingPower(abstain)}</div>
                <div class="stat-percent">${total > 0 ? ((abstain / total) * 100).toFixed(2) : 0}%</div>
            </div>
            <div class="stat-item stat-veto">
                <div class="stat-label">No With Veto</div>
                <div class="stat-value">${this.formatVotingPower(veto)}</div>
                <div class="stat-percent">${total > 0 ? ((veto / total) * 100).toFixed(2) : 0}%</div>
            </div>
            <div class="stat-item stat-total">
                <div class="stat-label">Total Votes</div>
                <div class="stat-value">${this.formatVotingPower(total)}</div>
            </div>
        `;
    }

    renderDepositProgress(proposal, deposits) {
        const minDeposit = 10000000; // From parameters
        const currentDeposit = deposits.reduce((sum, d) => {
            return sum + parseInt(d.amount[0].amount);
        }, 0);
        const percentage = Math.min((currentDeposit / minDeposit) * 100, 100);

        return `
            <div class="progress-container">
                <div class="progress-bar-deposit">
                    <div class="progress-fill" style="width: ${percentage}%"></div>
                </div>
                <div class="progress-text">
                    ${this.formatVotingPower(currentDeposit)} / ${this.formatVotingPower(minDeposit)} PAW
                    (${percentage.toFixed(1)}%)
                </div>
            </div>
        `;
    }

    renderDepositsList(deposits) {
        if (deposits.length === 0) {
            return '<div class="empty-state">No deposits yet</div>';
        }

        return deposits.map(deposit => `
            <div class="deposit-item">
                <div class="deposit-address">${this.truncateAddress(deposit.depositor)}</div>
                <div class="deposit-amount">${this.formatVotingPower(deposit.amount[0].amount)} PAW</div>
                <div class="deposit-date">${this.formatDateTime(deposit.timestamp)}</div>
            </div>
        `).join('');
    }

    renderVotesList(votes) {
        return votes.map(vote => `
            <div class="vote-item">
                <div class="vote-address">${this.truncateAddress(vote.voter)}</div>
                <div class="vote-option vote-${this.getVoteClass(vote.option)}">
                    ${this.getVoteLabel(vote.option)}
                </div>
                <div class="vote-date">${this.formatDateTime(vote.timestamp)}</div>
            </div>
        `).join('');
    }

    getStatusInfo(status) {
        const statusMap = {
            'VOTING_PERIOD': { text: 'Voting', class: 'status-voting', icon: 'fas fa-vote-yea' },
            'DEPOSIT_PERIOD': { text: 'Deposit Period', class: 'status-deposit', icon: 'fas fa-coins' },
            'PASSED': { text: 'Passed', class: 'status-passed', icon: 'fas fa-check-circle' },
            'REJECTED': { text: 'Rejected', class: 'status-rejected', icon: 'fas fa-times-circle' },
            'FAILED': { text: 'Failed', class: 'status-failed', icon: 'fas fa-exclamation-circle' }
        };
        return statusMap[status] || { text: status, class: 'status-unknown', icon: 'fas fa-question-circle' };
    }

    getProposalType(typeString) {
        if (!typeString) return { text: 'Unknown', icon: 'fas fa-file' };
        if (typeString.includes('TextProposal')) return { text: 'Text Proposal', icon: 'fas fa-file-alt' };
        if (typeString.includes('ParameterChange')) return { text: 'Parameter Change', icon: 'fas fa-sliders-h' };
        if (typeString.includes('SoftwareUpgrade')) return { text: 'Software Upgrade', icon: 'fas fa-arrow-up' };
        if (typeString.includes('CommunityPoolSpend')) return { text: 'Community Pool Spend', icon: 'fas fa-hand-holding-usd' };
        return { text: 'Other', icon: 'fas fa-file' };
    }

    getVoteClass(option) {
        if (option.includes('YES')) return 'yes';
        if (option.includes('NO_WITH_VETO')) return 'veto';
        if (option.includes('NO')) return 'no';
        if (option.includes('ABSTAIN')) return 'abstain';
        return 'unknown';
    }

    getVoteLabel(option) {
        if (option.includes('YES')) return 'Yes';
        if (option.includes('NO_WITH_VETO')) return 'No With Veto';
        if (option.includes('NO')) return 'No';
        if (option.includes('ABSTAIN')) return 'Abstain';
        return option;
    }

    formatDescription(text) {
        return this.escapeHtml(text).replace(/\n/g, '<br>');
    }

    formatDateTime(dateString) {
        const date = new Date(dateString);
        return date.toLocaleString();
    }

    formatVotingPower(amount) {
        return (parseInt(amount) / 1000000).toLocaleString();
    }

    truncateAddress(address) {
        if (!address) return '';
        return address.substring(0, 10) + '...' + address.substring(address.length - 6);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    attachEventListeners() {
        // Vote buttons
        document.querySelectorAll('.btn-vote').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const option = e.currentTarget.dataset.vote;
                this.handleVote(option);
            });
        });

        // Deposit button
        const depositBtn = document.querySelector('.btn-deposit');
        if (depositBtn) {
            depositBtn.addEventListener('click', () => {
                this.handleDeposit();
            });
        }
    }

    handleVote(option) {
        this.app.votingPanel.show(this.currentProposal, option);
    }

    handleDeposit() {
        const modal = document.getElementById('depositModal');
        const content = document.getElementById('depositModalContent');

        content.innerHTML = `
            <h2>Add Deposit</h2>
            <p>Add deposit to Proposal #${this.currentProposal.proposal_id}</p>
            <div class="form-group">
                <label>Amount (PAW)</label>
                <input type="number" id="depositAmount" min="1" step="1" placeholder="Enter amount">
            </div>
            <div class="modal-actions">
                <button class="btn btn-secondary" onclick="document.getElementById('depositModal').style.display='none'">
                    Cancel
                </button>
                <button class="btn btn-primary" onclick="window.governanceApp.proposalDetail.submitDeposit()">
                    Submit Deposit
                </button>
            </div>
        `;

        modal.style.display = 'block';
    }

    async submitDeposit() {
        const amount = document.getElementById('depositAmount').value;
        if (!amount || amount <= 0) {
            this.app.showError('Please enter a valid amount');
            return;
        }

        try {
            await this.api.deposit(
                this.currentProposal.proposal_id,
                [{ denom: 'paw', amount: String(parseInt(amount) * 1000000) }],
                this.app.walletAddress
            );

            document.getElementById('depositModal').style.display = 'none';
            this.app.showSuccess('Deposit submitted successfully!');

            // Reload proposal
            setTimeout(() => {
                this.app.loadProposals();
            }, 2000);
        } catch (error) {
            this.app.showError('Failed to submit deposit: ' + error.message);
        }
    }
}
