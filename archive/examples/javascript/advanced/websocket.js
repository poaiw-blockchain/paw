#!/usr/bin/env node

/**
 * PAW Blockchain - WebSocket Subscription Example
 *
 * This example demonstrates how to subscribe to blockchain events via WebSocket.
 *
 * Usage:
 *   node websocket.js
 */

import WebSocket from 'ws';
import dotenv from 'dotenv';

dotenv.config();

const WS_ENDPOINT = (process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657')
  .replace('http://', 'ws://').replace('https://', 'wss://') + '/websocket';

async function subscribeToBlocks() {
  console.log('Subscribing to New Blocks...\n');
  console.log(`WebSocket: ${WS_ENDPOINT}\n`);

  const ws = new WebSocket(WS_ENDPOINT);

  ws.on('open', () => {
    console.log('✓ WebSocket connected\n');

    // Subscribe to new blocks
    const subscribeMsg = {
      jsonrpc: '2.0',
      method: 'subscribe',
      id: '1',
      params: {
        query: "tm.event='NewBlock'"
      }
    };

    ws.send(JSON.stringify(subscribeMsg));
    console.log('Subscribed to new blocks. Waiting for events...\n');
  });

  ws.on('message', (data) => {
    const message = JSON.parse(data.toString());

    if (message.result && message.result.data) {
      const blockData = message.result.data.value.block;
      const height = blockData.header.height;
      const numTxs = blockData.data.txs?.length || 0;
      const time = blockData.header.time;

      console.log(`New Block #${height}:`);
      console.log(`  Time: ${time}`);
      console.log(`  Transactions: ${numTxs}`);
      console.log('');
    }
  });

  ws.on('error', (error) => {
    console.error('WebSocket error:', error.message);
  });

  ws.on('close', () => {
    console.log('WebSocket disconnected');
  });

  // Handle cleanup
  process.on('SIGINT', () => {
    console.log('\nClosing WebSocket connection...');
    ws.close();
    process.exit(0);
  });

  return { success: true };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  subscribeToBlocks().catch(console.error);
}

export { subscribeToBlocks };
