/**
 * ProposalList Component
 * Displays a list of governance proposals with filtering and search
 */

class ProposalList {
    constructor(api, app) {
        this.api = api;
        this.app = app;
    }

    render(proposals) {
        const container = document.getElementById('proposalsList');
        if (!container) return;

        if (!proposals || proposals.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-inbox fa-3x"></i>
                    <h3>No Proposals Found</h3>
                    <p>There are no proposals matching your criteria.</p>
                </div>
            `;
            return;
        }

        const html = proposals.map(proposal => this.renderProposalCard(proposal)).join('');
        container.innerHTML = html;

        // Add click handlers
        this.attachEventListeners();
    }

    renderProposalCard(proposal) {
        const status = this.getStatusInfo(proposal.status);
        const type = this.getProposalType(proposal.content['@type']);
        const progress = this.calculateProgress(proposal);
        const timeRemaining = this.getTimeRemaining(proposal);

        return `
            <div class="proposal-card" data-proposal-id="${proposal.proposal_id}">
                <div class="proposal-header">
                    <div class="proposal-id-badge">
                        #${proposal.proposal_id}
                    </div>
                    <div class="proposal-status ${status.class}">
                        <i class="${status.icon}"></i> ${status.text}
                    </div>
                    <div class="proposal-type">
                        <i class="${type.icon}"></i> ${type.text}
                    </div>
                </div>

                <div class="proposal-content">
                    <h3 class="proposal-title">${this.escapeHtml(proposal.content.title)}</h3>
                    <p class="proposal-description">
                        ${this.truncateText(this.escapeHtml(proposal.content.description), 200)}
                    </p>
                </div>

                ${proposal.status === 'VOTING_PERIOD' ? `
                    <div class="proposal-voting">
                        <div class="voting-progress">
                            <div class="progress-bar">
                                <div class="progress-yes" style="width: ${progress.yes}%"></div>
                                <div class="progress-no" style="width: ${progress.no}%"></div>
                                <div class="progress-veto" style="width: ${progress.veto}%"></div>
                                <div class="progress-abstain" style="width: ${progress.abstain}%"></div>
                            </div>
                            <div class="progress-labels">
                                <span class="label-yes">Yes: ${progress.yes.toFixed(1)}%</span>
                                <span class="label-no">No: ${progress.no.toFixed(1)}%</span>
                                <span class="label-abstain">Abstain: ${progress.abstain.toFixed(1)}%</span>
                                <span class="label-veto">Veto: ${progress.veto.toFixed(1)}%</span>
                            </div>
                        </div>
                        <div class="voting-time">
                            <i class="fas fa-clock"></i> ${timeRemaining}
                        </div>
                    </div>
                ` : ''}

                ${proposal.status === 'DEPOSIT_PERIOD' ? `
                    <div class="proposal-deposit">
                        <div class="deposit-progress">
                            <span>Deposit: ${this.formatDeposit(proposal.total_deposit)}</span>
                            <span class="deposit-time">
                                <i class="fas fa-clock"></i> ${timeRemaining}
                            </span>
                        </div>
                    </div>
                ` : ''}

                <div class="proposal-footer">
                    <div class="proposal-dates">
                        <span class="date-item">
                            <i class="fas fa-calendar-plus"></i>
                            Submitted ${this.formatDate(proposal.submit_time)}
                        </span>
                    </div>
                    <button class="btn btn-view" data-action="view" data-proposal-id="${proposal.proposal_id}">
                        View Details <i class="fas fa-arrow-right"></i>
                    </button>
                </div>
            </div>
        `;
    }

    getStatusInfo(status) {
        const statusMap = {
            'VOTING_PERIOD': {
                text: 'Voting',
                class: 'status-voting',
                icon: 'fas fa-vote-yea'
            },
            'DEPOSIT_PERIOD': {
                text: 'Deposit Period',
                class: 'status-deposit',
                icon: 'fas fa-coins'
            },
            'PASSED': {
                text: 'Passed',
                class: 'status-passed',
                icon: 'fas fa-check-circle'
            },
            'REJECTED': {
                text: 'Rejected',
                class: 'status-rejected',
                icon: 'fas fa-times-circle'
            },
            'FAILED': {
                text: 'Failed',
                class: 'status-failed',
                icon: 'fas fa-exclamation-circle'
            }
        };

        return statusMap[status] || {
            text: status,
            class: 'status-unknown',
            icon: 'fas fa-question-circle'
        };
    }

    getProposalType(typeString) {
        if (!typeString) {
            return { text: 'Unknown', icon: 'fas fa-file' };
        }

        if (typeString.includes('TextProposal')) {
            return { text: 'Text', icon: 'fas fa-file-alt' };
        } else if (typeString.includes('ParameterChange')) {
            return { text: 'Parameter Change', icon: 'fas fa-sliders-h' };
        } else if (typeString.includes('SoftwareUpgrade')) {
            return { text: 'Software Upgrade', icon: 'fas fa-arrow-up' };
        } else if (typeString.includes('CommunityPoolSpend')) {
            return { text: 'Community Spend', icon: 'fas fa-hand-holding-usd' };
        }

        return { text: 'Other', icon: 'fas fa-file' };
    }

    calculateProgress(proposal) {
        const tally = proposal.final_tally_result;
        if (!tally) {
            return { yes: 0, no: 0, abstain: 0, veto: 0 };
        }

        const yes = parseInt(tally.yes || 0);
        const no = parseInt(tally.no || 0);
        const abstain = parseInt(tally.abstain || 0);
        const veto = parseInt(tally.no_with_veto || 0);
        const total = yes + no + abstain + veto;

        if (total === 0) {
            return { yes: 0, no: 0, abstain: 0, veto: 0 };
        }

        return {
            yes: (yes / total) * 100,
            no: (no / total) * 100,
            abstain: (abstain / total) * 100,
            veto: (veto / total) * 100
        };
    }

    getTimeRemaining(proposal) {
        let endTime;

        if (proposal.status === 'VOTING_PERIOD') {
            endTime = new Date(proposal.voting_end_time);
        } else if (proposal.status === 'DEPOSIT_PERIOD') {
            endTime = new Date(proposal.deposit_end_time);
        } else {
            return 'Ended';
        }

        const now = new Date();
        const diff = endTime - now;

        if (diff <= 0) {
            return 'Ended';
        }

        const days = Math.floor(diff / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

        if (days > 0) {
            return `${days}d ${hours}h remaining`;
        } else if (hours > 0) {
            return `${hours}h remaining`;
        } else {
            const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
            return `${minutes}m remaining`;
        }
    }

    formatDeposit(deposits) {
        if (!deposits || deposits.length === 0) {
            return '0 PAW';
        }
        const amount = parseInt(deposits[0].amount);
        return (amount / 1000000).toLocaleString() + ' PAW';
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        const days = Math.floor(diff / (1000 * 60 * 60 * 24));

        if (days === 0) {
            return 'Today';
        } else if (days === 1) {
            return 'Yesterday';
        } else if (days < 7) {
            return `${days} days ago`;
        } else {
            return date.toLocaleDateString();
        }
    }

    truncateText(text, maxLength) {
        if (text.length <= maxLength) {
            return text;
        }
        return text.substring(0, maxLength) + '...';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    attachEventListeners() {
        // View buttons
        document.querySelectorAll('[data-action="view"]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const proposalId = e.currentTarget.dataset.proposalId;
                this.app.showProposalDetail(proposalId);
            });
        });

        // Card click (except on buttons)
        document.querySelectorAll('.proposal-card').forEach(card => {
            card.addEventListener('click', (e) => {
                if (!e.target.closest('button')) {
                    const proposalId = card.dataset.proposalId;
                    this.app.showProposalDetail(proposalId);
                }
            });
        });
    }
}
