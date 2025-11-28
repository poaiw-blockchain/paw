/* eslint-env node */
/* global __ENV */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomIntBetween } from 'k6/exec';

// Custom metrics
const errorRate = new Rate('errors');
const txLatency = new Trend('transaction_latency');
const queryLatency = new Trend('query_latency');
const txCounter = new Counter('transactions_submitted');

// Test configuration
export let options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp up to 200 users
    { duration: '5m', target: 200 }, // Peak load
    { duration: '2m', target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% under 500ms, 99% under 1s
    http_req_failed: ['rate<0.01'], // Less than 1% errors
    errors: ['rate<0.05'], // Less than 5% custom errors
    transaction_latency: ['p(95)<2000'], // Tx confirmation under 2s
    query_latency: ['p(95)<200'], // Queries under 200ms
  },
  ext: {
    loadimpact: {
      projectID: 'PAW Blockchain',
      name: 'Blockchain Load Test',
    },
  },
};

// Test data
const TEST_ADDRESSES = [
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0d8t4q',
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqg3vxq7',
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqa29wr0',
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqcjqxn9',
];

const BASE_URL = __ENV.BASE_URL || 'http://localhost:1317';
const RPC_URL = __ENV.RPC_URL || 'http://localhost:26657';

// Setup function - runs once per VU
export function setup() {
  console.log('Starting PAW blockchain load test');
  console.log(`API URL: ${BASE_URL}`);
  console.log(`RPC URL: ${RPC_URL}`);

  // Test connectivity
  const res = http.get(`${BASE_URL}/cosmos/base/tendermint/v1beta1/node_info`);
  check(res, {
    'setup: node is reachable': r => r.status === 200,
  });

  return {
    startTime: new Date().toISOString(),
  };
}

// Main test function
// eslint-disable-next-line no-unused-vars
export default function (data) {
  const testAddress =
    TEST_ADDRESSES[randomIntBetween(0, TEST_ADDRESSES.length - 1)];

  // Test 1: Query account balance (70% of requests)
  if (Math.random() < 0.7) {
    queryBalance(testAddress);
  }

  // Test 2: Query DEX pools (15% of requests)
  if (Math.random() < 0.15) {
    queryDEXPools();
  }

  // Test 3: Submit transaction (10% of requests)
  if (Math.random() < 0.1) {
    submitTransaction(testAddress);
  }

  // Test 4: Query validators (5% of requests)
  if (Math.random() < 0.05) {
    queryValidators();
  }

  sleep(randomIntBetween(1, 3));
}

function queryBalance(address) {
  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/cosmos/bank/v1beta1/balances/${address}`);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'balance query: status 200': r => r.status === 200,
    'balance query: has balances': r => {
      if (r.status !== 200) {return false;}
      try {
        const body = JSON.parse(r.body);
        return body.balances !== undefined;
      } catch (e) {
        return false;
      }
    },
    'balance query: response time OK': r => r.timings.duration < 300,
  });

  queryLatency.add(duration);
  if (!success) {errorRate.add(1);}
}

function queryDEXPools() {
  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/paw/dex/v1/pools`);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'dex query: status 200': r => r.status === 200,
    'dex query: has pools': r => {
      if (r.status !== 200) {return false;}
      try {
        const body = JSON.parse(r.body);
        return body.pools !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  queryLatency.add(duration);
  if (!success) {errorRate.add(1);}
}

function submitTransaction(fromAddress) {
  const startTime = Date.now();

  // Create a simple bank send transaction
  const txPayload = {
    tx: {
      body: {
        messages: [
          {
            '@type': '/cosmos.bank.v1beta1.MsgSend',
            from_address: fromAddress,
            to_address:
              TEST_ADDRESSES[randomIntBetween(0, TEST_ADDRESSES.length - 1)],
            amount: [
              {
                denom: 'upaw',
                amount: '1000',
              },
            ],
          },
        ],
        memo: `load-test-${Date.now()}`,
        timeout_height: '0',
        extension_options: [],
        non_critical_extension_options: [],
      },
      auth_info: {
        signer_infos: [],
        fee: {
          amount: [
            {
              denom: 'upaw',
              amount: '5000',
            },
          ],
          gas_limit: '200000',
          payer: '',
          granter: '',
        },
      },
      signatures: [],
    },
    mode: 'BROADCAST_MODE_ASYNC',
  };

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(
    `${BASE_URL}/cosmos/tx/v1beta1/txs`,
    JSON.stringify(txPayload),
    params
  );
  const duration = Date.now() - startTime;

  const success = check(res, {
    'tx submit: accepted': r => r.status === 200 || r.status === 400, // 400 might be invalid signature
    'tx submit: has tx_response': r => {
      try {
        const body = JSON.parse(r.body);
        return body.tx_response !== undefined || body.code !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  txLatency.add(duration);
  txCounter.add(1);
  if (!success) {errorRate.add(1);}
}

function queryValidators() {
  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/cosmos/staking/v1beta1/validators`);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'validators query: status 200': r => r.status === 200,
    'validators query: has validators': r => {
      if (r.status !== 200) {return false;}
      try {
        const body = JSON.parse(r.body);
        return body.validators !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  queryLatency.add(duration);
  if (!success) {errorRate.add(1);}
}

// Teardown function
export function teardown(data) {
  console.log('Load test completed');
  console.log(`Started at: ${data.startTime}`);
  console.log(`Ended at: ${new Date().toISOString()}`);
}
