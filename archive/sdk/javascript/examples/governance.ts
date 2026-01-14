/**
 * PAW SDK Governance Example
 *
 * This example demonstrates how to interact with the governance module
 * on the PAW testnet (paw-mvp-1).
 *
 * Note: Governance is how disabled modules (DEX, Compute, Oracle) can be enabled.
 *
 * Prerequisites:
 * - Set MNEMONIC environment variable with your wallet mnemonic
 * - Have testnet tokens (get from https://testnet-faucet.poaiw.org)
 */
import { PawClient, PawWallet, PAW_TESTNET_CONFIG, VoteOption } from '@paw-chain/sdk';

async function main() {
  // Setup wallet and client
  if (!process.env.MNEMONIC) {
    console.error('Please set MNEMONIC environment variable');
    process.exit(1);
  }

  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(process.env.MNEMONIC);
  const address = await wallet.getAddress();

  console.log('Wallet address:', address);
  console.log('');

  // Connect to PAW testnet
  const client = new PawClient(PAW_TESTNET_CONFIG);
  await client.connectWithWallet(wallet);

  // 1. Get governance parameters first
  console.log('--- Governance Parameters ---');

  const votingParams = await client.governance.getParams('voting');
  console.log('Voting period:', votingParams?.voting_period);

  const depositParams = await client.governance.getParams('deposit');
  console.log('Min deposit:', JSON.stringify(depositParams?.min_deposit));
  console.log('Max deposit period:', depositParams?.max_deposit_period);

  const tallyParams = await client.governance.getParams('tallying');
  console.log('Quorum:', tallyParams?.quorum);
  console.log('Threshold:', tallyParams?.threshold);
  console.log('Veto threshold:', tallyParams?.veto_threshold);
  console.log('');

  // 2. Get all proposals
  console.log('--- Proposals ---');
  const proposals = await client.governance.getProposals();
  console.log('Total proposals:', proposals.length);

  if (proposals.length === 0) {
    console.log('No proposals found.');
    console.log('');
    console.log('To enable DEX/Compute/Oracle modules, you can submit a governance proposal.');
    console.log('');
  }

  // 3. Get active proposals (status = 2 means voting period)
  const activeProposals = await client.governance.getProposals(2);
  console.log('Active proposals (voting period):', activeProposals.length);

  // Display active proposals
  if (activeProposals.length > 0) {
    console.log('\nActive Proposals:');
    for (const proposal of activeProposals) {
      console.log(`\nProposal #${proposal.proposal_id}`);
      console.log(`  Status: ${proposal.status}`);
      console.log(`  Voting ends: ${proposal.voting_end_time}`);

      // Get tally
      const tally = await client.governance.getTally(proposal.proposal_id);
      if (tally) {
        console.log('  Current tally:');
        console.log(`    Yes: ${tally.yes}`);
        console.log(`    No: ${tally.no}`);
        console.log(`    Abstain: ${tally.abstain}`);
        console.log(`    No with Veto: ${tally.no_with_veto}`);
      }
    }
  }
  console.log('');

  // Check balance before submitting proposals
  const balance = await client.bank.getBalance(address, 'upaw');
  console.log(`Your balance: ${balance?.amount || '0'} upaw`);

  if (!balance || parseInt(balance.amount) < 10000000) {
    console.log('\nInsufficient balance to submit proposals (need 10+ PAW).');
    console.log('Get testnet tokens at: https://testnet-faucet.poaiw.org');
    console.log('');
    console.log('Skipping proposal submission and voting demos.');
    await client.disconnect();
    return;
  }

  // 4. Submit a text proposal
  console.log('\n--- Submit Proposal ---');
  console.log('Submitting a text proposal...');

  try {
    const submitResult = await client.governance.submitTextProposal(
      address,
      'Enable DEX Module',
      'This proposal enables the DEX module on paw-mvp-1 testnet, allowing token swaps and liquidity pools.',
      '10000000', // 10 PAW initial deposit
      'upaw'
    );
    console.log('Proposal submitted!');
    console.log(`TX Hash: ${submitResult.transactionHash}`);
  } catch (error) {
    console.error('Failed to submit proposal:', error);
  }
  console.log('');

  // 5. Vote on a proposal (if any active)
  if (activeProposals.length > 0) {
    console.log('--- Vote on Proposal ---');
    const proposalId = activeProposals[0].proposal_id;

    try {
      const voteResult = await client.governance.vote(address, {
        proposalId,
        option: VoteOption.YES,
        metadata: 'Supporting this proposal'
      });
      console.log(`Voted YES on proposal #${proposalId}`);
      console.log(`TX Hash: ${voteResult.transactionHash}`);

      // Check your vote
      const myVote = await client.governance.getVote(proposalId, address);
      console.log('Your recorded vote:', client.governance.getVoteOptionName(myVote?.option));
    } catch (error) {
      console.error('Failed to vote:', error);
    }
    console.log('');
  }

  // 6. Deposit to a proposal in deposit period
  const depositProposals = await client.governance.getProposals(1); // status = 1 means deposit period
  if (depositProposals.length > 0) {
    console.log('--- Deposit to Proposal ---');
    const proposalId = depositProposals[0].proposal_id;

    try {
      const depositResult = await client.governance.deposit(address, {
        proposalId,
        amount: '1000000' // 1 PAW
      });
      console.log(`Deposited 1 PAW to proposal #${proposalId}`);
      console.log(`TX Hash: ${depositResult.transactionHash}`);

      // Get all deposits for the proposal
      const deposits = await client.governance.getDeposits(proposalId);
      console.log('Total deposits on proposal:', deposits.length);
    } catch (error) {
      console.error('Failed to deposit:', error);
    }
    console.log('');
  }

  // Summary
  console.log('--- Summary ---');
  console.log('Governance allows the community to:');
  console.log('1. Enable disabled modules (DEX, Compute, Oracle)');
  console.log('2. Change chain parameters');
  console.log('3. Upgrade the chain software');
  console.log('4. Fund community initiatives');
  console.log('');

  await client.disconnect();
  console.log('Disconnected from testnet');
}

main().catch(console.error);
