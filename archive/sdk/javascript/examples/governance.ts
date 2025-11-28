import { PawClient, PawWallet, VoteOption } from '@paw-chain/sdk';

async function main() {
  // Setup wallet and client
  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(process.env.MNEMONIC!);
  const address = await wallet.getAddress();

  const client = new PawClient({
    rpcEndpoint: process.env.RPC_ENDPOINT || 'http://localhost:26657',
    restEndpoint: process.env.REST_ENDPOINT || 'http://localhost:1317',
    chainId: process.env.CHAIN_ID || 'paw-testnet-1'
  });

  await client.connectWithWallet(wallet);

  // 1. Get all proposals
  const proposals = await client.governance.getProposals();
  console.log('Total proposals:', proposals.length);

  // 2. Get active proposals (status = 2 means voting period)
  const activeProposals = await client.governance.getProposals(2);
  console.log('Active proposals:', activeProposals.length);

  // Display active proposals
  if (activeProposals.length > 0) {
    console.log('\nActive Proposals:');
    for (const proposal of activeProposals) {
      console.log(`\nProposal #${proposal.proposalId}`);
      console.log(`Status: ${proposal.status}`);
      console.log(`Voting ends: ${proposal.votingEndTime}`);

      // Get tally
      const tally = await client.governance.getTally(proposal.proposalId);
      if (tally) {
        console.log('Current tally:');
        console.log(`- Yes: ${tally.yes}`);
        console.log(`- No: ${tally.no}`);
        console.log(`- Abstain: ${tally.abstain}`);
        console.log(`- No with Veto: ${tally.no_with_veto}`);
      }
    }
  }

  // 3. Submit a text proposal
  const submitResult = await client.governance.submitTextProposal(
    address,
    'Improve Network Performance',
    'This proposal suggests optimizations to improve network throughput and reduce latency.',
    '10000000', // 10 PAW initial deposit
    'upaw'
  );
  console.log('\nProposal submitted! TX:', submitResult.transactionHash);

  // 4. Vote on a proposal
  if (activeProposals.length > 0) {
    const proposalId = activeProposals[0].proposalId;

    const voteResult = await client.governance.vote(address, {
      proposalId,
      option: VoteOption.YES,
      metadata: 'I support this proposal'
    });
    console.log('Vote submitted! TX:', voteResult.transactionHash);

    // Check your vote
    const myVote = await client.governance.getVote(proposalId, address);
    console.log('Your vote:', client.governance.getVoteOptionName(myVote?.option));
  }

  // 5. Deposit to a proposal in deposit period
  const depositProposals = await client.governance.getProposals(1); // status = 1 means deposit period
  if (depositProposals.length > 0) {
    const proposalId = depositProposals[0].proposalId;

    const depositResult = await client.governance.deposit(address, {
      proposalId,
      amount: '1000000' // 1 PAW
    });
    console.log('Deposit added! TX:', depositResult.transactionHash);

    // Get all deposits for the proposal
    const deposits = await client.governance.getDeposits(proposalId);
    console.log('Total deposits:', deposits.length);
  }

  // 6. Get governance parameters
  const votingParams = await client.governance.getParams('voting');
  console.log('\nVoting parameters:');
  console.log('- Voting period:', votingParams?.voting_period);

  const depositParams = await client.governance.getParams('deposit');
  console.log('\nDeposit parameters:');
  console.log('- Min deposit:', depositParams?.min_deposit);
  console.log('- Max deposit period:', depositParams?.max_deposit_period);

  const tallyParams = await client.governance.getParams('tallying');
  console.log('\nTally parameters:');
  console.log('- Quorum:', tallyParams?.quorum);
  console.log('- Threshold:', tallyParams?.threshold);
  console.log('- Veto threshold:', tallyParams?.veto_threshold);

  await client.disconnect();
}

main().catch(console.error);
