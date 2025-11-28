/**
 * Comprehensive Test Suite for PAW Governance Portal
 * Tests all components, API interactions, and user flows
 */

// Mock environment for testing
const mockEnvironment = {
    window: {
        governanceApp: null,
        Chart: class {
            constructor() {}
            destroy() {}
        }
    },
    document: {
        getElementById: () => ({ innerHTML: '', style: {}, classList: { add: () => {}, remove: () => {} } }),
        querySelectorAll: () => [],
        createElement: () => ({ innerHTML: '', textContent: '', appendChild: () => {} }),
        addEventListener: () => {}
    }
};

// Test utilities
class TestRunner {
    constructor() {
        this.tests = [];
        this.passed = 0;
        this.failed = 0;
    }

    test(name, fn) {
        this.tests.push({ name, fn });
    }

    async run() {
        console.log('='.repeat(80));
        console.log('PAW Governance Portal - Test Suite');
        console.log('='.repeat(80));

        for (const test of this.tests) {
            try {
                await test.fn();
                this.passed++;
                console.log(`✓ PASS: ${test.name}`);
            } catch (error) {
                this.failed++;
                console.log(`✗ FAIL: ${test.name}`);
                console.log(`  Error: ${error.message}`);
            }
        }

        console.log('='.repeat(80));
        console.log(`Results: ${this.passed} passed, ${this.failed} failed, ${this.tests.length} total`);
        console.log('='.repeat(80));

        return this.failed === 0;
    }
}

// Test assertions
class Assert {
    static equals(actual, expected, message = '') {
        if (actual !== expected) {
            throw new Error(`Expected ${expected}, got ${actual}. ${message}`);
        }
    }

    static notEquals(actual, expected, message = '') {
        if (actual === expected) {
            throw new Error(`Expected not to equal ${expected}. ${message}`);
        }
    }

    static isTrue(value, message = '') {
        if (value !== true) {
            throw new Error(`Expected true, got ${value}. ${message}`);
        }
    }

    static isFalse(value, message = '') {
        if (value !== false) {
            throw new Error(`Expected false, got ${value}. ${message}`);
        }
    }

    static isNull(value, message = '') {
        if (value !== null) {
            throw new Error(`Expected null, got ${value}. ${message}`);
        }
    }

    static notNull(value, message = '') {
        if (value === null) {
            throw new Error(`Expected not null. ${message}`);
        }
    }

    static isDefined(value, message = '') {
        if (value === undefined) {
            throw new Error(`Expected value to be defined. ${message}`);
        }
    }

    static isArray(value, message = '') {
        if (!Array.isArray(value)) {
            throw new Error(`Expected array, got ${typeof value}. ${message}`);
        }
    }

    static isObject(value, message = '') {
        if (typeof value !== 'object' || value === null) {
            throw new Error(`Expected object, got ${typeof value}. ${message}`);
        }
    }

    static includes(array, value, message = '') {
        if (!array.includes(value)) {
            throw new Error(`Expected array to include ${value}. ${message}`);
        }
    }

    static greaterThan(actual, expected, message = '') {
        if (actual <= expected) {
            throw new Error(`Expected ${actual} to be greater than ${expected}. ${message}`);
        }
    }

    static async throws(fn, message = '') {
        try {
            await fn();
            throw new Error(`Expected function to throw. ${message}`);
        } catch (error) {
            if (error.message.includes('Expected function to throw')) {
                throw error;
            }
        }
    }
}

// Initialize test runner
const runner = new TestRunner();

// ========================================
// GovernanceAPI Tests
// ========================================

runner.test('GovernanceAPI: Constructor initializes correctly', () => {
    const api = new GovernanceAPI();
    Assert.equals(api.baseURL, 'http://localhost:1317', 'Base URL should be set');
    Assert.equals(api.rpcURL, 'http://localhost:26657', 'RPC URL should be set');
    Assert.isTrue(api.mockMode, 'Mock mode should be enabled');
    Assert.isFalse(api.connected, 'Should not be connected initially');
});

runner.test('GovernanceAPI: checkConnection returns true in mock mode', async () => {
    const api = new GovernanceAPI();
    const result = await api.checkConnection();
    Assert.isTrue(result, 'Connection check should return true in mock mode');
    Assert.isTrue(api.connected, 'Connected flag should be set');
});

runner.test('GovernanceAPI: getAllProposals returns array', async () => {
    const api = new GovernanceAPI();
    const proposals = await api.getAllProposals();
    Assert.isArray(proposals, 'Should return an array');
    Assert.greaterThan(proposals.length, 0, 'Should have proposals');
});

runner.test('GovernanceAPI: getAllProposals returns valid proposal structure', async () => {
    const api = new GovernanceAPI();
    const proposals = await api.getAllProposals();
    const proposal = proposals[0];

    Assert.isDefined(proposal.proposal_id, 'Proposal should have ID');
    Assert.isObject(proposal.content, 'Proposal should have content');
    Assert.isDefined(proposal.status, 'Proposal should have status');
    Assert.isDefined(proposal.content.title, 'Proposal should have title');
    Assert.isDefined(proposal.content.description, 'Proposal should have description');
});

runner.test('GovernanceAPI: getProposal returns specific proposal', async () => {
    const api = new GovernanceAPI();
    const proposal = await api.getProposal('1');
    Assert.notNull(proposal, 'Should return a proposal');
    Assert.equals(proposal.proposal_id, '1', 'Should return correct proposal');
});

runner.test('GovernanceAPI: getProposalVotes returns votes array', async () => {
    const api = new GovernanceAPI();
    const votes = await api.getProposalVotes('1');
    Assert.isArray(votes, 'Should return an array of votes');
});

runner.test('GovernanceAPI: getProposalDeposits returns deposits array', async () => {
    const api = new GovernanceAPI();
    const deposits = await api.getProposalDeposits('1');
    Assert.isArray(deposits, 'Should return an array of deposits');
});

runner.test('GovernanceAPI: getProposalTally returns tally object', async () => {
    const api = new GovernanceAPI();
    const tally = await api.getProposalTally('1');
    Assert.isObject(tally, 'Should return a tally object');
    Assert.isDefined(tally.yes, 'Should have yes votes');
    Assert.isDefined(tally.no, 'Should have no votes');
    Assert.isDefined(tally.abstain, 'Should have abstain votes');
    Assert.isDefined(tally.no_with_veto, 'Should have veto votes');
});

runner.test('GovernanceAPI: getGovernanceParameters returns all parameter types', async () => {
    const api = new GovernanceAPI();
    const params = await api.getGovernanceParameters();
    Assert.isObject(params, 'Should return parameters object');
    Assert.isDefined(params.deposit, 'Should have deposit parameters');
    Assert.isDefined(params.voting, 'Should have voting parameters');
    Assert.isDefined(params.tally, 'Should have tally parameters');
});

runner.test('GovernanceAPI: submitProposal returns success in mock mode', async () => {
    const api = new GovernanceAPI();
    const result = await api.submitProposal(
        { title: 'Test', description: 'Test proposal' },
        [{ denom: 'paw', amount: '10000000' }],
        'paw1test'
    );
    Assert.isTrue(result.success, 'Should return success');
    Assert.isDefined(result.proposal_id, 'Should return proposal ID');
    Assert.isDefined(result.txhash, 'Should return transaction hash');
});

runner.test('GovernanceAPI: vote returns success in mock mode', async () => {
    const api = new GovernanceAPI();
    const result = await api.vote('1', 'VOTE_OPTION_YES', 'paw1test');
    Assert.isTrue(result.success, 'Should return success');
    Assert.isDefined(result.txhash, 'Should return transaction hash');
});

runner.test('GovernanceAPI: deposit returns success in mock mode', async () => {
    const api = new GovernanceAPI();
    const result = await api.deposit(
        '1',
        [{ denom: 'paw', amount: '5000000' }],
        'paw1test'
    );
    Assert.isTrue(result.success, 'Should return success');
    Assert.isDefined(result.txhash, 'Should return transaction hash');
});

runner.test('GovernanceAPI: getUserVotes returns user vote history', async () => {
    const api = new GovernanceAPI();
    const votes = await api.getUserVotes('paw1test');
    Assert.isArray(votes, 'Should return an array of votes');
});

// ========================================
// ProposalList Component Tests
// ========================================

runner.test('ProposalList: getStatusInfo returns correct status for VOTING_PERIOD', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const status = list.getStatusInfo('VOTING_PERIOD');

    Assert.equals(status.text, 'Voting', 'Status text should be Voting');
    Assert.equals(status.class, 'status-voting', 'CSS class should be correct');
    Assert.isDefined(status.icon, 'Should have icon');
});

runner.test('ProposalList: getStatusInfo returns correct status for PASSED', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const status = list.getStatusInfo('PASSED');

    Assert.equals(status.text, 'Passed', 'Status text should be Passed');
    Assert.equals(status.class, 'status-passed', 'CSS class should be correct');
});

runner.test('ProposalList: getProposalType identifies text proposals', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const type = list.getProposalType('/cosmos.gov.v1beta1.TextProposal');

    Assert.equals(type.text, 'Text', 'Type should be Text');
    Assert.isDefined(type.icon, 'Should have icon');
});

runner.test('ProposalList: getProposalType identifies parameter change proposals', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const type = list.getProposalType('/cosmos.params.v1beta1.ParameterChangeProposal');

    Assert.equals(type.text, 'Parameter Change', 'Type should be Parameter Change');
});

runner.test('ProposalList: calculateProgress computes percentages correctly', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const tally = {
        yes: '50000000',
        no: '30000000',
        abstain: '10000000',
        no_with_veto: '10000000'
    };

    const progress = list.calculateProgress({ final_tally_result: tally });

    Assert.equals(progress.yes, 50, 'Yes should be 50%');
    Assert.equals(progress.no, 30, 'No should be 30%');
    Assert.equals(progress.abstain, 10, 'Abstain should be 10%');
    Assert.equals(progress.veto, 10, 'Veto should be 10%');
});

runner.test('ProposalList: calculateProgress handles empty tally', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const tally = { yes: '0', no: '0', abstain: '0', no_with_veto: '0' };

    const progress = list.calculateProgress({ final_tally_result: tally });

    Assert.equals(progress.yes, 0, 'All percentages should be 0');
    Assert.equals(progress.no, 0, 'All percentages should be 0');
});

runner.test('ProposalList: formatDeposit formats amount correctly', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const deposits = [{ amount: '10000000', denom: 'paw' }];

    const formatted = list.formatDeposit(deposits);

    Assert.isTrue(formatted.includes('10'), 'Should include amount');
    Assert.isTrue(formatted.includes('PAW'), 'Should include denomination');
});

runner.test('ProposalList: escapeHtml prevents XSS', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const malicious = '<script>alert("xss")</script>';

    const escaped = list.escapeHtml(malicious);

    Assert.isFalse(escaped.includes('<script>'), 'Should escape script tags');
    Assert.isTrue(escaped.includes('&lt;'), 'Should use HTML entities');
});

// ========================================
// ProposalDetail Component Tests
// ========================================

runner.test('ProposalDetail: getVoteClass returns correct class for YES', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});
    const voteClass = detail.getVoteClass('VOTE_OPTION_YES');

    Assert.equals(voteClass, 'yes', 'Should return yes class');
});

runner.test('ProposalDetail: getVoteClass returns correct class for NO_WITH_VETO', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});
    const voteClass = detail.getVoteClass('VOTE_OPTION_NO_WITH_VETO');

    Assert.equals(voteClass, 'veto', 'Should return veto class');
});

runner.test('ProposalDetail: getVoteLabel returns readable labels', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});

    Assert.equals(detail.getVoteLabel('VOTE_OPTION_YES'), 'Yes');
    Assert.equals(detail.getVoteLabel('VOTE_OPTION_NO'), 'No');
    Assert.equals(detail.getVoteLabel('VOTE_OPTION_ABSTAIN'), 'Abstain');
    Assert.equals(detail.getVoteLabel('VOTE_OPTION_NO_WITH_VETO'), 'No With Veto');
});

runner.test('ProposalDetail: formatVotingPower formats large amounts', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});

    const formatted = detail.formatVotingPower('100000000000');
    Assert.isTrue(formatted.includes('100'), 'Should format large numbers');
});

runner.test('ProposalDetail: truncateAddress shortens addresses', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});
    const address = 'paw1234567890abcdefghijklmnopqrstuvwxyz';

    const truncated = detail.truncateAddress(address);

    Assert.isTrue(truncated.includes('...'), 'Should include ellipsis');
    Assert.isTrue(truncated.length < address.length, 'Should be shorter');
});

// ========================================
// VotingPanel Component Tests
// ========================================

runner.test('VotingPanel: getVoteLabel returns correct labels', () => {
    const api = new GovernanceAPI();
    const panel = new VotingPanel(api, {});

    Assert.equals(panel.getVoteLabel('VOTE_OPTION_YES'), 'Yes');
    Assert.equals(panel.getVoteLabel('VOTE_OPTION_NO'), 'No');
    Assert.equals(panel.getVoteLabel('VOTE_OPTION_ABSTAIN'), 'Abstain');
    Assert.equals(panel.getVoteLabel('VOTE_OPTION_NO_WITH_VETO'), 'No With Veto');
});

runner.test('VotingPanel: escapeHtml prevents XSS', () => {
    const api = new GovernanceAPI();
    const panel = new VotingPanel(api, {});
    const malicious = '<img src=x onerror=alert(1)>';

    const escaped = panel.escapeHtml(malicious);

    Assert.isFalse(escaped.includes('onerror'), 'Should escape event handlers');
});

// ========================================
// TallyChart Component Tests
// ========================================

runner.test('TallyChart: formatVotingPower formats millions', () => {
    const chart = new TallyChart();
    const formatted = chart.formatVotingPower('1000000000000');

    Assert.isTrue(formatted.includes('M'), 'Should use M for millions');
});

runner.test('TallyChart: formatVotingPower formats thousands', () => {
    const chart = new TallyChart();
    const formatted = chart.formatVotingPower('1000000000');

    Assert.isTrue(formatted.includes('K'), 'Should use K for thousands');
});

runner.test('TallyChart: formatVotingPower formats small amounts', () => {
    const chart = new TallyChart();
    const formatted = chart.formatVotingPower('1000000');

    Assert.isFalse(formatted.includes('K'), 'Should not use K for small amounts');
    Assert.isFalse(formatted.includes('M'), 'Should not use M for small amounts');
});

// ========================================
// Integration Tests
// ========================================

runner.test('Integration: Proposal listing flow', async () => {
    const api = new GovernanceAPI();
    const proposals = await api.getAllProposals();

    Assert.isArray(proposals, 'Should get proposals');
    Assert.greaterThan(proposals.length, 0, 'Should have at least one proposal');

    const proposal = proposals[0];
    Assert.isDefined(proposal.proposal_id, 'Proposal should have ID');
    Assert.isDefined(proposal.content.title, 'Proposal should have title');
});

runner.test('Integration: Proposal detail flow', async () => {
    const api = new GovernanceAPI();
    const proposals = await api.getAllProposals();
    const proposalId = proposals[0].proposal_id;

    const proposal = await api.getProposal(proposalId);
    const votes = await api.getProposalVotes(proposalId);
    const tally = await api.getProposalTally(proposalId);

    Assert.notNull(proposal, 'Should get proposal');
    Assert.isArray(votes, 'Should get votes');
    Assert.isObject(tally, 'Should get tally');
});

runner.test('Integration: Create proposal flow', async () => {
    const api = new GovernanceAPI();

    const proposalData = {
        '@type': '/cosmos.gov.v1beta1.TextProposal',
        title: 'Test Proposal',
        description: 'This is a test proposal'
    };

    const deposit = [{ denom: 'paw', amount: '10000000' }];
    const result = await api.submitProposal(proposalData, deposit, 'paw1test');

    Assert.isTrue(result.success, 'Should submit successfully');
    Assert.isDefined(result.proposal_id, 'Should return proposal ID');
});

runner.test('Integration: Voting flow', async () => {
    const api = new GovernanceAPI();

    const result = await api.vote('1', 'VOTE_OPTION_YES', 'paw1test');

    Assert.isTrue(result.success, 'Should vote successfully');
    Assert.isDefined(result.txhash, 'Should return transaction hash');
});

runner.test('Integration: Deposit flow', async () => {
    const api = new GovernanceAPI();

    const amount = [{ denom: 'paw', amount: '5000000' }];
    const result = await api.deposit('1', amount, 'paw1test');

    Assert.isTrue(result.success, 'Should deposit successfully');
    Assert.isDefined(result.txhash, 'Should return transaction hash');
});

runner.test('Integration: Parameter retrieval', async () => {
    const api = new GovernanceAPI();
    const params = await api.getGovernanceParameters();

    Assert.isDefined(params.deposit.min_deposit, 'Should have min deposit');
    Assert.isDefined(params.voting.voting_period, 'Should have voting period');
    Assert.isDefined(params.tally.quorum, 'Should have quorum');
    Assert.isDefined(params.tally.threshold, 'Should have threshold');
    Assert.isDefined(params.tally.veto_threshold, 'Should have veto threshold');
});

// ========================================
// Edge Cases and Error Handling
// ========================================

runner.test('Edge Case: Empty tally calculation', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const emptyTally = { yes: '0', no: '0', abstain: '0', no_with_veto: '0' };

    const progress = list.calculateProgress({ final_tally_result: emptyTally });

    Assert.equals(progress.yes, 0);
    Assert.equals(progress.no, 0);
    Assert.equals(progress.abstain, 0);
    Assert.equals(progress.veto, 0);
});

runner.test('Edge Case: Null proposal content', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const type = list.getProposalType(null);

    Assert.equals(type.text, 'Unknown');
});

runner.test('Edge Case: Empty deposit array', () => {
    const api = new GovernanceAPI();
    const list = new ProposalList(api, {});
    const formatted = list.formatDeposit([]);

    Assert.equals(formatted, '0 PAW');
});

runner.test('Edge Case: Undefined address truncation', () => {
    const api = new GovernanceAPI();
    const detail = new ProposalDetail(api, {});
    const truncated = detail.truncateAddress(undefined);

    Assert.equals(truncated, '');
});

// Run all tests
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { TestRunner, Assert, runner };
} else {
    // Run tests immediately if in browser
    runner.run().then(success => {
        if (success) {
            console.log('All tests passed!');
        } else {
            console.log('Some tests failed!');
        }
    });
}
