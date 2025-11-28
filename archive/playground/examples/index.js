// Example Code Repository

export const examples = {
    'hello-world': {
        title: 'Hello World',
        description: 'Simple example to get started',
        category: 'getting-started',
        language: 'javascript',
        code: `// Welcome to PAW Playground!
// This is a simple example to get you started

console.log('Hello, PAW Blockchain!');

// Query node information
const nodeInfo = await api.getNodeInfo();
console.log('Node Info:', nodeInfo);

// Return the result
return nodeInfo;`
    },

    'query-balance': {
        title: 'Query Balance',
        description: 'Check account balance',
        category: 'getting-started',
        language: 'javascript',
        code: `// Query account balance
const address = 'paw1...'; // Replace with actual address

// Get all balances
const balances = await api.getAllBalances(address);
console.log('All Balances:', balances);

// Get specific denom balance
const pawBalance = await api.getBalance(address, 'upaw');
console.log('PAW Balance:', pawBalance);

return balances;`
    },

    'bank-transfer': {
        title: 'Bank Transfer',
        description: 'Send tokens to another address',
        category: 'bank',
        language: 'javascript',
        code: `// Send tokens (requires wallet connection)
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Transaction message
const msg = {
    type: 'cosmos-sdk/MsgSend',
    value: {
        from_address: wallet.address,
        to_address: 'paw1...', // Recipient address
        amount: [{
            denom: 'upaw',
            amount: '1000000' // 1 PAW (1,000,000 upaw)
        }]
    }
};

console.log('Transaction message:', msg);
console.log('Note: This is a preview. Use Keplr to sign and broadcast.');

return { transaction: msg };`
    },

    'multi-send': {
        title: 'Multi Send',
        description: 'Send tokens to multiple addresses',
        category: 'bank',
        language: 'javascript',
        code: `// Multi-send tokens
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

const msg = {
    type: 'cosmos-sdk/MsgMultiSend',
    value: {
        inputs: [{
            address: wallet.address,
            coins: [{
                denom: 'upaw',
                amount: '3000000'
            }]
        }],
        outputs: [
            {
                address: 'paw1...', // Recipient 1
                coins: [{
                    denom: 'upaw',
                    amount: '1000000'
                }]
            },
            {
                address: 'paw1...', // Recipient 2
                coins: [{
                    denom: 'upaw',
                    amount: '1000000'
                }]
            },
            {
                address: 'paw1...', // Recipient 3
                coins: [{
                    denom: 'upaw',
                    amount: '1000000'
                }]
            }
        ]
    }
};

console.log('Multi-send transaction:', msg);
return { transaction: msg };`
    },

    'dex-swap': {
        title: 'DEX Swap',
        description: 'Swap tokens on the DEX',
        category: 'dex',
        language: 'javascript',
        code: `// Swap tokens on PAW DEX
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get available pools
const pools = await api.getPools();
console.log('Available pools:', pools);

// Estimate swap
const poolId = 1;
const tokenIn = 'upaw';
const amountIn = '1000000'; // 1 PAW

const estimate = await api.estimateSwap(poolId, tokenIn, amountIn);
console.log('Swap estimate:', estimate);

// Create swap message
const swapMsg = {
    type: 'paw/dex/MsgSwap',
    value: {
        sender: wallet.address,
        pool_id: poolId,
        token_in: {
            denom: tokenIn,
            amount: amountIn
        },
        min_token_out: estimate.amount_out
    }
};

console.log('Swap transaction:', swapMsg);
return { transaction: swapMsg, estimate };`
    },

    'add-liquidity': {
        title: 'Add Liquidity',
        description: 'Add liquidity to a DEX pool',
        category: 'dex',
        language: 'javascript',
        code: `// Add liquidity to DEX pool
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

const poolId = 1;

// Get pool info
const pool = await api.getPool(poolId);
console.log('Pool info:', pool);

// Create add liquidity message
const msg = {
    type: 'paw/dex/MsgAddLiquidity',
    value: {
        sender: wallet.address,
        pool_id: poolId,
        tokens: [
            {
                denom: 'upaw',
                amount: '1000000' // 1 PAW
            },
            {
                denom: 'uusdc',
                amount: '1000000' // 1 USDC
            }
        ],
        min_liquidity: '0'
    }
};

console.log('Add liquidity transaction:', msg);
return { transaction: msg, pool };`
    },

    'remove-liquidity': {
        title: 'Remove Liquidity',
        description: 'Remove liquidity from a DEX pool',
        category: 'dex',
        language: 'javascript',
        code: `// Remove liquidity from DEX pool
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

const poolId = 1;
const liquidityAmount = '1000000'; // Amount of LP tokens to burn

// Create remove liquidity message
const msg = {
    type: 'paw/dex/MsgRemoveLiquidity',
    value: {
        sender: wallet.address,
        pool_id: poolId,
        liquidity: liquidityAmount,
        min_tokens: [] // Minimum tokens to receive
    }
};

console.log('Remove liquidity transaction:', msg);
return { transaction: msg };`
    },

    'staking': {
        title: 'Delegate Tokens',
        description: 'Stake tokens with a validator',
        category: 'staking',
        language: 'javascript',
        code: `// Delegate tokens to a validator
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get validators
const validators = await api.getValidators('BOND_STATUS_BONDED');
console.log('Active validators:', validators);

// Select a validator (use first one for example)
const validator = validators.validators[0];
console.log('Delegating to:', validator.description.moniker);

// Create delegation message
const msg = {
    type: 'cosmos-sdk/MsgDelegate',
    value: {
        delegator_address: wallet.address,
        validator_address: validator.operator_address,
        amount: {
            denom: 'upaw',
            amount: '1000000' // 1 PAW
        }
    }
};

console.log('Delegation transaction:', msg);
return { transaction: msg, validator };`
    },

    'unstaking': {
        title: 'Undelegate Tokens',
        description: 'Unstake tokens from a validator',
        category: 'staking',
        language: 'javascript',
        code: `// Undelegate tokens from a validator
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get current delegations
const delegations = await api.getDelegations(wallet.address);
console.log('Current delegations:', delegations);

// Create undelegation message (use first delegation for example)
if (delegations.delegation_responses.length === 0) {
    console.error('No active delegations found');
    return;
}

const delegation = delegations.delegation_responses[0];

const msg = {
    type: 'cosmos-sdk/MsgUndelegate',
    value: {
        delegator_address: wallet.address,
        validator_address: delegation.delegation.validator_address,
        amount: {
            denom: 'upaw',
            amount: '1000000' // 1 PAW
        }
    }
};

console.log('Undelegation transaction:', msg);
return { transaction: msg };`
    },

    'claim-rewards': {
        title: 'Claim Rewards',
        description: 'Claim staking rewards',
        category: 'staking',
        language: 'javascript',
        code: `// Claim staking rewards
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get pending rewards
const rewards = await api.getDelegationRewards(wallet.address);
console.log('Pending rewards:', rewards);

// Get delegations to claim from
const delegations = await api.getDelegations(wallet.address);

// Create claim rewards messages for each validator
const messages = delegations.delegation_responses.map(delegation => ({
    type: 'cosmos-sdk/MsgWithdrawDelegationReward',
    value: {
        delegator_address: wallet.address,
        validator_address: delegation.delegation.validator_address
    }
}));

console.log('Claim rewards transactions:', messages);
return { transaction: messages, rewards };`
    },

    'governance': {
        title: 'Submit Proposal',
        description: 'Create a governance proposal',
        category: 'governance',
        language: 'javascript',
        code: `// Submit a governance proposal
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get governance parameters
const params = await api.getGovParams('deposit');
console.log('Governance params:', params);

// Create proposal message
const msg = {
    type: 'cosmos-sdk/MsgSubmitProposal',
    value: {
        content: {
            type: 'cosmos-sdk/TextProposal',
            value: {
                title: 'Example Proposal',
                description: 'This is an example governance proposal'
            }
        },
        initial_deposit: [{
            denom: 'upaw',
            amount: '10000000' // 10 PAW minimum deposit
        }],
        proposer: wallet.address
    }
};

console.log('Proposal transaction:', msg);
return { transaction: msg, params };`
    },

    'vote': {
        title: 'Vote on Proposal',
        description: 'Cast a vote on a governance proposal',
        category: 'governance',
        language: 'javascript',
        code: `// Vote on a governance proposal
if (!wallet.connected) {
    console.error('Please connect your wallet first');
    return;
}

// Get active proposals
const proposals = await api.getProposals('PROPOSAL_STATUS_VOTING_PERIOD');
console.log('Active proposals:', proposals);

if (proposals.proposals.length === 0) {
    console.log('No active proposals to vote on');
    return;
}

// Vote on first proposal
const proposalId = proposals.proposals[0].proposal_id;

// Create vote message
// Vote options: VOTE_OPTION_YES, VOTE_OPTION_NO, VOTE_OPTION_ABSTAIN, VOTE_OPTION_NO_WITH_VETO
const msg = {
    type: 'cosmos-sdk/MsgVote',
    value: {
        proposal_id: proposalId,
        voter: wallet.address,
        option: 'VOTE_OPTION_YES'
    }
};

console.log('Vote transaction:', msg);
return { transaction: msg, proposal: proposals.proposals[0] };`
    }
};
