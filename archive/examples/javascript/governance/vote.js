#!/usr/bin/env node

/**
 * PAW Blockchain - Vote on Proposal Example
 *
 * This example demonstrates how to vote on a governance proposal.
 *
 * Usage:
 *   node vote.js <proposal_id> <vote_option>
 *
 * Vote Options: yes, no, abstain, no_with_veto
 */

import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';
const WALLET_PREFIX = 'paw';
const GAS_PRICE = '0.025upaw';

const VOTE_OPTIONS = {
  'yes': 1,
  'no': 3,
  'abstain': 2,
  'no_with_veto': 4
};

async function vote(mnemonic, proposalId, voteOption) {
  console.log('Voting on Governance Proposal...\n');

  try {
    // Validate vote option
    const normalizedOption = voteOption.toLowerCase();
    if (!VOTE_OPTIONS[normalizedOption]) {
      throw new Error(`Invalid vote option. Use: ${Object.keys(VOTE_OPTIONS).join(', ')}`);
    }

    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath("m/44'/118'/0'/0/0")]
    });

    const accounts = await wallet.getAccounts();
    const voter = accounts[0].address;

    console.log(`Voter: ${voter}`);
    console.log(`Proposal ID: ${proposalId}`);
    console.log(`Vote: ${normalizedOption.toUpperCase()}\n`);

    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    // Create vote message
    const voteMsg = {
      typeUrl: '/cosmos.gov.v1beta1.MsgVote',
      value: {
        proposalId: proposalId,
        voter: voter,
        option: VOTE_OPTIONS[normalizedOption]
      }
    };

    console.log('Broadcasting vote...');

    const result = await client.signAndBroadcast(
      voter,
      [voteMsg],
      'auto',
      `Voting ${normalizedOption} on proposal ${proposalId}`
    );

    if (result.code !== 0) {
      throw new Error(`Vote failed: ${result.rawLog}`);
    }

    console.log('\n✓ Vote recorded successfully!\n');
    console.log(`Tx Hash: ${result.transactionHash}`);
    console.log(`Height: ${result.height}`);
    console.log(`Gas Used: ${result.gasUsed}\n`);

    console.log('Your vote has been counted towards the proposal tally.');

    client.disconnect();

    return {
      success: true,
      txHash: result.transactionHash
    };

  } catch (error) {
    console.error('\n✗ Error:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  if (args.length < 2) {
    console.error('Usage: node vote.js <proposal_id> <vote_option>');
    console.log('\nVote Options: yes, no, abstain, no_with_veto');
    console.log('\nExample:');
    console.log('  node vote.js 1 yes');
    process.exit(1);
  }

  const mnemonic = process.env.MNEMONIC;
  if (!mnemonic) {
    console.error('Error: MNEMONIC not set');
    process.exit(1);
  }

  vote(mnemonic, args[0], args[1]).then(r => {
    if (!r.success) process.exit(1);
  });
}

export { vote, VOTE_OPTIONS };
