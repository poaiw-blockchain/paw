/* eslint-env node */
/* global __ENV, __VU */
import ws from 'k6/ws';
import { check } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';
import { randomIntBetween } from 'k6/exec';

// Custom metrics for WebSocket testing
const wsErrorRate = new Rate('ws_errors');
const wsConnections = new Counter('ws_connections');
const wsMessages = new Counter('ws_messages');
const wsLatency = new Trend('ws_message_latency');
const connectionDuration = new Trend('ws_connection_duration');

export let options = {
  stages: [
    { duration: '30s', target: 50 }, // Ramp up connections
    { duration: '2m', target: 50 }, // Steady WebSocket load
    { duration: '30s', target: 100 }, // Increase connections
    { duration: '2m', target: 100 }, // Peak WebSocket load
    { duration: '30s', target: 0 }, // Disconnect all
  ],
  thresholds: {
    ws_errors: ['rate<0.05'],
    ws_message_latency: ['p(95)<100'],
  },
};

const WS_URL = __ENV.WS_URL || 'ws://localhost:26657/websocket';

export function setup() {
  console.log('Starting PAW WebSocket load test');
  console.log(`WebSocket URL: ${WS_URL}`);
}

export default function () {
  const connectionStart = Date.now();
  const url = WS_URL;

  const params = {
    tags: { test_type: 'tendermint_websocket' },
  };

  const res = ws.connect(url, params, function (socket) {
    wsConnections.add(1);

    socket.on('open', function () {
      console.log(`VU ${__VU}: Connected to ${url}`);

      // Subscribe to new blocks
      subscribeToBlocks(socket);

      // Subscribe to new transactions
      subscribeToTransactions(socket);

      // Keep connection alive with periodic pings
      socket.setInterval(function () {
        socket.ping();
      }, 10000); // Ping every 10 seconds
    });

    socket.on('message', function (msg) {
      wsMessages.add(1);
      const latency = Date.now() - connectionStart;
      wsLatency.add(latency);

      try {
        const data = JSON.parse(msg);

        const success = check(data, {
          'ws: valid message': d => d !== undefined,
          'ws: has result or error': d =>
            d.result !== undefined || d.error !== undefined,
        });

        if (!success) {
          wsErrorRate.add(1);
        }

        // Log interesting events
        if (data.result && data.result.data) {
          if (data.result.data.type === 'tendermint/event/NewBlock') {
            console.log(`VU ${__VU}: New block received`);
          } else if (data.result.data.type === 'tendermint/event/Tx') {
            console.log(`VU ${__VU}: New transaction received`);
          }
        }
      } catch (e) {
        console.error(`VU ${__VU}: Failed to parse message: ${e}`);
        wsErrorRate.add(1);
      }
    });

    socket.on('ping', function () {
      console.log(`VU ${__VU}: Received ping`);
      socket.pong();
    });

    socket.on('pong', function () {
      console.log(`VU ${__VU}: Received pong`);
    });

    socket.on('close', function () {
      const duration = Date.now() - connectionStart;
      connectionDuration.add(duration);
      console.log(`VU ${__VU}: Disconnected after ${duration}ms`);
    });

    socket.on('error', function (e) {
      console.error(`VU ${__VU}: WebSocket error: ${e.error()}`);
      wsErrorRate.add(1);
    });

    // Keep connection open for random duration
    const connectionTime = randomIntBetween(30, 90);
    socket.setTimeout(function () {
      console.log(`VU ${__VU}: Closing connection after ${connectionTime}s`);
      socket.close();
    }, connectionTime * 1000);
  });

  check(res, {
    'ws: connection established': r => r && r.status === 101,
  });

  if (!res || res.status !== 101) {
    wsErrorRate.add(1);
  }
}

function subscribeToBlocks(socket) {
  const subscribeMsg = JSON.stringify({
    jsonrpc: '2.0',
    method: 'subscribe',
    id: `${__VU}-blocks`,
    params: {
      query: 'tm.event=\'NewBlock\'',
    },
  });

  socket.send(subscribeMsg);
  console.log(`VU ${__VU}: Subscribed to new blocks`);
}

function subscribeToTransactions(socket) {
  const subscribeMsg = JSON.stringify({
    jsonrpc: '2.0',
    method: 'subscribe',
    id: `${__VU}-txs`,
    params: {
      query: 'tm.event=\'Tx\'',
    },
  });

  socket.send(subscribeMsg);
  console.log(`VU ${__VU}: Subscribed to transactions`);
}

// Test query subscription
// eslint-disable-next-line no-unused-vars
function subscribeToQuery(socket, query) {
  const subscribeMsg = JSON.stringify({
    jsonrpc: '2.0',
    method: 'subscribe',
    id: `${__VU}-${Date.now()}`,
    params: { query },
  });

  socket.send(subscribeMsg);
}

// eslint-disable-next-line no-unused-vars
export function teardown(data) {
  console.log('WebSocket load test completed');
  console.log(`Total connections: ${wsConnections.value}`);
  console.log(`Total messages: ${wsMessages.value}`);
}
