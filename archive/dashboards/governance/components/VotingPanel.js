/**
 * VotingPanel Component
 * Handles the voting interface and vote submission
 */

class VotingPanel {
    constructor(api, app) {
        this.api = api;
        this.app = app;
        this.currentProposal = null;
        this.selectedOption = null;
    }

    show(proposal, preselectedOption = null) {
        if (!this.app.walletConnected) {
            this.app.showError('Please connect your wallet to vote');
            return;
        }

        this.currentProposal = proposal;
        this.selectedOption = preselectedOption;

        const modal = document.getElementById('voteModal');
        const content = document.getElementById('voteModalContent');

        content.innerHTML = this.renderVotingForm();
        modal.style.display = 'block';

        this.attachEventListeners();
    }

    renderVotingForm() {
        const optionLabels = {
            'VOTE_OPTION_YES': 'Yes',
            'VOTE_OPTION_NO': 'No',
            'VOTE_OPTION_ABSTAIN': 'Abstain',
            'VOTE_OPTION_NO_WITH_VETO': 'No With Veto'
        };

        const optionDescriptions = {
            'VOTE_OPTION_YES': 'I support this proposal and want it to pass',
            'VOTE_OPTION_NO': 'I oppose this proposal but do not want to veto it',
            'VOTE_OPTION_ABSTAIN': 'I am abstaining from this vote but want to contribute to quorum',
            'VOTE_OPTION_NO_WITH_VETO': 'I strongly oppose this proposal and want to veto it (deposits will be burned if veto threshold is met)'
        };

        const optionIcons = {
            'VOTE_OPTION_YES': 'fa-thumbs-up',
            'VOTE_OPTION_NO': 'fa-thumbs-down',
            'VOTE_OPTION_ABSTAIN': 'fa-minus',
            'VOTE_OPTION_NO_WITH_VETO': 'fa-ban'
        };

        const optionColors = {
            'VOTE_OPTION_YES': 'vote-yes',
            'VOTE_OPTION_NO': 'vote-no',
            'VOTE_OPTION_ABSTAIN': 'vote-abstain',
            'VOTE_OPTION_NO_WITH_VETO': 'vote-veto'
        };

        return `
            <div class="voting-modal">
                <h2>Cast Your Vote</h2>

                <div class="proposal-summary">
                    <div class="summary-id">Proposal #${this.currentProposal.proposal_id}</div>
                    <div class="summary-title">${this.escapeHtml(this.currentProposal.content.title)}</div>
                </div>

                <div class="voting-power-info">
                    <i class="fas fa-balance-scale"></i>
                    <span>Your Voting Power: <strong>${this.app.votingPower.toLocaleString()} PAW</strong></span>
                </div>

                <div class="vote-options">
                    ${Object.keys(optionLabels).map(option => `
                        <label class="vote-option-card ${this.selectedOption === option ? 'selected' : ''}">
                            <input
                                type="radio"
                                name="voteOption"
                                value="${option}"
                                ${this.selectedOption === option ? 'checked' : ''}
                            >
                            <div class="option-content ${optionColors[option]}">
                                <div class="option-icon">
                                    <i class="fas ${optionIcons[option]}"></i>
                                </div>
                                <div class="option-text">
                                    <h4>${optionLabels[option]}</h4>
                                    <p>${optionDescriptions[option]}</p>
                                </div>
                            </div>
                        </label>
                    `).join('')}
                </div>

                <div class="vote-memo">
                    <label for="voteMemo">
                        <i class="fas fa-comment"></i> Memo (Optional)
                    </label>
                    <textarea
                        id="voteMemo"
                        rows="3"
                        placeholder="Add a comment about your vote (publicly visible)"
                        maxlength="256"
                    ></textarea>
                    <small>Maximum 256 characters</small>
                </div>

                <div class="vote-warning">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>
                        Your vote is final and cannot be changed once submitted.
                        Make sure you understand the proposal before voting.
                    </p>
                </div>

                <div class="modal-actions">
                    <button class="btn btn-secondary" onclick="document.getElementById('voteModal').style.display='none'">
                        Cancel
                    </button>
                    <button class="btn btn-primary" id="confirmVoteBtn">
                        <i class="fas fa-check"></i> Confirm Vote
                    </button>
                </div>
            </div>
        `;
    }

    attachEventListeners() {
        // Radio button changes
        document.querySelectorAll('input[name="voteOption"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.selectedOption = e.target.value;

                // Update selected state
                document.querySelectorAll('.vote-option-card').forEach(card => {
                    card.classList.remove('selected');
                });
                e.target.closest('.vote-option-card').classList.add('selected');
            });
        });

        // Confirm button
        const confirmBtn = document.getElementById('confirmVoteBtn');
        if (confirmBtn) {
            confirmBtn.addEventListener('click', () => this.confirmVote());
        }
    }

    async confirmVote() {
        if (!this.selectedOption) {
            this.app.showError('Please select a vote option');
            return;
        }

        const memo = document.getElementById('voteMemo')?.value || '';

        // Show confirmation
        if (!confirm(`Are you sure you want to vote "${this.getVoteLabel(this.selectedOption)}" on this proposal?`)) {
            return;
        }

        const confirmBtn = document.getElementById('confirmVoteBtn');
        if (confirmBtn) {
            confirmBtn.disabled = true;
            confirmBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Submitting...';
        }

        try {
            const result = await this.api.vote(
                this.currentProposal.proposal_id,
                this.selectedOption,
                this.app.walletAddress
            );

            // Close modal
            document.getElementById('voteModal').style.display = 'none';

            // Show success message
            this.app.showSuccess('Vote submitted successfully!');

            // Reload proposals
            await this.app.loadProposals();

            // Update proposal detail if viewing
            if (this.app.currentSection === 'proposals') {
                setTimeout(() => {
                    this.app.showProposalDetail(this.currentProposal.proposal_id);
                }, 1000);
            }

        } catch (error) {
            console.error('Failed to submit vote:', error);
            this.app.showError('Failed to submit vote: ' + error.message);

            if (confirmBtn) {
                confirmBtn.disabled = false;
                confirmBtn.innerHTML = '<i class="fas fa-check"></i> Confirm Vote';
            }
        }
    }

    getVoteLabel(option) {
        const labels = {
            'VOTE_OPTION_YES': 'Yes',
            'VOTE_OPTION_NO': 'No',
            'VOTE_OPTION_ABSTAIN': 'Abstain',
            'VOTE_OPTION_NO_WITH_VETO': 'No With Veto'
        };
        return labels[option] || option;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
