#!/usr/bin/env node

/**
 * PAW Blockchain - Query Balance Example
 *
 * This example demonstrates how to query account balances.
 *
 * Usage:
 *   node query-balance.js <address>
 *   node query-balance.js  # Uses address from .env
 */

import { StargateClient } from '@cosmjs/stargate';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';

/**
 * Query account balance
 * @param {string} address - Account address
 * @returns {Promise<Object>} Balance information
 */
async function queryBalance(address) {
  console.log('Querying Account Balance...\n');
  console.log(`Address: ${address}`);
  console.log(`RPC: ${RPC_ENDPOINT}\n`);

  try {
    // Connect to the network
    const client = await StargateClient.connect(RPC_ENDPOINT);

    // Get all balances for the account
    const balances = await client.getAllBalances(address);

    if (balances.length === 0) {
      console.log('✓ Account exists but has no tokens');
      console.log('\nTo fund this account:');
      console.log('  - Use the testnet faucet');
      console.log('  - Request tokens from another address');
    } else {
      console.log('✓ Balances retrieved successfully:\n');

      // Display each token balance
      balances.forEach((balance, index) => {
        console.log(`${index + 1}. ${formatAmount(balance.amount, balance.denom)}`);
      });

      // Calculate total value (if we had price data)
      console.log('\nTotal Balances:');
      const totalTypes = balances.length;
      const totalAmount = balances.reduce((sum, b) =>
        sum + parseInt(b.amount), 0);
      console.log(`  Token Types: ${totalTypes}`);
      console.log(`  Total Units: ${totalAmount.toLocaleString()}`);
    }

    // Get account information
    try {
      const account = await client.getAccount(address);
      if (account) {
        console.log('\nAccount Information:');
        console.log(`  Account Number: ${account.accountNumber}`);
        console.log(`  Sequence: ${account.sequence}`);
        console.log(`  Type: ${account['@type'] || 'base'}`);
      }
    } catch (error) {
      console.log('\n✓ Account not yet initialized (no transactions)');
    }

    client.disconnect();

    return {
      success: true,
      address,
      balances
    };

  } catch (error) {
    console.error('✗ Error querying balance:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

/**
 * Format token amount with denomination
 * @param {string} amount - Raw amount
 * @param {string} denom - Token denomination
 * @returns {string} Formatted string
 */
function formatAmount(amount, denom) {
  const amt = parseInt(amount);

  // Convert micro units to base units for known denominations
  if (denom.startsWith('u')) {
    const baseDenom = denom.substring(1).toUpperCase();
    const baseAmount = amt / 1_000_000;
    return `${baseAmount.toLocaleString()} ${baseDenom} (${amt.toLocaleString()} ${denom})`;
  }

  return `${amt.toLocaleString()} ${denom}`;
}

/**
 * Query specific denomination balance
 * @param {string} address - Account address
 * @param {string} denom - Token denomination
 * @returns {Promise<Object>} Balance information
 */
async function queryDenomBalance(address, denom) {
  try {
    const client = await StargateClient.connect(RPC_ENDPOINT);
    const balance = await client.getBalance(address, denom);
    client.disconnect();

    console.log(`Balance of ${denom}:`);
    console.log(formatAmount(balance.amount, balance.denom));

    return {
      success: true,
      balance
    };

  } catch (error) {
    console.error('✗ Error:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

// Run the example if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  let address = args[0] || process.env.PAW_ADDRESS;

  if (!address) {
    console.error('Error: No address provided');
    console.log('Usage: node query-balance.js <address>');
    console.log('   or: Set PAW_ADDRESS in .env file');
    process.exit(1);
  }

  const denom = args[1]; // Optional specific denomination

  const queryFunc = denom ?
    queryDenomBalance(address, denom) :
    queryBalance(address);

  queryFunc.then(result => {
    if (!result.success) {
      process.exit(1);
    }
  });
}

export { queryBalance, queryDenomBalance, formatAmount };
