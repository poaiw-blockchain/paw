/**
 * PAW Governance API Service
 * Handles all API interactions for governance operations
 */

class GovernanceAPI {
    constructor() {
        this.baseURL = window.PAW_API_URL || 'http://localhost:1317'; // REST endpoint
        this.rpcURL = window.PAW_RPC_URL || 'http://localhost:26657'; // RPC endpoint
        this.connected = false;
        this.mockMode = window.PAW_MOCK_MODE !== undefined ? window.PAW_MOCK_MODE : true; // Enable mock mode for development
    }

    /**
     * Check connection to the blockchain
     */
    async checkConnection() {
        try {
            if (this.mockMode) {
                this.connected = true;
                return true;
            }

            const response = await fetch(`${this.baseURL}/cosmos/base/tendermint/v1beta1/node_info`);
            this.connected = response.ok;
            return this.connected;
        } catch (error) {
            console.error('Connection check failed:', error);
            this.connected = false;
            return false;
        }
    }

    /**
     * Get all governance proposals
     */
    async getAllProposals() {
        try {
            if (this.mockMode) {
                return this.getMockProposals();
            }

            const response = await fetch(`${this.baseURL}/cosmos/gov/v1beta1/proposals`);
            if (!response.ok) throw new Error('Failed to fetch proposals');

            const data = await response.json();
            return data.proposals || [];
        } catch (error) {
            console.error('Failed to fetch proposals:', error);
            return this.getMockProposals(); // Fallback to mock data
        }
    }

    /**
     * Get a specific proposal by ID
     */
    async getProposal(proposalId) {
        try {
            if (this.mockMode) {
                const proposals = this.getMockProposals();
                return proposals.find(p => p.proposal_id === proposalId);
            }

            const response = await fetch(`${this.baseURL}/cosmos/gov/v1beta1/proposals/${proposalId}`);
            if (!response.ok) throw new Error('Failed to fetch proposal');

            const data = await response.json();
            return data.proposal;
        } catch (error) {
            console.error('Failed to fetch proposal:', error);
            throw error;
        }
    }

    /**
     * Get votes for a proposal
     */
    async getProposalVotes(proposalId) {
        try {
            if (this.mockMode) {
                return this.getMockVotes(proposalId);
            }

            const response = await fetch(`${this.baseURL}/cosmos/gov/v1beta1/proposals/${proposalId}/votes`);
            if (!response.ok) throw new Error('Failed to fetch votes');

            const data = await response.json();
            return data.votes || [];
        } catch (error) {
            console.error('Failed to fetch votes:', error);
            return [];
        }
    }

    /**
     * Get deposits for a proposal
     */
    async getProposalDeposits(proposalId) {
        try {
            if (this.mockMode) {
                return this.getMockDeposits(proposalId);
            }

            const response = await fetch(`${this.baseURL}/cosmos/gov/v1beta1/proposals/${proposalId}/deposits`);
            if (!response.ok) throw new Error('Failed to fetch deposits');

            const data = await response.json();
            return data.deposits || [];
        } catch (error) {
            console.error('Failed to fetch deposits:', error);
            return [];
        }
    }

    /**
     * Get tally results for a proposal
     */
    async getProposalTally(proposalId) {
        try {
            if (this.mockMode) {
                const proposals = this.getMockProposals();
                const proposal = proposals.find(p => p.proposal_id === proposalId);
                return proposal?.final_tally_result || this.getEmptyTally();
            }

            const response = await fetch(`${this.baseURL}/cosmos/gov/v1beta1/proposals/${proposalId}/tally`);
            if (!response.ok) throw new Error('Failed to fetch tally');

            const data = await response.json();
            return data.tally;
        } catch (error) {
            console.error('Failed to fetch tally:', error);
            return this.getEmptyTally();
        }
    }

    /**
     * Get governance parameters
     */
    async getGovernanceParameters() {
        try {
            if (this.mockMode) {
                return this.getMockParameters();
            }

            const [depositParams, votingParams, tallyParams] = await Promise.all([
                fetch(`${this.baseURL}/cosmos/gov/v1beta1/params/deposit`).then(r => r.json()),
                fetch(`${this.baseURL}/cosmos/gov/v1beta1/params/voting`).then(r => r.json()),
                fetch(`${this.baseURL}/cosmos/gov/v1beta1/params/tallying`).then(r => r.json())
            ]);

            return {
                deposit: depositParams.deposit_params,
                voting: votingParams.voting_params,
                tally: tallyParams.tally_params
            };
        } catch (error) {
            console.error('Failed to fetch parameters:', error);
            return this.getMockParameters();
        }
    }

    /**
     * Submit a new proposal
     */
    async submitProposal(proposalData, initialDeposit, proposerAddress) {
        try {
            if (this.mockMode) {
                console.log('Mock: Submitting proposal', proposalData);
                return {
                    success: true,
                    proposal_id: Math.floor(Math.random() * 1000) + 1,
                    txhash: 'mock_' + Math.random().toString(36).substring(7)
                };
            }

            // In production, this would broadcast a transaction
            const tx = {
                type: 'cosmos-sdk/MsgSubmitProposal',
                value: {
                    content: proposalData,
                    initial_deposit: initialDeposit,
                    proposer: proposerAddress
                }
            };

            // This would use CosmJS or similar to broadcast
            throw new Error('Not implemented - use CosmJS to broadcast transaction');
        } catch (error) {
            console.error('Failed to submit proposal:', error);
            throw error;
        }
    }

    /**
     * Vote on a proposal
     */
    async vote(proposalId, option, voterAddress) {
        try {
            if (this.mockMode) {
                console.log(`Mock: Voting ${option} on proposal ${proposalId}`);
                return {
                    success: true,
                    txhash: 'mock_' + Math.random().toString(36).substring(7)
                };
            }

            // In production, this would broadcast a vote transaction
            const tx = {
                type: 'cosmos-sdk/MsgVote',
                value: {
                    proposal_id: proposalId,
                    voter: voterAddress,
                    option: option
                }
            };

            throw new Error('Not implemented - use CosmJS to broadcast transaction');
        } catch (error) {
            console.error('Failed to vote:', error);
            throw error;
        }
    }

    /**
     * Deposit to a proposal
     */
    async deposit(proposalId, amount, depositorAddress) {
        try {
            if (this.mockMode) {
                console.log(`Mock: Depositing ${amount} to proposal ${proposalId}`);
                return {
                    success: true,
                    txhash: 'mock_' + Math.random().toString(36).substring(7)
                };
            }

            // In production, this would broadcast a deposit transaction
            const tx = {
                type: 'cosmos-sdk/MsgDeposit',
                value: {
                    proposal_id: proposalId,
                    depositor: depositorAddress,
                    amount: amount
                }
            };

            throw new Error('Not implemented - use CosmJS to broadcast transaction');
        } catch (error) {
            console.error('Failed to deposit:', error);
            throw error;
        }
    }

    /**
     * Get user's votes
     */
    async getUserVotes(address) {
        try {
            if (this.mockMode) {
                return this.getMockUserVotes(address);
            }

            const proposals = await this.getAllProposals();
            const votes = [];

            for (const proposal of proposals) {
                try {
                    const response = await fetch(
                        `${this.baseURL}/cosmos/gov/v1beta1/proposals/${proposal.proposal_id}/votes/${address}`
                    );
                    if (response.ok) {
                        const data = await response.json();
                        votes.push({
                            proposal_id: proposal.proposal_id,
                            option: data.vote.option,
                            timestamp: proposal.voting_start_time
                        });
                    }
                } catch (error) {
                    // Vote not found for this proposal
                    continue;
                }
            }

            return votes;
        } catch (error) {
            console.error('Failed to fetch user votes:', error);
            return [];
        }
    }

    // Mock data generators
    getMockProposals() {
        return [
            {
                proposal_id: '1',
                content: {
                    '@type': '/cosmos.gov.v1beta1.TextProposal',
                    title: 'Increase Block Size Limit',
                    description: 'This proposal aims to increase the block size limit from 22KB to 50KB to improve network throughput and reduce transaction costs during peak usage periods.'
                },
                status: 'VOTING_PERIOD',
                final_tally_result: {
                    yes: '45000000',
                    abstain: '5000000',
                    no: '8000000',
                    no_with_veto: '2000000'
                },
                submit_time: '2024-01-15T10:00:00Z',
                deposit_end_time: '2024-01-29T10:00:00Z',
                total_deposit: [{ denom: 'paw', amount: '10000000' }],
                voting_start_time: '2024-01-20T10:00:00Z',
                voting_end_time: '2024-02-05T10:00:00Z'
            },
            {
                proposal_id: '2',
                content: {
                    '@type': '/cosmos.params.v1beta1.ParameterChangeProposal',
                    title: 'Update Governance Voting Period',
                    description: 'Reduce the voting period from 14 days to 7 days to accelerate governance decisions while maintaining adequate time for community participation.'
                },
                status: 'VOTING_PERIOD',
                final_tally_result: {
                    yes: '38000000',
                    abstain: '12000000',
                    no: '15000000',
                    no_with_veto: '5000000'
                },
                submit_time: '2024-01-18T14:30:00Z',
                deposit_end_time: '2024-02-01T14:30:00Z',
                total_deposit: [{ denom: 'paw', amount: '10000000' }],
                voting_start_time: '2024-01-22T14:30:00Z',
                voting_end_time: '2024-02-08T14:30:00Z'
            },
            {
                proposal_id: '3',
                content: {
                    '@type': '/cosmos.gov.v1beta1.TextProposal',
                    title: 'Community Pool Fund Allocation',
                    description: 'Allocate 500,000 PAW from the community pool to fund development of a mobile wallet application and related infrastructure improvements.'
                },
                status: 'PASSED',
                final_tally_result: {
                    yes: '75000000',
                    abstain: '10000000',
                    no: '8000000',
                    no_with_veto: '1000000'
                },
                submit_time: '2024-01-05T09:00:00Z',
                deposit_end_time: '2024-01-19T09:00:00Z',
                total_deposit: [{ denom: 'paw', amount: '10000000' }],
                voting_start_time: '2024-01-10T09:00:00Z',
                voting_end_time: '2024-01-26T09:00:00Z'
            },
            {
                proposal_id: '4',
                content: {
                    '@type': '/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal',
                    title: 'Network Upgrade v2.0',
                    description: 'Upgrade to version 2.0 including new DEX features, improved oracle integration, and enhanced security measures. Scheduled for block height 1,000,000.'
                },
                status: 'DEPOSIT_PERIOD',
                final_tally_result: {
                    yes: '0',
                    abstain: '0',
                    no: '0',
                    no_with_veto: '0'
                },
                submit_time: '2024-01-25T16:00:00Z',
                deposit_end_time: '2024-02-08T16:00:00Z',
                total_deposit: [{ denom: 'paw', amount: '5000000' }],
                voting_start_time: '0001-01-01T00:00:00Z',
                voting_end_time: '0001-01-01T00:00:00Z'
            },
            {
                proposal_id: '5',
                content: {
                    '@type': '/cosmos.gov.v1beta1.TextProposal',
                    title: 'Adjust Minimum Deposit Requirement',
                    description: 'Lower the minimum deposit requirement for proposals from 10,000 PAW to 5,000 PAW to encourage more community participation in governance.'
                },
                status: 'REJECTED',
                final_tally_result: {
                    yes: '25000000',
                    abstain: '15000000',
                    no: '45000000',
                    no_with_veto: '8000000'
                },
                submit_time: '2023-12-20T11:00:00Z',
                deposit_end_time: '2024-01-03T11:00:00Z',
                total_deposit: [{ denom: 'paw', amount: '10000000' }],
                voting_start_time: '2023-12-28T11:00:00Z',
                voting_end_time: '2024-01-13T11:00:00Z'
            }
        ];
    }

    getMockVotes(proposalId) {
        const voteOptions = ['VOTE_OPTION_YES', 'VOTE_OPTION_NO', 'VOTE_OPTION_ABSTAIN', 'VOTE_OPTION_NO_WITH_VETO'];
        const votes = [];

        for (let i = 0; i < 50; i++) {
            votes.push({
                proposal_id: proposalId,
                voter: `paw1${Math.random().toString(36).substring(2, 42)}`,
                option: voteOptions[Math.floor(Math.random() * voteOptions.length)],
                timestamp: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()
            });
        }

        return votes;
    }

    getMockDeposits(proposalId) {
        const deposits = [];

        for (let i = 0; i < 5; i++) {
            deposits.push({
                proposal_id: proposalId,
                depositor: `paw1${Math.random().toString(36).substring(2, 42)}`,
                amount: [{ denom: 'paw', amount: String(Math.floor(Math.random() * 5000000) + 1000000) }],
                timestamp: new Date(Date.now() - Math.random() * 14 * 24 * 60 * 60 * 1000).toISOString()
            });
        }

        return deposits;
    }

    getMockParameters() {
        return {
            deposit: {
                min_deposit: [{ denom: 'paw', amount: '10000000' }],
                max_deposit_period: '1209600s' // 14 days
            },
            voting: {
                voting_period: '1209600s' // 14 days
            },
            tally: {
                quorum: '0.334000000000000000',
                threshold: '0.500000000000000000',
                veto_threshold: '0.334000000000000000'
            }
        };
    }

    getMockUserVotes(address) {
        return [
            {
                proposal_id: '1',
                option: 'VOTE_OPTION_YES',
                timestamp: '2024-01-22T15:30:00Z'
            },
            {
                proposal_id: '3',
                option: 'VOTE_OPTION_YES',
                timestamp: '2024-01-12T10:15:00Z'
            },
            {
                proposal_id: '5',
                option: 'VOTE_OPTION_NO',
                timestamp: '2023-12-30T14:20:00Z'
            }
        ];
    }

    getEmptyTally() {
        return {
            yes: '0',
            abstain: '0',
            no: '0',
            no_with_veto: '0'
        };
    }
}
