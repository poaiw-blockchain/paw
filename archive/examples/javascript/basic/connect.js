#!/usr/bin/env node

/**
 * PAW Blockchain - Connect to Network Example
 *
 * This example demonstrates how to connect to the PAW blockchain network
 * and retrieve basic network information.
 *
 * Usage:
 *   node connect.js
 *
 * Environment Variables:
 *   PAW_RPC_ENDPOINT - RPC endpoint URL (default: http://localhost:26657)
 */

import { StargateClient } from '@cosmjs/stargate';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

// Configuration
const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';

/**
 * Connect to PAW network and display network information
 */
async function connectToNetwork() {
  console.log('Connecting to PAW Network...');
  console.log(`RPC Endpoint: ${RPC_ENDPOINT}\n`);

  try {
    // Create a client connection to the blockchain
    const client = await StargateClient.connect(RPC_ENDPOINT);
    console.log('✓ Successfully connected to PAW network\n');

    // Get chain ID
    const chainId = await client.getChainId();
    console.log(`Chain ID: ${chainId}`);

    // Get current block height
    const height = await client.getHeight();
    console.log(`Current Block Height: ${height}`);

    // Get block information
    const block = await client.getBlock(height);
    console.log(`\nLatest Block Info:`);
    console.log(`  Block Hash: ${block.id}`);
    console.log(`  Time: ${block.header.time}`);
    console.log(`  Num Transactions: ${block.txs.length}`);
    console.log(`  Proposer: ${block.header.proposerAddress || 'N/A'}`);

    // Get a few previous blocks to calculate block time
    if (height > 5) {
      const previousBlock = await client.getBlock(height - 5);
      const timeDiff = new Date(block.header.time).getTime() -
                       new Date(previousBlock.header.time).getTime();
      const avgBlockTime = timeDiff / 5000; // Convert to seconds
      console.log(`  Average Block Time: ${avgBlockTime.toFixed(2)}s`);
    }

    // Disconnect from the client
    client.disconnect();
    console.log('\n✓ Disconnected from network');

    return {
      success: true,
      chainId,
      height,
      blockHash: block.id
    };

  } catch (error) {
    console.error('✗ Error connecting to network:', error.message);

    // Provide helpful error messages
    if (error.message.includes('ECONNREFUSED')) {
      console.error('\nTroubleshooting:');
      console.error('  1. Check if the RPC endpoint is correct');
      console.error('  2. Ensure the node is running');
      console.error('  3. Check firewall settings');
    }

    return {
      success: false,
      error: error.message
    };
  }
}

// Run the example if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  connectToNetwork()
    .then(result => {
      if (!result.success) {
        process.exit(1);
      }
    })
    .catch(error => {
      console.error('Unexpected error:', error);
      process.exit(1);
    });
}

// Export for testing
export { connectToNetwork };
