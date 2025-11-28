/* eslint-env node */
/* global __ENV */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomIntBetween } from 'k6/exec';

// Custom metrics for DEX operations
const swapErrorRate = new Rate('swap_errors');
const poolQueryLatency = new Trend('pool_query_latency');
const swapLatency = new Trend('swap_latency');
const liquidityLatency = new Trend('liquidity_latency');
const swapCounter = new Counter('swaps_submitted');
const liquidityCounter = new Counter('liquidity_operations');

export let options = {
  stages: [
    { duration: '1m', target: 50 }, // Ramp up
    { duration: '3m', target: 50 }, // Steady DEX load
    { duration: '1m', target: 100 }, // Peak DEX activity
    { duration: '3m', target: 100 }, // Peak steady
    { duration: '1m', target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<800'],
    http_req_failed: ['rate<0.02'],
    swap_errors: ['rate<0.03'],
    pool_query_latency: ['p(95)<300'],
    swap_latency: ['p(95)<3000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:1317';

// Test token pairs
const TOKEN_PAIRS = [
  { tokenA: 'upaw', tokenB: 'uatom' },
  { tokenA: 'upaw', tokenB: 'uosmo' },
  { tokenA: 'uatom', tokenB: 'uosmo' },
  { tokenA: 'upaw', tokenB: 'uusdc' },
];

const TEST_ADDRESSES = [
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0d8t4q',
  'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqg3vxq7',
];

export function setup() {
  console.log('Starting PAW DEX load test');
  console.log(`Testing ${TOKEN_PAIRS.length} token pairs`);

  // Verify DEX module is available
  const res = http.get(`${BASE_URL}/paw/dex/v1/params`);
  check(res, {
    'setup: DEX module available': r => r.status === 200,
  });

  return { startTime: new Date().toISOString() };
}

// eslint-disable-next-line no-unused-vars
export default function (data) {
  // Test distribution:
  // 60% - Query pools
  // 25% - Simulate swaps
  // 10% - Query pool liquidity
  // 5% - Query pool prices

  const rand = Math.random();

  if (rand < 0.6) {
    queryAllPools();
  } else if (rand < 0.85) {
    simulateSwap();
  } else if (rand < 0.95) {
    queryPoolLiquidity();
  } else {
    queryPoolPrices();
  }

  sleep(randomIntBetween(1, 2));
}

function queryAllPools() {
  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/paw/dex/v1/pools`);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'query pools: status 200': r => r.status === 200,
    'query pools: has data': r => {
      try {
        const body = JSON.parse(r.body);
        return body.pools !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  poolQueryLatency.add(duration);
  if (!success) {swapErrorRate.add(1);}
}

function queryPoolLiquidity() {
  // Random pool ID between 1-10
  const poolId = randomIntBetween(1, 10);
  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/paw/dex/v1/pools/${poolId}`);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'query pool: status ok': r => r.status === 200 || r.status === 404,
    'query pool: valid response': r => {
      if (r.status === 404) {return true;} // Pool doesn't exist is OK
      try {
        const body = JSON.parse(r.body);
        return body.pool !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  liquidityLatency.add(duration);
  if (!success) {swapErrorRate.add(1);}
}

function queryPoolPrices() {
  const pair = TOKEN_PAIRS[randomIntBetween(0, TOKEN_PAIRS.length - 1)];
  const startTime = Date.now();

  const res = http.get(
    `${BASE_URL}/paw/dex/v1/spot-price?token_in=${pair.tokenA}&token_out=${pair.tokenB}`
  );
  const duration = Date.now() - startTime;

  const success = check(res, {
    'query price: status ok': r => r.status === 200 || r.status === 404,
  });

  poolQueryLatency.add(duration);
  if (!success) {swapErrorRate.add(1);}
}

function simulateSwap() {
  const startTime = Date.now();
  const pair = TOKEN_PAIRS[randomIntBetween(0, TOKEN_PAIRS.length - 1)];
  const sender = TEST_ADDRESSES[randomIntBetween(0, TEST_ADDRESSES.length - 1)];

  // Create swap transaction
  const swapPayload = {
    tx: {
      body: {
        messages: [
          {
            '@type': '/paw.dex.v1.MsgSwapExactAmountIn',
            sender: sender,
            routes: [
              {
                pool_id: '1',
                token_out_denom: pair.tokenB,
              },
            ],
            token_in: {
              denom: pair.tokenA,
              amount: `${randomIntBetween(1000, 100000)}`,
            },
            token_out_min_amount: '1',
          },
        ],
        memo: `dex-load-test-${Date.now()}`,
        timeout_height: '0',
      },
      auth_info: {
        signer_infos: [],
        fee: {
          amount: [
            {
              denom: 'upaw',
              amount: '10000',
            },
          ],
          gas_limit: '300000',
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
    JSON.stringify(swapPayload),
    params
  );
  const duration = Date.now() - startTime;

  const success = check(res, {
    'swap: accepted': r => r.status === 200 || r.status === 400,
    'swap: has response': r => {
      try {
        const body = JSON.parse(r.body);
        return body.tx_response !== undefined || body.code !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  swapLatency.add(duration);
  swapCounter.add(1);
  if (!success) {swapErrorRate.add(1);}
}

// Test adding liquidity
export function addLiquidity() {
  const startTime = Date.now();
  const sender = TEST_ADDRESSES[randomIntBetween(0, TEST_ADDRESSES.length - 1)];

  const liquidityPayload = {
    tx: {
      body: {
        messages: [
          {
            '@type': '/paw.dex.v1.MsgJoinPool',
            sender: sender,
            pool_id: '1',
            share_out_amount: '1000000',
            token_in_maxs: [
              { denom: 'upaw', amount: '1000000' },
              { denom: 'uatom', amount: '1000000' },
            ],
          },
        ],
        memo: `liquidity-test-${Date.now()}`,
      },
      auth_info: {
        fee: {
          amount: [{ denom: 'upaw', amount: '15000' }],
          gas_limit: '400000',
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
    JSON.stringify(liquidityPayload),
    params
  );
  const duration = Date.now() - startTime;

  check(res, {
    'add liquidity: accepted': r => r.status === 200 || r.status === 400,
  });

  liquidityLatency.add(duration);
  liquidityCounter.add(1);
}

// eslint-disable-next-line no-unused-vars
export function teardown(data) {
  console.log('DEX load test completed');
  console.log(`Total swaps simulated: ${swapCounter.value}`);
  console.log(`Total liquidity operations: ${liquidityCounter.value}`);
}
