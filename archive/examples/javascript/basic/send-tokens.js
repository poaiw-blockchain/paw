#!/usr/bin/env node

/**
 * PAW Blockchain - Send Tokens Example
 *
 * This example demonstrates how to send tokens from one address to another.
 *
 * Usage:
 *   node send-tokens.js <recipient> <amount> [denom]
 *
 * Environment Variables:
 *   MNEMONIC - Sender wallet mnemonic
 *   PAW_RPC_ENDPOINT - RPC endpoint
 */

import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';
const WALLET_PREFIX = 'paw';
const HD_PATH = "m/44'/118'/0'/0/0";
const DEFAULT_DENOM = 'upaw';
const GAS_PRICE = process.env.GAS_PRICE || '0.025upaw';

/**
 * Send tokens to recipient
 * @param {string} mnemonic - Sender mnemonic
 * @param {string} recipient - Recipient address
 * @param {string} amount - Amount to send
 * @param {string} denom - Token denomination
 * @param {string} memo - Optional transaction memo
 * @returns {Promise<Object>} Transaction result
 */
async function sendTokens(mnemonic, recipient, amount, denom = DEFAULT_DENOM, memo = '') {
  console.log('Sending Tokens...\n');

  try {
    // Create wallet from mnemonic
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath(HD_PATH)]
    });

    const accounts = await wallet.getAccounts();
    const sender = accounts[0].address;

    console.log(`From: ${sender}`);
    console.log(`To: ${recipient}`);
    console.log(`Amount: ${formatAmount(amount, denom)}`);
    if (memo) console.log(`Memo: ${memo}`);
    console.log();

    // Connect to the network
    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    // Check sender balance before sending
    const balance = await client.getBalance(sender, denom);
    const balanceAmount = parseInt(balance.amount);
    const sendAmount = parseInt(amount);

    if (balanceAmount < sendAmount) {
      throw new Error(
        `Insufficient balance. Have ${balance.amount} ${denom}, need ${amount} ${denom}`
      );
    }

    console.log(`Current Balance: ${formatAmount(balance.amount, denom)}`);
    console.log(`After Transaction: ${formatAmount(balanceAmount - sendAmount, denom)}\n`);

    // Prepare the transaction
    const sendAmount_obj = {
      denom: denom,
      amount: amount
    };

    console.log('Broadcasting transaction...');

    // Send the tokens
    const result = await client.sendTokens(
      sender,
      recipient,
      [sendAmount_obj],
      'auto', // Auto-calculate gas
      memo
    );

    // Check transaction result
    if (result.code !== undefined && result.code !== 0) {
      throw new Error(`Transaction failed with code ${result.code}: ${result.rawLog}`);
    }

    console.log('\n✓ Transaction successful!\n');
    console.log('Transaction Details:');
    console.log(`  Transaction Hash: ${result.transactionHash}`);
    console.log(`  Block Height: ${result.height}`);
    console.log(`  Gas Used: ${result.gasUsed}`);
    console.log(`  Gas Wanted: ${result.gasWanted}`);

    // Calculate transaction fee
    if (result.events) {
      const feeEvent = result.events.find(e => e.type === 'tx');
      if (feeEvent) {
        const feeAttr = feeEvent.attributes.find(a => a.key === 'fee');
        if (feeAttr) {
          console.log(`  Fee: ${feeAttr.value}`);
        }
      }
    }

    client.disconnect();

    return {
      success: true,
      txHash: result.transactionHash,
      height: result.height,
      gasUsed: result.gasUsed
    };

  } catch (error) {
    console.error('\n✗ Error sending tokens:', error.message);

    // Provide helpful error messages
    if (error.message.includes('account does not exist')) {
      console.error('\nThe recipient account may not be initialized.');
      console.error('This is normal - the transaction will initialize it.');
    } else if (error.message.includes('insufficient')) {
      console.error('\nPlease ensure you have enough tokens for:');
      console.error('  1. The transfer amount');
      console.error('  2. Transaction fees');
    }

    return {
      success: false,
      error: error.message
    };
  }
}

/**
 * Format amount with denomination
 */
function formatAmount(amount, denom) {
  const amt = parseInt(amount);
  if (denom.startsWith('u')) {
    const base = denom.substring(1).toUpperCase();
    return `${(amt / 1_000_000).toFixed(6)} ${base}`;
  }
  return `${amt} ${denom}`;
}

/**
 * Simulate transaction to estimate gas
 */
async function simulateTransaction(mnemonic, recipient, amount, denom = DEFAULT_DENOM) {
  try {
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX
    });

    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    const accounts = await wallet.getAccounts();
    const sender = accounts[0].address;

    // Simulate without broadcasting
    const gasEstimate = await client.simulate(
      sender,
      [{
        typeUrl: '/cosmos.bank.v1beta1.MsgSend',
        value: {
          fromAddress: sender,
          toAddress: recipient,
          amount: [{ denom, amount }]
        }
      }],
      ''
    );

    console.log(`Estimated gas: ${gasEstimate}`);
    client.disconnect();

    return { success: true, gasEstimate };

  } catch (error) {
    console.error('Simulation error:', error.message);
    return { success: false, error: error.message };
  }
}

// Run the example if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);

  if (args.length < 2) {
    console.error('Usage: node send-tokens.js <recipient> <amount> [denom] [memo]');
    console.log('\nExample:');
    console.log('  node send-tokens.js paw1abc...xyz 1000000 upaw "Hello PAW"');
    process.exit(1);
  }

  const mnemonic = process.env.MNEMONIC;
  if (!mnemonic) {
    console.error('Error: MNEMONIC not set in environment');
    process.exit(1);
  }

  const recipient = args[0];
  const amount = args[1];
  const denom = args[2] || DEFAULT_DENOM;
  const memo = args[3] || '';

  sendTokens(mnemonic, recipient, amount, denom, memo)
    .then(result => {
      if (!result.success) {
        process.exit(1);
      }
    });
}

export { sendTokens, simulateTransaction, formatAmount };
